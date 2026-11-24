package app

import (
	"math"
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"yoho-music/internal/music"
)

func (m *model) playerArtwork(cx *core.Scope, track *music.Track, style string, size core.Points) core.Node {
	image := track.Album.ImageURL
	cover := m.artwork(cx, image, size, track.Source)
	if style != "vinyl" && style != "cassette" {
		return cover
	}
	ready := core.NewSignal(cx, false)
	go func() {
		retroBitmaps()
		cx.Post(func() { ready.Set(true) })
	}()
	return core.Show{When: ready.Get, Else: func(*core.Scope) core.Node { return cover }, Then: func(cx *core.Scope) core.Node {
		pos := m.playbackClock(cx)
		bitmaps := retroBitmaps()
		coverSource := m.coverSource(cx, image, int(size*2), track.Source)
		height, title := float32(size), i18n.T("Turntable")
		if style == "cassette" {
			height, title = float32(size)*tapeHeight, i18n.T("Cassette")
		}
		playing := core.NewMemo(cx, func() bool { return m.playing.Get().playing })
		arm := core.NewSignal(cx, 0.0)
		var stop func()
		cx.Effect(func() {
			target := 0.0
			if playing.Get() {
				target = 1
			}
			if stop != nil {
				stop()
			}
			from := arm.GetUntracked()
			if cx.Env().ReduceMotion() || style != "vinyl" || from == target {
				arm.Set(target)
				return
			}
			stop = cx.Animate(700*time.Millisecond, core.Curve{X1: .18, Y1: 1, X2: .35, Y2: 1}, func(f float64) {
				arm.Set(from + (target-from)*f)
			})
		})
		canvas := core.Canvas{Label: title, Style: core.Style{Width: core.Pt(size), Height: core.Pt(core.Points(height))}, Draw: func(d *core.Drawing, bounds core.Size) {
			s := float32(bounds.Width)
			position := pos.Get()
			progress := 0.0
			if length := m.playing.Get().duration; length > 0 {
				progress = min(1, max(0, float64(position)/float64(length)))
			}
			seconds := position.Seconds()
			if cx.Env().ReduceMotion() {
				seconds = 0
			}
			if style == "vinyl" {
				drawTurntable(d, bitmaps, coverSource(), s, seconds, progress, arm.Get())
			} else {
				drawCassette(d, bitmaps, coverSource(), s, seconds, progress)
			}
		}}
		if style != "cassette" {
			return canvas
		}
		trackLabel := func(text string, top, font core.Points, weight core.FontWeight) core.Node {
			return core.Text{Content: text, MaxLines: 1, PointerEvents: core.PointerEventsNone, Style: core.Style{
				Position: core.Absolute, Left: core.Pt(size * .1), Right: core.Pt(size * .1), Top: core.Pt(top),
				FontSize: font, FontWeight: weight, FontStyle: core.FontStyleItalic, Color: core.RGB(0x1d2a6b),
			}}
		}
		return core.View{Children: []core.Node{canvas,
			trackLabel(track.Name, size*.103, size*.044, core.FontWeightBold),
			trackLabel(track.ArtistNames(), size*.172, size*.03, core.FontWeightMedium)}}
	}}
}

func drawTurntable(d *core.Drawing, parts retroArt, cover core.ImageSource, s float32, seconds, progress, on float64) {
	d.DrawImageIn(core.ImageSource{Pixels: parts.deck}, 0, 0, s, s)
	cx, cy, radius := deckCX*s, deckCY*s, deckRecord*deckLabel*s
	d.Save().Translate(cx, cy).Rotate(float32(math.Mod(seconds*200, 360) * math.Pi / 180))
	d.BeginPath().Ellipse(0, 0, radius, radius).Clip().SetFillColor(core.RGB(0xb84460)).FillRect(-radius, -radius, 2*radius, 2*radius)
	if cover.URI != "" || len(cover.Data) > 0 {
		d.DrawImageIn(cover, -radius, -radius, 2*radius, 2*radius)
	} else {
		d.SetFillColor(core.RGBA(0xffffff, .24)).FillRect(-radius, -radius*.13, 2*radius, radius*.26)
	}
	d.Restore()
	d.BeginPath().Ellipse(cx, cy, s*.012, s*.012).SetFillColor(core.RGB(0x99999e)).Fill()
	if on > .01 {
		d.BeginPath().Ellipse(deckLampX*s, deckLampY*s, s*.011, s*.011).SetFillColor(core.RGBA(0xff6048, float32(on))).Fill()
	}
	stx, sty := armStylus()
	reach, offset := math.Hypot(stx, sty), math.Atan2(sty, stx)
	distance := math.Hypot(deckCX-deckPivotX, deckCY-deckPivotY)
	needle := deckRecord * (.955 + (deckLabel*1.2-.955)*progress)
	cos := (reach*reach + distance*distance - needle*needle) / (2 * reach * distance)
	played := math.Atan2(deckCY-deckPivotY, deckCX-deckPivotX) - math.Acos(max(-1, min(1, cos))) - offset
	angle := float32(deckRest + (played-deckRest)*on)
	px, py := deckPivotX*s, deckPivotY*s
	lift := s * (.01 + .016*float32(1-on))
	d.Save().Translate(px+lift*.7, py+lift).Rotate(angle).
		DrawImageIn(core.ImageSource{Pixels: parts.armShadow}, -armBack*s, -armSide*s, (armBack+armFront)*s, 2*armSide*s).Restore()
	d.Save().Translate(px, py).Rotate(angle).
		DrawImageIn(core.ImageSource{Pixels: parts.arm}, -armBack*s, -armSide*s, (armBack+armFront)*s, 2*armSide*s).Restore()
}

