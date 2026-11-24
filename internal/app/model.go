package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"yoho-music/internal/i18n"
	"yoho-music/internal/local"
	"yoho-music/internal/lyrics"
	"yoho-music/internal/music"
	"yoho-music/internal/player"
	"yoho-music/internal/plugin"
)

type route struct{ page, id string }

type preferences struct {
	Appearance    string  `json:"appearance"`
	Language      string  `json:"language"`
	Volume        float64 `json:"volume"`
	Normalize     bool    `json:"normalize"`
	Quality       string  `json:"quality,omitempty"`
	CacheMB       int     `json:"cacheMB"`
	PlayerStyle   string  `json:"playerStyle,omitempty"`
	LyricsStage   bool    `json:"lyricsStage,omitempty"`
	StageStyle    string  `json:"stageStyle,omitempty"`
	StageSparks   bool    `json:"stageSparks,omitempty"`
	NotchEnabled  bool    `json:"notchEnabled,omitempty"`
	NotchVirtual  bool    `json:"notchVirtual,omitempty"`
	NotchActivity bool    `json:"notchActivity"`
	NotchGreeted  bool    `json:"notchGreeted,omitempty"`
	Captions      bool    `json:"captions,omitempty"`
}

type playbackView struct {
	track            *music.Track
	playing, loading bool
	shuffle          bool
	repeat           player.RepeatMode
	duration         time.Duration
	err              string
	queue            []music.Track
	order            []int
	index            int
	lines            []lyrics.Line
}

type model struct {
	cx                 *core.Scope
	context            context.Context
	dir                string
	lib                *local.Source
	player             *player.Player
	page               *core.Signal[route]
	query              *core.Signal[string]
	updated            *core.Signal[int]
	playing            *core.Signal[playbackView]
	pos                *core.Signal[time.Duration]
	prefs              *core.Signal[preferences]
	stageDismiss       func()
	stageCovered       *core.Signal[bool]
	stage              *core.Signal[string]
	error              *core.Signal[string]
	source             *core.Signal[string]
	plugins            *core.Signal[[]*plugin.NSGoSource]
	nsgoStore          *core.Signal[*plugin.NSGoStore]
	nsgoPluginBusy     *core.Signal[bool]
	marketURL          string
	accounts           *core.Signal[map[string]pluginAccount]
	sourceVersion      *core.Signal[int]
	signInDialog       *core.Signal[signInDialogState]
	back               []route
	forward            []route
	history            *core.Signal[int]
	display            *core.Signal[core.Bounds]
	notchQuick         *core.Signal[bool]
	notchState         *core.Signal[notchPresentation]
	volumeTimer        func() bool
	captions           *core.Signal[captionView]
	captionCancel      context.CancelFunc
	captionModel       *core.Signal[captionModelState]
	captionModelStop   context.CancelFunc
	captionWorker      sync.Mutex
	captionGeneration  int
	captionLoadVersion int
}

func newModel(cx *core.Scope, dir string) *model {
	prefs := preferences{Appearance: "system", Language: "system", Volume: .8, Normalize: true, Quality: "high", CacheMB: 2048, NotchActivity: true}
	if dir != "" {
		readJSON(filepath.Join(dir, "settings.json"), &prefs)
	}
	m := &model{
		cx: cx, context: cx.Context(), dir: dir, lib: local.New(dir),
		page: core.NewSignal(cx, route{page: "home"}), query: core.NewSignal(cx, ""),
		updated: core.NewSignal(cx, 0), playing: core.NewSignalFunc(cx, playbackView{}, nil),
		pos: core.NewSignal(cx, time.Duration(0)), prefs: core.NewSignal(cx, prefs),
		stage: core.NewSignal(cx, ""), stageCovered: core.NewSignal(cx, false), error: core.NewSignal(cx, ""), history: core.NewSignal(cx, 0),
		source:         core.NewSignal(cx, local.ID),
		plugins:        core.NewSignalFunc(cx, []*plugin.NSGoSource(nil), nil),
		nsgoStore:      core.NewSignal(cx, (*plugin.NSGoStore)(nil)),
		nsgoPluginBusy: core.NewSignal(cx, false),
		captions:       core.NewSignalFunc(cx, captionView{}, nil),
		captionModel:   core.NewSignal(cx, captionModelState{}),
		display:        core.NewSignal(cx, core.Bounds{}),
		notchQuick:     core.NewSignal(cx, false),
		notchState:     core.NewSignal(cx, notchPresentation{}),
		accounts:       core.NewSignalFunc(cx, map[string]pluginAccount{}, nil), sourceVersion: core.NewSignal(cx, 0),
		signInDialog: core.NewSignalFunc(cx, signInDialogState{}, nil),
	}
	m.applyPreferences(cx)
	m.lib.OnChange(func() { cx.Post(m.invalidate) })
	m.player = player.New(func(fn func()) { cx.Post(func() { fn(); m.refreshPlayback() }) })
	m.player.Source = m.sourceByID
	m.player.Normalize = func() bool { return m.prefs.Get().Normalize }
	m.player.Quality = func() string { return m.prefs.Get().Quality }
	if dir != "" {
		m.player.Cache = player.NewCache(filepath.Join(dir, "audio-cache"), int64(max(prefs.CacheMB, 0))<<20)
	}
	m.player.SetVolume(prefs.Volume)
	var saved player.Session
	var restore *player.Session
	if dir != "" && readJSON(filepath.Join(dir, "session.json"), &saved) {
		restore = &saved
	}
	m.refreshPlayback()
	if len(m.lib.Folders()) > 0 {
		m.lib.Rescan()
	}
	if dir != "" {
		m.discoverPluginsThen(func() {
			if restore != nil && m.player.Current() == nil {
				m.player.Restore(*restore)
				m.refreshPlayback()
			}
		})
		var tick func()
		lastSave := time.Now()
		tick = func() {
			if m.player.Current() != nil {
				m.pos.Set(m.player.Position())
				if time.Since(lastSave) >= 5*time.Second {
					m.saveSession()
					lastSave = time.Now()
				}
			}
			cx.AfterFunc(200*time.Millisecond, tick)
		}
		cx.AfterFunc(200*time.Millisecond, tick)
	}
	cx.Effect(m.syncCaptions)
	cx.Cleanup(func() {
		m.lib.OnChange(nil)
		if m.captionCancel != nil {
			m.captionCancel()
		}
		if m.captionModelStop != nil {
			m.captionModelStop()
		}
		m.player.Close()
	})
	return m
}

