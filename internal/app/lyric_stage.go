package app

import (
	"math"
	"time"
	"unicode/utf8"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/lyrics"
)

func (m *model) playbackClock(cx *core.Scope) *core.Signal[time.Duration] {
	position := core.NewSignal(cx, m.pos.GetUntracked())
	var stop func()
	cx.Effect(func() {
		if stop != nil {
			stop()
			stop = nil
		}
		if !m.playing.Get().playing || cx.Env().ReduceMotion() {
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

func (m *model) lyricStageStyle(cx *core.Scope) core.Node {
	style := ui.Select[string]{
		Label: i18n.T("Stage Style"), TestID: "lyric-stage-style",
		Options: []ui.Option[string]{
			{Value: "", Label: i18n.T("Flow")},
			{Value: "partita", Label: i18n.T("Steps")},
			{Value: "tilt", Label: i18n.T("Tilt")},
		},
		Value:    func() string { return m.prefs.Get().StageStyle },
		OnChange: func(value string) { m.updatePreferences(cx, func(p *preferences) { p.StageStyle = value }) },
		Style:    core.Style{Width: core.Pt(100)},
	}
	sparks := iconButton(lucide.Sparkles, i18n.T("Toggle Lyric Sparks"), "lyric-stage-sparks", func() {
		m.updatePreferences(cx, func(p *preferences) { p.StageSparks = !p.StageSparks })
	})
	sparks.Tint = func() core.Color {
		p := colors(cx)
		if m.prefs.Get().StageSparks {
			return p.accent
		}
		return p.muted
	}
	return row(style, sparks)
}

func (m *model) lyricStage(cx *core.Scope, lines []lyrics.Line) core.Node {
	if len(lines) == 0 {
		return m.empty(cx, lucide.AudioLines, i18n.T("No lyrics. Enjoy the music."), "", nil)
	}
	p := colors(cx)
	position := m.playbackClock(cx)
	active := core.NewMemo(cx, func() int { return lyrics.Index(lines, position.Get()) })
	interlude := core.NewMemo(cx, func() bool {
		i, at := active.Get(), position.Get()
		return i < 0 || i+1 < len(lines) && at > lines[i].End+time.Second && lines[i+1].At-lines[i].End > 4*time.Second
	})
	body := core.Dynamic(func(cx *core.Scope) core.Node {
		i := active.Get()
		if interlude.Get() {
			dots := label("•••", 40, core.FontWeightBold, p.muted)
			dots.Style.TextAlign = core.TextAlignCenter
			dots.StyleFunc = func() core.Style {
				if cx.Env().ReduceMotion() {
					return core.Style{}
				}
				pulse := float32(.70 + .25*math.Sin(position.Get().Seconds()*math.Pi))
				return core.Style{Opacity: core.Alpha(pulse)}
			}
			dots.Label = i18n.T("Instrumental")
			return dots
		}
		line := lines[i]
		style := m.prefs.Get().StageStyle
		parts := stageWordRows(line, style)
		reduce := cx.Env().ReduceMotion()
		children := make([]core.Node, 0, len(parts)+1)
		for index, words := range parts {
			text := core.Text{Style: core.Style{
				Color: p.muted, FontSize: 52, FontWeight: core.FontWeightBold, TextAlign: core.TextAlignCenter,
				Padding: core.Symmetric(0, 12),
			}}
			if len(parts) > 1 {
				text.Style.FontSize = 45
			}
			text.SpansFunc = func() []core.Span {
				at := position.Get()
				spans := make([]core.Span, 0, len(words))
				for _, word := range words {
					tint := p.muted
					if at >= word.End {
						tint = p.text
					} else if at >= word.Start {
						tint = p.accent
					}
					spans = append(spans, core.Span{Content: word.Text, Style: core.Style{Color: tint}})
				}
				return spans
			}
			lineRow := core.View{Style: core.Style{AlignSelf: core.AlignStretch}, Children: []core.Node{text}}
			lineRow.StyleFunc = func() core.Style {
				if reduce {
					return core.Style{}
				}
				elapsed := position.Get() - words[0].Start
				phase := min(max(float64(elapsed)/float64(330*time.Millisecond), 0), 1)
				rest := math.Pow(1-phase, 3)
				var x, y, angle float32
				switch style {
				case "partita":
					x = float32(index-(len(parts)-1)/2) * 24
					y = float32(18 * rest)
				case "tilt":
					angle = -5
					if index%2 == 0 {
						angle = 5
					}
					y = float32(22 * rest)
				default:
					y = float32(10 * rest)
				}
				return core.Style{Transform: core.Transform{TranslateX: core.Points(x), TranslateY: core.Points(y), Rotate: angle},
					Opacity: core.Alpha(float32(1 - .35*rest))}
			}
			children = append(children, lineRow)
		}
		if m.prefs.Get().StageSparks && !reduce {
			children = append(children, lyricSparks(line.Words, position.Get))
		}
		return core.View{Style: core.Style{AlignSelf: core.AlignStretch, JustifyContent: core.JustifyCenter, Gap: 2}, Children: children}
	})
	footer := core.Dynamic(func(cx *core.Scope) core.Node {
		i := active.Get()
		var children []core.Node
		if i >= 0 && lines[i].Translation != "" && !interlude.Get() {
			text := label(lines[i].Translation, 18, core.FontWeightMedium, p.text)
			text.Style.TextAlign = core.TextAlignCenter
			children = append(children, text)
		}
		if i+1 < len(lines) {
			text := label(lines[i+1].Text, 14, core.FontWeightMedium, p.muted)
			text.Style.TextAlign = core.TextAlignCenter
			text.OnPress = func(core.PressEvent) { m.seek(lines[i+1].At) }
			children = append(children, text)
		}
		return core.View{Style: core.Style{Gap: 6, MinHeight: core.Pt(44), AlignSelf: core.AlignStretch}, Children: children}
	})
	return core.View{TestID: "lyric-stage", Style: core.Style{FlexGrow: 1, FlexShrink: 1, Padding: core.Symmetric(12, 54)}, Children: []core.Node{
		core.View{Style: core.Style{FlexGrow: 1, JustifyContent: core.JustifyCenter, Overflow: core.Hidden}, Children: []core.Node{body}},
		footer,
	}}
}

func stageWordRows(line lyrics.Line, style string) [][]lyrics.Word {
	words := line.Words
	if len(words) == 0 {
		words = []lyrics.Word{{Text: line.Text, Start: line.At, End: line.End}}
	}
	rows := 1
	if style == "partita" {
		rows = 3
	} else if style == "tilt" {
		rows = 2
	}
	rows = min(rows, len(words))
	parts := make([][]lyrics.Word, 0, rows)
	for i := range rows {
		start, end := i*len(words)/rows, (i+1)*len(words)/rows
		parts = append(parts, words[start:end])
	}
	return parts
}

func lyricSparks(words []lyrics.Word, position func() time.Duration) core.Node {
	return core.Canvas{Style: core.Style{Position: core.Absolute, Left: core.Pt(0), Right: core.Pt(0), Top: core.Pt(0), Bottom: core.Pt(0)}, Draw: func(d *core.Drawing, size core.Size) {
		at := position()
		total, offset := 0, 0
		for _, word := range words {
			total += utf8.RuneCountInString(word.Text)
		}
		for _, word := range words {
			width := utf8.RuneCountInString(word.Text)
			elapsed := at - word.Start
			if elapsed >= 0 && elapsed < 420*time.Millisecond {
				t := float32(elapsed) / float32(420*time.Millisecond)
				x := float32(size.Width) * (.16 + .68*float32(offset+width/2)/float32(max(total, 1)))
				y := float32(size.Height) * .47
				d.SetFillColor(core.RGBA(0xf4a6b8, (1-t)*(1-t)))
				for i := range 7 {
					angle := float64(i)*2*math.Pi/7 + float64(offset)
					d.FillRect(x+float32(math.Cos(angle))*t*44, y+float32(math.Sin(angle))*t*30, 2, 2)
				}
			}
			offset += width
		}
	}}
}
