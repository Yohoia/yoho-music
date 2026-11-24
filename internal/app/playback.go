package app

import (
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/lyrics"
	"yoho-music/internal/music"

	"yoho-music/internal/player"
)

func (m *model) openPlaybackPage() {
	page := "lyrics"
	if m.prefs.Get().LyricsStage {
		page = "stage"
	}
	if track := m.playing.Get().track; track != nil && track.ShowID != "" {
		page = "episode"
	}
	if m.stage.Get() != "" {
		m.closePlaybackPage()
		return
	}
	m.stage.Set(page)
}

func (m *model) selectLyricsPage(cx *core.Scope, page string) {
	m.stage.Set(page)
	want := page == "stage"
	if m.prefs.Get().LyricsStage != want {
		m.updatePreferences(cx, func(p *preferences) { p.LyricsStage = want })
	}
}

func (m *model) toggleStage(page string) {
	if m.stage.Get() == page {
		m.closePlaybackPage()
		return
	}
	m.stage.Set(page)
}

func (m *model) transport(cx *core.Scope, full bool) core.Node {
	p := colors(cx)
	play := iconButton(func() *core.SVG {
		if m.playing.Get().playing {
			return lucide.Pause()
		}
		return lucide.Play()
	}, i18n.T("Play/Pause"), "play-toggle", m.playback(m.player.TogglePlay))
	play.Size, play.IconSize = 40, 22
	play.Disabled = func() bool { return m.playing.Get().track == nil }
	previous := iconButton(lucide.SkipBack, i18n.T("Previous"), "previous", m.playback(m.player.Prev))
	next := iconButton(lucide.SkipForward, i18n.T("Next"), "next", m.playback(m.player.Next))
	previous.Disabled, next.Disabled = play.Disabled, play.Disabled
	if !full {
		return row(previous, play, next)
	}
	shuffle := iconButton(lucide.Shuffle, i18n.T("Shuffle"), "shuffle", m.playback(m.player.ToggleShuffle))
	shuffle.Tint = func() core.Color {
		if m.playing.Get().shuffle {
			return p.accent
		}
		return p.muted
	}
	repeat := iconButton(func() *core.SVG {
		if m.playing.Get().repeat == player.RepeatOne {
			return lucide.Repeat1()
		}
		return lucide.Repeat()
	}, i18n.T("Repeat"), "repeat", m.playback(m.player.CycleRepeat))
	repeat.Tint = func() core.Color {
		if m.playing.Get().repeat != player.RepeatOff {
			return p.accent
		}
		return p.muted
	}
	return row(shuffle, previous, play, next, repeat)
}

func (m *model) progress(cx *core.Scope) core.Node {
	p := colors(cx)
	clock := func(value func() time.Duration) core.Node {
		return core.Text{ContentFunc: func() string { return formatTime(value()) }, Style: core.Style{Width: core.Pt(42), Color: p.muted, FontSize: 10, TextAlign: core.TextAlignCenter}}
	}
	return row(clock(m.pos.Get), ui.Slider{Label: i18n.T("Playback Position"), TestID: "position",
		Value: func() float64 { return m.pos.Get().Seconds() }, MaxFunc: func() float64 { return m.playing.Get().duration.Seconds() },
		OnChange: func(seconds float64) {
			m.seek(time.Duration(seconds * float64(time.Second)))
		},
		Disabled: func() bool { return m.playing.Get().duration <= 0 }, Style: fill,
	}, clock(func() time.Duration { return m.playing.Get().duration }))
}

func (m *model) trackInfo(cx *core.Scope, size core.Points) core.Node {
	p := colors(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		state := m.playing.Get()
		title, detail, cover := i18n.T("Not Playing"), "", ""
		var source string
		if state.track != nil {
			source = state.track.Source
			title, detail, cover = state.track.Name, state.track.ArtistNames(), state.track.Album.ImageURL
			if state.loading {
				detail = i18n.T("Loading…")
			}
			if state.err != "" {
				detail = state.err
			}
		}
		name := label(title, 13, core.FontWeightMedium, p.text)
		name.MaxLines = 1
		name.TestID = "current-track"
		meta := label(detail, 11, core.FontWeightNormal, p.muted)
		meta.MaxLines = 1
		return row(m.artwork(cx, cover, size, source), core.View{Style: fill, Children: []core.Node{name, meta}})
	})
}

