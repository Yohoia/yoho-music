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

	importPackage := button(i18n.T("Import Package"), "market-import", lucide.PackagePlus, func() { m.pickNSGoPlugin(cx) })
	importPackage.Disabled = func() bool { return m.nsgoStore.Get() == nil || m.nsgoPluginBusy.Get() }
	manage := button(i18n.T("Manage"), "market-manage", nil, func() { m.navigate(route{page: "settings"}) })
	manage.Variant = ui.ButtonGhost

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

	filters := row(core.Dynamic(func(cx *core.Scope) core.Node {
		installed := len(m.plugins.Get())
		total := len(offered())
		tabs := row(
			marketFilter(cx, p, i18n.T("All"), "market-all", total, installedOnly, false),
			marketFilter(cx, p, i18n.T("Installed"), "market-installed", installed, installedOnly, true),
		)
		tabs.Style.Gap = 2
		tabs.Style.Padding = core.All(3)
		tabs.Style.Background = p.hover
		tabs.Style.CornerRadius = 10
		return tabs
	}), spacer(), marketSearch(p, searchRef, query))
	filters.Style.Padding = core.Edges{Left: 28, Right: 28, Bottom: 14}
	filters.Style.Borders.Bottom = 1
	filters.Style.BorderColor = p.line

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
		columns := wide.Get()
		if len(installed) > 0 {
			sections = append(sections, marketSection(p, i18n.T("Installed Plugins"), columns, installed...))
		}
		if len(available) > 0 {
			sections = append(sections, marketSection(p, i18n.T("Available to Install"), columns, available...))
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
		return core.View{Style: core.Style{Padding: core.Edges{Top: 22, Left: 28, Right: 28, Bottom: 28}, Gap: 24}, Children: sections}
	})
	return core.View{TestID: "plugin-market", Style: fill.Merge(core.Style{Padding: core.Edges{Top: 36}}),
		OnLayout: func(bounds core.Bounds) {
			columns := bounds.Size.Width >= 960
			if columns != wide.GetUntracked() {
				wide.Set(columns)
			}
		}, Children: []core.Node{
			m.pageHeader(cx, i18n.T("Plugin Market"), "", "", manage, importPackage),
			filters,
			core.Scroll{Style: fill.Merge(core.Style{Background: p.sidebar}), Children: []core.Node{list}},
			m.marketDetail(cx, p, visibleEntry, closeDetail),
		}}
}

func marketFilter(cx *core.Scope, p palette, text, id string, count int, installedOnly *core.Signal[bool], installed bool) core.Node {
	selected := func() bool { return installedOnly.Get() == installed }
	number := label(strconv.Itoa(count), 12, core.FontWeightMedium, p.muted)
	number.Style.FontVariant = core.FontTabularNums
	return base.Button{Label: text + " " + strconv.Itoa(count), TestID: id, Selected: selected,
		OnPress: func(core.PressEvent) { installedOnly.Set(installed) },
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, Height: core.Pt(30),
			Padding: core.Symmetric(0, 10), CornerRadius: 7, BorderWidth: 1, BorderColor: core.Transparent,
			Hover: &core.Style{Background: p.hover}, Active: &core.Style{Background: p.selected},
			Selected: &core.Style{Background: p.background, BorderColor: p.line}, FocusVisible: ui.FocusRing(cx, 2)},
		Children: []core.Node{label(text, 13, core.FontWeightMedium, p.text), number}}
}

func marketSearch(p palette, ref *core.Ref, query *core.Signal[string]) core.Node {
	name := i18n.T("Search Plugins")
	search := ui.Input{Ref: ref, Value: query.Get, OnChange: query.Set, Placeholder: name, Label: name, TestID: "market-search",
		Style: core.Style{Height: core.Pt(36), CornerRadius: 9, Padding: core.Edges{Left: 32, Right: 10},
			PlaceholderColor: p.muted}}
	icon := core.View{Style: core.Style{Position: core.Absolute, Top: core.Pt(11), Left: core.Pt(11), PointerEvents: core.PointerEventsNone},
		Children: []core.Node{ui.Icon(lucide.Search, 14, func() core.Color { return p.muted })}}
	return core.View{Style: core.Style{Width: core.Pt(248), FlexShrink: 1}, Children: []core.Node{search, icon}}
}

