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
	return core.View{Style: fill, Children: []core.Node{m.pageHeader(cx, i18n.T("Settings"), "", ""),
		core.Scroll{Style: fill, Children: []core.Node{core.View{Style: core.Style{Gap: 15, Padding: core.Edges{Top: 4, Left: 28, Right: 28, Bottom: 24}}, Children: controls}}},
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

func (m *model) pluginNavigation(cx *core.Scope) core.Node {
	src := m.activeSource()
	if src == nil {
		return nil
	}
	info := src.Info()
	items := []core.Node{m.sectionTitle(cx, info.Name)}
	names := map[string]string{"home": i18n.T("Home"), "liked": i18n.T("Liked Songs"), "shows": i18n.T("Podcasts"),
		"stations": i18n.T("Stations"), "playlists": i18n.T("Playlists")}
	icons := map[string]glyph{"home": lucide.House, "liked": lucide.Heart, "shows": lucide.Mic, "stations": lucide.Radio, "playlists": lucide.ListMusic}
	nav := info.Navigation
	if len(nav) == 0 {
		for _, page := range []string{"home", "liked", "playlists", "shows", "stations"} {
			if src.Can(page) {
				nav = append(nav, music.NavItem{Builtin: page})
			}
		}
	}
	for _, item := range nav {
		title := item.Title
		if item.Page != "" {
			items = append(items, m.navTo(cx, lucide.LayoutGrid, title, route{page: "browse", id: item.Page}))
			continue
		}
		icon := icons[item.Builtin]
		if icon == nil || !src.Can(item.Builtin) {
			continue
		}
		if title == "" {
			title = names[item.Builtin]
		}
		items = append(items, m.navTo(cx, icon, title, route{page: item.Builtin}))
	}
	items = append(items, core.Dynamic(func(cx *core.Scope) core.Node {
		state := m.accountOf(src)
		var controls []core.Node
		if state.account != nil && state.account.Name != "" {
			controls = append(controls, m.sectionTitle(cx, state.account.Name))
		}
		if state.busy {
			controls = append(controls, m.sectionTitle(cx, i18n.T("Checking account…")))
		} else {
			if src.Can("signIn") && state.account == nil {
				controls = append(controls, m.nav(cx, lucide.LogIn, i18n.T("Sign In"), "sign-in", false, func() { m.changeAccount(src, true) }))
			}
			if src.Can("signOut") && (state.account != nil || !src.Can("account")) {
				controls = append(controls, m.nav(cx, lucide.LogOut, i18n.T("Sign Out"), "sign-out", false, func() { m.changeAccount(src, false) }))
			}
		}
		return core.Group(controls)
	}))
	return core.View{Style: core.Style{Gap: 2}, Children: items}
}

func (m *model) playStation(src music.Source, station music.Station) {
	go func() {
		tracks, err := src.StationNext(station.ID)
		m.cx.Post(func() {
			if err != nil {
				m.report(err)
				return
			}
			sourceID := src.Info().ID
			m.player.PlayRadio(station.Name, withSource(tracks, sourceID), func(got func([]music.Track)) {
				go func() {
					next, err := src.StationNext(station.ID)
					m.cx.Post(func() { m.report(err); got(withSource(next, sourceID)); m.refreshPlayback() })
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