func (m *model) discoverPluginsThen(done func()) {
	go func() {
		store, err := plugin.OpenNSGoStore(m.dir)
		var packages []*plugin.NSGoPackage
		if err == nil {
			packages, err = store.Discover()
		}
		m.cx.Post(func() {
			m.report(err)
			var sources []*plugin.NSGoSource
			for _, p := range packages {
				source, err := plugin.LoadNSGo(m.cx, p)
				if err != nil {
					m.report(err)
					continue
				}
				sources = append(sources, source)
			}
			m.plugins.Set(sources)
			m.nsgoStore.Set(store)
			done()
		})
	}()
}

func (m *model) sourceByID(id string) music.Source {
	if id == "" || id == local.ID {
		return m.lib
	}
	for _, source := range m.plugins.Get() {
		if source.Info().ID == id {
			return source
		}
	}
	return nil
}

func (m *model) invalidate() { m.updated.Update(func(n int) int { return n + 1 }) }

func (m *model) refreshPlayback() {
	p := m.player
	m.loadEpisodeTranscript(p.Current())
	lines, _ := p.Lyrics()
	order, index := p.PlayOrder()
	m.playing.Set(playbackView{track: p.Current(), playing: p.Playing(), loading: p.Loading(),
		shuffle: p.Shuffle(), repeat: p.Repeat(), duration: p.Duration(), err: p.Err(),
		queue: p.Queue(), order: order, index: index, lines: lines})
	m.pos.Set(p.Position())
}

func (m *model) playback(fn func()) func() {
	return func() {
		fn()
		m.refreshPlayback()
	}
}

func (m *model) navigate(next route) {
	m.stage.Set("")
	if current := m.page.Get(); current != next {
		m.back = append(m.back, current)
		m.forward = nil
		m.page.Set(next)
		m.history.Update(func(n int) int { return n + 1 })
	}
}

func (m *model) goBack() {
	if m.stage.Get() != "" {
		m.stage.Set("")
		return
	}
	if n := len(m.back); n > 0 {
		m.forward = append(m.forward, m.page.Get())
		m.page.Set(m.back[n-1])
		m.back = m.back[:n-1]
		m.history.Update(func(n int) int { return n + 1 })
	}
}

func (m *model) goForward() {
	if n := len(m.forward); n > 0 {
		m.back = append(m.back, m.page.Get())
		m.page.Set(m.forward[n-1])
		m.forward = m.forward[:n-1]
		m.history.Update(func(n int) int { return n + 1 })
	}
}

func (m *model) search(query string) {
	m.query.Set(query)
	if terms := strings.TrimSpace(query); terms != "" {
		m.navigate(route{page: "search", id: terms})
	} else if m.page.Get().page == "search" {
		m.navigate(route{page: "home"})
	}
}

func (m *model) report(err error) {
	if err != nil {
		m.error.Set(err.Error())
	}
}

func (m *model) run(fn func() error) {
	go func() {
		err := fn()
		m.cx.Post(func() { m.report(err) })
	}()
}

func (m *model) saveSession() {
	if session, ok := m.player.Session(); ok && m.dir != "" {
		m.report(writeJSON(filepath.Join(m.dir, "session.json"), session))
	}
}

func (m *model) applyPreferences(cx *core.Scope) {
	p := m.prefs.Get()
	appearances := map[string]core.Appearance{"light": core.AppearanceLight, "dark": core.AppearanceDark}
	cx.SetAppearance(appearances[p.Appearance])
	i18n.SetLanguage(i18n.Resolve(p.Language, []string{cx.Env().Locale()}))
}

func (m *model) updatePreferences(cx *core.Scope, update func(*preferences)) {
	before := m.prefs.GetUntracked()
	p := before
	update(&p)
	if p == before {
		return
	}
	m.prefs.Set(p)
	if p.Volume != before.Volume {
		m.player.SetVolume(p.Volume)
	}
	if p.Appearance != before.Appearance || p.Language != before.Language {
		m.applyPreferences(cx)
	}
	if m.dir != "" {
		m.report(writeJSON(filepath.Join(m.dir, "settings.json"), p))
	}
}

func readJSON(path string, value any) bool {
	b, err := os.ReadFile(path)
	return err == nil && json.Unmarshal(b, value) == nil
}

func writeJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".yoho-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(b)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}
