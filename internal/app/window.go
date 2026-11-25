package app

import (
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

type glyph = func() *core.SVG

type palette struct {
	background, sidebar, text, muted, line, hover, selected, accent core.Color
	chip, button, onButton, bar, barLine                            core.Color
}

func colors(cx *core.Scope) palette {
	if cx.Dark() {
		return palette{
			background: core.RGB(0x151517),
			sidebar:    core.RGB(0x1c1c20),
			text:       core.RGB(0xf4f4f5),
			muted:      core.RGB(0xb0b0b9),
			line:       core.RGB(0x343438),
			hover:      core.RGBA(0xffffff, .06),
			selected:   core.RGBA(0xffffff, .11),
			accent:     core.RGB(0xf5f5f7),
			chip:       core.RGB(0x242427),
			button:     core.RGB(0xededf0),
			onButton:   core.RGB(0x18181b),
			bar:        core.RGB(0x18181b),
			barLine:    core.RGB(0x27272a),
		}
	}
	return palette{
		background: core.RGB(0xffffff),
		sidebar:    core.RGB(0xf5f5f7),
		text:       core.RGB(0x09090b),
		muted:      core.RGB(0x71717a),
		line:       core.RGB(0xe4e4e7),
		hover:      core.RGBA(0x09090b, .04),
		selected:   core.RGBA(0x09090b, .08),
		accent:     core.RGB(0x1d1d1f),
		chip:       core.RGB(0xfafaf9),
		button:     core.RGB(0x27272a),
		onButton:   core.RGB(0xffffff),
		bar:        core.RGB(0xffffff),
		barLine:    core.RGB(0xebebeb),
	}
}

var fill = core.Style{FlexGrow: 1, FlexShrink: 1}

func label(text string, size core.Points, weight core.FontWeight, tint core.Color) core.Text {
	return core.Text{Content: text, Style: core.Style{FontSize: size, FontWeight: weight, Color: tint, FlexShrink: 1}}
}

func iconButton(icon glyph, name, id string, action func()) ui.IconButton {
	return ui.IconButton{Icon: icon, Label: name, TestID: id, OnPress: action, Size: 32, IconSize: 16}
}

func button(text, id string, icon glyph, action func()) ui.Button {
	return ui.Button{Text: text, TestID: id, Icon: icon, OnPress: action}
}

func row(children ...core.Node) core.View {
	return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8}, Children: children}
}

func spacer() core.View { return core.View{Style: core.Style{FlexGrow: 1}} }

func paperWrap(children ...core.Node) core.View {
	return core.View{
		Style: core.Style{
			Width: core.Pct(78), AlignSelf: core.AlignCenter,
		},
		Children: children,
	}
}

