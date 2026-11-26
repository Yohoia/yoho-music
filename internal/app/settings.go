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

	appearanceSeg := settingsSegment(p, "appearance",
		[]settingsSegOption{
			{value: "system", label: system},
			{value: "light", label: i18n.T("Light")},
			{value: "dark", label: i18n.T("Dark")},
		},
		func() string { return m.prefs.Get().Appearance },
		func(value string) { m.updatePreferences(cx, func(pr *preferences) { pr.Appearance = value }) },
	)

	playerStyleSeg := settingsSegment(p, "player-style",
		[]settingsSegOption{
			{value: "", label: i18n.T("Artwork")},
			{value: "vinyl", label: i18n.T("Vinyl")},
			{value: "cassette", label: i18n.T("Cassette")},
		},
		func() string { return m.prefs.Get().PlayerStyle },
		func(value string) { m.updatePreferences(cx, func(pr *preferences) { pr.PlayerStyle = value }) },
	)

	qualitySeg := settingsSegment(p, "streaming-quality",
		[]settingsSegOption{
			{value: "standard", label: i18n.T("Standard")},
			{value: "high", label: i18n.T("High")},
			{value: "lossless", label: i18n.T("Lossless")},
		},
		func() string { return m.prefs.Get().Quality },
		func(value string) { m.updatePreferences(cx, func(pr *preferences) { pr.Quality = value }) },
	)

	appearanceSection := settingsPaperSection(p, appearance,
		settingsRowWithDesc(cx, appearance, "", appearanceSeg),
		settingsRowWithDesc(cx, language, "", ui.Select[string]{
			Label: language, TestID: "language",
			Options: languageOptions(system),
			Value:   func() string { return m.prefs.Get().Language },
			OnChange: func(value string) {
				m.updatePreferences(cx, func(pr *preferences) { pr.Language = value })
			},
		}),
		settingsRowWithDesc(cx, i18n.T("Player Style"), "", playerStyleSeg),
	)

	playbackRows := []core.Node{
		settingsRowWithDesc(cx, normalize,
			i18n.T("Adjust each song's volume by what its plugin measured of it, for them to sound as loud as each other"),
			ui.Switch{
				Label: normalize, TestID: "normalize",
				Checked: func() bool { return m.prefs.Get().Normalize },
				OnChange: func(value bool) {
					m.updatePreferences(cx, func(pr *preferences) { pr.Normalize = value })
					m.player.SetVolume(m.prefs.Get().Volume)
				},
			}),
		settingsRowWithDesc(cx, i18n.T("Streaming Quality"), "", qualitySeg),
	}
	if cache := m.cacheSettings(cx); cache != nil {
		playbackRows = append(playbackRows, cache)
	}
	if captions := m.captionsSettings(cx); captions != nil {
		playbackRows = append(playbackRows, captions)
	}
	playbackRows = append(playbackRows,
		settingsRowWithDesc(cx, i18n.T("Top Player"),
			i18n.T("A small player at the notch on top of the screen, which opens as the pointer rests on it"),
			ui.Switch{
				Label: i18n.T("Top Player"), TestID: "notch-enabled",
				Checked:  func() bool { return m.prefs.Get().NotchEnabled },
				OnChange: func(on bool) { m.updatePreferences(cx, func(pr *preferences) { pr.NotchEnabled = on }) },
			}),
		core.Show{When: func() bool { return m.prefs.Get().NotchEnabled }, Then: func(cx *core.Scope) core.Node {
			activity := i18n.T("Show Activity at the Notch")
			virtual := i18n.T("Show on Displays Without a Notch")
			return core.View{Children: []core.Node{
				core.View{Style: core.Style{Height: core.Pt(1), Background: p.line}},
				settingsRowWithDesc(cx, activity, "", ui.Switch{
					Label: activity, TestID: "notch-activity",
					Checked:  func() bool { return m.prefs.Get().NotchActivity },
					OnChange: func(on bool) { m.updatePreferences(cx, func(pr *preferences) { pr.NotchActivity = on }) },
				}),
				core.View{Style: core.Style{Height: core.Pt(1), Background: p.line}},
				settingsRowWithDesc(cx, virtual, "", ui.Switch{
					Label: virtual, TestID: "notch-virtual",
					Checked:  func() bool { return m.prefs.Get().NotchVirtual },
					OnChange: func(on bool) { m.updatePreferences(cx, func(pr *preferences) { pr.NotchVirtual = on }) },
				}),
			}}
		}},
	)
	playbackSection := settingsPaperSection(p, i18n.T("Playback"), playbackRows...)

	rescanBtn := base.Button{
		Label:   i18n.T("Rescan"),
		TestID:  "rescan",
		OnPress: func(core.PressEvent) { m.lib.Rescan() },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(32), Padding: core.Symmetric(0, 14), CornerRadius: 16,
			Background: p.sidebar,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
			Hover: &core.Style{Background: p.hover},
		},
		Children: []core.Node{
			ui.Icon(lucide.RefreshCw, 13, func() core.Color { return p.text }),
			label(i18n.T("Rescan"), 12, core.FontWeightMedium, p.text),
		},
	}
	addFolderBtn := base.Button{
		Label:   i18n.T("Add a Music Folder"),
		TestID:  "settings-add-folder",
		OnPress: func(core.PressEvent) { m.pickMusic(cx) },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(32), Padding: core.Symmetric(0, 16), CornerRadius: 16,
			Background: p.button, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
			Hover: &core.Style{Opacity: core.Alpha(.88)},
		},
		Children: []core.Node{
			ui.Icon(lucide.FolderPlus, 13, func() core.Color { return p.onButton }),
			label(i18n.T("Add a Music Folder"), 12, core.FontWeightSemibold, p.onButton),
		},
	}
	foldersHeader := core.View{
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, Padding: core.Symmetric(12, 4)},
		Children: []core.Node{
			core.View{
				Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 3},
				Children: []core.Node{
					label(i18n.T("Music Folders"), 14, core.FontWeightMedium, p.text),
					label(i18n.T("Add the folders where your music lives"), 12, core.FontWeightNormal, p.muted),
				},
			},
			rescanBtn,
			addFolderBtn,
		},
	}
	foldersList := core.Dynamic(func(cx *core.Scope) core.Node {
		m.updated.Get()
		folders := m.lib.Folders()
		if len(folders) == 0 {
			return nil
		}
		var rows []core.Node
		for _, folder := range folders {
			path := folder
			remove := i18n.T("Remove Folder")
			pathText := label(path, 12, core.FontWeightNormal, p.text)
			pathText.Style.FontFamily = core.FontMonospace
			pathText.MaxLines = 1
			item := row(pathText, spacer(), iconButton(lucide.FolderMinus, remove, "remove-folder", func() {
				confirm(cx, remove, path, func() { m.run(func() error { m.lib.RemoveFolder(path); return nil }) })
			}))
			item.Style.Background = p.sidebar
			item.Style.CornerRadius = 10
			item.Style.Padding = core.Symmetric(4, 12)
			rows = append(rows, item)
		}
		return core.View{
			Style:    core.Style{Padding: core.Edges{Left: 4, Right: 4, Bottom: 4}, Gap: 6},
			Children: rows,
		}
	})
	foldersSection := core.View{Style: core.Style{Gap: 4}, Children: []core.Node{foldersHeader, foldersList}}

	aboutRow := row(
		core.Image{Source: core.ImageSource{Data: settingsIcon}, Style: core.Style{Width: core.Pt(36), Height: core.Pt(36), CornerRadius: 8}},
		core.View{
			Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 2},
			Children: []core.Node{
				label("Yoho Music", 14, core.FontWeightSemibold, p.text),
				func() core.Node {
					v := label("Version "+core.App().Version, 12, core.FontWeightNormal, p.muted)
					v.Style.FontFamily = core.FontMonospace
					return v
				}(),
			},
		},
	)
	aboutRow.Style.Gap = 12
	aboutRow.Style.Padding = core.Symmetric(8, 4)

	header := core.View{
		Style: core.Style{Gap: 4},
		Children: []core.Node{
			label(i18n.T("Settings"), 22, core.FontWeightSemibold, p.text),
			label(i18n.T("Appearance")+" · "+i18n.T("Playback")+" · "+i18n.T("Music Folders"), 12, core.FontWeightNormal, p.muted),
		},
	}

	body := paperWrap(core.View{
		Style: core.Style{Padding: core.Edges{Top: 62, Bottom: 108}, Gap: 20},
		Children: []core.Node{
			header,
			dashedDivider(p.line),
			appearanceSection,
			dashedDivider(p.line),
			playbackSection,
			dashedDivider(p.line),
			foldersSection,
			dashedDivider(p.line),
			aboutRow,
		},
	})
	body.TestID = "settings-content"
	return core.View{
		Style:    fill.Merge(core.Style{Background: p.background}),
		Children: []core.Node{core.Scroll{Style: fill, Children: []core.Node{body}}},
	}
}

