package app

import (
	"cmp"
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"slices"
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/lyrics"
	"yoho-music/internal/music"
	"yoho-music/internal/speech"
)

type captionSpan struct{ From, Until time.Duration }

type episodeTranscript struct {
	Model    string
	Segments []speech.Segment
	Spans    []captionSpan
}

type captionView struct {
	key                      string
	lines                    []lyrics.Line
	transcript               episodeTranscript
	problem                  string
	loaded, active, complete bool
}

type captionModelState struct {
	done, total        int64
	loading, installed bool
	problem            string
}

func captionKey(track *music.Track) string {
	if track == nil || track.ShowID == "" {
		return ""
	}
	return track.Source + "\x00" + track.ID
}

func (m *model) transcriptPath(key string) string {
	return filepath.Join(m.dir, "transcripts", fmt.Sprintf("%x.json", sha256.Sum256([]byte(key))))
}

func (m *model) loadEpisodeTranscript(track *music.Track) {
	key := captionKey(track)
	if m.captions.GetUntracked().key == key {
		return
	}
	m.stopCaptions()
	m.captionLoadVersion++
	loading := m.captionLoadVersion
	state := captionView{key: key}
	m.captions.Set(state)
	if key == "" || m.dir == "" {
		return
	}
	path := m.transcriptPath(key)
	go func() {
		m.captionWorker.Lock()
		defer m.captionWorker.Unlock()
		cached := episodeTranscript{Model: speech.ModelName}
		if !readJSON(path, &cached) || cached.Model != speech.ModelName {
			cached = episodeTranscript{Model: speech.ModelName}
		}
		m.cx.Post(func() {
			if m.captionLoadVersion != loading {
				return
			}
			state.loaded, state.transcript = true, cached
			state.lines = speech.Lines(cached.Segments)
			m.captions.Set(state)
		})
	}()
}

func (m *model) stopCaptions() {
	if m.captionCancel != nil {
		m.captionCancel()
		m.captionCancel = nil
	}
	m.captionGeneration++
	state := m.captions.GetUntracked()
	if state.active {
		state.active = false
		m.captions.Set(state)
	}
}

func (m *model) seek(position time.Duration) {
	if state := m.captions.GetUntracked(); state.active {
		if distance := position - m.player.Position(); distance > 3*time.Second || distance < -3*time.Second {
			m.stopCaptions()
		}
	}
	m.player.Seek(position)
	m.refreshPlayback()
}

func (m *model) requestCaptions(cx *core.Scope) {
	if !speech.Available() || m.dir == "" {
		return
	}
	root := filepath.Join(m.dir, "speech")
	if speech.Installed(root) {
		m.captionModel.Set(captionModelState{installed: true})
		m.enableCaptions(true)
		return
	}
	if m.captionModel.GetUntracked().loading {
		return
	}
	detail := fmt.Sprintf(i18n.T("Download %.1f GB for on-device transcription."), float64(speech.ModelSize())/1e9)
	confirm(cx, i18n.T("Download Caption Model?"), detail, func() {
		if m.captionModel.GetUntracked().loading {
			return
		}
		ctx, cancel := context.WithCancel(m.context)
		m.captionModelStop = cancel
		total := speech.ModelSize()
		m.captionModel.Set(captionModelState{loading: true, total: total})
		go func() {
			var shown int64
			err := speech.Download(ctx, root, func(done, total int64) {
				if done-shown < 1<<20 && done < total {
					return
				}
				shown = done
				m.cx.Post(func() { m.captionModel.Set(captionModelState{loading: true, done: done, total: total}) })
			})
			m.cx.Post(func() {
				m.captionModelStop = nil
				state := captionModelState{installed: err == nil}
				if err != nil && ctx.Err() == nil {
					state.problem = err.Error()
				}
				m.captionModel.Set(state)
				if err == nil {
					m.enableCaptions(true)
				}
			})
		}()
	})
}

