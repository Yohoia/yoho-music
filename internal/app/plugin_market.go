package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

type marketPlugin struct {
	music.Info
	source  *plugin.NSGoSource
	listing plugin.MarketPlugin
	update  bool
}

func (entry marketPlugin) packageID() string {
	return strings.TrimPrefix(entry.ID, plugin.NSGoSourcePrefix)
}

func (entry marketPlugin) identity() (glyph, string) {
	switch entry.ID {
	case plugin.DemoSourceID:
		return lucide.Disc3, i18n.T("Plugin Market")
	case plugin.NetEaseSourceID:
		return lucide.Music2, i18n.T("Plugin Market")
	default:
		if entry.listing.ID != "" {
			return lucide.AudioLines, i18n.T("Plugin Market")
		}
		return lucide.AudioLines, i18n.T("Imported NSGo Package")
	}
}

func (entry marketPlugin) description() string {
	switch entry.ID {
	case plugin.DemoSourceID:
		return i18n.T("Try a demo login and play a sample track offline.")
	case plugin.NetEaseSourceID:
		return i18n.T("Scan to sign in to NetEase Cloud Music and listen to your library in Yoho.")
	default:
		return entry.listing.Description
	}
}

type marketCatalog struct {
	entries []plugin.MarketPlugin
	loading bool
	failed  bool
}

