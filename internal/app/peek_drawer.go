package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
)

type peekTarget struct {
	sourceID string
	kind     string // "artist" | "album"
	id       string
	title    string
	image    string
}

type peekState struct {
	open    bool
	current peekTarget
	history []peekTarget
}

type peekResult struct {
	done   bool
	err    error
	artist *music.ArtistPage
	album  *music.AlbumPage
}

func (m *model) openPeek(sourceID, kind, id, title, image string) {
	if id == "" {
		return
	}
	if sourceID == "" {
		sourceID = m.source.GetUntracked()
	}
	m.peek.Set(peekState{
		open:    true,
		current: peekTarget{sourceID: sourceID, kind: kind, id: id, title: title, image: image},
	})
}

func (m *model) drillPeek(target peekTarget) {
	if target.id == "" {
		return
	}
	st := m.peek.GetUntracked()
	hist := append(append([]peekTarget(nil), st.history...), st.current)
	m.peek.Set(peekState{open: true, current: target, history: hist})
}

func (m *model) popPeek() {
	st := m.peek.GetUntracked()
	if len(st.history) == 0 {
		m.closePeek()
		return
	}
	last := st.history[len(st.history)-1]
	m.peek.Set(peekState{open: true, current: last, history: st.history[:len(st.history)-1]})
}

func (m *model) closePeek() {
	st := m.peek.GetUntracked()
	if !st.open {
		return
	}
	st.open = false
	m.peek.Set(st)
}

func (m *model) peekOverlay(cx *core.Scope) core.Node {
	return core.Show{
		When: func() bool { return m.peek.Get().open },
		Then: func(cx *core.Scope) core.Node {
			p := colors(cx)
			dark := cx.Dark()

			slideX := core.NewAnimatedValue(cx, 460)
			fade := core.NewAnimatedValue(cx, 0)
			ease := core.BezierEasing(core.Curve{X1: 0.22, Y1: 1, X2: 0.36, Y2: 1})
			cx.StartAnimation(core.Parallel(
				core.Timing(slideX, 0, core.TimingConfig{Duration: 280 * time.Millisecond, Easing: ease}),
				core.Timing(fade, 1, core.TimingConfig{Duration: 220 * time.Millisecond, Easing: ease}),
			), nil)
			cx.Cleanup(func() {
				slideX.Stop()
				fade.Stop()
			})

			veilColor := core.RGBA(0x000000, 0.14)
			shadowColor := core.RGBA(0x000000, 0.18)
			if dark {
				veilColor = core.RGBA(0x000000, 0.44)
				shadowColor = core.RGBA(0x000000, 0.58)
			}

			veil := base.Button{
				Label:   i18n.T("Close"),
				TestID:  "peek-veil",
				OnPress: func(core.PressEvent) { m.closePeek() },
				Style: core.Style{
					Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0),
					Background: veilColor,
					Animated:   &core.AnimatedStyle{Opacity: fade},
				},
			}

			sheet := core.View{
				TestID: "peek-drawer",
				Style: core.Style{
					Position:     core.Absolute,
					Top:          core.Pt(14),
					Right:        core.Pt(14),
					Bottom:       core.Pt(14),
					Width:        core.Pct(52),
					MinWidth:     core.Pt(420),
					MaxWidth:     core.Pt(560),
					CornerRadius: 20,
					Background:   p.background,
					BorderWidth:  1,
					BorderColor:  p.line,
					ShadowColor:  shadowColor,
					ShadowRadius: 36,
					ShadowOffset: core.Point{Y: 12},
					Overflow:     core.Hidden,
					Animated:     &core.AnimatedStyle{TranslateX: slideX, Opacity: fade},
				},
				Children: []core.Node{
					core.Dynamic(func(cx *core.Scope) core.Node {
						st := m.peek.Get()
						m.sourceVersion.Get()
						return m.peekBody(cx, p, st)
					}),
				},
			}

			return core.View{
				Style:    core.Style{Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0)},
				Children: []core.Node{veil, sheet},
			}
		},
	}
}