func (m *model) enableCaptions(on bool) {
	if m.prefs.GetUntracked().Captions != on {
		m.updatePreferences(m.cx, func(p *preferences) { p.Captions = on })
	}
	if on {
		state := m.captions.GetUntracked()
		state.problem = ""
		m.captions.Set(state)
	} else {
		m.stopCaptions()
	}
}

func (m *model) captionActions(cx *core.Scope) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		download := m.captionModel.Get()
		caption := m.captions.Get()
		p := colors(cx)
		if download.loading {
			detail := fmt.Sprintf("%s %d%%", i18n.T("Downloading model…"), download.done*100/max(download.total, 1))
			stop := button(i18n.T("Cancel"), "cancel-caption-download", lucide.X, func() {
				if m.captionModelStop != nil {
					m.captionModelStop()
				}
			})
			return row(label(detail, 13, core.FontWeightNormal, p.muted), spacer(), stop)
		}
		on := m.prefs.Get().Captions && (download.installed || m.dir != "" && speech.Installed(filepath.Join(m.dir, "speech")))
		text, id, action := i18n.T("Generate Captions"), "start-captions", func() { m.requestCaptions(cx) }
		if on && caption.problem == "" {
			text, id, action = i18n.T("Turn Off Captions"), "stop-captions", func() { m.enableCaptions(false) }
		}
		if on && caption.problem != "" {
			text = i18n.T("Retry")
		}
		control := button(text, id, lucide.Mic, action)
		problem := caption.problem
		if download.problem != "" {
			problem = download.problem
		}
		if problem != "" {
			return core.View{Style: core.Style{Gap: 6}, Children: []core.Node{label(problem, 13, core.FontWeightNormal, p.accent), row(control)}}
		}
		return row(control)
	})
}

func (m *model) captionPane(cx *core.Scope) core.Node {
	if !speech.Available() {
		return m.empty(cx, lucide.Mic, i18n.T("Captions are unavailable on this Mac."), "", nil)
	}
	content := core.Dynamic(func(cx *core.Scope) core.Node {
		state := m.captions.Get()
		if len(state.lines) > 0 {
			return m.lyricList(cx, state.lines)
		}
		message := ""
		if state.active {
			message = i18n.T("Transcribing…")
		}
		return m.empty(cx, lucide.Mic, i18n.T("No Captions"), message, nil)
	})
	return core.View{Style: fill, Children: []core.Node{m.captionActions(cx), content}}
}

func (m *model) captionsSettings(cx *core.Scope) core.Node {
	if !speech.Available() {
		return nil
	}
	name := i18n.T("Podcast Captions")
	toggle := ui.Switch{Label: name, TestID: "podcast-captions",
		Checked:  func() bool { return m.prefs.Get().Captions },
		Disabled: func() bool { return m.captionModel.Get().loading },
		OnChange: func(on bool) {
			if on {
				m.requestCaptions(cx)
			} else {
				m.enableCaptions(false)
			}
		}}
	return core.View{Children: []core.Node{settingsRow(cx, name, toggle),
		core.Show{When: func() bool { model := m.captionModel.Get(); return model.loading || model.problem != "" },
			Then: m.captionActions}}}
}