func (m *model) root(cx *core.Scope) core.Node {
	theme := *core.DefaultTheme()
	applyAccent := func(p *core.Palette, accent, hover, pressed, onAccent uint32) {
		p.Accent = core.RGB(accent)
		p.AccentHover = core.RGB(hover)
		p.AccentPressed = core.RGB(pressed)
		p.AccentText = core.RGB(onAccent)
		p.Focus = core.RGB(accent)
		p.Selection = core.RGBA(accent, .18)
	}
	applyAccent(&theme.Light, 0x1d1d1f, 0x27272a, 0x09090b, 0xffffff)
	applyAccent(&theme.Dark, 0xf5f5f7, 0xffffff, 0xd4d4d8, 0x141416)
	cx.SetTheme(&theme)
	cx.Effect(func() {
		displays := cx.Env().Displays()
		window := cx.Env().WindowFrame()
		at := core.Point{X: window.X + window.Width/2, Y: window.Y + window.Height/2}
		for _, display := range displays {
			if display.Frame.Contains(at) {
				m.display.Set(display.VisibleFrame)
				return
			}
		}
		for _, display := range displays {
			if display.Main {
				m.display.Set(display.VisibleFrame)
				return
			}
		}
	})
	desktop.SetTitleBar(cx, desktop.TitleBarHidden, 36)
	desktop.SetWindowControls(cx, desktop.WindowControls{TrafficLights: core.Point{X: 18, Y: 12}})
	searchRef := &core.Ref{}
	marketSearchRef := &core.Ref{}
	menuBtnRef := &core.Ref{}
	openSearch := func() {
		m.stage.Set("")
		m.searchOpen.Set(true)
		cx.Post(searchRef.Focus)
	}
	cx.OnKey("/", openSearch)
	cx.OnKey("mod+k", openSearch)
	cx.OnKey("mod+f", func() {
		m.stage.Set("")
		if m.page.GetUntracked().page == "plugin-market" {
			marketSearchRef.Focus()
			return
		}
		openSearch()
	})
	cx.OnKey("alt+left", m.goBack)
	cx.OnKey("alt+right", m.goForward)
	cx.OnKey("escape", func() {
		if m.sourceMenuOpen.GetUntracked() {
			m.sourceMenuOpen.Set(false)
		} else if m.searchOpen.GetUntracked() {
			m.searchOpen.Set(false)
		} else if m.signInDialog.GetUntracked().source != nil {
			m.closeSignInDialog()
		} else {
			m.closePlaybackPage()
		}
	})
	cx.OnKey("space", m.playback(m.player.TogglePlay))
	m.applyPreferences(cx)
	m.appMenu(cx)
	language := core.NewMemo(cx, func() string { return m.prefs.Get().Language })
	return core.Dynamic(func(cx *core.Scope) core.Node {
		_ = language.Get()
		p := colors(cx)
		return core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Background: p.background}, Children: []core.Node{
			core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1},
				AccessibilityElementsHidden: func() bool { return m.stage.Get() != "" },
				StyleFunc: func() core.Style {
					if m.stageCovered.Get() {
						return core.Style{Visibility: core.VisibilityHidden}
					}
					return core.Style{}
				}, Children: []core.Node{
					m.topNav(cx, menuBtnRef, openSearch),
					core.Show{When: func() bool { return m.error.Get() != "" }, Then: func(cx *core.Scope) core.Node {
						return row(core.Text{ContentFunc: m.error.Get, Style: core.Style{FlexGrow: 1, FlexShrink: 1, Color: colors(cx).accent, Padding: core.All(12)}},
							iconButton(lucide.X, i18n.T("Dismiss"), "dismiss", func() { m.error.Set("") }))
					}},
					m.subNav(cx, menuBtnRef, openSearch),
					core.View{Style: fill, Children: []core.Node{
						core.Dynamic(func(cx *core.Scope) core.Node { return m.pageView(cx, marketSearchRef) }),
					}},
					m.playerBar(cx),
				}},
			core.Show{When: func() bool { return m.stage.Get() != "" }, Then: m.nowPlaying},
			m.sourceMenuOverlay(cx, menuBtnRef),
			m.searchOverlay(cx, searchRef),
			m.signInDialogView(cx),
		}}
	})
}

func circleIconButton(p palette, icon glyph, name, id string, size, iconSize core.Points, bordered bool, active func() bool, action func()) core.Node {
	style := core.Style{
		Width: core.Pt(size), Height: core.Pt(size), CornerRadius: size / 2,
		Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
		Hover: &core.Style{Background: p.hover},
	}
	if bordered {
		style.BorderWidth = 1
		style.BorderColor = p.barLine
	}
	tint := func() core.Color {
		if active != nil && active() {
			return p.text
		}
		return p.muted
	}
	return base.Button{
		OnPress: func(core.PressEvent) { action() },
		Label:   name,
		TestID:  id,
		Style:   style,
		StyleFunc: func() core.Style {
			if active != nil && active() {
				return core.Style{Background: p.selected}
			}
			return core.Style{}
		},
		Children: []core.Node{ui.Icon(icon, iconSize, tint)},
	}
}