func (m *model) playerBar(cx *core.Scope) core.Node {
	p := colors(cx)
	left := core.View{Style: core.Style{Width: core.Pt(174), FlexShrink: 1}, Children: []core.Node{m.trackInfo(cx, 42)}}
	center := core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, AlignItems: core.AlignCenter}, Children: []core.Node{m.transport(cx, true),
		core.View{Style: core.Style{AlignSelf: core.AlignStretch}, Children: []core.Node{m.progress(cx)}},
	}}
	volume := ui.Slider{Label: i18n.T("Volume"), TestID: "volume", Max: 1, Value: func() float64 { return m.prefs.Get().Volume },
		OnChange: m.setVolume, Style: core.Style{Width: core.Pt(62)}}
	right := row(m.remoteDevicePicker(cx), ui.Icon(lucide.Volume2, 16, func() core.Color { return p.muted }), volume,
		iconButton(lucide.ListMusic, i18n.T("Queue"), "queue", func() { m.toggleStage("queue") }),
		iconButton(lucide.Maximize2, i18n.T("Now Playing"), "now-playing", m.openPlaybackPage))
	right.Style.Gap = 2
	bar := row(left, center, right)
	bar.Style.Height = core.Pt(77)
	bar.Style.Padding = core.Symmetric(8, 16)
	bar.Style.Gap = 12
	return core.View{Children: []core.Node{core.View{Style: core.Style{Height: core.Pt(1), Background: p.line}}, bar}}
}

func (m *model) miniPlayer(cx *core.Scope) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		p := colors(cx)
		view := row(m.trackInfo(cx, 42), spacer(), m.transport(cx, false))
		return core.View{Style: core.Style{FlexGrow: 1, Background: p.sidebar, Padding: core.All(14), Gap: 10, JustifyContent: core.JustifyCenter},
			Children: []core.Node{view, m.progress(cx)}}
	})
}