func marketSection(p palette, title string, wide bool, cards ...core.Node) core.Node {
	count := label(strconv.Itoa(len(cards)), 12, core.FontWeightMedium, p.muted)
	count.Style.FontVariant = core.FontTabularNums
	columns := core.GridTracks("minmax(0, 1fr)")
	if wide {
		columns = "repeat(2, minmax(0, 1fr))"
	}
	return core.View{Style: core.Style{Gap: 10}, Children: []core.Node{
		row(label(title, 15, core.FontWeightBold, p.text), count),
		core.View{Style: core.Style{Display: core.DisplayGrid, GridTemplateColumns: columns, Gap: 12}, Children: cards},
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

func (m *model) marketCard(cx *core.Scope, p palette, entry marketPlugin, open func(string)) core.Node {
	icon, origin := entry.identity()
	title := label(entry.Name, 16, core.FontWeightBold, p.text)
	title.MaxLines = 1
	subtitle := label(origin, 12, core.FontWeightMedium, p.muted)
	subtitle.MaxLines = 1
	head := row(marketArtwork(p, icon, 44),
		core.View{Style: fill.Merge(core.Style{Gap: 3}), Children: []core.Node{title, subtitle}},
		ui.Icon(lucide.ChevronRight, 17, func() core.Color { return p.accent }))
	head.Style.Gap = 11

	description := entry.description()
	if description == "" {
		description = entry.packageID()
	}
	summary := label(description, 13, core.FontWeightNormal, p.muted)
	summary.MaxLines = 2
	summary.Style.LineHeight = 18
	body := base.Button{TestID: "market-card-" + entry.ID, Label: fmt.Sprintf(i18n.T("View details for %s"), entry.Name),
		OnPress: func(core.PressEvent) { open(entry.ID) },
		Style: core.Style{Padding: core.Edges{Top: 14, Left: 14, Right: 14, Bottom: 12}, Gap: 10,
			MinHeight: core.Pt(100), FlexGrow: 1,
			Hover: &core.Style{Background: p.hover}, Active: &core.Style{Background: p.selected},
			FocusVisible: &core.Style{OutlineColor: p.accent, OutlineWidth: 2, OutlineOffset: -2, CornerRadius: 13}},
		Children: []core.Node{head, summary}}

	footer := core.Dynamic(func(*core.Scope) core.Node {
		state := pluginAccount{}
		if entry.source != nil {
			state = m.accountOf(entry.source)
		}
		version := label(entry.Version, 12, core.FontWeightMedium, p.muted)
		version.MaxLines = 1
		version.Style.FontVariant = core.FontTabularNums
		info := row(version)
		info.Style.FlexShrink = 1
		if entry.source != nil {
			status, statusIcon, tint := m.marketInstallation(cx, p, entry)
			switch {
			case state.busy:
				status, statusIcon, tint = i18n.T("Checking account…"), lucide.LoaderCircle, p.muted
			case state.account != nil:
				status, statusIcon, tint = marketAccountName(state.account), lucide.Check, marketSuccess(cx)
			case entry.source.Can("signIn") && m.source.Get() != entry.ID:
				status, statusIcon = i18n.T("Signed Out"), lucide.User
			}
			info.Style = core.Style{FlexShrink: 1, Gap: 3}
			info.Children = []core.Node{marketStatusLabel(status, statusIcon, tint), version}
		}
		bar := row(info, spacer(), m.marketAccountAction(entry, state, "market-"), m.marketPrimaryAction(entry, "market-", ui.ButtonOutline))
		bar.Style.Padding = core.Edges{Top: 8, Bottom: 8, Left: 14, Right: 14}
		bar.Style.MinHeight = core.Pt(48)
		bar.Style.Borders.Top = 1
		bar.Style.BorderColor = p.line
		return bar
	})
	return core.View{Style: core.Style{Background: p.background, BorderWidth: 1, BorderColor: p.line,
		CornerRadius: 14, Overflow: core.Hidden}, Children: []core.Node{body, footer}}
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
