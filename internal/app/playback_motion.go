package app

import (
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

func (m *model) closePlaybackPage() {
	if m.stageDismiss != nil {
		m.stageDismiss()
	} else {
		m.stage.Set("")
	}
}

func (m *model) playbackMotion(cx *core.Scope) (func() core.Style, func(core.PanEvent)) {
	offset := core.NewSignal(cx, core.Points(0))
	var stop func()
	var closing bool
	move := func(to core.Points, duration time.Duration, done func()) {
		if stop != nil {
			stop()
		}
		from := offset.GetUntracked()
		if cx.Env().ReduceMotion() || from == to {
			offset.Set(to)
			if done != nil {
				done()
			}
			return
		}
		stop = cx.Animate(duration, core.Curve{X1: .16, Y1: 1, X2: .3, Y2: 1}, func(t float64) {
			offset.Set(from + (to-from)*core.Points(t))
			if t >= 1 && done != nil {
				done()
			}
		})
	}
	if !cx.Env().ReduceMotion() {
		offset.Set(50)
		move(0, 280*time.Millisecond, func() { m.stageCovered.Set(true) })
	} else {
		m.stageCovered.Set(true)
	}
	m.stageDismiss = func() {
		if closing {
			return
		}
		closing = true
		m.stageCovered.Set(false)
		move(max(110, offset.GetUntracked()+60), 180*time.Millisecond, func() { m.stage.Set("") })
	}
	cx.Cleanup(func() {
		m.stageDismiss = nil
		m.stageCovered.Set(false)
	})
	return func() core.Style {
			y := offset.Get()
			return core.Style{Transform: core.Transform{TranslateY: y}, Opacity: core.Alpha(float32(max(.10, 1-y/150)))}
		}, func(e core.PanEvent) {
			if closing || cx.Env().ReduceMotion() {
				return
			}
			switch e.Phase {
			case core.GestureBegan:
				if stop != nil {
					stop()
				}
			case core.GestureChanged:
				offset.Set(max(0, e.Translation.Y))
			case core.GestureEnded:
				if offset.GetUntracked() > 85 || e.Velocity.Y > 550 {
					m.closePlaybackPage()
				} else {
					move(0, 230*time.Millisecond, func() { m.stageCovered.Set(true) })
				}
			case core.GestureCancelled:
				move(0, 230*time.Millisecond, func() { m.stageCovered.Set(true) })
			}
		}
}