func (m *model) pluginMarket(cx *core.Scope, searchRef *core.Ref) core.Node {
	query := core.NewSignal(cx, "")
	installedOnly := core.NewSignal(cx, false)
	selectedID := core.NewSignal(cx, "")
	wide := core.NewSignal(cx, false)
	p := colors(cx)
	catalog := core.NewSignalFunc(cx, marketCatalog{}, nil)
	ctx := cx.Context()
	refresh := func() {
		catalog.Set(marketCatalog{loading: true})
		go func() {
			entries, err := plugin.FetchMarket(ctx, m.marketURL)
			if ctx.Err() == nil {
				cx.Post(func() { catalog.Set(marketCatalog{entries: entries, failed: err != nil}) })
			}
		}()
	}
	if m.marketURL != "" {
		cx.Post(refresh)
	}
	offered := func() []marketPlugin {
		sources := m.plugins.Get()
		listings := catalog.Get().entries
		byID := make(map[string]plugin.MarketPlugin, len(listings))
		for _, listing := range listings {
			byID[listing.SourceID()] = listing
		}
		entries := make([]marketPlugin, 0, len(sources)+len(listings))
		for _, source := range sources {
			info := source.Info()
			listing := byID[info.ID]
			entries = append(entries, marketPlugin{
				Info: info, source: source, listing: listing, update: source.Build() < listing.Build,
			})
			delete(byID, info.ID)
		}
		for _, listing := range listings {
			id := listing.SourceID()
			if byID[id].ID != "" {
				entries = append(entries, marketPlugin{
					Info: music.Info{ID: id, Name: listing.Name, Version: listing.Version}, listing: listing,
				})
			}
		}
		return entries
	}

	importBtn := base.Button{
		Label:    i18n.T("Import Package"),
		TestID:   "market-import",
		Disabled: func() bool { return m.nsgoStore.Get() == nil || m.nsgoPluginBusy.Get() },
		OnPress:  func(core.PressEvent) { m.pickNSGoPlugin(cx) },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(32), Padding: core.Symmetric(0, 16), CornerRadius: 16,
			Background: p.button, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
			Hover: &core.Style{Opacity: core.Alpha(.88)},
		},
		Children: []core.Node{
			ui.Icon(lucide.PackagePlus, 13, func() core.Color { return p.onButton }),
			label(i18n.T("Import Package"), 12, core.FontWeightSemibold, p.onButton),
		},
	}
	manageBtn := base.Button{
		Label:   i18n.T("Manage"),
		TestID:  "market-manage",
		OnPress: func(core.PressEvent) { m.navigate(route{page: "settings"}) },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(32), Padding: core.Symmetric(0, 14), CornerRadius: 16,
			Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Hover: &core.Style{Background: p.hover},
		},
		Children: []core.Node{label(i18n.T("Manage"), 12, core.FontWeightMedium, p.text)},
	}

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
	visibleEntry := func() marketPlugin {
		id := selectedID.Get()
		if id == "" {
			return marketPlugin{}
		}
		for _, entry := range offered() {
			if id == entry.ID {
				return entry
			}
		}
		return marketPlugin{}
	}
	closeDetail := func() { selectedID.Set("") }

	headerLeft := core.View{
		Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 4},
		Children: []core.Node{
			label(i18n.T("Plugin Market"), 22, core.FontWeightSemibold, p.text),
			label(i18n.T("Browse all plugins or import a signed Yoho package."), 12, core.FontWeightNormal, p.muted),
		},
	}
	headerRow := row(headerLeft, manageBtn, importBtn)
	headerRow.Style.AlignItems = core.AlignEnd
	headerRow.Style.Gap = 8

	filters := row(core.Dynamic(func(cx *core.Scope) core.Node {
		installed := len(m.plugins.Get())
		total := len(offered())
		tabs := row(
			marketFilter(cx, p, i18n.T("All"), "market-all", total, installedOnly, false),
			marketFilter(cx, p, i18n.T("Installed"), "market-installed", installed, installedOnly, true),
		)
		tabs.Style.Gap = 4
		return tabs
	}), spacer(), marketSearch(p, searchRef, query))

	list := core.Dynamic(func(cx *core.Scope) core.Node {
		term := strings.ToLower(strings.TrimSpace(query.Get()))
		matches := func(entry marketPlugin) bool {
			_, origin := entry.identity()
			return strings.Contains(strings.ToLower(strings.Join([]string{entry.Name, entry.ID, entry.description(), origin}, " ")), term)
		}
		open := func(id string) { selectedID.Set(id) }
		var sections, installed, available []core.Node
		onlyInstalled := installedOnly.Get()
		for _, entry := range offered() {
			if !matches(entry) {
				continue
			}
			if entry.source != nil {
				installed = append(installed, m.marketCard(cx, p, entry, open))
			} else if !onlyInstalled {
				available = append(available, m.marketCard(cx, p, entry, open))
			}
		}
		if len(installed) > 0 {
			sections = append(sections, marketSection(p, i18n.T("Installed Plugins"), installed...))
		}
		if len(available) > 0 {
			if len(sections) > 0 {
				sections = append(sections, dashedDivider(p.line))
			}
			sections = append(sections, marketSection(p, i18n.T("Available to Install"), available...))
		}
		if state := catalog.Get(); !onlyInstalled && state.loading {
			sections = append(sections, label(i18n.T("Loading plugin market…"), 13, core.FontWeightNormal, p.muted))
		} else if !onlyInstalled && state.failed {
			retry := button(i18n.T("Retry"), "market-retry", lucide.RefreshCw, refresh)
			retry.Variant = ui.ButtonOutline
			sections = append(sections, row(label(i18n.T("Couldn't load the plugin market."), 13, core.FontWeightNormal, p.muted), spacer(), retry))
		}
		if len(sections) == 0 {
			sections = append(sections, marketEmpty(p, term != "", func() {
				query.Set("")
				searchRef.Focus()
			}, func() { installedOnly.Set(false) }))
		}
		return core.View{Style: core.Style{Gap: 20}, Children: sections}
	})

	_ = wide
	pageBody := paperWrap(core.View{
		Style: core.Style{Padding: core.Edges{Top: 62, Bottom: 108}, Gap: 18},
		Children: []core.Node{
			headerRow,
			filters,
			dashedDivider(p.line),
			list,
		},
	})
	return core.View{
		TestID: "plugin-market",
		Style:  fill.Merge(core.Style{Background: p.background}),
		Children: []core.Node{
			core.Scroll{Style: fill, Children: []core.Node{pageBody}},
			m.marketDetail(cx, p, visibleEntry, closeDetail),
		},
	}
}

