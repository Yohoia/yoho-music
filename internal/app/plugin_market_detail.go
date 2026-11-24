package app

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/plugin"
)

func (m *model) marketDetail(cx *core.Scope, p palette, current func() marketPlugin, close func()) core.Node {
	popup := core.Style{Width: core.Pct(92), MaxWidth: core.Pt(620), MaxHeight: core.Pct(90), FlexShrink: 1,
		Background: p.background, CornerRadius: 18, Overflow: core.Hidden}
	if cx.Dark() {
		popup.BorderWidth, popup.BorderColor = 1, p.line
	} else {
		popup.ShadowColor, popup.ShadowOffset, popup.ShadowRadius = core.RGBA(0x000000, .27), core.Point{Y: 12}, 24
	}
	return base.Dialog{
		Open: func() bool { return current().ID != "" },
		OnOpenChange: func(open bool, _ base.Reason) {
			if !open {
				close()
			}
		},
		LabelFunc:      func() string { return current().Name },
		TestID:         "market-detail",
		BackdropTestID: "market-detail-backdrop",
		BackdropStyle:  core.Style{Background: core.RGBA(0x111015, .36), BackdropFilters: []core.Filter{core.Blur(8)}},
		PopupStyle:     popup,
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
	icon, origin := entry.identity()
	title := label(entry.Name, 22, core.FontWeightBold, p.text)
	title.MaxLines = 2
	dismiss := iconButton(lucide.X, i18n.T("Close"), "market-detail-close", close)
	header := row(marketArtwork(p, icon, 54),
		core.View{Style: fill.Merge(core.Style{Gap: 4}), Children: []core.Node{
			title, label(origin, 13, core.FontWeightMedium, p.muted),
		}}, dismiss)
	header.Style.Gap = 14
	header.Style.Padding = core.Symmetric(17, 22)
	header.Style.Borders.Bottom = 1
	header.Style.BorderColor = p.line

	identity := label(entry.packageID(), 13, core.FontWeightNormal, p.text)
	identity.Style.FontFamily = core.FontMonospace
	identity.Selectable = core.SelectableOn
	packageID := core.View{Style: core.Style{Gap: 5}, Children: []core.Node{
		label(i18n.T("Plugin ID"), 12, core.FontWeightMedium, p.muted), identity,
	}}

	content := []core.Node{m.marketDetailsFacts(cx, p, entry), packageID, marketFeatures(p, entry)}
	if description := entry.description(); description != "" {
		summary := label(description, 15, core.FontWeightNormal, p.text)
		summary.Style.LineHeight = 22
		content = append([]core.Node{summary}, content...)
	}
	body := core.Scroll{Style: core.Style{FlexShrink: 1, MinHeight: core.Pt(0)},
		Children: []core.Node{core.View{Style: core.Style{Padding: core.Symmetric(20, 22), Gap: 18}, Children: content}}}

	footer := core.Dynamic(func(*core.Scope) core.Node {
		var account core.Node
		if entry.source != nil {
			account = m.marketAccountAction(entry, m.accountOf(entry.source), "market-detail-")
		}
		actions := row(spacer(), account, m.marketPrimaryAction(entry, "market-detail-", ui.ButtonPrimary))
		actions.Style.Padding = core.Symmetric(12, 22)
		actions.Style.Borders.Top = 1
		actions.Style.BorderColor = p.line
		return actions
	})
	return core.View{Style: core.Style{FlexShrink: 1, MinHeight: core.Pt(0)}, Children: []core.Node{header, body, footer}}
}

func (m *model) marketDetailsFacts(cx *core.Scope, p palette, entry marketPlugin) core.Node {
	return core.Dynamic(func(*core.Scope) core.Node {
		state := pluginAccount{}
		if entry.source != nil {
			state = m.accountOf(entry.source)
		}
		status, icon, tint := m.marketInstallation(cx, p, entry)
		version := entry.Version
		if version == "" {
			version = i18n.T("Unknown")
		}
		versionText := label(version, 14, core.FontWeightMedium, p.text)
		versionText.Style.FontVariant = core.FontTabularNums
		facts := []core.Node{
			marketFact(p, i18n.T("Version"), versionText),
			marketFact(p, i18n.T("Status"), marketStatusLabel(status, icon, tint)),
		}
		if entry.source != nil && (entry.source.Can("account") || entry.source.Can("signIn") || state.account != nil) {
			account := i18n.T("Signed Out")
			switch {
			case state.busy:
				account = i18n.T("Checking account…")
			case state.account != nil:
				account = marketAccountName(state.account)
			}
			accountText := label(account, 14, core.FontWeightMedium, p.text)
			accountText.MaxLines = 2
			facts = append(facts, marketFact(p, i18n.T("Account"), accountText))
		}
		return core.View{Style: core.Style{Direction: core.Row, AlignItems: core.AlignStretch, Gap: 1,
			Background: p.line, BorderWidth: 1, BorderColor: p.line, CornerRadius: 10, Overflow: core.Hidden},
			Children: facts}
	})
}

func marketFact(p palette, name string, value core.Node) core.Node {
	return core.View{Style: fill.Merge(core.Style{FlexBasis: core.Pt(0), MinWidth: core.Pt(0), Padding: core.Symmetric(11, 12),
		Background: p.sidebar, Gap: 5}), Children: []core.Node{label(name, 12, core.FontWeightMedium, p.muted), value}}
}

func marketFeatures(p palette, entry marketPlugin) core.Node {
	var items []core.Node
	for _, feature := range []struct {
		method string
		name   string
		icon   glyph
		inDemo bool
	}{
		{"search", i18n.T("Search"), lucide.Search, true},
		{"stream", i18n.T("Playback"), lucide.Play, true},
		{"home", i18n.T("Home"), lucide.House, true},
		{"lyrics", i18n.T("Lyrics"), lucide.FileText, false},
		{"playlists", i18n.T("Playlists"), lucide.ListMusic, false},
		{"shows", i18n.T("Podcasts"), lucide.Mic, false},
		{"stations", i18n.T("Stations"), lucide.Radio, false},
		{"signIn", i18n.T("Sign In"), lucide.LogIn, true},
	} {
		if entry.source != nil {
			if !entry.source.Can(feature.method) {
				continue
			}
		} else if entry.ID != plugin.NetEaseSourceID && (entry.ID != plugin.DemoSourceID || !feature.inDemo) {
			continue
		}
		chip := row(ui.Icon(feature.icon, 14, func() core.Color { return p.muted }),
			label(feature.name, 12, core.FontWeightMedium, p.text))
		chip.Style.Gap = 5
		chip.Style.Padding = core.Symmetric(6, 9)
		chip.Style.Background = p.hover
		chip.Style.CornerRadius = 7
		items = append(items, chip)
	}
	return core.View{Style: core.Style{Gap: 8}, Children: []core.Node{
		label(i18n.T("Features"), 13, core.FontWeightBold, p.text),
		core.View{Style: core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 6}, Children: items},
	}}
}
