package app

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
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

const paperWidth = core.Points(900)

func paperWrap(children ...core.Node) core.View {
	return core.View{
		Style: core.Style{
			Width: core.Pct(100), MaxWidth: core.Pt(paperWidth),
			AlignSelf: core.AlignCenter,
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
	desktop.SetTitleBar(cx, desktop.TitleBarHidden, 52)
	desktop.SetWindowControls(cx, desktop.WindowControls{TrafficLights: core.Point{X: 20, Y: 20}})
	searchRef := &core.Ref{}
	marketSearchRef := &core.Ref{}
	openSearch := func() {
		m.stage.Set("")
		if m.page.GetUntracked().page == "plugin-market" {
			marketSearchRef.Focus()
			return
		}
		m.searchOpen.Set(true)
		cx.Post(searchRef.Focus)
	}
	cx.OnKey("mod+k", openSearch)
	cx.OnKey("mod+f", openSearch)
	cx.OnKey("alt+left", m.goBack)
	cx.OnKey("alt+right", m.goForward)
	cx.OnKey("escape", func() {
		if m.searchOpen.GetUntracked() {
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
					m.topNav(cx, openSearch),
					core.Show{When: func() bool { return m.error.Get() != "" }, Then: func(cx *core.Scope) core.Node {
						return row(core.Text{ContentFunc: m.error.Get, Style: core.Style{FlexGrow: 1, FlexShrink: 1, Color: colors(cx).accent, Padding: core.All(12)}},
							iconButton(lucide.X, i18n.T("Dismiss"), "dismiss", func() { m.error.Set("") }))
					}},
					m.subNav(cx),
					core.View{Style: fill, Children: []core.Node{
						core.Dynamic(func(cx *core.Scope) core.Node { return m.pageView(cx, marketSearchRef) }),
					}},
					m.playerBar(cx),
				}},
			core.Show{When: func() bool { return m.stage.Get() != "" }, Then: m.nowPlaying},
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

func (m *model) topNav(cx *core.Scope, openSearch func()) core.Node {
	p := colors(cx)
	navCircle := func(icon glyph, name, id string, disabled func() bool, action func()) core.Node {
		return base.Button{
			OnPress:  func(core.PressEvent) { action() },
			Label:    name,
			TestID:   id,
			Disabled: disabled,
			Style: core.Style{
				Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14,
				Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				Hover: &core.Style{Background: p.hover},
			},
			StyleFunc: func() core.Style {
				if disabled != nil && disabled() {
					return core.Style{Opacity: core.Alpha(.40)}
				}
				return core.Style{}
			},
			Children: []core.Node{ui.Icon(icon, 14, func() core.Color { return p.muted })},
		}
	}
	back := navCircle(lucide.ChevronLeft, i18n.T("Back"), "back", func() bool { m.history.Get(); return len(m.back) == 0 }, m.goBack)
	forward := navCircle(lucide.ChevronRight, i18n.T("Forward"), "forward", func() bool { m.history.Get(); return len(m.forward) == 0 }, m.goForward)
	left := row(back, forward)
	left.Style.Gap = 4

	// Ian's Daily segmented control for source switching.
	sources := core.Dynamic(func(cx *core.Scope) core.Node {
		items := []core.Node{
			m.sourceSegment(cx, "Yoho Music", local.ID, "source-local"),
		}
		for _, plug := range m.plugins.Get() {
			info := plug.Info()
			items = append(items, m.sourceSegment(cx, info.Name, info.ID, "source-"+info.ID))
		}
		seg := row(items...)
		seg.Style.Gap = 2
		seg.Style.Padding = core.All(3)
		seg.Style.CornerRadius = 9999
		seg.Style.Background = p.sidebar
		seg.Style.BorderWidth = 1
		seg.Style.BorderColor = p.barLine
		return seg
	})

	search := core.Show{When: func() bool { return m.page.Get().page != "plugin-market" }, Then: func(*core.Scope) core.Node {
		shortcut := core.Text{Content: "⌘K", Style: core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: p.muted}}
		return base.Button{
			OnPress: func(core.PressEvent) { openSearch() },
			Label:   i18n.T("Search"),
			TestID:  "search-trigger",
			Style: core.Style{
				Direction: core.Row, Height: core.Pt(32), AlignItems: core.AlignCenter, Gap: 6,
				Padding: core.Symmetric(0, 12), CornerRadius: 16,
				Background: p.sidebar, BorderWidth: 1, BorderColor: p.barLine,
				Hover: &core.Style{Background: p.hover},
			},
			Children: []core.Node{ui.Icon(lucide.Search, 13, func() core.Color { return p.muted }), shortcut},
		}
	}}
	market := circleIconButton(p, lucide.Store, i18n.T("Plugin Market"), "nav-plugin-market", 32, 14, true,
		func() bool { return m.page.Get().page == "plugin-market" },
		func() { m.navigate(route{page: "plugin-market"}) })
	settings := circleIconButton(p, lucide.Settings, i18n.T("Settings"), "settings", 32, 14, true,
		func() bool { return m.page.Get().page == "settings" },
		func() { m.navigate(route{page: "settings"}) })
	right := row(search, market, settings)
	right.Style.Gap = 8

	bar := row(left, spacer(), sources, spacer(), right)
	bar.Style.Padding = core.Edges{Left: 88, Right: 20}
	bar.Style.Height = core.Pt(52)
	bar.Desktop.WindowDrag = core.WindowDragOn
	return bar
}

func (m *model) sourceSegment(cx *core.Scope, text, sourceID, id string) core.Node {
	p := colors(cx)
	isSelected := func() bool {
		page := m.page.Get().page
		return m.source.Get() == sourceID && page != "settings" && page != "plugin-market"
	}
	t := core.Text{Content: text, MaxLines: 1,
		Style: core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: p.muted},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Color: p.text}
			}
			return core.Style{}
		},
	}
	return base.Button{
		OnPress:  func(core.PressEvent) { m.selectSource(sourceID) },
		Label:    text,
		TestID:   id,
		Selected: isSelected,
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(26), AlignItems: core.AlignCenter,
			Padding: core.Symmetric(0, 14), CornerRadius: 13,
			Hover: &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Background: p.background, ShadowColor: core.RGBA(0x000000, .08), ShadowOffset: core.Point{Y: 1}, ShadowRadius: 2}
			}
			return core.Style{}
		},
		Children: []core.Node{t},
	}
}

