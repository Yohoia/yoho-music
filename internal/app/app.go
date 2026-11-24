package app

import (
	_ "embed"
	"os"
	"path/filepath"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"
)

//go:embed assets/SettingsIcon.png
var settingsIcon []byte

//go:embed assets/Tray.png
var trayIcon []byte

// Run starts the app and blocks until it exits.
func Run() error {
	nsgo.Run(App)
	return nil
}

// App wires the window, panels, tray and media controls.
func App(cx *core.Scope) {
	dir := os.Getenv("YOHO_MUSIC_DATA_DIR")
	if dir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			panic(err)
		}
		dir = filepath.Join(base, "Yoho Music")
	}
	m := newModel(cx, dir)
	window := desktop.OpenWindow(cx, desktop.WindowOptions{Title: "Yoho Music", Key: "main",
		Width: 1180, Height: 780, Constraints: core.WindowConstraints{MinWidth: 880, MinHeight: 580}}, m.root)
	if window == nil {
		nsgo.SetMainView(cx, m.root)
		return
	}
	desktop.SetMainWindow(cx, window)
	desktop.OnReopen(cx, window.Focus)
	m.installMediaControls()
	panel := desktop.OpenPanel(cx, desktop.PanelOptions{Width: 350, Height: 144, HideOnBlur: true}, m.miniPlayer)
	notch := desktop.OpenPanel(cx, desktop.PanelOptions{Width: notchWidth, Height: notchHeight, NoActivate: true, Transparent: true,
		PointerEvents: desktop.WindowPointerPainted}, m.notchPlayer)
	if notch != nil {
		m.bindNotch(cx, notch)
		desktop.RegisterHotKey(cx, "alt+shift+p", m.toggleNotch)
	}
	desktop.OnOpenLinks(cx, func(links []string) {
		for _, link := range links {
			if link == "yoho-music://open" {
				window.Focus()
				break
			}
		}
	})
	if tray := desktop.NewTray(cx, desktop.TrayOptions{Icon: trayIcon, Template: true, Tooltip: "Yoho Music", Panel: panel,
		Menu: func() []desktop.MenuItem {
			return []desktop.MenuItem{
				{Label: i18n.T("Open Yoho Music"), OnSelect: window.Focus},
				{Label: i18n.T("Play/Pause"), OnSelect: m.playback(m.player.TogglePlay)},
				{Label: i18n.T("Toggle Top Player"), OnSelect: m.toggleNotch},
				{Separator: true}, {Label: i18n.T("Quit Yoho Music"), OnSelect: func() { m.saveSession(); m.savePreferences(); cx.Quit() }},
			}
		}}); tray != nil {
		desktop.SetShutdownMode(cx, desktop.ShutdownOnExplicit)
		window.OnClosing(cx, func(closing *desktop.WindowClosing) { closing.Cancel(); window.Hide() })
	}
	desktop.OnQuitRequest(cx, func(*desktop.QuitRequest) { m.saveSession(); m.savePreferences() })
}
