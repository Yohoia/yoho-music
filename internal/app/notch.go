package app

import (
	"math"
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
)

const (
	notchTitle  = "Yoho Music Top Player"
	notchWidth  = 528
	notchHeight = 228
)

type notchMode uint8

const (
	notchClosed notchMode = iota
	notchCompact
	notchGreeting
	notchExpanded
)

type notchPresentation struct {
	mode          notchMode
	width, height core.Points
}

type notchScreen struct {
	cursor           core.Point // Relative to the top center of the menu-bar display.
	width, height    core.Points
	real, positioned bool
}

func (m *model) toggleNotch() {
	m.notchQuick.Update(func(open bool) bool { return !open })
}

func (m *model) bindNotch(cx *core.Scope, panel *desktop.Window) {
	var hovered, greetedUntil, changedAt time.Time
	var hadPinned bool
	var timer func() bool
	var probing bool
	var closed bool
	set := func(screen notchScreen, now time.Time) {
		prefs := m.prefs.GetUntracked()
		previous := m.notchState.GetUntracked()
		open := previous.mode == notchExpanded
		mode := notchClosed
		if screen.real && prefs.NotchEnabled || prefs.NotchVirtual && prefs.NotchEnabled {
			mode = notchCompact
		}
		if now.Before(greetedUntil) {
			mode = notchGreeting
		}
		width, height := screen.width, screen.height
		if width == 0 {
			width, height = 200, 28
		}
		if prefs.NotchActivity && m.playing.GetUntracked().track != nil {
			width += 2*max(16, height-10) + 16
		}
		if prefs.NotchEnabled && mode != notchGreeting {
			hitWidth, hitHeight := width, height
			if open {
				hitWidth, hitHeight = 468, screen.height+164
			} else if mode == notchClosed {
				hitHeight = 3
			}
			inside := math.Abs(float64(screen.cursor.X)) <= float64(hitWidth/2+5) && screen.cursor.Y >= -2 && screen.cursor.Y <= hitHeight+5
			if inside {
				if hovered.IsZero() {
					hovered = now
				}
				if open || now.Sub(hovered) >= 300*time.Millisecond {
					mode = notchExpanded
					changedAt = now
				}
			} else {
				hovered = time.Time{}
				if open && now.Sub(changedAt) < 300*time.Millisecond {
					mode = notchExpanded
				}
			}
		}
		if m.notchQuick.GetUntracked() {
			mode = notchExpanded
		}
		m.notchState.Set(notchPresentation{mode: mode, width: width, height: height})
		if mode == notchClosed {
			if panel.Visible() {
				panel.Hide()
			}
		} else if !panel.Visible() {
			if screen.positioned {
				panel.Show()
			} else {
				frame := m.display.GetUntracked()
				if frame.Size.Width >= notchWidth {
					at := core.Point{X: frame.Origin.X + frame.Size.Width/2, Y: frame.Origin.Y}
					m.report(panel.ShowAt(core.Bounds{Origin: at}, core.PlaceBottom))
				} else {
					panel.Show()
				}
			}
		}
	}
	var poll func()
	poll = func() {
		timer = nil
		if closed || probing {
			return
		}
		now := time.Now()
		if !m.prefs.GetUntracked().NotchEnabled && !m.notchQuick.GetUntracked() && !now.Before(greetedUntil) {
			set(notchScreen{}, now)
			return
		}
		probing = true
		sampleNotch(cx, func(screen notchScreen) {
			probing = false
			if !closed {
				set(screen, time.Now())
				timer = cx.AfterFunc(90*time.Millisecond, poll)
			}
		})
	}
	cx.Effect(func() {
		prefs := m.prefs.Get()
		if !prefs.NotchEnabled && !prefs.NotchGreeted && m.playing.Get().playing {
			greetedUntil = time.Now().Add(7 * time.Second)
			m.updatePreferences(cx, func(p *preferences) { p.NotchGreeted = true })
		}
		if m.notchQuick.Get() || prefs.NotchEnabled && !hadPinned {
			greetedUntil = time.Time{}
		}
		hadPinned = prefs.NotchEnabled
		if timer != nil {
			timer()
			timer = nil
		}
		poll()
	})
	cx.Cleanup(func() {
		closed = true
		if timer != nil {
			timer()
		}
	})
}

func (m *model) notchPlayer(cx *core.Scope) core.Node {
	desktop.SetTitle(cx, notchTitle)
	desktop.SetWindowWorkspaces(cx, desktop.WindowWorkspacesAll)
	desktop.SetWindowLevel(cx, desktop.WindowLevelFloating)
	cx.SetSubtreeAppearance(core.AppearanceDark)
	background := core.RGB(0x09090a)
	return core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignCenter}, Children: []core.Node{
		core.Dynamic(func(cx *core.Scope) core.Node {
			state := m.notchState.Get()
			if state.mode == notchClosed {
				return nil
			}
			width, height := state.width, state.height
			var children []core.Node
			switch state.mode {
			case notchExpanded:
				width, height = 468, state.height+149
				dismiss := iconButton(lucide.X, i18n.T("Close"), "close-top-player", func() {
					m.notchQuick.Set(false)
					m.notchState.Set(notchPresentation{})
					desktop.CurrentWindow(cx).Hide()
				})
				dismiss.Tint = func() core.Color { return core.RGB(0xffffff) }
				info := core.View{Style: fill, Children: []core.Node{m.trackInfo(cx, 42)}}
				children = []core.Node{row(info, m.transport(cx, false), dismiss), m.progress(cx)}
			case notchGreeting:
				width, height = 340, state.height+54
				keep := button(i18n.T("Keep Top Player"), "keep-top-player", lucide.Check, func() {
					m.updatePreferences(m.cx, func(p *preferences) { p.NotchEnabled = true })
				})
				info := core.View{Style: fill, Children: []core.Node{m.trackInfo(cx, 34)}}
				children = []core.Node{row(info, keep)}
			case notchCompact:
				if m.prefs.Get().NotchActivity && m.playing.Get().track != nil {
					track := m.playing.Get().track
					cover := m.artwork(cx, track.Album.ImageURL, max(12, height-10), track.Source)
					children = []core.Node{row(cover, spacer(), m.notchLevels(cx))}
				}
			}
			style := core.Style{Width: core.Pt(width), Height: core.Pt(height), Background: background,
				Overflow: core.Hidden, JustifyContent: core.JustifyEnd, Gap: 8,
				Padding: core.Symmetric(8, 11), CornerRadii: core.Corners{BottomLeft: 19, BottomRight: 19}}
			if state.mode == notchCompact {
				style.Padding = core.Symmetric(0, 12)
				style.JustifyContent = core.JustifyCenter
			}
			return core.View{Style: style, TestID: "top-player", Children: children}
		}),
	}}
}

func (m *model) notchLevels(cx *core.Scope) core.Node {
	position := m.playbackClock(cx)
	return core.Canvas{Style: core.Style{Width: core.Pt(27), Height: core.Pt(20)}, Draw: func(d *core.Drawing, _ core.Size) {
		_ = position.Get()
		levels, live := m.player.Levels()
		d.SetFillColor(core.RGB(0xe57689))
		for i := range 5 {
			height := float32(3)
			if live {
				height += min(15, levels[i*len(levels)/5]*15)
			}
			d.BeginPath().RoundRect(float32(i*5), (20-height)/2, 3, height, 1.5).Fill()
		}
	}}
}
