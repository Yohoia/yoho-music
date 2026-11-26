package app

import (
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/plugin"
)

func (m *model) marketDetail(cx *core.Scope, p palette, current func() marketPlugin, close func()) core.Node {
	smooth := core.Curve{X1: 0.22, Y1: 1, X2: 0.36, Y2: 1}
	popup := core.Style{
		Width: core.Pct(92), MaxWidth: core.Pt(660), MaxHeight: core.Pct(90), FlexShrink: 1,
		Background: p.background, BorderWidth: 1, BorderColor: p.line, CornerRadius: 20, Overflow: core.Hidden,
		Transition: core.Transition{Duration: 220 * time.Millisecond, Easing: smooth},
	}
	if !cx.Dark() {
		popup.ShadowColor, popup.ShadowOffset, popup.ShadowRadius = core.RGBA(0x000000, .24), core.Point{Y: 14}, 28
	}
	return base.Dialog{
		Open: func() bool { return current().ID != "" },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				close()
			}
		},
		Animation:             core.ModalNone,
		LabelFunc:             func() string { return current().Name },
		TestID:                "market-detail",
		BackdropTestID:        "market-detail-backdrop",
		BackdropStyle:         core.Style{Background: core.RGBA(0x111015, .36), BackdropFilters: []core.Filter{core.Blur(8)}, Transition: core.Transition{Duration: 200 * time.Millisecond, Easing: core.EaseOut}},
		BackdropStartingStyle: core.Style{Opacity: core.Alpha(0)},
		BackdropEndingStyle:   core.Style{Opacity: core.Alpha(0)},
		PopupStyle:            popup,
		StartingStyle:         core.Style{Opacity: core.Alpha(0), Transform: core.Transform{Scale: 0.94}},
		EndingStyle:           core.Style{Opacity: core.Alpha(0), Transform: core.Transform{Scale: 0.96}},
		Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			entry := current()
			if entry.ID == "" {
				return nil
			}
			return m.marketDetailContent(cx, p, entry, close)
		})},
	}
}

