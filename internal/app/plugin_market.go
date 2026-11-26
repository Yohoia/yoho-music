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

	filters := core.Dynamic(func(cx *core.Scope) core.Node {
		installed := len(m.plugins.Get())
		total := len(offered())
		tabs := row(
			marketFilter(cx, p, i18n.T("All"), "market-all", total, installedOnly, false),
			marketFilter(cx, p, i18n.T("Installed"), "market-installed", installed, installedOnly, true),
		)
		tabs.Style.Gap = 4
		return tabs
	})

	pageTitle := label(i18n.T("Plugin Market"), 22, core.FontWeightSemibold, p.text)

	rightHeader := row(marketSearch(p, searchRef, query), importBtn)
	rightHeader.Style.AlignItems = core.AlignCenter
	rightHeader.Style.Gap = 8

	toolbarRow := row(filters, spacer(), rightHeader)
	toolbarRow.Style.AlignItems = core.AlignCenter
	toolbarRow.Style.Gap = 12

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
			sections = append(sections, marketSection(p, "", installed...))
		}
		if len(available) > 0 {
			if len(sections) > 0 {
				sections = append(sections, dashedDivider(p.line))
			}
			sections = append(sections, marketSection(p, i18n.T("Available to Install"), available...))
		}
		if state := catalog.Get(); !onlyInstalled && state.loading {
			if len(sections) > 0 {
				sections = append(sections, dashedDivider(p.line))
			}
			sections = append(sections, marketLoadingSkeleton(cx, p))
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

	pageBody := paperWrap(core.View{
		Style: core.Style{Padding: core.Edges{Top: 62, Bottom: 108}, Gap: 16},
		Children: []core.Node{
			pageTitle,
			toolbarRow,
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

func marketSection(p palette, title string, cards ...core.Node) core.Node {
	grid := core.View{
		Style: core.Style{
			Display:             core.DisplayGrid,
			GridTemplateColumns: core.GridTracks("repeat(2, minmax(0, 1fr))"),
			Gap:                 16,
		},
		Children: cards,
	}
	content := []core.Node{grid}
	if title != "" {
		count := label(strconv.Itoa(len(cards)), 12, core.FontWeightMedium, p.muted)
		count.Style.FontVariant = core.FontTabularNums
		content = append([]core.Node{row(label(title, 15, core.FontWeightSemibold, p.text), count)}, content...)
	}
	return core.View{Style: core.Style{Gap: 12}, Children: content}
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
	_, origin := entry.identity()
	cardBg := core.RGB(0xffffff)
	trayBg := core.RGB(0xf2f2f5)
	descColor := core.RGB(0x3f3f46)
	kindColor := core.RGB(0x4f46e5)
	borderHover := core.RGB(0xc7c7d1)
	if cx.Dark() {
		cardBg = core.RGB(0x1c1c20)
		trayBg = core.RGB(0x26262c)
		descColor = core.RGB(0xc4c4cc)
		kindColor = core.RGB(0x818cf8)
		borderHover = core.RGB(0x4a4a55)
	}
	green := marketSuccess(cx)

	iconTray := core.View{
		Style: core.Style{
			Width: core.Pt(46), Height: core.Pt(46), CornerRadius: 12, FlexShrink: 0,
			Background: trayBg, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
		},
		Children: []core.Node{marketPluginArtwork(p, entry, 28)},
	}

	title := label(entry.Name, 16, core.FontWeightBold, p.text)
	title.MaxLines = 1
	verified := row(
		ui.Icon(lucide.ShieldCheck, 13, func() core.Color { return green }),
		label(origin, 12, core.FontWeightMedium, green),
	)
	verified.Style.Gap = 4

	topRight := core.Dynamic(func(*core.Scope) core.Node {
		if entry.source == nil || entry.update {
			install := m.marketPrimaryAction(entry, "market-", ui.ButtonPrimary)
			install.Style.Height = core.Pt(28)
			install.Style.CornerRadius = 14
			return install
		}
		state := m.accountOf(entry.source)
		switch {
		case state.busy:
			return label(i18n.T("Checking account…"), 12, core.FontWeightNormal, p.muted)
		case state.account != nil:
			status := row(ui.Icon(lucide.Check, 14, func() core.Color { return green }), label(i18n.T("Signed In"), 12.5, core.FontWeightMedium, green))
			status.Style.Gap = 4
			return status
		case entry.source.Can("signIn"):
			return label(i18n.T("Signed Out"), 12, core.FontWeightNormal, p.muted)
		default:
			status := row(ui.Icon(lucide.Check, 13, func() core.Color { return p.muted }), label(i18n.T("Installed"), 12, core.FontWeightMedium, p.muted))
			status.Style.Gap = 4
			return status
		}
	})

	cardHovered := core.NewSignal(cx, false)
	softRed := core.RGB(0xe56b6f)
	softRedHover := core.RGBA(0xef4444, .10)
	if cx.Dark() {
		softRed = core.RGB(0xf28b8d)
		softRedHover = core.RGBA(0xf87171, .14)
	}

	headerRow := row(
		iconTray,
		core.View{
			Style: core.Style{
				Height: core.Pt(46), FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0),
				Padding: core.Symmetric(2, 0),
			},
			Children: []core.Node{title, spacer(), verified},
		},
		topRight,
	)
	headerRow.Style.Gap = 12

	description := entry.description()
	if description == "" {
		description = entry.packageID()
	}
	desc := label(description, 13, core.FontWeightNormal, descColor)
	desc.MaxLines = 2
	desc.Style.LineHeight = 19

	upperClick := base.Button{
		TestID:   "market-card-" + entry.ID,
		Label:    fmt.Sprintf(i18n.T("View details for %s"), entry.Name),
		OnPress:  func(core.PressEvent) { open(entry.ID) },
		Style:    core.Style{Gap: 12, FlexGrow: 1},
		Children: []core.Node{headerRow, desc},
	}

	footer := core.Dynamic(func(*core.Scope) core.Node {
		var metaNodes []core.Node
		kind := row(ui.Icon(lucide.Plug, 12.5, func() core.Color { return kindColor }), label(i18n.T("Sources"), 12, core.FontWeightMedium, kindColor))
		kind.Style.Gap = 4
		metaNodes = append(metaNodes, kind)

		if entry.source != nil {
			state := m.accountOf(entry.source)
			if state.account != nil {
				acct := row(ui.Icon(lucide.User, 12.5, func() core.Color { return p.muted }), label(marketAccountName(state.account), 12, core.FontWeightNormal, p.muted))
				acct.Style.Gap = 4
				metaNodes = append(metaNodes, acct)
			}
		}

		if entry.Version != "" {
			ver := label("v"+entry.Version, 12, core.FontWeightNormal, p.muted)
			ver.Style.FontFamily = core.FontMonospace
			ver.Style.FontVariant = core.FontTabularNums
			metaNodes = append(metaNodes, ver)
		}

		metadata := core.View{
			Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, AlignItems: core.AlignCenter, Gap: 10, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
			Children: metaNodes,
		}

		var remove core.Node
		if entry.source != nil {
			removeBtn := base.Button{
				Label:    i18n.T("Remove Plugin"),
				TestID:   "market-remove-" + entry.ID,
				Disabled: func() bool { return m.nsgoPluginBusy.Get() || !cardHovered.Get() },
				OnPress: func(core.PressEvent) {
					confirm(cx, i18n.T("Remove Plugin?"), entry.Name, func() { m.removeNSGoPlugin(entry.source) })
				},
				Style: core.Style{
					Width: core.Pt(30), Height: core.Pt(30), CornerRadius: 15, FlexShrink: 0,
					AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
					Hover: &core.Style{Background: softRedHover},
				},
				StyleFunc: func() core.Style {
					if cardHovered.Get() {
						return core.Style{Opacity: core.Alpha(1)}
					}
					return core.Style{Opacity: core.Alpha(0), PointerEvents: core.PointerEventsNone}
				},
				Children: []core.Node{ui.Icon(lucide.Trash, 15, func() core.Color { return softRed })},
			}
			remove = ui.Tooltip{Text: i18n.T("Remove Plugin"), Trigger: removeBtn}
		}

		bar := row(metadata, remove)
		bar.Style.Gap = 8
		return bar
	})

	return core.View{
		TestID:       "market-tile-" + entry.ID,
		OnMouseEnter: func(core.MouseEvent) { cardHovered.Set(true) },
		OnMouseLeave: func(core.MouseEvent) { cardHovered.Set(false) },
		Style: core.Style{
			Background: cardBg, BorderWidth: 1, BorderColor: p.line, CornerRadius: 16,
			Padding: core.All(18), Gap: 14,
			Hover: &core.Style{BorderColor: borderHover},
		},
		Children: []core.Node{upperClick, footer},
	}
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