type settingsSegOption struct {
	value, label string
}

func settingsSegment(p palette, id string, options []settingsSegOption, value func() string, onChange func(string)) core.Node {
	items := make([]core.Node, 0, len(options))
	for _, opt := range options {
		o := opt
		selected := func() bool { return value() == o.value }
		t := core.Text{
			Content: o.label,
			Style:   core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: p.muted},
			StyleFunc: func() core.Style {
				if selected() {
					return core.Style{Color: p.onButton}
				}
				return core.Style{}
			},
		}
		items = append(items, base.Button{
			Label:    o.label,
			Selected: selected,
			OnPress:  func(core.PressEvent) { onChange(o.value) },
			Style: core.Style{
				Height: core.Pt(26), Padding: core.Symmetric(0, 12), CornerRadius: 13,
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			},
			StyleFunc: func() core.Style {
				return p.selectableButtonStyle(selected())
			},
			Children: []core.Node{t},
		})
	}
	return core.View{
		TestID: id,
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 2,
			Padding: core.All(3), CornerRadius: 16, Background: p.sidebar,
		},
		Children: items,
	}
}

func settingsPaperSection(p palette, heading string, rows ...core.Node) core.Node {
	var children []core.Node
	for i, r := range rows {
		if r == nil {
			continue
		}
		if i > 0 && len(children) > 0 {
			if _, isShow := r.(core.Show); !isShow {
				children = append(children, core.View{Style: core.Style{Height: core.Pt(1), Background: p.line}})
			}
		}
		children = append(children, r)
	}
	title := label(heading, 12, core.FontWeightSemibold, p.muted)
	title.Style.Padding = core.Edges{Left: 4, Bottom: 4}
	return core.View{Style: core.Style{Gap: 2}, Children: append([]core.Node{title}, children...)}
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
	return settingsRowWithDesc(cx, name, "", control)
}