func drawCassette(d *core.Drawing, parts retroArt, cover core.ImageSource, s float32, seconds, progress float64) {
	d.Save().BeginPath().RoundRect(.2*s, .24*s, .6*s, .15*s, .075*s).Clip()
	d.SetFillColor(core.RGB(0x151517)).FillRect(.2*s, .24*s, .6*s, .15*s)
	for i, x := range []float32{tapeReelL * s, tapeReelR * s} {
		y := float32(tapeReelY) * s
		fraction := []float64{1 - progress, progress}[i]
		radius := float32(math.Sqrt(tapeHub*tapeHub+(tapeFull*tapeFull-tapeHub*tapeHub)*fraction)) * s
		d.BeginPath().Ellipse(x, y, radius, radius).SetFillColor(core.RGB(0x382819)).Fill()
		var hubSize = float32(parts.hub.Bounds().Dx()) / tapePixels * s
		d.Save().Translate(x, y).Rotate(float32(math.Mod(seconds*150, 360)*math.Pi/180)).
			DrawImageIn(core.ImageSource{Pixels: parts.hub}, -hubSize/2, -hubSize/2, hubSize, hubSize).Restore()
	}
	d.Restore()
	d.DrawImageIn(core.ImageSource{Pixels: parts.cassette}, 0, 0, s, s*tapeHeight)
	d.SetFillColor(core.RGB(0xa44657)).FillRect(.081*s, .399*s, .838*s, .018*s)
	d.SetFillColor(core.RGB(0xc4747f)).FillRect(.081*s, .418*s, .838*s, .009*s)
	if cover.URI != "" || len(cover.Data) > 0 {
		d.DrawImageIn(cover, .088*s, .262*s, .095*s, .095*s)
	}
}

func (m *model) dragArtwork(cx *core.Scope, content core.Node) core.Node {
	offset := core.NewSignal(cx, core.Point{})
	var start core.Point
	var stop func()
	return core.View{TestID: "draggable-artwork", Children: []core.Node{content}, StyleFunc: func() core.Style {
		p := offset.Get()
		return core.Style{Transform: core.Transform{TranslateX: p.X, TranslateY: p.Y, Rotate: float32(p.X) * .15,
			RotateX: -float32(p.Y) * .16, RotateY: float32(p.X) * .16, Perspective: 650}}
	}, OnPan: func(event core.PanEvent) {
		if cx.Env().ReduceMotion() {
			return
		}
		switch event.Phase {
		case core.GestureBegan:
			if stop != nil {
				stop()
			}
			start = offset.GetUntracked()
		case core.GestureChanged:
			offset.Set(core.Point{X: start.X + event.Translation.X*.35, Y: start.Y + event.Translation.Y*.35})
		case core.GestureEnded, core.GestureCancelled:
			from := offset.GetUntracked()
			velocity := event.Velocity
			stop = cx.Animate(600*time.Millisecond, core.Curve{X1: 0, Y1: 0, X2: 1, Y2: 1}, func(t float64) {
				rest := core.Points(math.Pow(1-t, 3))
				overshoot := core.Points(math.Sin(t*math.Pi) * float64(rest) * .12)
				offset.Set(core.Point{X: from.X*rest + velocity.X*overshoot, Y: from.Y*rest + velocity.Y*overshoot})
			})
		}
	}}
}