func (m *model) peekBody(cx *core.Scope, p palette, st peekState) core.Node {
	cur := st.current
	src := m.sourceByID(cur.sourceID)
	if src == nil {
		src = m.activeSource()
	}

	result := core.NewSignalFunc(cx, peekResult{}, nil)
	tab := core.NewSignal(cx, "tracks")

	if src != nil {
		go func() {
			var res peekResult
			res.done = true
			if cur.kind == "artist" {
				res.artist, res.err = src.Artist(cur.id)
			} else {
				res.album, res.err = src.Album(cur.id)
			}
			cx.Post(func() { result.Set(res) })
		}()
	} else {
		result.Set(peekResult{done: true, err: fmt.Errorf("source unavailable")})
	}

	kicker := "ALBUM"
	if cur.kind == "artist" {
		kicker = "ARTIST"
	}
	leftTop := []core.Node{}
	if len(st.history) > 0 {
		leftTop = append(leftTop, base.Button{
			Label:   i18n.T("Back"),
			TestID:  "peek-back",
			OnPress: func(core.PressEvent) { m.popPeek() },
			Style: core.Style{
				Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14,
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				Hover: &core.Style{Background: p.hover},
			},
			Children: []core.Node{ui.Icon(lucide.ChevronLeft, 15, func() core.Color { return p.text })},
		})
	}
	leftTop = append(leftTop, label(kicker, 11, core.FontWeightMedium, p.muted))

	expandBtn := ui.Tooltip{
		Text: i18n.T("Open"),
		Trigger: base.Button{
			Label:  i18n.T("Open"),
			TestID: "peek-expand",
			OnPress: func(core.PressEvent) {
				m.closePeek()
				m.revealSourceTrack(cur.sourceID, route{page: cur.kind, id: cur.id})
			},
			Style: core.Style{
				Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14,
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				Hover: &core.Style{Background: p.hover},
			},
			Children: []core.Node{ui.Icon(lucide.ArrowUpRight, 14, func() core.Color { return p.muted })},
		},
	}
	closeBtn := base.Button{
		Label:   i18n.T("Close"),
		TestID:  "peek-close",
		OnPress: func(core.PressEvent) { m.closePeek() },
		Style: core.Style{
			Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Hover: &core.Style{Background: p.hover},
		},
		Children: []core.Node{ui.Icon(lucide.X, 15, func() core.Color { return p.muted })},
	}

	topBar := core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter,
			Padding: core.Edges{Top: 14, Left: 20, Right: 16, Bottom: 8},
		},
		Children: []core.Node{
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6}, Children: leftTop},
			spacer(),
			core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4}, Children: []core.Node{expandBtn, closeBtn}},
		},
	}

	content := core.Dynamic(func(cx *core.Scope) core.Node {
		res := result.Get()
		if !res.done {
			return m.peekSkeleton(cx, p, cur.kind == "artist")
		}
		if res.err != nil {
			return m.empty(cx, lucide.CircleAlert, res.err.Error(), "", nil)
		}
		if cur.kind == "artist" && res.artist != nil {
			return m.peekArtistContent(cx, p, src, res.artist, tab)
		}
		if res.album != nil {
			return m.peekAlbumContent(cx, p, src, res.album)
		}
		return m.empty(cx, lucide.Music, i18n.T("Nothing here yet"), "", nil)
	})

	return core.View{
		Style:    fill,
		Children: []core.Node{topBar, core.View{Style: fill, Children: []core.Node{content}}},
	}
}