func (m *model) headerActionButton(p palette, ref *core.Ref, icon glyph, name, id string, active func() bool, action func()) core.Node {
	return base.Button{
		Ref:     ref,
		OnPress: func(core.PressEvent) { action() },
		Label:   name,
		TestID:  id,
		Style: core.Style{
			Width: core.Pt(32), Height: core.Pt(32), CornerRadius: 16,
			Background: core.Transparent, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: p.hover},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.94}},
		},
		StyleFunc: func() core.Style {
			if active != nil && active() {
				return core.Style{Background: p.selected}
			}
			return core.Style{}
		},
		Children: []core.Node{ui.Icon(icon, 15, func() core.Color {
			if active != nil && active() {
				return p.text
			}
			return p.muted
		})},
	}
}

func (m *model) hamburgerButton(p palette, ref *core.Ref) core.Node {
	return m.headerActionButton(p, ref, lucide.Menu, i18n.T("Sources"), "source-menu-trigger",
		m.sourceMenuOpen.Get,
		func() { m.sourceMenuOpen.Update(func(open bool) bool { return !open }) })
}

func (m *model) isSubNavVisible() bool {
	r := m.page.Get()
	if m.source.Get() == local.ID {
		return isLocalCategoryPage(r.page)
	}
	src := m.activeSource()
	if src == nil {
		return false
	}
	return isPluginCategoryPage(r, pluginNavEntries(src))
}

func (m *model) searchIconButton(cx *core.Scope, openSearch func()) core.Node {
	return m.headerActionButton(colors(cx), nil, lucide.Search, i18n.T("Search"), "search-trigger",
		m.searchOpen.Get, openSearch)
}

func (m *model) topNav(cx *core.Scope, menuBtnRef *core.Ref, openSearch func()) core.Node {
	p := colors(cx)
	left := core.Show{When: func() bool { return !m.isSubNavVisible() }, Then: func(*core.Scope) core.Node {
		back := circleIconButton(p, lucide.ChevronLeft, i18n.T("Back"), "back", 32, 14, true, nil, m.goBack)
		controls := row(m.hamburgerButton(p, menuBtnRef), back)
		controls.Style.Gap = 8
		return controls
	}}
	right := core.Show{When: func() bool { return !m.isSubNavVisible() }, Then: func(cx *core.Scope) core.Node {
		return m.searchIconButton(cx, openSearch)
	}}

	bar := row(left, spacer(), right)
	bar.Style.Padding = core.Edges{Left: 84, Right: 18}
	bar.Style.Height = core.Pt(36)
	bar.Desktop.WindowDrag = core.WindowDragOn
	return bar
}

const wordmarkFont = "Snell Roundhand, Bradley Hand, Georgia, serif"

func (m *model) subNav(cx *core.Scope, menuBtnRef *core.Ref, openSearch func()) core.Node {
	p := colors(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		r := m.page.Get()
		sourceID := m.source.Get()
		if sourceID == local.ID {
			if !isLocalCategoryPage(r.page) {
				return nil
			}
			m.updated.Get()
			sub := i18n.T("Local Music")
			if count := m.lib.TrackCount(); count > 0 {
				sub += " · " + i18n.Songs(count)
			}
			brand := row(
				core.Text{Content: "Yoho Music", MaxLines: 1, Style: core.Style{FontFamily: wordmarkFont, FontSize: 24, FontWeight: core.FontWeightNormal, LetterSpacing: -0.3, Color: p.text}},
				label(sub, 12, core.FontWeightNormal, p.muted),
			)
			brand.Style.Gap = 10
			items := []core.Node{
				m.subNavTo(cx, i18n.T("Home"), route{page: "home"}),
				m.subNavTo(cx, i18n.T("Songs"), route{page: "songs"}),
				m.subNavTo(cx, i18n.T("Albums"), route{page: "albums"}),
				m.subNavTo(cx, i18n.T("Artists"), route{page: "artists"}),
				m.subNavTo(cx, i18n.T("Liked Songs"), route{page: "liked"}),
				m.subNavTo(cx, i18n.T("Playlists"), route{page: "playlists"}),
			}
			return m.subNavBar(cx, p, menuBtnRef, brand, items, openSearch)
		}
		src := m.activeSource()
		if src == nil {
			return nil
		}
		entries := pluginNavEntries(src)
		if !isPluginCategoryPage(r, entries) {
			return nil
		}
		brand := row(
			core.Text{Content: sourceDisplayName(src.Info()), MaxLines: 1, Style: core.Style{FontFamily: wordmarkFont, FontSize: 24, FontWeight: core.FontWeightNormal, LetterSpacing: -0.3, Color: p.text}},
		)
		items := make([]core.Node, 0, len(entries))
		for _, entry := range entries {
			items = append(items, m.subNavTo(cx, entry.title, entry.target))
		}
		return m.subNavBar(cx, p, menuBtnRef, brand, items, openSearch)
	})
}

