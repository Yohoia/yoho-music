package app

import (
	"math"
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/local"
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

var (
	playbarRewindSVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M11.6 18.2a1 1 0 0 1-1.6.8L2.6 13.2a1.5 1.5 0 0 1 0-2.4L10 5a1 1 0 0 1 1.6.8v12.4zm9.6 0a1 1 0 0 1-1.6.8l-7.4-5.8a1.5 1.5 0 0 1 0-2.4L19.6 5a1 1 0 0 1 1.6.8v12.4z"/></svg>`))

	playbarFastForwardSVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M2.8 5.8A1 1 0 0 1 4.4 5l7.4 5.8a1.5 1.5 0 0 1 0 2.4L4.4 19a1 1 0 0 1-1.6-.8V5.8zm9.6 0A1 1 0 0 1 14 5l7.4 5.8a1.5 1.5 0 0 1 0 2.4L14 19a1 1 0 0 1-1.6-.8V5.8z"/></svg>`))

	playbarPauseSVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><rect x="6" y="4.5" width="4.4" height="15" rx="1.4"/><rect x="13.6" y="4.5" width="4.4" height="15" rx="1.4"/></svg>`))

	// Custom [untitled] queue icon from PlaybarFeature Assets.car: rounded capsule bar on top + two rounded bars below.
	playbarQueueSVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><rect x="3.5" y="4.5" width="17" height="4.8" rx="2.4"/><line x1="4" y1="14" x2="20" y2="14"/><line x1="4" y1="18.6" x2="20" y2="18.6"/></svg>`))

	// 4-state [untitled] / SF Symbols volume icons: mute (slash), 1 wave, 2 waves, 3 waves.
	playbarVolMuteSVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5.5 7.5 8.5H4.5a1 1 0 0 0-1 1v5a1 1 0 0 0 1 1h3l4.5 3.8V16"/><path d="M12 10.2V5.5"/><line x1="3.5" y1="3.5" x2="20.5" y2="20.5"/></svg>`))

	playbarVol1SVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5.5 6.8 9H4a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h2.8L11 18.5a.6.6 0 0 0 1-.45V5.95a.6.6 0 0 0-1-.45z"/><path d="M15.2 9.4a4.2 4.2 0 0 1 0 5.2"/></svg>`))

	playbarVol2SVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5.5 6.8 9H4a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h2.8L11 18.5a.6.6 0 0 0 1-.45V5.95a.6.6 0 0 0-1-.45z"/><path d="M15.1 9.4a4.2 4.2 0 0 1 0 5.2"/><path d="M17.8 7a7.4 7.4 0 0 1 0 10"/></svg>`))

	playbarVol3SVG = core.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round"><path d="M10.2 5.5 6.2 9H3.5a1 1 0 0 0-1 1v4a1 1 0 0 0 1 1h2.7l4 3.5a.6.6 0 0 0 1-.45V5.95a.6.6 0 0 0-1-.45z"/><path d="M14.2 9.5a3.8 3.8 0 0 1 0 5"/><path d="M16.9 7.2a7 7 0 0 1 0 9.6"/><path d="M19.6 4.9a10.2 10.2 0 0 1 0 14.2"/></svg>`))
)

func waveformHeights(seed string, count int) []float32 {
	h := uint32(2166136261)
	for i := 0; i < len(seed); i++ {
		h ^= uint32(seed[i])
		h *= 16777619
	}
	bars := make([]float32, count)
	for i := 0; i < count; i++ {
		h ^= h << 13
		h ^= h >> 17
		h ^= h << 5
		r1 := float32(h%1000) / 1000.0
		h ^= h << 13
		h ^= h >> 17
		h ^= h << 5
		r2 := float32(h%1000) / 1000.0
		// Macro musical phrasing envelope + crisp local needle dynamics matching [untitled] waveform.
		t := float64(i) / float64(max(1, count-1))
		edge := float32(0.38 + 0.62*math.Sin(t*math.Pi))
		phrase := float32(0.76 + 0.24*math.Sin(t*math.Pi*5.2+float64(seed[0]%7)))
		amp := (0.14 + 0.86*(0.65*r1+0.35*r2)) * edge * phrase
		bars[i] = min(1, max(0.12, amp))
	}
	return bars
}

func (m *model) playbarClock(cx *core.Scope) *core.Signal[time.Duration] {
	position := core.NewSignal(cx, m.pos.GetUntracked())
	var stop func()
	cx.Effect(func() {
		if stop != nil {
			stop()
			stop = nil
		}
		if !m.playing.Get().playing || m.stage.Get() != "" || cx.Env().ReduceMotion() {
			position.Set(m.pos.Get())
			return
		}
		var last time.Time
		var frame func(time.Time)
		frame = func(at time.Time) {
			if at.Sub(last) >= time.Second/30 {
				position.Set(m.player.Position())
				last = at
			}
			stop = cx.RequestAnimationFrame(frame)
		}
		position.Set(m.player.Position())
		stop = cx.RequestAnimationFrame(frame)
	})
	cx.Cleanup(func() {
		if stop != nil {
			stop()
		}
	})
	return position
}