func (m *model) coverWall(cx *core.Scope, state playbackView) core.Node {
	tracks := state.queue
	if len(tracks) == 0 {
		return m.empty(cx, lucide.LayoutGrid, i18n.T("No Covers"), "", nil)
	}
	const tile, pitch = 124, 138
	const inset = 28
	current := -1
	if state.index >= 0 && state.index < len(state.order) {
		current = state.order[state.index]
	}
	bounds := core.NewSignal(cx, core.Size{})
	offset := core.NewSignal(cx, core.Point{})
	p := colors(cx)
	var start core.Point
	var stop func()
	columns := core.NewMemo(cx, func() int {
		fit := max(1, int((bounds.Get().Width-2*inset)/pitch))
		return min(len(tracks), max(fit, int(math.Ceil(math.Sqrt(float64(len(tracks))*2.5)))))
	})
	clamp := func(at core.Point) core.Point {
		size, cols := bounds.GetUntracked(), columns.Get()
		rows := (len(tracks) + cols - 1) / cols
		x, y := core.Points(cols*pitch-(pitch-tile)), core.Points(rows*pitch-(pitch-tile))
		axis := func(value, viewport, content core.Points) core.Points {
			if content+2*inset <= viewport {
				return (viewport - content) / 2
			}
			return max(viewport-inset-content, min(inset, value))
		}
		return core.Point{X: axis(at.X, size.Width, x), Y: axis(at.Y, size.Height, y)}
	}
	glide := func(to core.Point) {
		if stop != nil {
			stop()
		}
		to = clamp(to)
		from := offset.GetUntracked()
		if from == to || cx.Env().ReduceMotion() {
			offset.Set(to)
			return
		}
		stop = cx.Animate(480*time.Millisecond, core.Curve{X1: .16, Y1: 1, X2: .3, Y2: 1}, func(t float64) {
			f := core.Points(t)
			offset.Set(core.Point{X: from.X + (to.X-from.X)*f, Y: from.Y + (to.Y-from.Y)*f})
		})
	}
	center := func() {
		size, cols := bounds.GetUntracked(), columns.Get()
		index := max(0, current)
		glide(core.Point{X: (size.Width-tile)/2 - core.Points(index%cols*pitch), Y: (size.Height-tile)/2 - core.Points(index/cols*pitch)})
	}
	type wallWindow struct{ cols, x0, x1, y0, y1 int }
	visible := core.NewMemo(cx, func() wallWindow {
		cols, size, at := columns.Get(), bounds.Get(), offset.Get()
		return wallWindow{cols: cols, x0: max(0, int(-at.X/pitch)-1), x1: min(cols, int((size.Width-at.X)/pitch)+2),
			y0: max(0, int(-at.Y/pitch)-1), y1: min((len(tracks)+cols-1)/cols, int((size.Height-at.Y)/pitch)+2)}
	})
	tiles := core.Dynamic(func(cx *core.Scope) core.Node {
		f := visible.Get()
		var children []core.Node
		for y := f.y0; y < f.y1; y++ {
			for x := f.x0; x < f.x1; x++ {
				index := y*f.cols + x
				if index >= len(tracks) {
					break
				}
				track := tracks[index]
				style := core.Style{Position: core.Absolute, Left: core.Pt(core.Points(x * pitch)), Top: core.Pt(core.Points(y * pitch)),
					Width: core.Pt(tile), Height: core.Pt(tile), CornerRadius: 9, Overflow: core.Hidden,
					Hover: &core.Style{Transform: core.Transform{Scale: 1.05}}, Transition: core.Transition{Duration: 150 * time.Millisecond}}
				if index == current {
					style.BorderWidth = 2
					style.BorderColor = p.accent
				}
				cover := m.artwork(cx, track.Album.ImageURL, tile-4, track.Source)
				children = append(children, base.Button{Label: track.Name, Style: style, Children: []core.Node{cover},
					OnPress: func(core.PressEvent) { m.playback(func() { m.player.Jump(index) })() }})
			}
		}
		return core.View{Style: fill, StyleFunc: func() core.Style {
			at := offset.Get()
			return core.Style{Transform: core.Transform{TranslateX: at.X, TranslateY: at.Y}}
		}, Children: children}
	})
	return core.View{TestID: "cover-wall", Style: core.Style{FlexGrow: 1, FlexShrink: 1, Overflow: core.Hidden},
		Focusable: func() bool { return true }, Label: i18n.T("Cover Wall"),
		OnLayout: func(b core.Bounds) {
			if bounds.GetUntracked() != b.Size {
				first := bounds.GetUntracked().Width == 0
				bounds.Set(b.Size)
				if first {
					cols := columns.Get()
					index := max(0, current)
					offset.Set(clamp(core.Point{X: (b.Size.Width-tile)/2 - core.Points(index%cols*pitch), Y: (b.Size.Height-tile)/2 - core.Points(index/cols*pitch)}))
				} else {
					center()
				}
			}
		},
		OnPan: func(event core.PanEvent) {
			switch event.Phase {
			case core.GestureBegan:
				if stop != nil {
					stop()
				}
				start = offset.GetUntracked()
			case core.GestureChanged:
				offset.Set(clamp(core.Point{X: start.X + event.Translation.X, Y: start.Y + event.Translation.Y}))
			case core.GestureEnded:
				at := offset.GetUntracked()
				glide(core.Point{X: at.X + event.Velocity.X*.22, Y: at.Y + event.Velocity.Y*.22})
			}
		}, OnWheel: func(event core.WheelEvent) {
			if stop != nil {
				stop()
			}
			at := offset.GetUntracked()
			offset.Set(clamp(core.Point{X: at.X - event.Delta.X, Y: at.Y - event.Delta.Y}))
		},
		OnKeyDown: func(event core.KeyEvent) bool {
			at := offset.GetUntracked()
			switch event.Key {
			case "ArrowLeft":
				at.X += pitch
			case "ArrowRight":
				at.X -= pitch
			case "ArrowUp":
				at.Y += pitch
			case "ArrowDown":
				at.Y -= pitch
			default:
				return false
			}
			glide(at)
			return true
		}, Children: []core.Node{tiles},
	}
}