func settingsRowWithDesc(cx *core.Scope, name, desc string, control core.Node) core.Node {
	p := colors(cx)
	leftChildren := []core.Node{label(name, 14, core.FontWeightMedium, p.text)}
	if desc != "" {
		d := label(desc, 12, core.FontWeightNormal, p.muted)
		d.Style.LineHeight = 16
		leftChildren = append(leftChildren, d)
	}
	left := core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 3}, Children: leftChildren}
	setting := row(left, control)
	setting.Style.MinHeight = core.Pt(48)
	setting.Style.Padding = core.Symmetric(14, 4)
	setting.Style.Gap = 16
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
		icon := item.Icon
		if item.Page != "" {
			if title != "" && src.Can("browse") {
				if icon == "" && item.Page == "playlists-hub" {
					icon = "playlists"
				}
				entries = append(entries, pluginNavEntry{title: title, target: route{page: "browse", id: item.Page}, icon: icon})
			}
			continue
		}
		if _, ok := names[item.Builtin]; !ok || !src.Can(item.Builtin) {
			continue
		}
		if title == "" {
			title = names[item.Builtin]
		}
		if icon == "" {
			switch item.Builtin {
			case "home":
				icon = "home"
			case "playlists":
				icon = "playlists"
			case "recent":
				icon = "history"
			}
		}
		entries = append(entries, pluginNavEntry{title: title, target: route{page: item.Builtin}, icon: icon})
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