func (m *model) playbarWaveform(cx *core.Scope) core.Node {
	waveRef := &core.Ref{}
	clock := m.playbarClock(cx)
	hoverRatio := core.NewSignal(cx, float32(-1))
	hoverX := core.NewSignal(cx, core.Points(0))
	scrubRatio := core.NewSignal(cx, float32(-1))

	ratioForX := func(pageX core.Points) (float32, core.Points, bool) {
		bounds, ok := waveRef.Bounds()
		if !ok || bounds.Size.Width <= 0 {
			return 0, 0, false
		}
		localX := min(max(pageX-bounds.Origin.X, 0), bounds.Size.Width)
		return float32(localX / bounds.Size.Width), localX, true
	}
	commitSeek := func(r float32) {
		dur := m.playing.GetUntracked().duration
		if dur <= 0 {
			return
		}
		m.seek(time.Duration(float64(r) * float64(dur)))
	}

	canvas := core.Canvas{
		Style: core.Style{FlexGrow: 1, FlexShrink: 1, Height: core.Pt(34)},
		Draw: func(d *core.Drawing, size core.Size) {
			state := m.playing.Get()
			pos := clock.Get()
			var ratio float32
			if sr := scrubRatio.Get(); sr >= 0 {
				ratio = sr
			} else if state.duration > 0 {
				ratio = float32(pos.Seconds() / state.duration.Seconds())
				ratio = min(1, max(0, ratio))
			}
			hr := hoverRatio.Get()
			seed := "yoho"
			if state.track != nil {
				seed = state.track.ID + state.track.Name
			}
			w := float32(size.Width)
			h := float32(size.Height)
			if w <= 4 || h <= 4 {
				return
			}
			const barW float32 = 1.5
			const gap float32 = 1.5
			step := barW + gap
			count := max(12, int(w/step))
			bars := waveformHeights(seed, count)
			playX := ratio * w
			hoverPosX := hr * w
			playedColor := core.RGB(0xe5e5ea)
			previewColor := core.RGB(0x9e9ea6)
			unplayedColor := core.RGB(0x636369)

			for i, amp := range bars {
				x := float32(i) * step
				if x+barW > w {
					break
				}
				bh := max(float32(3), amp*(h-5))
				y := (h - bh) / 2
				switch {
				case x <= playX:
					d.FillFromView(core.PaintColor(playedColor))
				case hr >= 0 && x <= hoverPosX:
					d.FillFromView(core.PaintColor(previewColor))
				default:
					d.FillFromView(core.PaintColor(unplayedColor))
				}
				d.FillRect(x, y, barW, bh)
			}

			// Subtle hover hairline cursor.
			if hr >= 0 && scrubRatio.Get() < 0 {
				hx := min(max(hoverPosX, 0), w-1)
				d.FillFromView(core.PaintColor(core.RGBA(0xffffff, .42)))
				d.FillRect(hx, 2, 1, h-4)
			}

			// Signature [untitled] warm-yellow playhead with rounded caps.
			headX := min(max(playX-1, 0), w-2)
			d.FillFromView(core.PaintColor(core.RGB(0xf5d033)))
			d.BeginPath()
			d.RoundRect(headX, 0, 2, h, 1)
			d.Fill()
		},
	}

	// PlaybarPreviewPill: floating timestamp pill above hovered/scrubbed waveform position.
	previewPill := core.View{
		PointerEvents: core.PointerEventsNone,
		Style: core.Style{
			Position: core.Absolute, Bottom: core.Pt(44), Left: core.Pt(0),
			Height: core.Pt(24), Padding: core.Symmetric(0, 8), CornerRadius: 12,
			Background: core.RGB(0x1c1c1e), BorderWidth: 1, BorderColor: core.RGBA(0xffffff, .14),
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			ShadowColor: core.RGBA(0x000000, .32), ShadowOffset: core.Point{Y: 4}, ShadowRadius: 10,
			Transition: core.Transition{Duration: 140 * time.Millisecond, Easing: core.EaseOut},
		},
		StyleFunc: func() core.Style {
			r := scrubRatio.Get()
			if r < 0 {
				r = hoverRatio.Get()
			}
			if r < 0 || m.playing.Get().duration <= 0 {
				return core.Style{Opacity: core.Alpha(0), Transform: core.Transform{TranslateX: max(0, hoverX.Get()-22), TranslateY: 4}}
			}
			return core.Style{Opacity: core.Alpha(1), Transform: core.Transform{TranslateX: max(0, hoverX.Get()-22), TranslateY: 0}}
		},
		Children: []core.Node{
			core.Text{
				ContentFunc: func() string {
					r := scrubRatio.Get()
					if r < 0 {
						r = hoverRatio.Get()
					}
					if r < 0 {
						return ""
					}
					dur := m.playing.Get().duration
					return formatTime(time.Duration(float64(r) * float64(dur)))
				},
				Style: core.Style{
					FontFamily: "SF Mono, Menlo, Monaco, monospace", FontSize: 10.5,
					FontVariant: core.FontTabularNums, Color: core.RGB(0xf4f4f5),
				},
			},
		},
	}

	var panStartX core.Points
	waveBox := core.View{
		Ref:   waveRef,
		Style: core.Style{FlexGrow: 1, FlexShrink: 1, Height: core.Pt(34), MinWidth: core.Pt(0), JustifyContent: core.JustifyCenter},
		OnMouseEnter: func(e core.MouseEvent) {
			if r, lx, ok := ratioForX(e.PagePosition.X); ok {
				hoverRatio.Set(r)
				hoverX.Set(lx)
			}
		},
		OnMouseMove: func(e core.MouseEvent) {
			if r, lx, ok := ratioForX(e.PagePosition.X); ok {
				hoverRatio.Set(r)
				hoverX.Set(lx)
			}
		},
		OnMouseLeave: func(core.MouseEvent) {
			hoverRatio.Set(-1)
		},
		OnPress: func(e core.PressEvent) {
			if r, lx, ok := ratioForX(e.PagePosition.X); ok {
				hoverX.Set(lx)
				commitSeek(r)
			}
		},
		PanActivation: core.PanActivation{ActiveOffsetX: 3},
		OnPan: func(e core.PanEvent) {
			bounds, ok := waveRef.Bounds()
			if !ok || bounds.Size.Width <= 0 {
				return
			}
			switch e.Phase {
			case core.GestureBegan:
				panStartX = hoverX.GetUntracked()
				lx := min(max(panStartX+e.Translation.X, 0), bounds.Size.Width)
				hoverX.Set(lx)
				scrubRatio.Set(float32(lx / bounds.Size.Width))
			case core.GestureChanged:
				lx := min(max(panStartX+e.Translation.X, 0), bounds.Size.Width)
				hoverX.Set(lx)
				scrubRatio.Set(float32(lx / bounds.Size.Width))
			case core.GestureEnded:
				lx := min(max(panStartX+e.Translation.X, 0), bounds.Size.Width)
				r := float32(lx / bounds.Size.Width)
				hoverX.Set(lx)
				scrubRatio.Set(-1)
				commitSeek(r)
			case core.GestureCancelled:
				scrubRatio.Set(-1)
			}
		},
		Children: []core.Node{canvas, previewPill},
	}

	clockText := core.Text{
		ContentFunc: func() string {
			dur := m.playing.Get().duration
			at := clock.Get()
			if sr := scrubRatio.Get(); sr >= 0 && dur > 0 {
				at = time.Duration(float64(sr) * float64(dur))
			}
			return formatTime(at) + " / " + formatTime(dur)
		},
		Style: core.Style{
			FontFamily: "SF Mono, Menlo, Monaco, monospace", FontSize: 12,
			FontVariant: core.FontTabularNums, Color: core.RGB(0xd4d4d8),
			Margin: core.Edges{Left: 8},
		},
	}
	scrubber := row(waveBox, clockText)
	scrubber.Style.FlexGrow = 1
	scrubber.Style.FlexShrink = 1
	scrubber.Style.Gap = 4
	return scrubber
}