func (m *model) peekSkeleton(cx *core.Scope, p palette, isArtist bool) core.Node {
	s := newSkeletonShimmer(cx)
	radius := core.Points(14)
	if isArtist {
		radius = 40
	}
	rows := []core.Node{
		core.View{
			Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 16, Padding: core.Edges{Bottom: 16}},
			Children: []core.Node{
				s.box(core.Style{Width: core.Pt(80), Height: core.Pt(80), CornerRadius: radius, FlexShrink: 0}),
				core.View{
					Style: core.Style{FlexGrow: 1, Gap: 8},
					Children: []core.Node{
						s.bar(148, 20, 6),
						s.bar(112, 12, 5),
						core.View{
							Style:    core.Style{Direction: core.Row, Gap: 8, Padding: core.Edges{Top: 4}},
							Children: []core.Node{s.bar(76, 30, 15), s.bar(68, 30, 15)},
						},
					},
				},
			},
		},
		dashedDivider(p.line),
	}
	for i := 0; i < 6; i++ {
		rows = append(rows, s.trackRow(i))
	}
	return core.View{
		Style:    core.Style{Padding: core.Edges{Top: 6, Left: 20, Right: 20}, Gap: 6},
		Children: rows,
	}
}

func (m *model) peekActionCapsules(cx *core.Scope, p palette, tracks []music.Track, extra ...core.Node) core.Node {
	var children []core.Node
	if len(tracks) > 0 {
		children = append(children,
			base.Button{
				Label:   i18n.T("Play"),
				TestID:  "peek-play-all",
				OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayQueue(tracks, 0) })() },
				Style: core.Style{
					Direction: core.Row, Height: core.Pt(30), Padding: core.Symmetric(0, 14), CornerRadius: 15,
					Background: p.button, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 5,
					Hover: &core.Style{Opacity: core.Alpha(.88)},
				},
				Children: []core.Node{
					ui.Icon(lucide.Play, 12, func() core.Color { return p.onButton }),
					label(i18n.T("Play"), 12, core.FontWeightSemibold, p.onButton),
				},
			},
			base.Button{
				Label:   i18n.T("Shuffle"),
				TestID:  "peek-shuffle-all",
				OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayShuffled(tracks) })() },
				Style: core.Style{
					Direction: core.Row, Height: core.Pt(30), Padding: core.Symmetric(0, 12), CornerRadius: 15,
					BorderWidth: 1, BorderColor: p.line, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 5,
					Hover: &core.Style{Background: p.hover},
				},
				Children: []core.Node{
					ui.Icon(lucide.Shuffle, 12, func() core.Color { return p.text }),
					label(i18n.T("Shuffle"), 12, core.FontWeightMedium, p.text),
				},
			},
		)
	}
	children = append(children, extra...)
	return core.View{
		Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Edges{Top: 10}},
		Children: children,
	}
}

