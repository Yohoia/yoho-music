package app

import (
	"time"

	"github.com/nsgo-dev/nsgo/core"
)

type skeletonShimmer struct {
	base  core.Color
	hi    core.Color
	sweep *core.AnimatedValue
	pulse *core.AnimatedValue
}

func newSkeletonShimmer(cx *core.Scope) skeletonShimmer {
	base := core.RGB(0xefeff2)
	hi := core.RGBA(0xffffff, 0.72)
	if cx.Dark() {
		base = core.RGB(0x222227)
		hi = core.RGBA(0xffffff, 0.08)
	}
	sweep := core.NewAnimatedValue(cx, -240)
	pulse := core.NewAnimatedValue(cx, 0.88)

	ease := core.BezierEasing(core.Curve{X1: 0.4, Y1: 0.0, X2: 0.2, Y2: 1.0})
	cx.StartAnimation(core.AtFrameRate(core.Loop(core.Sequence(
		core.Timing(sweep, 460, core.TimingConfig{Duration: 1450 * time.Millisecond, Easing: ease}),
		core.Delay(260*time.Millisecond),
	), 0), 30), nil)
	cx.StartAnimation(core.AtFrameRate(core.Loop(core.Sequence(
		core.Timing(pulse, 1.0, core.TimingConfig{Duration: 900 * time.Millisecond, Easing: ease}),
		core.Timing(pulse, 0.86, core.TimingConfig{Duration: 900 * time.Millisecond, Easing: ease}),
	), 0), 30), nil)

	cx.Cleanup(func() {
		sweep.Stop()
		pulse.Stop()
	})

	return skeletonShimmer{base: base, hi: hi, sweep: sweep, pulse: pulse}
}

func (s skeletonShimmer) box(style core.Style) core.View {
	st := core.Style{
		Background:   s.base,
		CornerRadius: 8,
		Overflow:     core.Hidden,
		Animated:     &core.AnimatedStyle{Opacity: s.pulse},
	}.Merge(style)
	sheen := core.View{
		Style: core.Style{
			Position:  core.Absolute,
			Top:       core.Pt(0),
			Bottom:    core.Pt(0),
			Left:      core.Pt(0),
			Width:     core.Pt(220),
			Gradients: []core.Gradient{core.LinearGradient(105, core.Transparent, s.hi, core.Transparent)},
			Animated:  &core.AnimatedStyle{TranslateX: s.sweep},
		},
	}
	return core.View{Style: st, Children: []core.Node{sheen}}
}

func (s skeletonShimmer) bar(w, h core.Points, radius core.Points) core.View {
	return s.box(core.Style{Width: core.Pt(w), Height: core.Pt(h), CornerRadius: radius})
}

func (s skeletonShimmer) tileRow(count int) core.View {
	cols := make([]core.Node, 0, count)
	titleWidths := []core.Points{110, 96, 122, 88, 108}
	subWidths := []core.Points{58, 48, 66, 52, 60}
	for i := 0; i < count; i++ {
		cols = append(cols, core.View{
			Style: core.Style{FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0), Gap: 8},
			Children: []core.Node{
				s.box(core.Style{Width: core.Pct(100), AspectRatio: 1, CornerRadius: 12}),
				s.bar(titleWidths[i%len(titleWidths)], 12, 6),
				s.bar(subWidths[i%len(subWidths)], 10, 5),
			},
		})
	}
	return core.View{
		Style:    core.Style{Direction: core.Row, Gap: 16, AlignItems: core.AlignStart},
		Children: cols,
	}
}

func (s skeletonShimmer) trackRow(i int) core.View {
	titleW := []core.Points{184, 224, 156, 206, 172, 238, 164, 198}[i%8]
	artistW := []core.Points{92, 118, 84, 104, 96, 112, 88, 100}[i%8]
	return core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14,
			Padding: core.Symmetric(8, 6),
		},
		Children: []core.Node{
			s.box(core.Style{Width: core.Pt(42), Height: core.Pt(42), CornerRadius: 8, FlexShrink: 0}),
			core.View{
				Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 7},
				Children: []core.Node{
					s.bar(titleW, 13, 6),
					s.bar(artistW, 10, 5),
				},
			},
			s.bar(36, 11, 5),
		},
	}
}