func (m *model) playbarSpinAngle(cx *core.Scope) *core.Signal[float32] {
	spinAngle := core.NewSignal(cx, float32(0))
	spinning := core.NewMemo(cx, func() bool {
		return m.playing.Get().playing && m.stage.Get() == "" && !cx.Env().ReduceMotion()
	})
	var stopSpin func()
	cx.Effect(func() {
		if stopSpin != nil {
			stopSpin()
			stopSpin = nil
		}
		if !spinning.Get() {
			return
		}
		var last time.Time
		var frame func(time.Time)
		frame = func(at time.Time) {
			if !last.IsZero() {
				dt := at.Sub(last).Seconds()
				if dt > 0 && dt < 0.25 {
					spinAngle.Update(func(deg float32) float32 {
						next := deg + float32(dt*36.0)
						if next >= 360 {
							next -= 360
						}
						return next
					})
				}
			}
			last = at
			stopSpin = cx.RequestAnimationFrame(frame)
		}
		stopSpin = cx.RequestAnimationFrame(frame)
	})
	cx.Cleanup(func() {
		if stopSpin != nil {
			stopSpin()
		}
	})
	return spinAngle
}

func (m *model) playbarTrackInfo(cx *core.Scope, spinAngle *core.Signal[float32]) core.Node {
	type artwork struct{ cover, source, fallback string }
	currentArt := core.NewMemo(cx, func() artwork {
		track := m.playing.Get().track
		if track == nil {
			return artwork{}
		}
		if track.Album.ImageURL != "" {
			return artwork{cover: track.Album.ImageURL, source: track.Source}
		}
		return artwork{fallback: track.Name}
	})

	spinningArt := core.View{
		Style: core.Style{Width: core.Pt(44), Height: core.Pt(44), CornerRadius: 22, Overflow: core.Hidden},
		StyleFunc: func() core.Style {
			return core.Style{Transform: core.Transform{Rotate: spinAngle.Get()}}
		},
		Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			art := currentArt.Get()
			face := core.Style{Width: core.Pt(44), Height: core.Pt(44), CornerRadius: 22}
			if art.cover != "" {
				return core.Image{SourceFunc: m.coverSource(cx, art.cover, 88, art.source), Style: face, ResizeMode: core.Cover}
			}
			face.Gradients = []core.Gradient{coverGradientFor(art.fallback)}
			return core.View{Style: face}
		})},
	}
	name := label("", 14, core.FontWeightMedium, core.RGB(0xffffff))
	name.ContentFunc = func() string {
		if track := m.playing.Get().track; track != nil {
			return track.Name
		}
		return ""
	}
	name.MaxLines = 1
	name.TestID = "current-track"
	meta := label("", 12.5, core.FontWeightNormal, core.RGB(0x98989d))
	meta.ContentFunc = func() string {
		state := m.playing.Get()
		switch {
		case state.track == nil:
			return ""
		case state.err != "":
			return state.err
		case state.loading:
			return i18n.T("Loading…")
		case state.track.Album.Name != "":
			return state.track.Album.Name
		default:
			return state.track.ArtistNames()
		}
	}
	meta.MaxLines = 1
	return base.Button{
		Label:   i18n.T("Now Playing"),
		TestID:  "now-playing",
		OnPress: func(core.PressEvent) { m.openPlaybackPage() },
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12,
			Width: core.Pt(170), MinWidth: core.Pt(170), CornerRadius: 22,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Opacity: core.Alpha(.88)},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.98}},
		},
		Children: []core.Node{spinningArt, core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2}, Children: []core.Node{name, meta}}},
	}
}