func (m *model) subNav(cx *core.Scope) core.Node {
	p := colors(cx)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		r := m.page.Get()
		sourceID := m.source.Get()
		if sourceID == local.ID {
			if !isLocalCategoryPage(r.page) {
				return nil
			}
			items := []core.Node{
				m.subNavTo(cx, i18n.T("Home"), route{page: "home"}),
				m.subNavTo(cx, i18n.T("Songs"), route{page: "songs"}),
				m.subNavTo(cx, i18n.T("Albums"), route{page: "albums"}),
				m.subNavTo(cx, i18n.T("Artists"), route{page: "artists"}),
				m.subNavTo(cx, i18n.T("Liked Songs"), route{page: "liked"}),
				m.subNavTo(cx, i18n.T("Playlists"), route{page: "playlists"}),
			}
			return subNavBar(p, items, nil)
		}
		src := m.activeSource()
		if src == nil {
			return nil
		}
		entries := pluginNavEntries(src)
		if !isPluginCategoryPage(r, entries) {
			return nil
		}
		items := make([]core.Node, 0, len(entries))
		for _, entry := range entries {
			items = append(items, m.subNavTo(cx, entry.title, entry.target))
		}
		return subNavBar(p, items, m.pluginAccountControls(cx, src))
	})
}

func isLocalCategoryPage(page string) bool {
	switch page {
	case "home", "songs", "albums", "artists", "liked", "playlists":
		return true
	}
	return false
}

func subNavBar(p palette, items []core.Node, trailing core.Node) core.Node {
	_ = p
	tabs := row(items...)
	tabs.Style.Gap = 6
	scroll := core.Scroll{Horizontal: true, HideIndicator: true, Style: fill, Children: []core.Node{tabs}}
	children := []core.Node{scroll}
	if trailing != nil {
		children = append(children, trailing)
	}
	bar := row(children...)
	bar.Style.Padding = core.Edges{Top: 18, Right: 28, Bottom: 8, Left: 28}
	bar.Style.Gap = 12
	return paperWrap(bar)
}

func (m *model) subNavTo(cx *core.Scope, text string, target route) core.Node {
	return m.subNavChip(cx, text, "nav-"+target.page+target.id, true, func() { m.navigate(target) }, func() bool { return m.page.Get() == target })
}

func (m *model) subNavChip(cx *core.Scope, text, id string, selectable bool, action func(), selected ...func() bool) core.Node {
	p := colors(cx)
	isSelected := func() bool { return selectable && len(selected) > 0 && selected[0]() }
	t := core.Text{Content: text, MaxLines: 1,
		Style: core.Style{FontSize: 13, FontWeight: core.FontWeightNormal, Color: p.text},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Color: p.onButton, FontWeight: core.FontWeightMedium}
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
			Direction: core.Row, Height: core.Pt(28), AlignItems: core.AlignCenter,
			Padding: core.Symmetric(0, 10), CornerRadius: 0, Background: p.chip,
			Hover: &core.Style{Background: p.line},
		},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Background: p.button, CornerRadius: 2}
			}
			return core.Style{}
		},
		Children: []core.Node{t},
	}
}

func (m *model) searchOverlay(cx *core.Scope, searchRef *core.Ref) core.Node {
	return core.Show{When: m.searchOpen.Get, Then: func(cx *core.Scope) core.Node {
		p := colors(cx)
		closeModal := func() { m.searchOpen.Set(false) }
		input := ui.Input{
			Ref:         searchRef,
			Value:       m.query.Get,
			OnChange:    m.query.Set,
			OnSubmit:    func(q string) { m.searchOpen.Set(false); m.search(q) },
			Placeholder: i18n.T("Search"),
			TestID:      "search",
			Style:       fill,
		}
		card := core.View{
			Style: core.Style{
				Width: core.Pt(440), MaxWidth: core.Pct(90),
				Background: p.background, CornerRadius: 2,
				BorderWidth: 1, BorderColor: p.line,
				Padding: core.Symmetric(10, 14),
				ShadowColor: core.RGBA(0x000000, .20), ShadowOffset: core.Point{Y: 12}, ShadowRadius: 24,
			},
			Children: []core.Node{row(ui.Icon(lucide.Search, 15, func() core.Color { return p.muted }), input,
				iconButton(lucide.X, i18n.T("Close"), "search-close", closeModal))},
		}
		backdrop := base.Button{
			Label:   i18n.T("Close"),
			TestID:  "search-backdrop",
			OnPress: func(core.PressEvent) { closeModal() },
			Style:   core.Style{Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0), Background: core.RGBA(0x000000, .30)},
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
