package app

import (
	"math"
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

// Ported from MyGo Music v0.1.25 internal/app/ui_like.go under AGPL-3.0.
// favoriteMotion gives hearts and library actions the same pop, ring and confetti.
func favoriteMotion(cx *core.Scope, id string, size, iconSize core.Points) (*core.AnimatedValue, func(bool), core.Node) {
	scale := core.NewAnimatedValue(cx, 1)
	burst := core.NewSignal(cx, time.Duration(-1))
	var stopFrame func()
	animationContext := cx.Context()
	animate := func(on bool) {
		if animationContext.Err() != nil {
			return
		}
		if stopFrame != nil {
			stopFrame()
		}
		burst.Set(-1)
		if cx.Env().ReduceMotion() {
			scale.Set(1)
			return
		}
		from := .8
		if on {
			from = .2
		}
		scale.Set(from)
		cx.StartAnimation(core.Spring(scale, 1, core.SpringConfig{Stiffness: 310, Damping: 13}), nil)
		if !on {
			return
		}
		start := cx.Now()
		burst.Set(0)
		var frame func(time.Time)
		frame = func(at time.Time) {
			elapsed := at.Sub(start)
			if elapsed >= 750*time.Millisecond {
				burst.Set(-1)
				return
			}
			burst.Set(elapsed)
			stopFrame = cx.RequestAnimationFrame(frame)
		}
		stopFrame = cx.RequestAnimationFrame(frame)
	}
	cx.Cleanup(func() {
		scale.Stop()
		if stopFrame != nil {
			stopFrame()
		}
	})
	particles := core.View{PointerEvents: core.PointerEventsNone, Style: core.Style{Position: core.Absolute,
		Left: core.Pt(size/2 - iconSize*2), Top: core.Pt(size/2 - iconSize*2), Width: core.Pt(iconSize * 4), Height: core.Pt(iconSize * 4)},
		Children: []core.Node{core.Canvas{TestID: id + "-burst", Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)},
			Draw: func(d *core.Drawing, bounds core.Size) { drawLikeBurst(d, bounds, float32(iconSize), burst.Get()) }}},
	}
	return scale, animate, particles
}

// The timing, expanding ring and two seven-dot rings follow MyGo Music's ui_like.go.
func drawLikeBurst(d *core.Drawing, bounds core.Size, size float32, elapsed time.Duration) {
	if elapsed < 0 || elapsed >= 750*time.Millisecond {
		return
	}
	t := float32(elapsed) / float32(750*time.Millisecond)
	x, y := float32(bounds.Width)/2, float32(bounds.Height)/2
	out := func(v float32) float32 { return 1 - (1-v)*(1-v)*(1-v) }
	if rt := t * 2; rt < 1 {
		radius := size * (.35 + .65*out(rt))
		d.SetStrokeColor(core.RGBA(0xff4d6d, .9*(1-rt))).SetLineWidth(size * .22 * (1 - rt)).
			StrokePath((&core.Path{}).Ellipse(x, y, radius, radius))
	}
	colors := [...]uint32{0xff4d6d, 0xff9f1c, 0xffd166, 0xc77dff, 0x4cc9f0, 0xff70a6}
	for ring := range 2 {
		lag := float32(ring) * .08
		pt := (t - .12 - lag) / (.88 - lag)
		if pt <= 0 {
			continue
		}
		pt = min(pt, 1)
		distance := size * (.55 + (1.15-.25*float32(ring))*out(pt))
		radius := size * (.16 - .04*float32(ring)) * (1 - pt*pt)
		for i := range 7 {
			angle := (float64(i)/7 + float64(ring)*.5/7 - .25) * 2 * math.Pi
			d.SetFillColor(core.RGBA(colors[(i+ring*3)%len(colors)], 1-pt*pt)).
				FillPath((&core.Path{}).Ellipse(x+distance*float32(math.Cos(angle)), y+distance*float32(math.Sin(angle)), radius, radius), core.NonZero)
		}
	}
}