func (m *model) playbarControls(cx *core.Scope, onCollapse func()) core.Node {
	// Observe changing glyphs in the icon nodes so control state survives playback updates.
	btn := func(icon glyph, name, id string, size core.Points, tint func() core.Color, action func()) core.Node {
		return base.Button{
			Label:   name,
			TestID:  id,
			OnPress: func(core.PressEvent) { action() },
			Style: core.Style{
				Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 15,
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				Transition: core.Transition{Duration: 140 * time.Millisecond, Easing: core.EaseOut},
				Hover:      &core.Style{Background: core.RGBA(0xffffff, .10)},
				Active:     &core.Style{Transform: core.Transform{Scale: 0.88}},
			},
			Children: []core.Node{ui.IconOf(icon, size, tint)},
		}
	}
	white := func() core.Color { return core.RGB(0xffffff) }
	hasNext := func() bool {
		st := m.playing.Get()
		return st.repeat == player.RepeatAll || (st.index >= 0 && st.index+1 < len(st.order))
	}

	prev := btn(func() *core.SVG { return playbarRewindSVG }, i18n.T("Previous"), "previous", 16, white, m.playback(m.player.Prev))
	play := btn(func() *core.SVG {
		if m.playing.Get().playing {
			return playbarPauseSVG
		}
		return playFillSVG
	}, i18n.T("Play/Pause"), "play-toggle", 16, white, m.playback(m.player.TogglePlay))
	next := btn(func() *core.SVG { return playbarFastForwardSVG }, i18n.T("Next"), "next", 16, func() core.Color {
		if hasNext() {
			return core.RGB(0xffffff)
		}
		return core.RGB(0x7a7a80)
	}, m.playback(m.player.Next))
	shuffle := btn(lucide.Shuffle, i18n.T("Shuffle"), "shuffle", 15, func() core.Color {
		if m.playing.Get().shuffle {
			return core.RGB(0xffffff)
		}
		return core.RGB(0x8e8e93)
	}, m.playback(m.player.ToggleShuffle))
	repeat := btn(func() *core.SVG {
		if m.playing.Get().repeat == player.RepeatOne {
			return lucide.Repeat1()
		}
		return lucide.Repeat()
	}, i18n.T("Repeat"), "repeat", 15, func() core.Color {
		if m.playing.Get().repeat != player.RepeatOff {
			return core.RGB(0xffffff)
		}
		return core.RGB(0x8e8e93)
	}, m.playback(m.player.CycleRepeat))
	queueBtn := btn(func() *core.SVG { return playbarQueueSVG }, i18n.T("Queue"), "queue", 16, func() core.Color {
		if m.stage.Get() == "queue" {
			return core.RGB(0xffffff)
		}
		return core.RGB(0xd4d4d8)
	}, func() { m.toggleStage("queue") })

	// Vertical [untitled] volume capsule unfolding upward out of the volume icon.
	// Mounted in a full-window PointerEventsBoxNone Overlay anchored to volBtnRef.Bounds()
	// so the portion rising above the 60pt Playbar receives full native hit-testing/hover/drag.
	volBtnRef := &core.Ref{}
	volOpen := core.NewSignal(cx, false)
	volAnchor := core.NewSignal(cx, core.Bounds{})
	volBtnHovered := false
	volPopHovered := false
	volDragging := false
	var volCloseTimer *time.Timer
	cancelVolClose := func() {
		if volCloseTimer != nil {
			volCloseTimer.Stop()
			volCloseTimer = nil
		}
	}
	updateAnchor := func() {
		if b, ok := volBtnRef.Bounds(); ok && b.Size.Width > 0 {
			volAnchor.Set(b)
		}
	}
	syncVolOpen := func() {
		if volBtnHovered || volPopHovered || volDragging {
			cancelVolClose()
			updateAnchor()
			volOpen.Set(true)
			return
		}
		cancelVolClose()
		volCloseTimer = time.AfterFunc(180*time.Millisecond, func() {
			cx.Post(func() {
				if !volBtnHovered && !volPopHovered && !volDragging {
					volOpen.Set(false)
				}
			})
		})
	}
	cx.Cleanup(cancelVolClose)

	lastVol := 0.8
	volIcon := func() *core.SVG {
		v := m.prefs.Get().Volume
		switch {
		case v <= 0.001:
			return playbarVolMuteSVG
		case v <= 0.33:
			return playbarVol1SVG
		case v <= 0.66:
			return playbarVol2SVG
		default:
			return playbarVol3SVG
		}
	}
	toggleMute := func() {
		cur := m.prefs.Get().Volume
		if cur > 0.001 {
			lastVol = cur
			m.setVolume(0)
		} else {
			if lastVol <= 0.001 {
				lastVol = 0.8
			}
			m.setVolume(lastVol)
		}
	}
	volBtn := base.Button{
		Ref:    volBtnRef,
		Label:  i18n.T("Volume"),
		TestID: "volume-toggle",
		OnPress: func(core.PressEvent) {
			cancelVolClose()
			updateAnchor()
			volOpen.Update(func(v bool) bool { return !v })
		},
		Style: core.Style{
			Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 15,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Transition: core.Transition{Duration: 140 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: core.RGBA(0xffffff, .10)},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.88}},
		},
		Children: []core.Node{ui.IconOf(volIcon, 16, white)},
	}

	trackRef := &core.Ref{}
	const trackTopPad core.Points = 8
	const trackBotPad core.Points = 8
	volForPageY := func(pageY core.Points) (float64, bool) {
		bounds, ok := trackRef.Bounds()
		if !ok || bounds.Size.Height <= trackTopPad+trackBotPad {
			return 0, false
		}
		usable := bounds.Size.Height - trackTopPad - trackBotPad
		rel := min(max(pageY-bounds.Origin.Y-trackTopPad, 0), usable)
		return float64(1 - rel/usable), true
	}
	var panStartVol float64
	volTrack := core.View{
		Ref:    trackRef,
		TestID: "volume",
		Label:  i18n.T("Volume"),
		Style:  core.Style{Width: core.Pt(40), Height: core.Pt(114), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
		OnPress: func(e core.PressEvent) {
			if v, ok := volForPageY(e.PagePosition.Y); ok {
				m.setVolume(v)
			}
		},
		PanActivation: core.PanActivation{ActiveOffsetY: 2},
		OnPan: func(e core.PanEvent) {
			bounds, ok := trackRef.Bounds()
			if !ok || bounds.Size.Height <= trackTopPad+trackBotPad {
				return
			}
			usable := float64(bounds.Size.Height - trackTopPad - trackBotPad)
			switch e.Phase {
			case core.GestureBegan:
				volDragging = true
				syncVolOpen()
				panStartVol = m.prefs.Get().Volume
			case core.GestureChanged:
				next := min(1, max(0, panStartVol-float64(e.Translation.Y)/usable))
				m.setVolume(next)
			case core.GestureEnded:
				next := min(1, max(0, panStartVol-float64(e.Translation.Y)/usable))
				m.setVolume(next)
				volDragging = false
				syncVolOpen()
			case core.GestureCancelled:
				volDragging = false
				syncVolOpen()
			}
		},
		Children: []core.Node{
			core.Canvas{
				Style: core.Style{Width: core.Pt(40), Height: core.Pt(114)},
				Draw: func(d *core.Drawing, size core.Size) {
					v := float32(min(1, max(0, m.prefs.Get().Volume)))
					cxPos := float32(size.Width) / 2
					topY := float32(trackTopPad)
					botY := float32(size.Height) - float32(trackBotPad)
					usableH := botY - topY
					if usableH <= 4 {
						return
					}
					thumbY := topY + (1-v)*usableH
					const trackW float32 = 3.2
					// Unfilled vertical track.
					d.FillFromView(core.PaintColor(core.RGBA(0xffffff, .24)))
					d.BeginPath()
					d.RoundRect(cxPos-trackW/2, topY, trackW, usableH, trackW/2)
					d.Fill()
					// Filled [untitled] warm-yellow vertical bar.
					if botY-thumbY > 0.5 {
						d.FillFromView(core.PaintColor(core.RGB(0xf5d033)))
						d.BeginPath()
						d.RoundRect(cxPos-trackW/2, thumbY, trackW, botY-thumbY, trackW/2)
						d.Fill()
					}
					// Warm-yellow circular thumb.
					const thumbR float32 = 5.5
					d.FillFromView(core.PaintColor(core.RGB(0xf5d033)))
					d.BeginPath()
					d.RoundRect(cxPos-thumbR, thumbY-thumbR, thumbR*2, thumbR*2, thumbR)
					d.Fill()
				},
			},
		},
	}
	volCapIcon := base.Button{
		Label:   i18n.T("Mute"),
		OnPress: func(core.PressEvent) { toggleMute() },
		Style: core.Style{
			Width: core.Pt(34), Height: core.Pt(34), CornerRadius: 17,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Hover:  &core.Style{Background: core.RGBA(0xffffff, .10)},
			Active: &core.Style{Transform: core.Transform{Scale: 0.90}},
		},
		Children: []core.Node{ui.IconOf(volIcon, 16, func() core.Color { return core.RGB(0xd4d4d8) })},
	}
	const volCapW core.Points = 44
	const volCapH core.Points = 174
	volOverlay := core.Overlay{
		PointerEvents: core.PointerEventsBoxNone,
		Children: []core.Node{
			core.View{
				OnMouseEnter: func(core.MouseEvent) {
					volPopHovered = true
					syncVolOpen()
				},
				OnMouseLeave: func(core.MouseEvent) {
					volPopHovered = false
					syncVolOpen()
				},
				Style: core.Style{
					Position: core.Absolute,
					Width:    core.Pt(volCapW), Height: core.Pt(volCapH), CornerRadius: 22,
					Padding: core.Edges{Top: 14, Bottom: 6}, Gap: 6,
					Background:      core.RGBA(0x4c4c52, .80),
					BackdropFilters: []core.Filter{core.Blur(24), core.Saturate(1.45)},
					Gradients:       []core.Gradient{core.LinearGradient(180, core.RGBA(0xffffff, .10), core.RGBA(0xffffff, .02))},
					BorderWidth:     1, BorderColor: core.RGBA(0xffffff, .16),
					AlignItems: core.AlignCenter, JustifyContent: core.JustifyEnd,
					ShadowColor: core.RGBA(0x000000, .28), ShadowOffset: core.Point{Y: 6}, ShadowRadius: 18,
					Transition: core.Transition{Duration: 200 * time.Millisecond, Easing: core.Curve{X1: .22, Y1: 1, X2: .36, Y2: 1}},
				},
				StyleFunc: func() core.Style {
					b := volAnchor.Get()
					left := b.Origin.X + (b.Size.Width-volCapW)/2
					top := b.Origin.Y + b.Size.Height + 8 - volCapH
					if volOpen.Get() && b.Size.Width > 0 && m.playing.Get().track != nil {
						return core.Style{
							Left: core.Pt(left), Top: core.Pt(top),
							Opacity: core.Alpha(1), Transform: core.Transform{TranslateY: 0, Scale: 1},
						}
					}
					return core.Style{
						Left: core.Pt(left), Top: core.Pt(top),
						Opacity: core.Alpha(0), Transform: core.Transform{TranslateY: 14, Scale: 0.92},
						PointerEvents: core.PointerEventsNone,
					}
				},
				Children: []core.Node{volTrack, volCapIcon},
			},
		},
	}
	volWrap := core.View{
		OnMouseEnter: func(core.MouseEvent) {
			volBtnHovered = true
			syncVolOpen()
		},
		OnMouseLeave: func(core.MouseEvent) {
			volBtnHovered = false
			syncVolOpen()
		},
		Style:    core.Style{Width: core.Pt(30), Height: core.Pt(30), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
		Children: []core.Node{volBtn, volOverlay},
	}

	collapseBtn := btn(lucide.Minimize2, i18n.T("Collapse Playbar"), "playbar-collapse", 14.5, func() core.Color { return core.RGB(0xb4b4ba) }, func() {
		cancelVolClose()
		volOpen.Set(false)
		if onCollapse != nil {
			onCollapse()
		}
	})

	controls := row(prev, play, next, shuffle, repeat, queueBtn, m.remoteDevicePicker(cx), volWrap, collapseBtn)
	controls.Style.Gap = 5
	return controls
}

func (m *model) playbarCompactRight(cx *core.Scope, spinAngle *core.Signal[float32], onExpand func()) core.Node {
	// 3-bar warm-yellow animated equalizer synced with spinAngle while playing.
	eq := core.Canvas{
		Style: core.Style{Width: core.Pt(14), Height: core.Pt(14)},
		Draw: func(d *core.Drawing, size core.Size) {
			playing := m.playing.Get().playing
			phase := float64(spinAngle.Get()) * math.Pi / 28.0
			d.FillFromView(core.PaintColor(core.RGB(0xf5d033)))
			const barW float32 = 2.4
			const gap float32 = 2.4
			h := float32(size.Height)
			for i := 0; i < 3; i++ {
				frac := float32(0.28)
				if playing {
					frac = 0.32 + 0.68*float32(0.5+0.5*math.Sin(phase+float64(i)*1.9))
				}
				bh := max(float32(3.5), frac*h)
				x := float32(i) * (barW + gap)
				y := h - bh
				d.BeginPath()
				d.RoundRect(x, y, barW, bh, barW/2)
				d.Fill()
			}
		},
	}

	playBtn := base.Button{
		Label:   i18n.T("Play/Pause"),
		TestID:  "playbar-compact-play",
		OnPress: func(core.PressEvent) { m.playback(m.player.TogglePlay)() },
		Style: core.Style{
			Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 15,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Transition: core.Transition{Duration: 140 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: core.RGBA(0xffffff, .10)},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.88}},
		},
		Children: []core.Node{ui.IconOf(func() *core.SVG {
			if m.playing.Get().playing {
				return playbarPauseSVG
			}
			return playFillSVG
		}, 15, func() core.Color { return core.RGB(0xffffff) })},
	}

	expandBtn := base.Button{
		Label:  i18n.T("Expand Playbar"),
		TestID: "playbar-expand",
		OnPress: func(core.PressEvent) {
			if onExpand != nil {
				onExpand()
			}
		},
		Style: core.Style{
			Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 15,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Transition: core.Transition{Duration: 140 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: core.RGBA(0xffffff, .10)},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.88}},
		},
		Children: []core.Node{ui.Icon(lucide.Maximize2, 14.5, func() core.Color { return core.RGB(0xb4b4ba) })},
	}

	right := row(eq, playBtn, expandBtn)
	right.Style.Gap = 6
	return right
}