func marketFilter(cx *core.Scope, p palette, text, id string, count int, installedOnly *core.Signal[bool], installed bool) core.Node {
	selected := func() bool { return installedOnly.Get() == installed }
	title := core.Text{
		Content: text + " " + strconv.Itoa(count),
		Style:   core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, FontVariant: core.FontTabularNums, Color: p.muted},
		StyleFunc: func() core.Style {
			if selected() {
				return core.Style{Color: p.onButton}
			}
			return core.Style{}
		},
	}
	return base.Button{
		Label: text + " " + strconv.Itoa(count), TestID: id, Selected: selected,
		OnPress: func(core.PressEvent) { installedOnly.Set(installed) },
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Height: core.Pt(30),
			Padding: core.Symmetric(0, 14), CornerRadius: 15,
		},
		StyleFunc: func() core.Style {
			return p.selectableButtonStyle(selected())
		},
		Children: []core.Node{title},
	}
}

func marketSearch(p palette, ref *core.Ref, query *core.Signal[string]) core.Node {
	name := i18n.T("Search Plugins")
	search := base.Input{
		Ref: ref, Value: query.Get, OnChange: query.Set, Placeholder: name, Label: name, TestID: "market-search",
		Style: core.Style{
			FlexGrow: 1, FlexShrink: 1, Height: core.Pt(28),
			Padding: core.All(0), Background: core.Transparent,
			FontSize: 12, Color: p.text, PlaceholderColor: p.muted, CaretColor: p.text,
		},
	}
	box := row(ui.Icon(lucide.Search, 13, func() core.Color { return p.muted }), search)
	box.Style.Width = core.Pt(224)
	box.Style.Height = core.Pt(32)
	box.Style.CornerRadius = 16
	box.Style.Background = p.sidebar
	box.Style.BorderWidth = 1
	box.Style.BorderColor = p.line
	box.Style.Padding = core.Symmetric(0, 12)
	box.Style.Gap = 8
	return box
}

func marketSection(p palette, title string, rows ...core.Node) core.Node {
	count := label(strconv.Itoa(len(rows)), 12, core.FontWeightMedium, p.muted)
	count.Style.FontVariant = core.FontTabularNums
	var children []core.Node
	for i, r := range rows {
		if i > 0 {
			children = append(children, core.View{Style: core.Style{Height: core.Pt(1), Background: p.line}})
		}
		children = append(children, r)
	}
	return core.View{Style: core.Style{Gap: 10}, Children: []core.Node{
		row(label(title, 15, core.FontWeightSemibold, p.text), count),
		core.View{Children: children},
	}}
}

func marketEmpty(p palette, searching bool, clearSearch, showAll func()) core.Node {
	icon, title := lucide.Plug, i18n.T("No installed plugins")
	detail := i18n.T("Browse all plugins or import a signed Yoho package.")
	action := button(i18n.T("Browse All"), "market-browse-all", lucide.ArrowRight, showAll)
	if searching {
		icon, title = lucide.Search, i18n.T("No matching plugins")
		detail = i18n.T("Try another plugin name.")
		action = button(i18n.T("Clear Search"), "market-clear-search", nil, clearSearch)
	}
	action.Variant = ui.ButtonOutline
	return core.View{Style: core.Style{Padding: core.Symmetric(48, 16), Gap: 10, AlignItems: core.AlignCenter},
		Children: []core.Node{marketArtwork(p, icon, 48),
			label(title, 17, core.FontWeightBold, p.text), label(detail, 13, core.FontWeightNormal, p.muted), action}}
}

