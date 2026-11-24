package app

import (
	"strings"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/plugin"
)

func (m *model) pluginMarket(cx *core.Scope, searchRef *core.Ref) core.Node {
	query := core.NewSignal(cx, "")
	installedOnly := core.NewSignal(cx, false)
	p := colors(cx)
	importPackage := button(i18n.T("Import Package"), "market-import", lucide.PackagePlus, func() { m.pickNSGoPlugin(cx) })
	importPackage.Disabled = func() bool { return m.nsgoStore.Get() == nil || m.nsgoPluginBusy.Get() }
	manage := button(i18n.T("Manage"), "market-manage", nil, func() { m.navigate(route{page: "settings"}) })
	manage.Variant = ui.ButtonSecondary

	cx.Effect(func() {
		sources := m.plugins.Get()
		cx.Post(func() {
			for _, source := range sources {
				if m.sourceByID(source.Info().ID) == source {
					m.refreshAccount(source)
				}
			}
		})
	})

	filter := func(text, id string, installed bool) core.Node {
		selected := func() bool { return installedOnly.Get() == installed }
		return base.Button{Label: text, TestID: id, Selected: selected,
			OnPress: func(core.PressEvent) { installedOnly.Set(installed) },
			Style: core.Style{Padding: core.Symmetric(6, 12), CornerRadius: 7, MinHeight: core.Pt(30), JustifyContent: core.JustifyCenter,
				Hover: &core.Style{Background: p.hover}},
			StyleFunc: func() core.Style {
				if selected() {
					return core.Style{Background: p.selected}
				}
				return core.Style{}
			},
			Children: []core.Node{label(text, 13, core.FontWeightMedium, p.text)}}
	}
	searchLabel := i18n.T("Search Plugins")
	filters := row(filter(i18n.T("All"), "market-all", false), filter(i18n.T("Installed"), "market-installed", true), spacer(),
		ui.Input{Ref: searchRef, Value: query.Get, OnChange: query.Set, Placeholder: searchLabel, Label: searchLabel,
			TestID: "market-search", Style: core.Style{Width: core.Pt(240), FlexShrink: 1}})
	filters.Style.Padding = core.Symmetric(6, 28)

	list := core.Dynamic(func(cx *core.Scope) core.Node {
		term := strings.ToLower(strings.TrimSpace(query.Get()))
		matches := func(parts ...string) bool { return strings.Contains(strings.ToLower(strings.Join(parts, " ")), term) }
		var sections, installed []core.Node
		demoInstalled := false
		for _, source := range m.plugins.Get() {
			info := source.Info()
			isDemo := info.ID == plugin.DemoSourceID
			demoInstalled = demoInstalled || isDemo
			if matches(info.ID, info.Name) {
				installed = append(installed, m.installedMarketRow(cx, source, isDemo))
			}
		}
		if len(installed) > 0 {
			sections = append(sections, marketSection(p, i18n.T("Installed Plugins"), installed...))
		}
		demoDetail := i18n.T("Try a demo login and play a sample track offline.")
		if !installedOnly.Get() && !demoInstalled && matches("Yoho Demo", plugin.DemoSourceID, demoDetail) {
			install := button(i18n.T("Install"), "market-install-demo", lucide.Download, func() { m.installMarketDemo(cx) })
			install.Disabled = importPackage.Disabled
			sections = append(sections, marketSection(p, i18n.T("Available to Install"),
				marketRow(p, lucide.Disc3, "Yoho Demo", demoDetail, install)))
		}
		if len(sections) == 0 {
			title := i18n.T("No installed plugins")
			detail := i18n.T("Browse all plugins or import a signed Yoho package.")
			if term != "" {
				title = i18n.T("No matching plugins")
				detail = i18n.T("Try another plugin name.")
			}
			sections = append(sections, core.View{Style: core.Style{Padding: core.Symmetric(30, 12), Gap: 6, AlignItems: core.AlignCenter},
				Children: []core.Node{label(title, 16, core.FontWeightBold, p.text), label(detail, 13, core.FontWeightNormal, p.muted)}})
		}
		return core.View{Style: core.Style{Padding: core.Edges{Top: 18, Left: 28, Right: 28, Bottom: 28}, Gap: 26}, Children: sections}
	})
	return core.View{TestID: "plugin-market", Style: fill, Children: []core.Node{
		m.pageHeader(cx, i18n.T("Plugin Market"), "", "", manage, importPackage),
		filters,
		core.Scroll{Style: fill, Children: []core.Node{list}},
	}}
}