func (m *model) playerBar(cx *core.Scope) core.Node {
	smooth := core.Curve{X1: .22, Y1: 1, X2: .36, Y2: 1}
	frost := []core.Filter{core.Blur(24), core.Saturate(1.45)}
	sheen := []core.Gradient{core.LinearGradient(180, core.RGBA(0xffffff, .09), core.RGBA(0xffffff, .01))}
	barBg := func() core.Color {
		if cx.Dark() {
			return core.RGBA(0x202024, .76)
		}
		return core.RGBA(0x18181b, .80)
	}
	barBorder := func() core.Color {
		if cx.Dark() {
			return core.RGBA(0xffffff, .16)
		}
		return core.RGBA(0xffffff, .14)
	}
	shadowColor := func() core.Color {
		if cx.Dark() {
			return core.RGBA(0x000000, .48)
		}
		return core.RGBA(0x000000, .24)
	}

	const morphDuration = 340 * time.Millisecond
	dockExpanded := core.NewSignal(cx, true)
	animating := core.NewSignal(cx, false)
	morph := core.NewAnimatedValue(cx, 1)
	expandedOpacity := core.Interpolate(morph, core.Interpolation{
		Input: []float64{.2, .8}, Output: []float64{0, 1}, Extrapolate: core.ExtrapolateClamp,
	})
	compactOpacity := core.Sub(core.Constant(1), expandedOpacity)
	setExpanded := func(want bool) {
		if dockExpanded.GetUntracked() == want {
			return
		}
		to := 0.0
		if want {
			to = 1.0
		}
		if cx.Env().ReduceMotion() {
			dockExpanded.Set(want)
			animating.Set(false)
			morph.Set(to)
			return
		}
		// Commit the destination layout once. The platform interpolates its
		// frames and the controls' opacity on the same frame clock.
		cx.AnimateLayout(core.LayoutAnimation{Update: &core.LayoutTransition{
			Duration: morphDuration, Curve: core.LayoutEaseInEaseOut,
		}})
		dockExpanded.Set(want)
		animating.Set(true)
		cx.StartAnimation(core.Timing(morph, to, core.TimingConfig{
			Duration: morphDuration, Easing: core.BezierEasing(core.EaseInOut),
		}), func(finished bool) {
			if finished {
				animating.Set(false)
			}
		})
	}

	// Idle centered "+ Add" pill (shown on Local Music when no track is playing).
	idleAddVisible := func() bool {
		if m.playing.Get().track != nil {
			return false
		}
		if m.source.Get() != local.ID {
			return false
		}
		pg := m.page.Get().page
		return pg != "settings" && pg != "plugin-market"
	}
	addPill := base.Button{
		Label:    i18n.T("Add a Music Folder"),
		TestID:   "bottom-add",
		Disabled: func() bool { return !idleAddVisible() },
		OnPress:  func(core.PressEvent) { m.pickMusic(cx) },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(50), Padding: core.Symmetric(0, 28),
			CornerRadius: 25, Background: barBg(), BackdropFilters: frost, Gradients: sheen,
			BorderWidth: 1, BorderColor: barBorder(),
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 8,
			ShadowColor: shadowColor(), ShadowOffset: core.Point{Y: 8}, ShadowRadius: 22,
			Transition: core.Transition{Duration: 280 * time.Millisecond, Easing: smooth},
			Hover:      &core.Style{Transform: core.Transform{Scale: 1.03}},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.96}},
		},
		StyleFunc: func() core.Style {
			if idleAddVisible() {
				return core.Style{
					Opacity:   core.Alpha(1),
					Transform: core.Transform{Scale: 1, TranslateY: 0},
				}
			}
			return core.Style{
				Opacity:       core.Alpha(0),
				Transform:     core.Transform{Scale: 0.86, TranslateY: 10},
				PointerEvents: core.PointerEventsNone,
			}
		},
		Children: []core.Node{
			ui.Icon(lucide.Plus, 16, func() core.Color { return core.RGB(0xffffff) }),
			label(i18n.T("Add"), 14.5, core.FontWeightSemibold, core.RGB(0xffffff)),
		},
	}
	idleLayer := core.View{
		PointerEvents: core.PointerEventsBoxNone,
		Style: core.Style{
			Position: core.Absolute, Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(4),
			AlignItems: core.AlignCenter,
		},
		Children: []core.Node{addPill},
	}

	spinAngle := m.playbarSpinAngle(cx)
	hasTrack := func() bool {
		track := m.playing.Get().track
		if track == nil {
			return false
		}
		if src := m.activeSource(); src != nil && src.Info().ID != local.ID && src.Can("signIn") && m.accountOf(src).account == nil {
			return false
		}
		if track.Source != "" && track.Source != local.ID {
			if src := m.sourceByID(track.Source); src != nil && src.Can("signIn") && m.accountOf(src).account == nil {
				return false
			}
		}
		return true
	}

	// Expanded body (waveform + full controls + collapse button) smoothly fades out during collapse.
	expandedBody := row(
		m.playbarWaveform(cx),
		m.playbarControls(cx, func() { setExpanded(false) }),
	)
	expandedBody.TestID = "playbar-expanded-controls"
	expandedBody.Style.FlexGrow = 1
	expandedBody.Style.FlexShrink = 1
	expandedBody.Style.MinWidth = core.Pt(0)
	expandedBody.Style.Gap = 14
	expandedBody.Style.Animated = &core.AnimatedStyle{Opacity: expandedOpacity}
	expandedBody.StyleFunc = func() core.Style {
		expanded := dockExpanded.Get()
		s := core.Style{}
		if !expanded || animating.Get() {
			s.Overflow = core.Hidden
		}
		if !expanded {
			s.PointerEvents = core.PointerEventsNone
		}
		return s
	}

	// Compact right controls (equalizer + play/pause + expand button) anchored to the right edge of the same 60pt pill.
	compactRight := core.View{
		TestID: "playbar-compact-controls",
		Style: core.Style{
			Position: core.Absolute, Right: core.Pt(12), Top: core.Pt(0), Bottom: core.Pt(0),
			JustifyContent: core.JustifyCenter,
			Animated: &core.AnimatedStyle{
				Opacity: compactOpacity, TranslateX: core.Mul(expandedOpacity, core.Constant(8)),
			},
		},
		StyleFunc: func() core.Style {
			if dockExpanded.Get() {
				return core.Style{PointerEvents: core.PointerEventsNone}
			}
			return core.Style{}
		},
		Children: []core.Node{m.playbarCompactRight(cx, spinAngle, func() { setExpanded(true) })},
	}

	// Single physical 60pt-tall frosted glass Playbar pill that smoothly morphs its width around center.
	const compactW core.Points = 328
	pill := core.View{
		TestID: "playbar-surface",
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter,
			Height: core.Pt(60), CornerRadius: 30,
			Background: barBg(), BackdropFilters: frost, Gradients: sheen,
			BorderWidth: 1, BorderColor: barBorder(),
			Padding: core.Edges{Left: 8, Right: 14}, Gap: 14,
			ShadowColor: shadowColor(), ShadowOffset: core.Point{Y: 10}, ShadowRadius: 26,
		},
		StyleFunc: func() core.Style {
			if !dockExpanded.Get() {
				return core.Style{Width: core.Pt(compactW)}
			}
			winW := cx.Env().Window().Width
			maxExpanded := core.Points(900)
			if winW > 0 {
				avail := winW - 128
				if m.source.Get() != local.ID {
					avail = winW - 56
				}
				maxExpanded = min(maxExpanded, max(compactW, avail))
			}
			return core.Style{Width: core.Pt(maxExpanded)}
		},
		Children: []core.Node{
			m.playbarTrackInfo(cx, spinAngle),
			expandedBody,
			compactRight,
		},
	}

	// Right circular "+" button smoothly shrinks/fades out alongside the pill so the dock stays dead-center.
	showCircleAdd := func() bool { return hasTrack() && dockExpanded.Get() && m.source.Get() == local.ID }
	addCircle := core.Show{
		When: func() bool { return m.source.Get() == local.ID },
		Then: func(cx *core.Scope) core.Node {
			return core.View{
				Style: core.Style{
					Height: core.Pt(60), JustifyContent: core.JustifyCenter, AlignItems: core.AlignCenter,
					Animated: &core.AnimatedStyle{
						Opacity: morph, Scale: core.Add(core.Constant(.85), core.Mul(morph, core.Constant(.15))),
					},
				},
				StyleFunc: func() core.Style {
					if dockExpanded.Get() {
						return core.Style{Width: core.Pt(60), Margin: core.Edges{Left: 10}}
					}
					return core.Style{Width: core.Pt(0), PointerEvents: core.PointerEventsNone}
				},
				Children: []core.Node{
					base.Button{
						Label:    i18n.T("Add a Music Folder"),
						TestID:   "playbar-add",
						Disabled: func() bool { return !showCircleAdd() },
						OnPress:  func(core.PressEvent) { m.pickMusic(cx) },
						Style: core.Style{
							Width: core.Pt(60), Height: core.Pt(60), CornerRadius: 30,
							Background: barBg(), BackdropFilters: frost, Gradients: sheen,
							BorderWidth: 1, BorderColor: barBorder(),
							AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
							ShadowColor: shadowColor(), ShadowOffset: core.Point{Y: 10}, ShadowRadius: 26,
							Transition: core.Transition{Duration: 180 * time.Millisecond, Easing: smooth},
							Hover:      &core.Style{Transform: core.Transform{Scale: 1.05}},
							Active:     &core.Style{Transform: core.Transform{Scale: 0.94}},
						},
						Children: []core.Node{ui.Icon(lucide.Plus, 21, func() core.Color { return core.RGB(0xffffff) })},
					},
				},
			}
		},
	}

	activeDock := row(pill, addCircle)
	activeDock.PointerEvents = core.PointerEventsBoxNone
	activeDock.Style.Gap = 0
	activeDock.Style.JustifyContent = core.JustifyCenter
	activeDock.Style.Transition = core.Transition{Duration: 340 * time.Millisecond, Easing: smooth}
	activeDock.StyleFunc = func() core.Style {
		if hasTrack() {
			return core.Style{
				Opacity:   core.Alpha(1),
				Transform: core.Transform{TranslateY: 0, Scale: 1},
			}
		}
		return core.Style{
			Opacity:       core.Alpha(0),
			Transform:     core.Transform{TranslateY: 24, Scale: 0.94},
			PointerEvents: core.PointerEventsNone,
		}
	}

	// Subtle progressive bottom frosted mist so scrolling covers dissolve gracefully under the dock.
	bottomMist := core.View{
		PointerEvents: core.PointerEventsNone,
		Style: core.Style{
			Position: core.Absolute, Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(-20),
			Height:          core.Pt(72),
			BackdropFilters: []core.Filter{core.Blur(12)},
			Mask: &core.Gradient{
				Kind:  core.GradientLinear,
				Angle: 0,
				Stops: []core.ColorStop{
					{Color: core.RGBA(0x000000, 1), At: core.Pct(0)},
					{Color: core.RGBA(0x000000, .65), At: core.Pct(45)},
					{Color: core.RGBA(0x000000, 0), At: core.Pct(100)},
				},
			},
		},
		StyleFunc: func() core.Style {
			if !hasTrack() && !idleAddVisible() {
				return core.Style{Opacity: core.Alpha(0)}
			}
			bgHex := uint32(0xffffff)
			if cx.Dark() {
				bgHex = 0x151517
			}
			return core.Style{
				Opacity: core.Alpha(1),
				Gradients: []core.Gradient{{
					Kind:  core.GradientLinear,
					Angle: 0,
					Stops: []core.ColorStop{
						{Color: core.RGBA(bgHex, .72), At: core.Pct(0)},
						{Color: core.RGBA(bgHex, .32), At: core.Pct(55)},
						{Color: core.RGBA(bgHex, 0), At: core.Pct(100)},
					},
				}},
			}
		},
	}

	return core.View{
		PointerEvents: core.PointerEventsBoxNone,
		Style: core.Style{
			Position: core.Absolute, Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(20),
			Padding: core.Symmetric(0, 24), AlignItems: core.AlignCenter,
		},
		Children: []core.Node{bottomMist, idleLayer, activeDock},
	}
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
							return m.songRow(cx, state.queue[j], i, nil, true, func() { m.player.Jump(j) })
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