func marketArtwork(p palette, icon glyph, size core.Points) core.Node {
	return core.View{Style: core.Style{Width: core.Pt(size), Height: core.Pt(size), CornerRadius: 12,
		Background: ui.Mix(p.background, p.accent, 0.10), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
		Children: []core.Node{ui.Icon(icon, size*.47, func() core.Color { return p.accent })}}
}

func marketGradientFor(id string) core.Gradient {
	switch id {
	case plugin.NetEaseSourceID:
		return core.LinearGradient(135, core.RGB(0xc2415c), core.RGB(0x7f1d32))
	case plugin.DemoSourceID:
		return core.LinearGradient(135, core.RGB(0x6366f1), core.RGB(0x312e81))
	default:
		return core.LinearGradient(135, core.RGB(0x475569), core.RGB(0x1e293b))
	}
}

func (m *model) marketCard(cx *core.Scope, p palette, entry marketPlugin, open func(string)) core.Node {
	icon, _ := entry.identity()
	iconBadge := core.View{
		Style: core.Style{
			Width: core.Pt(44), Height: core.Pt(44), CornerRadius: 12, FlexShrink: 0,
			Gradients:  []core.Gradient{marketGradientFor(entry.ID)},
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
		},
		Children: []core.Node{ui.Icon(icon, 20, func() core.Color { return core.RGB(0xffffff) })},
	}

	description := entry.description()
	if description == "" {
		description = entry.packageID()
	}

	infoArea := core.Dynamic(func(*core.Scope) core.Node {
		state := pluginAccount{}
		if entry.source != nil {
			state = m.accountOf(entry.source)
		}
		name := label(entry.Name, 14, core.FontWeightSemibold, p.text)
		name.MaxLines = 1
		topItems := []core.Node{name}
		if entry.Version != "" {
			ver := label("v"+entry.Version, 11, core.FontWeightNormal, p.muted)
			ver.Style.FontFamily = core.FontMonospace
			ver.Style.FontVariant = core.FontTabularNums
			verChip := core.View{
				Style:    core.Style{Background: p.chip, Padding: core.Symmetric(2, 6), CornerRadius: 4, FlexShrink: 0},
				Children: []core.Node{ver},
			}
			topItems = append(topItems, verChip)
		}
		if entry.source != nil {
			var statusText string
			statusColor := p.muted
			switch {
			case state.busy:
				statusText = i18n.T("Checking account…")
			case state.account != nil:
				statusText = marketAccountName(state.account)
				statusColor = marketSuccess(cx)
			case entry.source.Can("signIn"):
				statusText = i18n.T("Signed Out")
			}
			if statusText != "" {
				st := label("· "+statusText, 11, core.FontWeightNormal, statusColor)
				st.MaxLines = 1
				topItems = append(topItems, st)
			}
		}
		topLine := row(topItems...)
		topLine.Style.Gap = 6

		desc := label(description, 12, core.FontWeightNormal, p.muted)
		desc.MaxLines = 1

		return base.Button{
			TestID:  "market-card-" + entry.ID,
			Label:   fmt.Sprintf(i18n.T("View details for %s"), entry.Name),
			OnPress: func(core.PressEvent) { open(entry.ID) },
			Style: core.Style{
				Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14,
				FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0),
				Padding: core.Symmetric(12, 8), CornerRadius: 12,
				Hover: &core.Style{Background: p.hover},
			},
			Children: []core.Node{
				iconBadge,
				core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 4}, Children: []core.Node{topLine, desc}},
			},
		}
	})

	actions := core.Dynamic(func(*core.Scope) core.Node {
		state := pluginAccount{}
		if entry.source != nil {
			state = m.accountOf(entry.source)
		}
		variant := ui.ButtonOutline
		if entry.source == nil || entry.update {
			variant = ui.ButtonPrimary
		}
		primary := m.marketPrimaryAction(entry, "market-", variant)
		primary.Style.Height = core.Pt(30)
		primary.Style.CornerRadius = 15
		box := row(m.marketAccountAction(entry, state, "market-"), primary)
		box.Style.Gap = 6
		box.Style.FlexShrink = 0
		return box
	})

	entryRow := row(infoArea, actions)
	entryRow.Style.Gap = 10
	entryRow.Style.Padding = core.Symmetric(2, 0)
	return entryRow
}