func (m *model) nowPlaying(cx *core.Scope) core.Node {
	cx.SetSubtreeAppearance(core.AppearanceDark)
	p := colors(cx)
	motion, drag := m.playbackMotion(cx)
	tab := func(text, id string, icon glyph, action func()) ui.Button {
		b := button(text, id, icon, action)
		b.Variant = ui.ButtonGhost
		return b
	}
	header := row(iconButton(lucide.ChevronDown, i18n.T("Close Now Playing"), "close-stage", m.closePlaybackPage),
		label(i18n.T("Now Playing"), 18, core.FontWeightBold, p.text), spacer(),
		tab(i18n.T("Up Next"), "stage-queue", lucide.ListMusic, func() { m.stage.Set("queue") }))
	header.Children = append(header.Children, core.Dynamic(func(cx *core.Scope) core.Node {
		state := m.playing.Get()
		if state.track == nil {
			return nil
		}
		var tabs []core.Node
		if state.track.ShowID == "" {
			tabs = append(tabs,
				tab(i18n.T("Lyrics"), "stage-lyrics", lucide.ListMusic, func() { m.selectLyricsPage(cx, "lyrics") }),
				tab(i18n.T("Stage"), "stage-animation", lucide.AudioLines, func() { m.selectLyricsPage(cx, "stage") }),
				iconButton(lucide.LayoutGrid, i18n.T("Cover Wall"), "stage-wall", func() { m.stage.Set("wall") }))
			if m.stage.Get() == "stage" {
				tabs = append(tabs, m.lyricStageStyle(cx))
			}
		} else {
			tabs = append(tabs,
				tab(i18n.T("Episode"), "stage-episode", lucide.Info, func() { m.stage.Set("episode") }),
				tab(i18n.T("Captions"), "stage-captions", lucide.Mic, func() { m.stage.Set("captions") }))
		}
		src := m.sourceByID(state.track.Source)
		if src != nil && src.Can("comments") {
			tabs = append(tabs, iconButton(lucide.MessageCircle, i18n.T("Comments"), "stage-comments", func() { m.stage.Set("comments") }))
		}
		return core.Group(tabs)
	}))
	header.Style.Padding = core.Edges{Top: 35, Right: 24, Bottom: 14, Left: 24}
	header.TestID, header.OnPan = "stage-header", drag
	header.PanActivation = core.PanActivation{ActiveOffsetY: 12, FailOffsetX: 16}
	return core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0),
		Bottom: core.Pt(0), Left: core.Pt(0), Background: p.sidebar}, StyleFunc: motion, Children: []core.Node{
		m.playbackBackdrop(cx), header, core.Dynamic(func(cx *core.Scope) core.Node {
			state := m.playing.Get()
			if state.track == nil {
				return m.empty(cx, lucide.Music, i18n.T("Not Playing"), "", nil)
			}
			mode := m.stage.Get()
			if state.track.ShowID != "" && (mode == "stage" || mode == "lyrics" || mode == "wall") {
				mode = "episode"
			}
			if state.track.ShowID == "" && (mode == "episode" || mode == "captions") {
				mode = "lyrics"
			}
			if mode == "stage" {
				return core.View{Style: fill, Children: []core.Node{m.lyricStage(cx, state.lines), m.playerBar(cx)}}
			}
			if mode == "wall" {
				return core.View{Style: fill, Children: []core.Node{m.coverWall(cx, state), m.playerBar(cx)}}
			}
			window := cx.Env().Window()
			side := max(240, min(440, window.Width*.36))
			coverSize := max(120, min(side, window.Height-390))
			cover := m.playbackSide(cx, state.track, side, coverSize)
			var visible core.Node
			switch mode {
			case "queue":
				var indexes []int
				for _, index := range state.order {
					if index >= 0 && index < len(state.queue) {
						indexes = append(indexes, index)
					}
				}
				visible = core.List{Count: func() int { return len(indexes) }, EstimatedSize: 52, Style: fill,
					Row: func(cx *core.Scope, cell core.ListRow) core.Node {
						return core.Dynamic(func(cx *core.Scope) core.Node {
							i := cell.Index()
							if i < 0 || i >= len(indexes) {
								return nil
							}
							j := indexes[i]
							return m.songRow(cx, state.queue[j], i, nil, func() { m.player.Jump(j) })
						})
					}}
			case "captions":
				visible = m.captionPane(cx)
			case "comments":
				visible = m.commentsPane(cx, *state.track)
			case "episode":
				visible = m.episodePane(cx, *state.track)
			default:
				visible = m.lyricList(cx, state.lines)
			}
			inset := max(26, min(72, window.Width*.061))
			return core.View{Style: core.Style{Direction: core.Row, FlexGrow: 1, FlexShrink: 1, Gap: min(72, window.Width*.06),
				Padding: core.Edges{Left: inset, Right: inset, Bottom: 24}}, Children: []core.Node{cover, visible}}
		}),
	}}
}