func isLocalCategoryPage(page string) bool {
	switch page {
	case "home", "songs", "albums", "artists", "liked", "playlists":
		return true
	}
	return false
}

func (m *model) subNavBar(cx *core.Scope, p palette, menuBtnRef *core.Ref, brand core.Node, items []core.Node, openSearch func()) core.Node {
	// Hang the search icon and hamburger button together on the right of the paper column
	// so the wordmark aligns flush-left and the last category pill aligns flush-right with the content below.
	rightTools := row(m.searchIconButton(cx, openSearch), m.hamburgerButton(p, menuBtnRef))
	rightTools.Style.Gap = 6
	hangingTools := core.View{
		Style:    core.Style{Position: core.Absolute, Right: core.Pt(-80), Top: core.Pt(1)},
		Children: []core.Node{rightTools},
	}
	tabs := row(items...)
	tabs.Style.Gap = 2
	bar := row(brand, spacer(), tabs)
	bar.Style.Padding = core.Edges{Top: 2, Bottom: 8}
	bar.Style.Gap = 12
	return paperWrap(core.View{
		Children: []core.Node{hangingTools, bar},
	})
}

func (m *model) sourceMenuOverlay(cx *core.Scope, menuBtnRef *core.Ref) core.Node {
	return core.Show{When: m.sourceMenuOpen.Get, Then: func(cx *core.Scope) core.Node {
		p := colors(cx)
		closeMenu := func() { m.sourceMenuOpen.Set(false) }
		const menuWidth = core.Points(228)
		menuLeft, menuTop := core.Points(88), core.Points(50)
		if bounds, ok := menuBtnRef.Bounds(); ok {
			menuTop = bounds.Origin.Y + bounds.Size.Height + 8
			if bounds.Origin.X > 260 {
				menuLeft = max(core.Points(16), bounds.Origin.X+bounds.Size.Width-menuWidth)
			} else {
				menuLeft = bounds.Origin.X
			}
		}
		option := func(icon glyph, text, sourceID, id string) core.Node {
			selected := func() bool { return m.source.Get() == sourceID }
			check := core.Show{When: selected, Then: func(*core.Scope) core.Node {
				return ui.Icon(lucide.Check, 15, func() core.Color { return p.text })
			}}
			name := core.Text{Content: text, MaxLines: 1, Style: core.Style{FontSize: 14, FontWeight: core.FontWeightSemibold, Color: p.text, FlexGrow: 1, FlexShrink: 1}}
			return base.Button{
				OnPress: func(core.PressEvent) {
					closeMenu()
					m.selectSource(sourceID)
				},
				Label:    text,
				TestID:   id,
				Selected: selected,
				Style: core.Style{
					Direction: core.Row, Height: core.Pt(40), AlignItems: core.AlignCenter, Gap: 12,
					Padding: core.Symmetric(0, 12), CornerRadius: 14,
					Hover: &core.Style{Background: p.hover},
				},
				Children: []core.Node{ui.Icon(icon, 16, func() core.Color { return p.text }), name, check},
			}
		}

		var rows []core.Node
		rows = append(rows, option(lucide.Library, "Yoho Music", local.ID, "source-local"))
		for _, plug := range m.plugins.Get() {
			info := plug.Info()
			rows = append(rows, option(lucide.Radio, sourceDisplayName(info), info.ID, "source-"+info.ID))
		}
		if src := m.activeSource(); src != nil && src.Info().ID != local.ID {
			if acct := m.pluginAccountMenuItems(cx, src, closeMenu); acct != nil {
				rows = append(rows,
					core.View{Style: core.Style{Height: core.Pt(1), Background: p.line, Margin: core.Symmetric(4, 10)}},
					acct,
				)
			}
		}

		// Bottom utility section in the hamburger menu: Plugin Market & Settings.
		menuAction := func(icon glyph, text, shortcut, id string, action func()) core.Node {
			children := []core.Node{
				ui.Icon(icon, 15, func() core.Color { return p.muted }),
				label(text, 13, core.FontWeightMedium, p.text),
				spacer(),
			}
			if shortcut != "" {
				children = append(children, label(shortcut, 11, core.FontWeightNormal, p.muted))
			}
			return base.Button{
				OnPress: func(core.PressEvent) {
					closeMenu()
					action()
				},
				Label:  text,
				TestID: id,
				Style: core.Style{
					Direction: core.Row, Height: core.Pt(36), AlignItems: core.AlignCenter, Gap: 10,
					Padding: core.Symmetric(0, 12), CornerRadius: 12,
					Hover: &core.Style{Background: p.hover},
				},
				Children: children,
			}
		}
		rows = append(rows,
			core.View{Style: core.Style{Height: core.Pt(1), Background: p.line, Margin: core.Symmetric(4, 10)}},
			menuAction(lucide.Store, i18n.T("Plugin Market"), "", "nav-plugin-market", func() { m.navigate(route{page: "plugin-market"}) }),
			menuAction(lucide.Settings, i18n.T("Settings"), "⌘,", "settings", func() { m.navigate(route{page: "settings"}) }),
		)

		cardBg := p.background
		shadowAlpha := float32(.11)
		if cx.Dark() {
			cardBg = p.sidebar
			shadowAlpha = .42
		}
		card := core.View{
			TestID: "source-menu",
			Style: core.Style{
				Position: core.Absolute, Left: core.Pt(menuLeft), Top: core.Pt(menuTop),
				Width: core.Pt(228), CornerRadius: 22,
				Background: cardBg, BorderWidth: 1, BorderColor: p.barLine,
				Padding: core.All(8), Gap: 2,
				ShadowColor: core.RGBA(0x000000, shadowAlpha), ShadowOffset: core.Point{Y: 8}, ShadowRadius: 24,
			},
			Children: rows,
		}
		backdrop := base.Button{
			Label:   i18n.T("Close"),
			TestID:  "source-menu-backdrop",
			OnPress: func(core.PressEvent) { closeMenu() },
			Style:   core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0)},
		}
		return core.View{
			Style:    core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0)},
			Children: []core.Node{backdrop, card},
		}
	}}
}