func (m *model) peekArtistContent(cx *core.Scope, p palette, src music.Source, page *music.ArtistPage, tab *core.Signal[string]) core.Node {
	sourceID := src.Info().ID
	tracks := withSource(page.Tracks, sourceID)
	artist := page.Artist

	avatarStyle := core.Style{
		Width: core.Pt(80), Height: core.Pt(80), CornerRadius: 40, FlexShrink: 0,
		Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	var avatar core.Node
	if artist.ImageURL != "" {
		avatar = core.Image{SourceFunc: m.coverSource(cx, artist.ImageURL, 160, sourceID), Style: avatarStyle, ResizeMode: core.Cover}
	} else {
		avatar = core.View{Style: avatarStyle, Children: []core.Node{ui.Icon(lucide.UserRound, 28, func() core.Color { return p.muted })}}
	}

	var stats []string
	if len(tracks) > 0 {
		stats = append(stats, i18n.Songs(len(tracks)))
	}
	if len(page.Albums) > 0 {
		stats = append(stats, fmt.Sprintf("%d %s", len(page.Albums), i18n.T("Albums")))
	}

	var extraActions []core.Node
	if src.Can("followArtist") {
		following := artist.Following
		lbl := i18n.T("Follow")
		icon := lucide.Plus
		if following {
			lbl = i18n.T("Following")
			icon = lucide.Check
		}
		extraActions = append(extraActions, base.Button{
			Label:  lbl,
			TestID: "peek-follow-artist",
			OnPress: func(core.PressEvent) {
				m.changeSource(cx, src, func() error { return src.FollowArtist(artist.ID, !following) })
			},
			Style: core.Style{
				Direction: core.Row, Height: core.Pt(30), Padding: core.Symmetric(0, 12), CornerRadius: 15,
				BorderWidth: 1, BorderColor: p.line, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 5,
				Hover: &core.Style{Background: p.hover},
			},
			Children: []core.Node{
				ui.Icon(icon, 12, func() core.Color { return p.text }),
				label(lbl, 12, core.FontWeightMedium, p.text),
			},
		})
	}

	hero := core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 16,
			Padding: core.Edges{Top: 6, Left: 20, Right: 20, Bottom: 16},
		},
		Children: []core.Node{
			avatar,
			core.View{
				Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 3},
				Children: []core.Node{
					label(artist.Name, 20, core.FontWeightBold, p.text),
					label(strings.Join(stats, " · "), 12, core.FontWeightNormal, p.muted),
					m.peekActionCapsules(cx, p, tracks, extraActions...),
				},
			},
		},
	}

	tabBtn := func(key, text string) core.Node {
		selected := func() bool { return tab.Get() == key }
		t := core.Text{
			Content: text,
			Style:   core.Style{FontSize: 11, FontWeight: core.FontWeightMedium, Color: p.muted},
			StyleFunc: func() core.Style {
				if selected() {
					return core.Style{Color: p.onButton}
				}
				return core.Style{}
			},
		}
		return base.Button{
			Label:    text,
			TestID:   "peek-tab-" + key,
			Selected: selected,
			OnPress:  func(core.PressEvent) { tab.Set(key) },
			Style: core.Style{
				Height: core.Pt(26), Padding: core.Symmetric(0, 12), CornerRadius: 13,
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			},
			StyleFunc: func() core.Style { return p.selectableButtonStyle(selected()) },
			Children:  []core.Node{t},
		}
	}

	sectionHeader := core.View{
		Style: core.Style{
			Padding: core.Edges{Left: 20, Right: 20, Bottom: 10}, Gap: 12,
		},
		Children: []core.Node{
			dashedDivider(p.line),
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter},
				Children: []core.Node{
					core.Text{
						ContentFunc: func() string {
							if tab.Get() == "albums" {
								return i18n.T("Albums")
							}
							return i18n.T("Songs")
						},
						Style: core.Style{FontSize: 13, FontWeight: core.FontWeightSemibold, Color: p.text},
					},
					spacer(),
					core.View{
						Style: core.Style{
							Direction: core.Row, Padding: core.All(2), CornerRadius: 15,
							Background: p.sidebar, BorderWidth: 1, BorderColor: p.line,
						},
						Children: []core.Node{
							tabBtn("tracks", i18n.T("Songs")),
							tabBtn("albums", i18n.T("Albums")),
						},
					},
				},
			},
		},
	}

	body := core.Dynamic(func(cx *core.Scope) core.Node {
		if tab.Get() == "albums" {
			if len(page.Albums) == 0 {
				return m.empty(cx, lucide.Disc3, i18n.T("Nothing here yet"), "", nil)
			}
			cards := make([]core.Node, 0, len(page.Albums))
			for _, al := range page.Albums {
				album := al
				coverStyle := core.Style{
					Width: core.Pct(100), AspectRatio: 1, CornerRadius: 12,
					Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				}
				var cover core.Node
				if album.ImageURL != "" {
					cover = core.Image{SourceFunc: m.coverSource(cx, album.ImageURL, 240, sourceID), Style: coverStyle, ResizeMode: core.Cover}
				} else {
					cover = core.View{Style: coverStyle, Children: []core.Node{ui.Icon(lucide.Disc3, 24, func() core.Color { return p.muted })}}
				}
				alName := label(album.Name, 12, core.FontWeightMedium, p.text)
				alName.MaxLines = 1
				subText := album.Released
				if subText == "" {
					subText = albumSubtitle(album)
				}
				alSub := label(subText, 11, core.FontWeightNormal, p.muted)
				alSub.MaxLines = 1
				cards = append(cards, base.Button{
					Label:  album.Name,
					TestID: "peek-album-" + album.ID,
					OnPress: func(core.PressEvent) {
						m.drillPeek(peekTarget{sourceID: sourceID, kind: "album", id: album.ID, title: album.Name, image: album.ImageURL})
					},
					Style: core.Style{
						Padding: core.All(6), CornerRadius: 14, Gap: 6,
						Hover: &core.Style{Background: p.hover},
					},
					Children: []core.Node{cover, alName, alSub},
				})
			}
			grid := core.View{
				Style: core.Style{
					Display:             core.DisplayGrid,
					GridTemplateColumns: core.GridTracks("repeat(2, minmax(0, 1fr))"),
					ColumnGap:           10,
					RowGap:              10,
					Padding:             core.Edges{Left: 14, Right: 14, Bottom: 24},
				},
				Children: cards,
			}
			return core.Scroll{Style: fill, Children: []core.Node{grid}}
		}
		if len(tracks) == 0 {
			return m.empty(cx, lucide.Music, i18n.T("Nothing here yet"), "", nil)
		}
		return m.peekTrackList(cx, p, tracks, true)
	})

	return core.View{
		Style:    fill,
		Children: []core.Node{hero, sectionHeader, core.View{Style: fill, Children: []core.Node{body}}},
	}
}