func (m *model) playbackSide(cx *core.Scope, track *music.Track, side, size core.Points) core.Node {
	p := colors(cx)
	title := label(track.Name, 22, core.FontWeightBold, p.text)
	title.MaxLines = 2
	meta := track.ArtistNames()
	if track.Album.Name != "" {
		if meta != "" {
			meta += " · "
		}
		meta += track.Album.Name
	}
	detail := label(meta, 13, core.FontWeightMedium, p.muted)
	detail.MaxLines = 1
	cover := core.Dynamic(func(cx *core.Scope) core.Node {
		return m.dragArtwork(cx, m.playerArtwork(cx, track, m.prefs.Get().PlayerStyle, size))
	})
	titleLine := row(core.View{Style: fill, Children: []core.Node{title, detail}})
	if src := m.sourceByID(track.Source); canLike(src, *track) {
		titleLine.Children = append(titleLine.Children, iconButton(lucide.Heart, i18n.T("Like/Unlike"), "now-playing-like", func() { m.toggleLike(*track) }))
	}
	controls := row(spacer(), m.transport(cx, true), spacer())
	slider := ui.Slider{Label: i18n.T("Volume"), TestID: "volume", Max: 1,
		Value: func() float64 { return m.prefs.Get().Volume }, OnChange: m.setVolume, Style: core.Style{Width: core.Pt(156)}}
	volume := row(spacer(), m.remoteDevicePicker(cx), ui.Icon(lucide.Volume2, 14, func() core.Color { return p.muted }), slider, spacer())
	return core.View{Style: core.Style{Width: core.Pt(side), FlexShrink: 1, JustifyContent: core.JustifyCenter, Gap: 14},
		Children: []core.Node{core.View{Style: core.Style{AlignItems: core.AlignCenter}, Children: []core.Node{cover}},
			titleLine, m.progress(cx), controls, volume}}
}

func (m *model) playbackBackdrop(cx *core.Scope) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		track := m.playing.Get().track
		if track == nil || track.Album.ImageURL == "" {
			return core.View{PointerEvents: core.PointerEventsNone, Style: core.Style{
				Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Background: core.RGB(0x202128),
			}}
		}
		frame := cx.Env().Window()
		cover := core.Image{SourceFunc: m.coverSource(cx, track.Album.ImageURL, 256, track.Source),
			ResizeMode: core.Cover, BlurRadius: 58, ElementsHidden: true,
			Style: core.Style{Width: core.Pt(frame.Width), Height: core.Pt(frame.Height), Opacity: core.Alpha(.43)}}
		return core.View{PointerEvents: core.PointerEventsNone, Style: core.Style{
			Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
			Background: core.RGB(0x14141c), Overflow: core.Hidden,
		}, Children: []core.Node{cover}}
	})
}

func (m *model) lyricList(cx *core.Scope, lines []lyrics.Line) core.Node {
	if len(lines) == 0 {
		return m.empty(cx, lucide.AudioLines, i18n.T("No Lyrics"), "", nil)
	}
	p := colors(cx)
	ref := &core.Ref{}
	active := core.NewMemo(cx, func() int { return lyrics.Index(lines, m.pos.Get()) })
	cx.Effect(func() {
		if i := active.Get(); i >= 0 {
			ref.ScrollToIndexAt(i, .36, 0, true)
		}
	})
	return core.List{Ref: ref, Count: func() int { return len(lines) }, EstimatedSize: 56, Style: fill,
		Row: func(cx *core.Scope, cell core.ListRow) core.Node {
			return core.Dynamic(func(cx *core.Scope) core.Node {
				i := cell.Index()
				if i < 0 || i >= len(lines) {
					return nil
				}
				line := lines[i]
				tint := func() core.Style {
					if active.Get() == i {
						return core.Style{Color: p.text}
					}
					return core.Style{Color: p.muted}
				}
				text := core.Text{Content: line.Text, Style: core.Style{FontSize: 27, FontWeight: core.FontWeightBold}, StyleFunc: tint}
				if len(line.Words) > 0 {
					text.SpansFunc = func() []core.Span {
						if active.Get() != i {
							return []core.Span{{Content: line.Text}}
						}
						at := m.pos.Get()
						spans := make([]core.Span, 0, len(line.Words))
						for _, word := range line.Words {
							color := p.muted
							if at >= word.Start {
								color = p.text
							}
							spans = append(spans, core.Span{Content: word.Text, Style: core.Style{Color: color}})
						}
						return spans
					}
				}
				var children = []core.Node{text}
				if line.Translation != "" {
					children = append(children, label(line.Translation, 14, core.FontWeightNormal, p.muted))
				}
				return core.View{OnPress: func(core.PressEvent) { m.seek(line.At) }, Style: core.Style{Padding: core.Symmetric(8, 12), Gap: 3}, Children: children}
			})
		},
	}
}