func marketSection(p palette, title string, rows ...core.Node) core.Node {
	return core.View{Style: core.Style{Gap: 9}, Children: []core.Node{
		label(title, 16, core.FontWeightBold, p.text),
		core.View{Style: core.Style{Background: p.sidebar, CornerRadius: 12, Padding: core.Symmetric(2, 15)}, Children: rows},
	}}
}

func marketRow(p palette, icon glyph, name, detail string, actions ...core.Node) core.Node {
	title := label(name, 15, core.FontWeightBold, p.text)
	title.MaxLines = 1
	subtitle := label(detail, 13, core.FontWeightNormal, p.muted)
	subtitle.MaxLines = 2
	face := core.View{Style: core.Style{Width: core.Pt(44), Height: core.Pt(44), CornerRadius: 10, Background: p.background,
		AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}, Children: []core.Node{ui.Icon(icon, 21, func() core.Color { return p.accent })}}
	body := core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 4}, Children: []core.Node{title, subtitle}}
	view := row(append([]core.Node{face, body}, actions...)...)
	view.Style.Gap = 12
	view.Style.Padding = core.Symmetric(11, 0)
	return view
}

func (m *model) installedMarketRow(cx *core.Scope, source *plugin.NSGoSource, demo bool) core.Node {
	p := colors(cx)
	info := source.Info()
	state := m.accountOf(source)
	active := m.source.Get() == info.ID
	icon := lucide.Radio
	if demo {
		icon = lucide.Disc3
	}
	detail := i18n.T("Installed")
	if info.Version != "" {
		detail += " · " + info.Version
	}
	if state.account != nil {
		name := state.account.Name
		if name == "" {
			name = i18n.T("Signed In")
		}
		detail += " · " + name
	} else if state.busy {
		detail += " · " + i18n.T("Checking account…")
	} else if source.Can("signIn") {
		detail += " · " + i18n.T("Signed Out")
	}
	var actions []core.Node
	if source.Can("signIn") && state.account == nil {
		signIn := button(i18n.T("Sign In"), "market-sign-in-"+info.ID, lucide.LogIn, func() { m.changeAccount(source, true) })
		signIn.Variant = ui.ButtonSecondary
		signIn.Disabled = func() bool { return state.busy || m.nsgoPluginBusy.Get() }
		actions = append(actions, signIn)
	} else if source.Can("signOut") && (state.account != nil || !source.Can("account")) {
		signOut := button(i18n.T("Sign Out"), "market-sign-out-"+info.ID, nil, func() { m.changeAccount(source, false) })
		signOut.Variant = ui.ButtonGhost
		signOut.Disabled = func() bool { return state.busy || m.nsgoPluginBusy.Get() }
		actions = append(actions, signOut)
	}
	open := button(i18n.T("Use Source"), "market-open-"+info.ID, lucide.ArrowRight, func() { m.selectSource(info.ID) })
	if active {
		open.Text = i18n.T("Open Source")
		detail += " · " + i18n.T("In Use")
	}
	open.Variant = ui.ButtonOutline
	open.Disabled = m.nsgoPluginBusy.Get
	actions = append(actions, open)
	return marketRow(p, icon, info.Name, detail, actions...)
}

func (m *model) installMarketDemo(cx *core.Scope) {
	store := m.nsgoStore.GetUntracked()
	if store == nil || m.nsgoPluginBusy.GetUntracked() {
		return
	}
	go func() {
		p, err := store.BundledDemo()
		cx.Post(func() {
			if err != nil {
				m.report(err)
				return
			}
			if m.sourceByID(plugin.DemoSourceID) == nil && !m.nsgoPluginBusy.GetUntracked() {
				m.confirmPluginInstall(cx, store, p)
			}
		})
	}()
}