func (m *model) peekAlbumContent(cx *core.Scope, p palette, src music.Source, page *music.AlbumPage) core.Node {
	sourceID := src.Info().ID
	tracks := withSource(page.Tracks, sourceID)
	album := page.Album

	coverStyle := core.Style{
		Width: core.Pt(84), Height: core.Pt(84), CornerRadius: 14, FlexShrink: 0,
		Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	var cover core.Node
	if album.ImageURL != "" {
		cover = core.Image{SourceFunc: m.coverSource(cx, album.ImageURL, 168, sourceID), Style: coverStyle, ResizeMode: core.Cover}
	} else {
		cover = core.View{Style: coverStyle, Children: []core.Node{ui.Icon(lucide.Disc3, 28, func() core.Color { return p.muted })}}
	}

	var subParts []string
	if names := albumSubtitle(album); names != "" {
		subParts = append(subParts, names)
	}
	if len(tracks) > 0 {
		subParts = append(subParts, i18n.Songs(len(tracks)))
	}

	var extraActions []core.Node
	if src.Can("saveAlbum") {
		saved := album.Saved
		lbl := i18n.T("Save")
		icon := lucide.Bookmark
		if saved {
			lbl = i18n.T("Saved")
			icon = lucide.Check
		}
		extraActions = append(extraActions, base.Button{
			Label:  lbl,
			TestID: "peek-save-album",
			OnPress: func(core.PressEvent) {
				m.changeSource(cx, src, func() error { return src.SaveAlbum(album.ID, !saved) })
			},
			Style: core.Style{
				Direction: core.Row, Height: core.Pt(30), Padding: core.Symmetric(0, 12), CornerRadius: 15,
				BorderWidth: 1, BorderColor: p.line, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 5,
				Hover: &core.Style{Background: p.hover},
			},
			Children: []core.Node{
				ui.Icon(icon, 12, func() core.Color { return p.text }),
				label(lbl, 12, core.FontWeightMedium, p.text),
			},
		})
	}

	hero := core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 16,
			Padding: core.Edges{Top: 6, Left: 20, Right: 20, Bottom: 16},
		},
		Children: []core.Node{
			cover,
			core.View{
				Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 3},
				Children: []core.Node{
					label(album.Name, 20, core.FontWeightBold, p.text),
					label(strings.Join(subParts, " · "), 12, core.FontWeightNormal, p.muted),
					m.peekActionCapsules(cx, p, tracks, extraActions...),
				},
			},
		},
	}

	divider := core.View{
		Style:    core.Style{Padding: core.Edges{Left: 20, Right: 20, Bottom: 8}},
		Children: []core.Node{dashedDivider(p.line)},
	}

	var body core.Node
	if len(tracks) == 0 {
		body = m.empty(cx, lucide.Music, i18n.T("Nothing here yet"), "", nil)
	} else {
		body = m.peekTrackList(cx, p, tracks, false)
	}

	return core.View{
		Style:    fill,
		Children: []core.Node{hero, divider, core.View{Style: fill, Children: []core.Node{body}}},
	}
}

