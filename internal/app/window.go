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
}

func colors(cx *core.Scope) palette {
	if cx.Dark() {
		return palette{core.RGB(0x141416), core.RGB(0x1c1c1f), core.RGB(0xf5f5f7), core.RGB(0xa5a5ae),
			core.RGBA(0xffffff, .08), core.RGBA(0xffffff, .06), core.RGBA(0xffffff, .11), core.RGB(0xff5366)}
	}
	return palette{core.RGB(0xffffff), core.RGB(0xf4f4f6), core.RGB(0x1d1d1f), core.RGB(0x62626b),
		core.RGBA(0x000000, .08), core.RGBA(0x000000, .04), core.RGBA(0x000000, .08), core.RGB(0xce2848)}
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

func (m *model) root(cx *core.Scope) core.Node {
	theme := *core.DefaultTheme()
	applyAccent := func(p *core.Palette, accent, hover, pressed uint32) {
		p.Accent = core.RGB(accent)
		p.AccentHover = core.RGB(hover)
		p.AccentPressed = core.RGB(pressed)
		p.AccentText = core.RGB(0xffffff)
		p.Focus = core.RGB(accent)
		p.Selection = core.RGBA(accent, .26)
	}
	applyAccent(&theme.Light, 0xce2848, 0xbe2341, 0xad203b)
	applyAccent(&theme.Dark, 0xd63555, 0xe74765, 0xc72e4d)
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
	desktop.SetTitleBar(cx, desktop.TitleBarHidden, 42)
	desktop.SetWindowControls(cx, desktop.WindowControls{TrafficLights: core.Point{X: 20, Y: 15}})
	searchRef := &core.Ref{}
	marketSearchRef := &core.Ref{}
	cx.OnKey("mod+f", func() {
		m.stage.Set("")
		if m.page.GetUntracked().page == "plugin-market" {
			marketSearchRef.Focus()
		} else {
			searchRef.Focus()
		}
	})
	cx.OnKey("alt+left", m.goBack)
	cx.OnKey("alt+right", m.goForward)
	cx.OnKey("escape", func() {
		if m.signInDialog.GetUntracked().source != nil {
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
			core.View{Style: core.Style{Direction: core.Row, FlexGrow: 1, FlexShrink: 1},
				AccessibilityElementsHidden: func() bool { return m.stage.Get() != "" },
				StyleFunc: func() core.Style {
					if m.stageCovered.Get() {
						return core.Style{Visibility: core.VisibilityHidden}
					}
					return core.Style{}
				}, Children: []core.Node{
					m.sidebar(cx), core.View{Style: fill, Children: []core.Node{
						m.toolbar(cx, searchRef),
						core.Show{When: func() bool { return m.error.Get() != "" }, Then: func(cx *core.Scope) core.Node {
							return row(core.Text{ContentFunc: m.error.Get, Style: core.Style{FlexGrow: 1, FlexShrink: 1, Color: colors(cx).accent, Padding: core.All(12)}},
								iconButton(lucide.X, i18n.T("Dismiss"), "dismiss", func() { m.error.Set("") }))
						}},
						core.Dynamic(func(cx *core.Scope) core.Node { return m.pageView(cx, marketSearchRef) }), m.playerBar(cx),
					}},
				}},
			core.Show{When: func() bool { return m.stage.Get() != "" }, Then: m.nowPlaying},
			m.signInDialogView(cx),
		}}
	})
}

func (m *model) toolbar(cx *core.Scope, searchRef *core.Ref) core.Node {
	back := iconButton(lucide.ChevronLeft, i18n.T("Back"), "back", m.goBack)
	back.Disabled = func() bool { m.history.Get(); return len(m.back) == 0 }
	forward := iconButton(lucide.ChevronRight, i18n.T("Forward"), "forward", m.goForward)
	forward.Disabled = func() bool { m.history.Get(); return len(m.forward) == 0 }
	search := core.Show{When: func() bool { return m.page.Get().page != "plugin-market" }, Then: func(*core.Scope) core.Node {
		return ui.Input{Ref: searchRef, Value: m.query.Get, OnChange: m.query.Set, OnSubmit: m.search,
			Placeholder: i18n.T("Search"), TestID: "search", Style: core.Style{Width: core.Pt(244)}}
	}}
	bar := row(back, forward, spacer(), search,
		iconButton(lucide.Settings, i18n.T("Settings"), "settings", func() { m.navigate(route{page: "settings"}) }))
	bar.Style.Padding = core.Symmetric(4, 24)
	bar.Style.Height = core.Pt(46)
	bar.Desktop.WindowDrag = core.WindowDragOn
	return bar
}