func sourceDisplayName(info music.Info) string {
	if info.ID == plugin.NetEaseSourceID {
		return "NetEase Music"
	}
	return info.Name
}

func (m *model) subNavTo(cx *core.Scope, text string, target route) core.Node {
	return m.subNavPill(cx, text, "nav-"+target.page+target.id, true, func() { m.navigate(target) }, func() bool { return m.page.Get() == target })
}

func (m *model) subNavPill(cx *core.Scope, text, id string, selectable bool, action func(), selected ...func() bool) core.Node {
	p := colors(cx)
	isSelected := func() bool { return selectable && len(selected) > 0 && selected[0]() }
	t := core.Text{Content: text, MaxLines: 1,
		Style: core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: p.muted},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Color: p.onButton}
			}
			return core.Style{}
		},
	}
	return base.Button{
		OnPress:  func(core.PressEvent) { action() },
		Label:    text,
		TestID:   id,
		Selected: isSelected,
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(30), AlignItems: core.AlignCenter,
			Padding: core.Symmetric(0, 12), CornerRadius: 15,
			Hover: &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Background: p.button}
			}
			return core.Style{}
		},
		Children: []core.Node{t},
	}
}

type quickSearchState struct {
	query     string
	loading   bool
	tracks    []music.Track
	albums    []music.Album
	playlists []music.Playlist
}