func marketSuccess(cx *core.Scope) core.Color {
	if cx.Dark() {
		return core.RGB(0x76d4a2)
	}
	return core.RGB(0x17734a)
}

func (m *model) marketInstallation(cx *core.Scope, p palette, entry marketPlugin) (string, glyph, core.Color) {
	switch {
	case entry.source == nil:
		return i18n.T("Not Installed"), lucide.Download, p.muted
	case m.source.Get() == entry.ID:
		return i18n.T("In Use"), lucide.AudioLines, marketSuccess(cx)
	default:
		return i18n.T("Installed"), lucide.Check, p.muted
	}
}

func marketAccountName(account *music.Account) string {
	if account.Name != "" {
		return account.Name
	}
	return i18n.T("Signed In")
}

func marketStatusLabel(text string, icon glyph, tint core.Color) core.View {
	status := label(text, 12, core.FontWeightMedium, tint)
	status.MaxLines = 1
	view := row(ui.Icon(icon, 13, func() core.Color { return tint }), status)
	view.Style.Gap = 4
	view.Style.FlexShrink = 1
	return view
}

func (m *model) marketPrimaryAction(entry marketPlugin, prefix string, variant ui.ButtonVariant) ui.Button {
	var action ui.Button
	if entry.source == nil || entry.update {
		id := entry.ID
		if id == plugin.DemoSourceID {
			id = "demo"
		}
		title, verb := i18n.T("Install"), "install"
		if entry.update {
			title, verb = i18n.T("Update"), "update"
		}
		action = button(title, prefix+verb+"-"+id, lucide.Download, func() { m.installMarketPlugin(entry.listing) })
		action.Disabled = func() bool { return m.nsgoStore.Get() == nil || m.nsgoPluginBusy.Get() }
	} else {
		name := i18n.T("Use Source")
		if m.source.Get() == entry.ID {
			name = i18n.T("Open Source")
		}
		action = button(name, prefix+"open-"+entry.ID, lucide.ArrowRight, func() { m.selectSource(entry.ID) })
		action.Disabled = m.nsgoPluginBusy.Get
	}
	action.Variant = variant
	action.Style.CornerRadius = 14
	return action
}

func (m *model) marketAccountAction(entry marketPlugin, state pluginAccount, prefix string) core.Node {
	source := entry.source
	if source == nil {
		return nil
	}
	var action ui.Button
	if source.Can("signIn") && state.account == nil {
		action = button(i18n.T("Sign In"), prefix+"sign-in-"+entry.ID, nil, func() { m.changeAccount(source, true) })
	} else if source.Can("signOut") && (state.account != nil || !source.Can("account")) {
		action = button(i18n.T("Sign Out"), prefix+"sign-out-"+entry.ID, nil, func() { m.changeAccount(source, false) })
	} else {
		return nil
	}
	action.Variant = ui.ButtonGhost
	action.Disabled = func() bool { return state.busy || m.nsgoPluginBusy.Get() }
	return action
}

func (m *model) installMarketPlugin(entry plugin.MarketPlugin) {
	store := m.nsgoStore.GetUntracked()
	if store == nil || m.nsgoPluginBusy.GetUntracked() {
		return
	}
	m.nsgoPluginBusy.Set(true)
	go func() {
		p, err := store.DownloadMarketPlugin(m.context, entry)
		m.cx.Post(func() {
			m.nsgoPluginBusy.Set(false)
			if err != nil {
				m.report(err)
				return
			}
			current, _ := m.sourceByID(entry.SourceID()).(*plugin.NSGoSource)
			if current == nil || current.Build() < p.App().Build {
				m.confirmPluginInstall(m.cx, store, p)
			}
		})
	}()
}