func (m *model) sidebar(cx *core.Scope) core.Node {
	p := colors(cx)
	return core.View{Style: core.Style{Width: core.Pt(232), Background: p.sidebar, Padding: core.Edges{Top: 44, Right: 12, Bottom: 14, Left: 12}, Gap: 10}, Children: []core.Node{
		core.Dynamic(func(cx *core.Scope) core.Node {
			var items []core.Node
			plugins := m.plugins.Get()
			if len(plugins) > 0 {
				items = append(items, m.nav(cx, lucide.HardDrive, i18n.T("Local Music"), "source-local", false, func() { m.selectSource(local.ID) }))
				for _, p := range plugins {
					info := p.Info()
					items = append(items, m.nav(cx, lucide.Radio, info.Name, "source-"+info.ID, false, func() { m.selectSource(info.ID) }))
				}
			}
			return core.Group(items)
		}),
		core.Scroll{Style: fill, Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			if m.source.Get() != local.ID {
				return m.pluginNavigation(cx)
			}
			m.updated.Get()
			items := []core.Node{
				m.sectionTitle(cx, i18n.T("Local Music")),
				m.navTo(cx, lucide.House, i18n.T("Home"), route{page: "home"}),
				m.navTo(cx, lucide.Music, i18n.T("Songs"), route{page: "songs"}),
				m.navTo(cx, lucide.Disc3, i18n.T("Albums"), route{page: "albums"}),
				m.navTo(cx, lucide.Users, i18n.T("Artists"), route{page: "artists"}),
				m.navTo(cx, lucide.Heart, i18n.T("Liked Songs"), route{page: "liked"}),
			}
			header := row(m.sectionTitle(cx, i18n.T("My Playlists")), spacer(), iconButton(lucide.Plus, i18n.T("New Playlist"), "new-playlist", func() { m.navigate(route{page: "playlist-edit"}) }))
			header.Style.Margin.Top = 14
			items = append(items, header)
			lists, _ := m.lib.Playlists()
			for _, list := range lists {
				items = append(items, m.navTo(cx, lucide.ListMusic, list.Name, route{page: "playlist", id: list.ID}))
			}
			return core.View{Style: core.Style{Gap: 2}, Children: items}
		})}},
		core.View{Style: core.Style{Gap: 3}, Children: []core.Node{
			m.navTo(cx, lucide.Store, i18n.T("Plugin Market"), route{page: "plugin-market"}),
			m.navTo(cx, lucide.Settings, i18n.T("Settings"), route{page: "settings"}),
		}},
	}}
}

func (m *model) sectionTitle(cx *core.Scope, title string) core.Node {
	t := label(title, 11, core.FontWeightBold, colors(cx).muted)
	t.Style.Padding = core.Symmetric(4, 10)
	return t
}

func (m *model) navTo(cx *core.Scope, icon glyph, text string, target route) core.Node {
	return m.nav(cx, icon, text, "nav-"+target.page+target.id, true, func() { m.navigate(target) }, func() bool { return m.page.Get() == target })
}

func (m *model) nav(cx *core.Scope, icon glyph, text, id string, selectable bool, action func(), selected ...func() bool) core.Node {
	p := colors(cx)
	isSelected := func() bool { return selectable && len(selected) > 0 && selected[0]() }
	t := label(text, 13, core.FontWeightMedium, p.text)
	t.MaxLines = 1
	return base.Button{OnPress: func(core.PressEvent) { action() }, Label: text, TestID: id, Selected: isSelected,
		Style: core.Style{Direction: core.Row, Height: core.Pt(34), AlignItems: core.AlignCenter, Gap: 10, Padding: core.Symmetric(0, 10), CornerRadius: 7,
			Hover: &core.Style{Background: p.hover}},
		StyleFunc: func() core.Style {
			if isSelected() {
				return core.Style{Background: p.selected}
			}
			return core.Style{}
		},
		Children: []core.Node{ui.Icon(icon, 16, func() core.Color { return p.accent }), t},
	}
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