func (m *model) searchOverlay(cx *core.Scope, searchRef *core.Ref) core.Node {
	return core.Show{When: m.searchOpen.Get, Then: func(cx *core.Scope) core.Node {
		p := colors(cx)
		closeModal := func() { m.searchOpen.Set(false) }
		results := core.NewSignalFunc(cx, quickSearchState{}, nil)
		var seq int
		var debounce func() bool
		runQuick := func(raw string) {
			if debounce != nil {
				debounce()
				debounce = nil
			}
			q := strings.TrimSpace(raw)
			seq++
			gen := seq
			if q == "" {
				results.Set(quickSearchState{})
				return
			}
			src := m.activeSource()
			if src == nil || !src.Can("search") {
				results.Set(quickSearchState{query: q})
				return
			}
			results.Set(quickSearchState{query: q, loading: true})
			delay := 180 * time.Millisecond
			if src.Info().ID == local.ID {
				delay = 40 * time.Millisecond
			}
			debounce = cx.AfterFunc(delay, func() {
				go func() {
					sourceID := src.Info().ID
					var tracks []music.Track
					var albums []music.Album
					var playlists []music.Playlist
					if res, err := src.Search(q, music.SearchTracks, 0); err == nil && res != nil {
						tracks = withSource(res.Tracks[:min(len(res.Tracks), 4)], sourceID)
					}
					if res, err := src.Search(q, music.SearchAlbums, 0); err == nil && res != nil {
						albums = res.Albums[:min(len(res.Albums), 2)]
					}
					if res, err := src.Search(q, music.SearchPlaylists, 0); err == nil && res != nil {
						playlists = res.Playlists[:min(len(res.Playlists), 2)]
					}
					cx.Post(func() {
						if gen != seq {
							return
						}
						results.Set(quickSearchState{query: q, tracks: tracks, albums: albums, playlists: playlists})
					})
				}()
			})
		}
		cx.Cleanup(func() {
			if debounce != nil {
				debounce()
			}
		})
		if initial := m.query.GetUntracked(); strings.TrimSpace(initial) != "" {
			cx.Post(func() { runQuick(initial) })
		}

		spotThumb := func(cx *core.Scope, url, source string) core.Node {
			face := core.Style{
				Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 6,
				Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			}
			if url == "" {
				return core.View{Style: face, Children: []core.Node{ui.Icon(lucide.Music, 13, func() core.Color { return p.muted })}}
			}
			return core.Image{SourceFunc: m.coverSource(cx, url, 56, source), Style: face, ResizeMode: core.Cover}
		}

		sourceLabel := "Yoho Music"
		if src := m.activeSource(); src != nil && src.Info().ID != local.ID {
			sourceLabel = sourceDisplayName(src.Info())
		}
		sourceBadge := core.View{
			Style: core.Style{
				Height: core.Pt(22), Padding: core.Symmetric(0, 9), CornerRadius: 11,
				Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			},
			Children: []core.Node{label(sourceLabel, 11, core.FontWeightMedium, p.muted)},
		}
		input := base.Input{
			Ref:   searchRef,
			Value: m.query.Get,
			OnChange: func(q string) {
				m.query.Set(q)
				runQuick(q)
			},
			OnSubmit:    func(q string) { m.searchOpen.Set(false); m.search(q) },
			Placeholder: i18n.T("Search"),
			TestID:      "search",
			Style: core.Style{
				FlexGrow: 1, FlexShrink: 1, Height: core.Pt(30),
				Padding: core.All(0), Background: core.Transparent,
				FontSize: 14, Color: p.text, PlaceholderColor: p.muted,
				CaretColor: p.text, SelectionColor: cx.Palette().Selection,
			},
		}
		searchBar := row(ui.Icon(lucide.Search, 15, func() core.Color { return p.muted }), input, sourceBadge)
		searchBar.Style.Height = core.Pt(48)
		searchBar.Style.Padding = core.Symmetric(0, 16)
		searchBar.Style.Gap = 10

		body := core.Dynamic(func(cx *core.Scope) core.Node {
			st := results.Get()
			if st.query == "" {
				return nil
			}
			var sections []core.Node
			if st.loading {
				loading := label(i18n.T("Loading…"), 12, core.FontWeightNormal, p.muted)
				loading.Style.Padding = core.Symmetric(10, 10)
				sections = append(sections, loading)
			} else if len(st.tracks)+len(st.albums)+len(st.playlists) == 0 {
				empty := label(i18n.T("Nothing here yet"), 12, core.FontWeightNormal, p.muted)
				empty.Style.Padding = core.Symmetric(10, 10)
				sections = append(sections, empty)
			} else {
				if len(st.tracks) > 0 {
					heading := label(i18n.T("Songs"), 11, core.FontWeightNormal, p.muted)
					heading.Style.Padding = core.Edges{Top: 2, Right: 10, Bottom: 4, Left: 10}
					sections = append(sections, heading)
					for i, tr := range st.tracks {
						track := tr
						idx := i
						art := spotThumb(cx, track.Album.ImageURL, track.Source)
						name := label(track.Name, 13, core.FontWeightMedium, p.text)
						name.MaxLines = 1
						artist := label(track.ArtistNames(), 12, core.FontWeightNormal, p.muted)
						artist.MaxLines = 1
						artist.Style.FlexGrow = 1
						dur := label(formatTime(track.Length()), 11, core.FontWeightNormal, p.muted)
						dur.Style.FontVariant = core.FontTabularNums
						rowStyle := core.Style{
							Direction: core.Row, Height: core.Pt(38), AlignItems: core.AlignCenter, Gap: 10,
							Padding: core.Symmetric(0, 10), CornerRadius: 8,
							Hover: &core.Style{Background: p.hover},
						}
						if idx == 0 {
							rowStyle.Background = p.hover
						}
						sections = append(sections, base.Button{
							Label: track.Name,
							OnPress: func(core.PressEvent) {
								closeModal()
								m.playback(func() { m.player.PlayTrack(st.tracks, idx) })()
							},
							Style:    rowStyle,
							Children: []core.Node{art, name, artist, dur},
						})
					}
				}
				if len(st.albums)+len(st.playlists) > 0 {
					if len(st.tracks) > 0 {
						sections = append(sections, core.View{Style: core.Style{Height: core.Pt(1), Background: p.barLine, Margin: core.Symmetric(6, 8)}})
					}
					heading := label(i18n.T("Albums")+" / "+i18n.T("Playlists"), 11, core.FontWeightNormal, p.muted)
					heading.Style.Padding = core.Edges{Top: 2, Right: 10, Bottom: 4, Left: 10}
					sections = append(sections, heading)
					sourceID := m.source.GetUntracked()
					for _, al := range st.albums {
						album := al
						art := spotThumb(cx, album.ImageURL, sourceID)
						name := label(album.Name, 13, core.FontWeightMedium, p.text)
						name.MaxLines = 1
						sub := label(albumSubtitle(album), 12, core.FontWeightNormal, p.muted)
						sub.MaxLines = 1
						sub.Style.FlexGrow = 1
						kind := label(i18n.T("Album"), 11, core.FontWeightNormal, p.muted)
						sections = append(sections, base.Button{
							Label: album.Name,
							OnPress: func(core.PressEvent) {
								closeModal()
								m.navigate(route{page: "album", id: album.ID})
							},
							Style: core.Style{
								Direction: core.Row, Height: core.Pt(38), AlignItems: core.AlignCenter, Gap: 10,
								Padding: core.Symmetric(0, 10), CornerRadius: 8,
								Hover: &core.Style{Background: p.hover},
							},
							Children: []core.Node{art, name, sub, kind},
						})
					}
					for _, pl := range st.playlists {
						list := pl
						art := spotThumb(cx, list.ImageURL, sourceID)
						name := label(list.Name, 13, core.FontWeightMedium, p.text)
						name.MaxLines = 1
						sub := label(list.Owner, 12, core.FontWeightNormal, p.muted)
						sub.MaxLines = 1
						sub.Style.FlexGrow = 1
						kind := label(i18n.T("Playlist"), 11, core.FontWeightNormal, p.muted)
						sections = append(sections, base.Button{
							Label: list.Name,
							OnPress: func(core.PressEvent) {
								closeModal()
								m.navigate(route{page: "playlist", id: list.ID})
							},
							Style: core.Style{
								Direction: core.Row, Height: core.Pt(38), AlignItems: core.AlignCenter, Gap: 10,
								Padding: core.Symmetric(0, 10), CornerRadius: 8,
								Hover: &core.Style{Background: p.hover},
							},
							Children: []core.Node{art, name, sub, kind},
						})
					}
				}
			}
			listBox := core.View{
				Style:    core.Style{Padding: core.All(8), Gap: 2, Borders: core.Edges{Top: 1}, BorderColor: p.barLine},
				Children: sections,
			}
			footer := row(
				label("↵ "+i18n.T("Show All"), 11, core.FontWeightNormal, p.muted),
				spacer(),
				label("ESC", 11, core.FontWeightNormal, p.muted),
			)
			footer.Style.Height = core.Pt(34)
			footer.Style.Padding = core.Symmetric(0, 14)
			footer.Style.Background = p.sidebar
			footer.Style.Borders.Top = 1
			footer.Style.BorderColor = p.barLine
			return core.View{Children: []core.Node{listBox, footer}}
		})

		cardBg := p.background
		shadowAlpha := float32(.14)
		if cx.Dark() {
			cardBg = p.sidebar
			shadowAlpha = .48
		}
		inner := core.View{
			Style:    core.Style{CornerRadius: 14, Overflow: core.Hidden, Background: cardBg},
			Children: []core.Node{searchBar, body},
		}
		card := core.View{
			Style: core.Style{
				Width: core.Pt(470), MaxWidth: core.Pct(90),
				Background: cardBg, CornerRadius: 14,
				BorderWidth: 1, BorderColor: p.barLine,
				ShadowColor: core.RGBA(0x000000, shadowAlpha), ShadowOffset: core.Point{Y: 16}, ShadowRadius: 36,
			},
			Children: []core.Node{inner},
		}
		backdrop := base.Button{
			Label:   i18n.T("Close"),
			TestID:  "search-backdrop",
			OnPress: func(core.PressEvent) { closeModal() },
			Style:   core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0), Background: core.RGBA(0x000000, .24)},
		}
		return core.View{
			Style: core.Style{
				Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0),
				AlignItems: core.AlignCenter, Padding: core.Edges{Top: 84},
			},
			Children: []core.Node{backdrop, card},
		}
	}}
}

func (m *model) pickMusic(cx *core.Scope) {
	cx.PickDirectories(core.PickDirectoriesOptions{Multiple: true, Title: i18n.T("Add a Music Folder")}, func(dirs []core.Directory, err error) {
		m.report(err)
		var paths []string
		for _, d := range dirs {
			if d.Path != "" {
				paths = append(paths, d.Path)
			}
		}
		if len(paths) > 0 {
			m.run(func() error { m.lib.AddFolders(paths...); return nil })
		}
	})
}

func (m *model) selectSource(id string) {
	m.source.Set(id)
	m.page.Set(route{page: "home"})
	m.stage.Set("")
	m.back, m.forward = nil, nil
	m.history.Update(func(n int) int { return n + 1 })
	if source := m.activeSource(); source != nil {
		m.refreshAccount(source)
	}
}

func (m *model) activeSource() music.Source { return m.sourceByID(m.source.Get()) }
