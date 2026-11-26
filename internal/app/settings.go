package app

import (
	"path/filepath"
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

type desktopItem struct {
	label  string
	action func()
}

func confirm(cx *core.Scope, title, detail string, action func()) {
	label := i18n.T("Confirm")
	base.ShowAlertDialog(cx, base.AlertDialog{Title: title, Description: detail, Actions: []core.AlertAction{
		{Label: label}, {Label: i18n.T("Cancel"), Role: core.AlertCancel},
	}}, func(choice string) {
		if choice == label {
			action()
		}
	})
}

func (m *model) setVolume(volume float64) {
	p := m.prefs.Get()
	p.Volume = volume
	m.prefs.Set(p)
	m.player.SetVolume(volume)
	if m.dir != "" {
		if m.volumeTimer != nil {
			m.volumeTimer()
		}
		m.volumeTimer = m.cx.AfterFunc(250*time.Millisecond, m.savePreferences)
	}
}

func (m *model) savePreferences() {
	if m.dir != "" {
		m.report(writeJSON(filepath.Join(m.dir, "settings.json"), m.prefs.Get()))
	}
}

func (m *model) settingsPage(cx *core.Scope) core.Node {
	p := colors(cx)
	language := i18n.T("Language")
	appearance := i18n.T("Appearance")
	system := i18n.T("System")
	normalize := i18n.T("Normalize Volume")
	controls := []core.Node{
		label(i18n.T("General"), 19, core.FontWeightBold, p.text),
		settingsRow(cx, appearance, ui.Select[string]{Label: appearance, TestID: "appearance",
			Options: []ui.Option[string]{{Value: "system", Label: system}, {Value: "light", Label: i18n.T("Light")}, {Value: "dark", Label: i18n.T("Dark")}},
			Value:   func() string { return m.prefs.Get().Appearance }, OnChange: func(value string) { m.updatePreferences(cx, func(p *preferences) { p.Appearance = value }) }}),
		settingsRow(cx, language, ui.Select[string]{Label: language, TestID: "language",
			Options: languageOptions(system),
			Value:   func() string { return m.prefs.Get().Language }, OnChange: func(value string) { m.updatePreferences(cx, func(p *preferences) { p.Language = value }) }}),
		settingsRow(cx, normalize, ui.Switch{Label: normalize, TestID: "normalize", Checked: func() bool { return m.prefs.Get().Normalize },
			OnChange: func(value bool) {
				m.updatePreferences(cx, func(p *preferences) { p.Normalize = value })
				m.player.SetVolume(m.prefs.Get().Volume)
			}}),
		settingsRow(cx, i18n.T("Player Style"), ui.Select[string]{Label: i18n.T("Player Style"), TestID: "player-style",
			Options: []ui.Option[string]{{Value: "", Label: i18n.T("Artwork")}, {Value: "vinyl", Label: i18n.T("Vinyl")}, {Value: "cassette", Label: i18n.T("Cassette")}},
			Value:   func() string { return m.prefs.Get().PlayerStyle }, OnChange: func(value string) { m.updatePreferences(cx, func(p *preferences) { p.PlayerStyle = value }) }}),
		settingsRow(cx, i18n.T("Streaming Quality"), ui.Select[string]{Label: i18n.T("Streaming Quality"), TestID: "streaming-quality",
			Options:  []ui.Option[string]{{Value: "standard", Label: i18n.T("Standard")}, {Value: "high", Label: i18n.T("High")}, {Value: "lossless", Label: i18n.T("Lossless")}},
			Value:    func() string { return m.prefs.Get().Quality },
			OnChange: func(value string) { m.updatePreferences(cx, func(p *preferences) { p.Quality = value }) }}),
		m.cacheSettings(cx),
		m.captionsSettings(cx),
		settingsRow(cx, i18n.T("Top Player"), ui.Switch{Label: i18n.T("Top Player"), TestID: "notch-enabled",
			Checked:  func() bool { return m.prefs.Get().NotchEnabled },
			OnChange: func(on bool) { m.updatePreferences(cx, func(p *preferences) { p.NotchEnabled = on }) }}),
		core.Show{When: func() bool { return m.prefs.Get().NotchEnabled }, Then: func(cx *core.Scope) core.Node {
			activity := i18n.T("Show Activity at the Notch")
			virtual := i18n.T("Show on Displays Without a Notch")
			return core.View{Children: []core.Node{
				settingsRow(cx, activity, ui.Switch{Label: activity, TestID: "notch-activity",
					Checked:  func() bool { return m.prefs.Get().NotchActivity },
					OnChange: func(on bool) { m.updatePreferences(cx, func(p *preferences) { p.NotchActivity = on }) }}),
				settingsRow(cx, virtual, ui.Switch{Label: virtual, TestID: "notch-virtual",
					Checked:  func() bool { return m.prefs.Get().NotchVirtual },
					OnChange: func(on bool) { m.updatePreferences(cx, func(p *preferences) { p.NotchVirtual = on }) }}),
			}}
		}},
		label(i18n.T("Music Folders"), 19, core.FontWeightBold, p.text),
		row(button(i18n.T("Add a Music Folder"), "settings-add-folder", lucide.FolderPlus, func() { m.pickMusic(cx) }),
			button(i18n.T("Rescan"), "rescan", lucide.RefreshCw, m.lib.Rescan)),
		core.Dynamic(func(cx *core.Scope) core.Node {
			m.updated.Get()
			var rows []core.Node
			for _, folder := range m.lib.Folders() {
				remove := i18n.T("Remove Folder")
				rows = append(rows, settingsRow(cx, folder, iconButton(lucide.FolderMinus, remove, "remove-folder", func() {
					confirm(cx, remove, folder, func() { m.run(func() error { m.lib.RemoveFolder(folder); return nil }) })
				})))
			}
			return core.Group(rows)
		}),
		label(i18n.T("Plugins"), 19, core.FontWeightBold, p.text),
		m.nsgoPluginSettings(cx),
		row(core.Image{Source: core.ImageSource{Data: settingsIcon}, Style: core.Style{Width: core.Pt(30), Height: core.Pt(30)}},
			label("Yoho Music "+core.App().Version, 12, core.FontWeightMedium, p.muted)),
	}
	return core.View{Style: fill.Merge(core.Style{Padding: core.Edges{Top: 36}}), Children: []core.Node{m.pageHeader(cx, i18n.T("Settings"), "", ""),
		core.Scroll{Style: fill, Children: []core.Node{core.View{Style: core.Style{Gap: 15, Padding: core.Edges{Top: 4, Left: 28, Right: 28, Bottom: 100}}, Children: controls}}},
	}}
}

// languageOptions lists the languages the listener can pick, after System.
func languageOptions(system string) []ui.Option[string] {
	options := make([]ui.Option[string], 0, len(i18n.Languages)+1)
	options = append(options, ui.Option[string]{Value: i18n.System, Label: system})
	for _, lang := range i18n.Languages {
		options = append(options, ui.Option[string]{Value: lang.Code, Label: lang.Name})
	}
	return options
}

func settingsRow(cx *core.Scope, name string, control core.Node) core.Node {
	setting := row(label(name, 13, core.FontWeightMedium, colors(cx).text), spacer(), control)
	setting.Style.MinHeight = core.Pt(34)
	return setting
}

type pluginNavEntry struct {
	title  string
	target route
	icon   string
}

func pluginNavEntries(src music.Source) []pluginNavEntry {
	info := src.Info()
	names := map[string]string{"home": i18n.T("Home"), "liked": i18n.T("Liked Songs"), "shows": i18n.T("Podcasts"),
		"stations": i18n.T("Stations"), "playlists": i18n.T("Playlists"), "recent": i18n.T("Recently Played"), "savedAlbums": i18n.T("Saved Albums")}
	nav := info.Navigation
	if len(nav) == 0 {
		for _, page := range []string{"home", "liked", "playlists", "savedAlbums", "recent", "shows", "stations"} {
			if src.Can(page) {
				nav = append(nav, music.NavItem{Builtin: page})
			}
		}
	}
	var entries []pluginNavEntry
	for _, item := range nav {
		if info.ID == plugin.NetEaseSourceID {
			if item.Page == "my-music" || item.Page == "for-you" || item.Page == "discovery" || item.Page == "charts" || item.Builtin == "stations" {
				continue
			}
		}
		title := item.Title
		if item.Page != "" {
			if title != "" && src.Can("browse") {
				entries = append(entries, pluginNavEntry{title: title, target: route{page: "browse", id: item.Page}, icon: item.Icon})
			}
			continue
		}
		if _, ok := names[item.Builtin]; !ok || !src.Can(item.Builtin) {
			continue
		}
		if title == "" {
			title = names[item.Builtin]
		}
		entries = append(entries, pluginNavEntry{title: title, target: route{page: item.Builtin}, icon: item.Icon})
	}
	return entries
}

func isPluginCategoryPage(r route, entries []pluginNavEntry) bool {
	if r.page == "home" {
		return true
	}
	for _, entry := range entries {
		if entry.target == r {
			return true
		}
	}
	return false
}

func (m *model) pluginAccountMenuItems(cx *core.Scope, src music.Source, closeMenu func()) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		p := colors(cx)
		state := m.accountOf(src)
		var rows []core.Node
		if state.account != nil && state.account.Name != "" {
			name := label(state.account.Name, 12, core.FontWeightNormal, p.muted)
			name.MaxLines = 1
			name.Style.Padding = core.Symmetric(4, 12)
			rows = append(rows, name)
		}
		if state.busy {
			busy := label(i18n.T("Checking account…"), 12, core.FontWeightNormal, p.muted)
			busy.Style.Padding = core.Symmetric(4, 12)
			rows = append(rows, busy)
		} else {
			item := func(icon glyph, text, id string, action func()) core.Node {
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
					Children: []core.Node{ui.Icon(icon, 15, func() core.Color { return p.muted }), label(text, 13, core.FontWeightMedium, p.text)},
				}
			}
			if src.Can("signIn") && state.account == nil {
				rows = append(rows, item(lucide.LogIn, i18n.T("Sign In"), "sign-in", func() { m.changeAccount(src, true) }))
			}
			if src.Can("signOut") && (state.account != nil || !src.Can("account")) {
				rows = append(rows, item(lucide.LogOut, i18n.T("Sign Out"), "sign-out", func() { m.changeAccount(src, false) }))
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return core.View{Style: core.Style{Gap: 2}, Children: rows}
	})
}

func (m *model) playStation(src music.Source, station music.Station) {
	go func() {
		tracks, err := src.StationNext(station.ID)
		m.cx.Post(func() {
			if !m.sourceActionResult(src, err) {
				return
			}
			sourceID := src.Info().ID
			m.player.PlayRadio(station.Name, withSource(tracks, sourceID), func(got func([]music.Track)) {
				go func() {
					next, err := src.StationNext(station.ID)
					m.cx.Post(func() {
						m.sourceActionResult(src, err)
						got(withSource(next, sourceID))
						m.refreshPlayback()
					})
				}()
			})
			m.refreshPlayback()
		})
	}()
}

func (m *model) revealSourceTrack(target string, next route) {
	if target == "" {
		target = local.ID
	}
	if m.source.Get() != target {
		m.selectSource(target)
	}
	m.navigate(next)
}

func trackMenu(cx *core.Scope, items []desktopItem, at core.Point) {
	menu := make([]desktop.MenuItem, 0, len(items))
	for _, item := range items {
		menu = append(menu, desktop.MenuItem{Label: item.label, OnSelect: item.action})
	}
	if len(menu) > 0 {
		desktop.PopupMenu(cx, menu, at, nil)
	}
}