func (m *model) syncCaptions() {
	state, playback := m.captions.Get(), m.playing.Get()
	if !m.prefs.Get().Captions || !playback.playing || captionKey(playback.track) == "" {
		if state.active {
			m.stopCaptions()
		}
		return
	}
	if state.active || !state.loaded || state.complete || state.problem != "" || playback.loading || m.dir == "" {
		return
	}
	src := m.sourceByID(playback.track.Source)
	if src != nil && src.Info().Playback == music.PlaybackRemote {
		return
	}
	root := filepath.Join(m.dir, "speech")
	if !m.captionModel.Get().installed && !speech.Installed(root) {
		return
	}
	from, until, ok := captionGap(state.transcript.Spans, m.player.Position(), playback.duration)
	if !ok {
		state.complete = true
		m.captions.Set(state)
		return
	}
	audio, err := m.player.OpenAudio()
	if err != nil {
		state.problem = err.Error()
		m.captions.Set(state)
		return
	}
	ctx, cancel := context.WithCancel(m.context)
	m.captionCancel = cancel
	m.captionGeneration++
	generation := m.captionGeneration
	state.active = true
	m.captions.Set(state)
	path := m.transcriptPath(state.key)
	go func() {
		defer audio.Close()
		m.captionWorker.Lock()
		defer m.captionWorker.Unlock()
		transcript := state.transcript
		transcript.Segments = slices.Clone(transcript.Segments)
		transcript.Spans = slices.Clone(transcript.Spans)
		// An interrupted worker may have saved more since this attempt began.
		var cached episodeTranscript
		if readJSON(path, &cached) && cached.Model == speech.ModelName {
			transcript = cached
		}
		lastSaved := from
		publish := func() {
			snapshot := transcript
			snapshot.Segments = slices.Clone(transcript.Segments)
			snapshot.Spans = slices.Clone(transcript.Spans)
			lines := speech.Lines(snapshot.Segments)
			m.cx.Post(func() {
				if m.captionGeneration == generation {
					active := m.captions.GetUntracked()
					active.transcript, active.lines = snapshot, lines
					m.captions.Set(active)
				}
			})
		}
		run := func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			recognizer, err := speech.NewRecognizer(root)
			if err != nil {
				return err
			}
			defer recognizer.Close()
			frame := int64(from.Seconds() * float64(audio.SampleRate()))
			if err := audio.SetPosition(frame); err != nil {
				return err
			}
			return recognizer.Transcribe(ctx, audio, from, until, func(segment speech.Segment) {
				transcript.Segments = addCaptionSegment(transcript.Segments, segment)
				publish()
			}, func(heard time.Duration) {
				if heard > from {
					transcript.Spans = addCaptionSpan(transcript.Spans, captionSpan{From: from, Until: heard})
					if heard-lastSaved >= 10*time.Second {
						if writeJSON(path, transcript) == nil {
							lastSaved = heard
						}
					}
				}
			})
		}
		err := run()
		if err == nil {
			end := until
			if end == 0 {
				end = playback.duration
			}
			if end > from {
				transcript.Spans = addCaptionSpan(transcript.Spans, captionSpan{From: from, Until: end})
			}
		}
		if len(transcript.Spans) > 0 || len(transcript.Segments) > 0 {
			if saveErr := writeJSON(path, transcript); err == nil {
				err = saveErr
			}
		}
		publish()
		m.cx.Post(func() {
			if m.captionGeneration != generation {
				return
			}
			m.captionCancel = nil
			state := m.captions.GetUntracked()
			state.active = false
			if err != nil && ctx.Err() == nil {
				state.problem = err.Error()
			}
			state.complete = err == nil && until == 0 && playback.duration <= 0
			m.captions.Set(state)
		})
	}()
}

func captionGap(spans []captionSpan, position, length time.Duration) (time.Duration, time.Duration, bool) {
	gap := func(from time.Duration) (time.Duration, time.Duration, bool) {
		for _, span := range spans {
			if span.From > from {
				return from, span.From, true
			}
			from = max(from, span.Until)
		}
		return from, 0, length == 0 || from < length-time.Second
	}
	from, until, ok := gap(position)
	if ok || position == 0 {
		return from, until, ok
	}
	return gap(0)
}

func addCaptionSpan(spans []captionSpan, next captionSpan) []captionSpan {
	spans = append(spans, next)
	slices.SortFunc(spans, func(a, b captionSpan) int { return cmp.Compare(a.From, b.From) })
	merged := spans[:0]
	for _, span := range spans {
		if i := len(merged) - 1; i >= 0 && span.From <= merged[i].Until {
			merged[i].Until = max(merged[i].Until, span.Until)
		} else {
			merged = append(merged, span)
		}
	}
	return merged
}

func addCaptionSegment(segments []speech.Segment, next speech.Segment) []speech.Segment {
	i, found := slices.BinarySearchFunc(segments, next.Start, func(s speech.Segment, at time.Duration) int { return cmp.Compare(s.Start, at) })
	if found {
		segments[i] = next
		return segments
	}
	return slices.Insert(segments, i, next)
}
