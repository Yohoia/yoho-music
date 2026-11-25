package app

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
	"yoho-music/internal/i18n"
)

func (m *model) appMenu(cx *core.Scope) {
	cx.Effect(func() {
		_ = m.prefs.Get().Language
		focus := func() {
			if window := desktop.CurrentWindow(cx); window != nil {
				window.Focus()
			}
		}
		separator := desktop.MenuItem{Separator: true}
		menus := []desktop.MenuItem{
			{Label: "Yoho Music", Items: []desktop.MenuItem{
				{Role: desktop.MenuAbout},
				separator,
				{Label: i18n.T("Settings…"), Shortcut: "mod+,", OnSelect: func() {
					focus()
					m.navigate(route{page: "settings"})
				}},
				separator,
				{Role: desktop.MenuServices}, {Role: desktop.MenuHide}, {Role: desktop.MenuHideOthers},
				{Role: desktop.MenuShowAll}, separator, {Role: desktop.MenuQuit},
			}},
			{Label: i18n.T("File"), Items: []desktop.MenuItem{
				{Label: i18n.T("New Playlist…"), Shortcut: "mod+n", OnSelect: func() {
					focus()
					m.navigate(route{page: "playlist-edit"})
				}},
			}},
			{Label: i18n.T("Edit"), Items: []desktop.MenuItem{
				{Role: desktop.MenuUndo}, {Role: desktop.MenuRedo}, separator,
				{Role: desktop.MenuCut}, {Role: desktop.MenuCopy}, {Role: desktop.MenuPaste}, {Role: desktop.MenuSelectAll},
			}},
			{Label: i18n.T("Controls"), Items: []desktop.MenuItem{
				{Label: i18n.T("Play/Pause"), OnSelect: m.playback(m.player.TogglePlay)},
				{Label: i18n.T("Next"), Shortcut: "mod+right", When: core.MenuAuto, OnSelect: m.playback(m.player.Next)},
				{Label: i18n.T("Previous"), Shortcut: "mod+left", When: core.MenuAuto, OnSelect: m.playback(m.player.Prev)},
				separator,
				{Label: i18n.T("Like/Unlike"), Shortcut: "mod+d", OnSelect: func() {
					if track := m.player.Current(); track != nil {
						m.toggleLike(*track)
					}
				}},
				{Label: i18n.T("Shuffle"), OnSelect: m.playback(m.player.ToggleShuffle)},
				{Label: i18n.T("Change Repeat Mode"), OnSelect: m.playback(m.player.CycleRepeat)},
				separator,
				{Label: i18n.T("Now Playing"), Shortcut: "mod+l", OnSelect: func() {
					focus()
					m.openPlaybackPage()
				}},
				{Label: i18n.T("Toggle Top Player"), OnSelect: m.toggleNotch},
			}},
		}
		if source := m.activeSource(); source != nil && len(source.Info().Navigation) > 0 {
			entries := pluginNavEntries(source)
			items := make([]desktop.MenuItem, 0, len(entries))
			for _, entry := range entries {
				items = append(items, desktop.MenuItem{Label: entry.title, OnSelect: func() {
					focus()
					m.navigate(entry.target)
				}})
			}
			if len(items) > 0 {
				menus = append(menus, desktop.MenuItem{Label: source.Info().Name, Items: items})
			}
		}
		menus = append(menus, desktop.MenuItem{Label: i18n.T("Window"), Items: []desktop.MenuItem{
			{Role: desktop.MenuMinimize}, {Role: desktop.MenuZoom}, {Role: desktop.MenuClose},
			separator,
			{Label: "Yoho Music", Shortcut: "mod+0", OnSelect: focus}, {Role: desktop.MenuBringAllToFront},
		}})
		desktop.SetApplicationMenu(cx, menus)
	})
}