func (m *model) loadingSkeleton(cx *core.Scope, r route) core.Node {
	p := colors(cx)
	s := newSkeletonShimmer(cx)

	var content core.Node
	var wrap func(...core.Node) core.View

	switch {
	case r.page == "album" || r.page == "playlist" || r.page == "show" || r.page == "artist":
		wrap = collectionWrap
		hero := core.View{
			Style: core.Style{Direction: core.Row, AlignItems: core.AlignEnd, Gap: 24, Padding: core.Edges{Bottom: 24}},
			Children: []core.Node{
				s.box(core.Style{Width: core.Pt(186), Height: core.Pt(186), CornerRadius: 14, FlexShrink: 0}),
				core.View{
					Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 10, Padding: core.Edges{Bottom: 4}},
					Children: []core.Node{
						s.bar(68, 11, 5),
						s.bar(248, 28, 8),
						s.bar(310, 13, 6),
						core.View{
							Style: core.Style{Direction: core.Row, Gap: 10, Padding: core.Edges{Top: 8}},
							Children: []core.Node{
								s.bar(106, 34, 17),
								s.bar(88, 34, 17),
								s.bar(34, 34, 17),
							},
						},
					},
				},
			},
		}
		rows := make([]core.Node, 0, 7)
		rows = append(rows, hero, dashedDivider(p.line))
		for i := 0; i < 6; i++ {
			rows = append(rows, s.trackRow(i))
		}
		content = core.Scroll{
			Style: fill,
			Children: []core.Node{
				collectionWrap(core.View{
					Style:    core.Style{Padding: core.Edges{Top: detailHeaderHeight + 12, Bottom: 108}, Gap: 6},
					Children: rows,
				}),
			},
		}

	case r.page == "home":
		wrap = paperWrap
		features := make([]core.Node, 0, 5)
		for i := 0; i < 5; i++ {
			features = append(features, s.box(core.Style{
				FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0),
				Height: core.Pt(168), CornerRadius: 16,
			}))
		}
		pairedCol := func(offset int) core.View {
			items := make([]core.Node, 0, 4)
			items = append(items, s.bar(96, 16, 6))
			for i := 0; i < 3; i++ {
				items = append(items, s.trackRow(i+offset))
			}
			return core.View{
				Style:    core.Style{FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0), Gap: 8},
				Children: items,
			}
		}
		topPad := detailHeaderHeight
		if m.hasSubNav() {
			topPad = 88
		}
		content = core.Scroll{
			Style: fill,
			Children: []core.Node{
				paperWrap(core.View{
					Style: core.Style{Padding: core.Edges{Top: topPad, Bottom: 108}, Gap: 24},
					Children: []core.Node{
						core.View{Style: core.Style{Direction: core.Row, Gap: 14}, Children: features},
						dashedDivider(p.line),
						core.View{
							Style: core.Style{Gap: 14},
							Children: []core.Node{
								s.bar(104, 16, 6),
								s.tileRow(5),
							},
						},
						dashedDivider(p.line),
						core.View{
							Style: core.Style{Direction: core.Row, Gap: 28, AlignItems: core.AlignStart},
							Children: []core.Node{
								pairedCol(0),
								pairedCol(3),
							},
						},
					},
				}),
			},
		}

	case r.page == "playlists" || r.page == "savedAlbums" || (r.page == "browse" && r.id == "playlists-hub"):
		wrap = paperWrap
		topPad := detailHeaderHeight
		if m.hasSubNav() {
			topPad = 88
		}
		content = core.Scroll{
			Style: fill,
			Children: []core.Node{
				paperWrap(core.View{
					Style: core.Style{Padding: core.Edges{Top: topPad, Bottom: 108}, Gap: 20},
					Children: []core.Node{
						core.View{
							Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
							Children: []core.Node{
								s.bar(86, 30, 15),
								s.bar(86, 30, 15),
							},
						},
						dashedDivider(p.line),
						s.tileRow(5),
						s.tileRow(5),
					},
				}),
			},
		}

	default:
		wrap = paperWrap
		topPad := detailHeaderHeight
		if m.hasSubNav() {
			topPad = 88
		}
		rows := make([]core.Node, 0, 10)
		rows = append(rows,
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, Padding: core.Edges{Bottom: 8}},
				Children: []core.Node{
					s.bar(140, 24, 8),
					spacer(),
					s.bar(84, 30, 15),
					s.bar(84, 30, 15),
				},
			},
			dashedDivider(p.line),
		)
		for i := 0; i < 8; i++ {
			rows = append(rows, s.trackRow(i))
		}
		content = core.Scroll{
			Style: fill,
			Children: []core.Node{
				paperWrap(core.View{
					Style:    core.Style{Padding: core.Edges{Top: topPad, Bottom: 108}, Gap: 6},
					Children: rows,
				}),
			},
		}
	}

	return m.withDetailHeader(cx, content, wrap)
}

func marketLoadingSkeleton(cx *core.Scope, p palette) core.Node {
	s := newSkeletonShimmer(cx)
	card := func() core.Node {
		return core.View{
			Style: core.Style{
				FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0),
				Background: p.background, BorderWidth: 1, BorderColor: p.line,
				CornerRadius: 16, Padding: core.All(16), Gap: 12,
			},
			Children: []core.Node{
				core.View{
					Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12},
					Children: []core.Node{
						s.box(core.Style{Width: core.Pt(46), Height: core.Pt(46), CornerRadius: 12, FlexShrink: 0}),
						core.View{
							Style: core.Style{FlexGrow: 1, Gap: 8},
							Children: []core.Node{
								s.bar(116, 15, 6),
								s.bar(76, 11, 5),
							},
						},
						s.bar(58, 12, 6),
					},
				},
				core.View{
					Style: core.Style{Gap: 6},
					Children: []core.Node{
						s.box(core.Style{Width: core.Pct(92), Height: core.Pt(12), CornerRadius: 5}),
						s.box(core.Style{Width: core.Pct(64), Height: core.Pt(12), CornerRadius: 5}),
					},
				},
				core.View{
					Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, Padding: core.Edges{Top: 4}},
					Children: []core.Node{
						s.bar(46, 11, 5),
						s.bar(64, 11, 5),
						s.bar(42, 11, 5),
					},
				},
			},
		}
	}
	return core.View{
		Style:    core.Style{Direction: core.Row, Gap: 14},
		Children: []core.Node{card(), card()},
	}
}