func (m *model) marketDetailContent(cx *core.Scope, p palette, entry marketPlugin, close func()) core.Node {
	_, origin := entry.identity()
	green := marketSuccess(cx)

	// Left Identity & Action Rail (224pt)
	artwork := marketPluginArtwork(p, entry, 56)
	title := label(entry.Name, 17, core.FontWeightBold, p.text)
	title.MaxLines = 2
	verified := row(
		ui.Icon(lucide.ShieldCheck, 12.5, func() core.Color { return green }),
		label(origin, 11.5, core.FontWeightMedium, green),
	)
	verified.Style.Gap = 4

	facts := core.Dynamic(func(*core.Scope) core.Node {
		state := pluginAccount{}
		if entry.source != nil {
			state = m.accountOf(entry.source)
		}
		status, _, tint := m.marketInstallation(cx, p, entry)
		version := entry.Version
		if version == "" {
			version = i18n.T("Unknown")
		} else {
			version = "v" + version
		}
		verText := label(version, 12, core.FontWeightNormal, p.text)
		verText.Style.FontFamily = core.FontMonospace
		verText.Style.FontVariant = core.FontTabularNums

		factRow := func(k string, val core.Node) core.Node {
			r := row(label(k, 12, core.FontWeightNormal, p.muted), spacer(), val)
			r.Style.Gap = 8
			return r
		}
		rows := []core.Node{
			factRow(i18n.T("Version"), verText),
			factRow(i18n.T("Status"), label(status, 12, core.FontWeightMedium, tint)),
		}
		if entry.source != nil && (entry.source.Can("account") || entry.source.Can("signIn") || state.account != nil) {
			account := i18n.T("Signed Out")
			acctColor := p.muted
			switch {
			case state.busy:
				account = i18n.T("Checking account…")
			case state.account != nil:
				account = marketAccountName(state.account)
				acctColor = p.text
			}
			acctText := label(account, 12, core.FontWeightMedium, acctColor)
			acctText.MaxLines = 1
			rows = append(rows, factRow(i18n.T("Account"), acctText))
		}
		return core.View{Style: core.Style{Gap: 8, Padding: core.Edges{Top: 8}}, Children: rows}
	})

	leftActions := core.Dynamic(func(*core.Scope) core.Node {
		primary := m.marketPrimaryAction(entry, "market-detail-", ui.ButtonPrimary)
		primary.Style.Width = core.Pct(100)
		primary.Style.Height = core.Pt(36)
		primary.Style.CornerRadius = 18
		btns := []core.Node{primary}
		if entry.source != nil {
			if acct := m.marketAccountAction(entry, m.accountOf(entry.source), "market-detail-"); acct != nil {
				if b, ok := acct.(ui.Button); ok {
					b.Variant = ui.ButtonOutline
					b.Style.Width = core.Pct(100)
					b.Style.Height = core.Pt(34)
					b.Style.CornerRadius = 17
					acct = b
				}
				btns = append(btns, acct)
			}
			remove := button(i18n.T("Remove Plugin"), "market-detail-remove-"+entry.ID, lucide.Trash, func() {
				confirm(cx, i18n.T("Remove Plugin?"), entry.Name, func() { m.removeNSGoPlugin(entry.source) })
			})
			remove.Variant = ui.ButtonGhost
			remove.Style.Width = core.Pct(100)
			remove.Style.Height = core.Pt(34)
			remove.Style.CornerRadius = 17
			remove.Disabled = m.nsgoPluginBusy.Get
			btns = append(btns, remove)
		}
		return core.View{Style: core.Style{Gap: 8}, Children: btns}
	})

	leftPane := core.View{
		Style: core.Style{
			Width: core.Pt(224), FlexShrink: 0, Background: p.sidebar,
			Borders: core.Edges{Right: 1}, BorderColor: p.line,
			Padding: core.All(22), Gap: 22,
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{Gap: 10},
				Children: []core.Node{
					artwork,
					core.View{Style: core.Style{Gap: 4}, Children: []core.Node{title, verified}},
					facts,
				},
			},
			spacer(),
			leftActions,
		},
	}

	// Right Detail Pane
	closeBtn := base.Button{
		Label:   i18n.T("Close"),
		TestID:  "market-detail-close",
		OnPress: func(core.PressEvent) { close() },
		Style: core.Style{
			Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14,
			Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Hover: &core.Style{Background: p.hover},
		},
		Children: []core.Node{ui.Icon(lucide.X, 14, func() core.Color { return p.muted })},
	}
	rightTop := row(label(i18n.T("About"), 12, core.FontWeightSemibold, p.muted), spacer(), closeBtn)

	var rightItems []core.Node
	rightItems = append(rightItems, rightTop)
	if description := entry.description(); description != "" {
		summary := label(description, 13.5, core.FontWeightNormal, p.text)
		summary.Style.LineHeight = 20
		rightItems = append(rightItems, summary)
	}
	rightItems = append(rightItems, dashedDivider(p.line))

	identity := label(entry.packageID(), 12, core.FontWeightNormal, p.text)
	identity.Style.FontFamily = core.FontMonospace
	identity.Selectable = core.SelectableOn
	idBox := core.View{
		Style:    core.Style{Background: p.sidebar, Padding: core.Symmetric(8, 12), CornerRadius: 10},
		Children: []core.Node{identity},
	}
	packageID := core.View{
		Style:    core.Style{Gap: 6},
		Children: []core.Node{label(i18n.T("Plugin ID"), 11.5, core.FontWeightMedium, p.muted), idBox},
	}
	rightItems = append(rightItems, packageID, marketFeatures(cx, p, entry))

	rightPane := core.Scroll{
		Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), MinHeight: core.Pt(0)},
		Children: []core.Node{
			core.View{
				Style:    core.Style{Padding: core.All(22), Gap: 16},
				Children: rightItems,
			},
		},
	}

	return core.View{
		Style:    core.Style{Direction: core.Row, AlignItems: core.AlignStretch, FlexShrink: 1, MinHeight: core.Pt(0)},
		Children: []core.Node{leftPane, rightPane},
	}
}

func marketFeatures(cx *core.Scope, p palette, entry marketPlugin) core.Node {
	green := marketSuccess(cx)
	var items []core.Node
	for _, feature := range []struct {
		method string
		name   string
		inDemo bool
	}{
		{"search", i18n.T("Search"), true},
		{"stream", i18n.T("Playback"), true},
		{"home", i18n.T("Home"), true},
		{"lyrics", i18n.T("Lyrics"), false},
		{"playlists", i18n.T("Playlists"), false},
		{"shows", i18n.T("Podcasts"), false},
		{"stations", i18n.T("Stations"), false},
		{"signIn", i18n.T("Sign In"), true},
	} {
		if entry.source != nil {
			if !entry.source.Can(feature.method) {
				continue
			}
		} else if entry.ID != plugin.NetEaseSourceID && (entry.ID != plugin.DemoSourceID || !feature.inDemo) {
			continue
		}
		chip := row(
			ui.Icon(lucide.Check, 13, func() core.Color { return green }),
			label(feature.name, 12, core.FontWeightMedium, p.text),
		)
		chip.Style.Gap = 6
		chip.Style.Padding = core.Symmetric(7, 10)
		chip.Style.Background = p.sidebar
		chip.Style.CornerRadius = 8
		items = append(items, chip)
	}
	grid := core.View{
		Style: core.Style{
			Display:             core.DisplayGrid,
			GridTemplateColumns: core.GridTracks("repeat(2, minmax(0, 1fr))"),
			Gap:                 8,
		},
		Children: items,
	}
	return core.View{
		Style:    core.Style{Gap: 8},
		Children: []core.Node{label(i18n.T("Features"), 11.5, core.FontWeightMedium, p.muted), grid},
	}
}