func (m *model) peekTrackList(cx *core.Scope, p palette, tracks []music.Track, showAlbumSub bool) core.Node {
	list := core.List{
		Count: func() int { return len(tracks) }, EstimatedSize: 52, Style: fill,
		Key: func(i int) any { return fmt.Sprintf("peek:%d:%s:%s", i, tracks[i].Source, tracks[i].ID) },
		Row: func(cx *core.Scope, cell core.ListRow) core.Node {
			return core.Dynamic(func(cx *core.Scope) core.Node {
				i := cell.Index()
				if i < 0 || i >= len(tracks) {
					return nil
				}
				track := tracks[i]
				isPlaying := func() bool {
					cur := m.playing.Get().track
					return cur != nil && cur.ID == track.ID && cur.Source == track.Source
				}

				num := core.Text{
					ContentFunc: func() string {
						if isPlaying() && m.playing.Get().playing {
							return "▶"
						}
						return fmt.Sprint(i + 1)
					},
					Style: core.Style{
						Width: core.Pt(22), FontSize: 11, FontVariant: core.FontTabularNums,
						TextAlign: core.TextAlignCenter, Color: p.muted, FlexShrink: 0,
					},
					StyleFunc: func() core.Style {
						if isPlaying() {
							return core.Style{Color: p.text, FontWeight: core.FontWeightSemibold}
						}
						return core.Style{}
					},
				}

				thumbStyle := core.Style{
					Width: core.Pt(36), Height: core.Pt(36), CornerRadius: 7, FlexShrink: 0,
					Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				}
				var thumb core.Node
				if track.Album.ImageURL != "" {
					thumb = core.Image{SourceFunc: m.coverSource(cx, track.Album.ImageURL, 72, track.Source), Style: thumbStyle, ResizeMode: core.Cover}
				} else {
					thumb = core.View{Style: thumbStyle, Children: []core.Node{ui.Icon(lucide.Music, 14, func() core.Color { return p.muted })}}
				}

				title := core.Text{
					Content:  track.Name,
					MaxLines: 1,
					Style:    core.Style{FontSize: 13, FontWeight: core.FontWeightMedium, Color: p.text},
				}
				subText := track.ArtistNames()
				if showAlbumSub && track.Album.Name != "" {
					subText = track.Album.Name
				}
				sub := label(subText, 11, core.FontWeightNormal, p.muted)
				sub.MaxLines = 1

				dur := label(formatTime(track.Length()), 11, core.FontWeightNormal, p.muted)
				dur.Style.FontVariant = core.FontTabularNums
				dur.Style.FlexShrink = 0

				btn := base.Button{
					Label:   track.Name,
					TestID:  "peek-track-" + fmt.Sprint(i),
					OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayTrack(tracks, i) })() },
					Style: core.Style{
						Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10,
						Height: core.Pt(50), Padding: core.Symmetric(0, 10), CornerRadius: 10,
						Hover: &core.Style{Background: p.hover},
					},
					StyleFunc: func() core.Style {
						if isPlaying() {
							return core.Style{Background: p.selected}
						}
						return core.Style{}
					},
					Children: []core.Node{
						num,
						thumb,
						core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 2}, Children: []core.Node{title, sub}},
						dur,
					},
				}
				return core.View{
					Style:    core.Style{Padding: core.Edges{Left: 10, Right: 10, Top: 1, Bottom: 1}},
					Children: []core.Node{btn},
				}
			})
		},
	}
	list.Footer = core.View{Style: core.Style{Height: core.Pt(24)}}
	return list
}
