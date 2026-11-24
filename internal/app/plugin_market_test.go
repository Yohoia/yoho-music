package app

import (
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"yoho-music/internal/plugin"
)

func TestPluginMarket(t *testing.T) {
	dir := t.TempDir()
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.OnAlert = func(core.AlertConfig) int { return 0 }
	app.Press("id:nav-plugin-market")
	app.WaitUntil(5*time.Second, func() bool { return app.Enabled("id:market-install-demo") })
	app.Screenshot("plugin-market")
	app.Press("id:market-installed")
	app.WaitForGone("id:market-install-demo", time.Second)
	app.Press("id:market-all")
	app.Type("id:market-search", "no matching source")
	app.WaitForGone("id:market-install-demo", time.Second)
	// Reopen the page with a fresh filter.
	app.Press("id:nav-settings")
	app.Press("id:nav-plugin-market")
	app.Press("id:market-install-demo")

	id := plugin.DemoSourceID
	signIn := "id:market-sign-in-" + id
	app.WaitFor(signIn, 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(signIn) && app.Enabled(signIn) })
	app.Press(signIn)
	app.Press("id:sign-in-confirm")
	app.WaitFor("id:market-sign-out-"+id, 5*time.Second)
	app.Resize(880, 550)
	app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = "dark" }) })
	app.Settle()
	app.Screenshot("plugin-market-installed-minimum")
	app.Press("id:market-open-" + id)
	app.WaitFor("id:track-0", 5*time.Second)
	app.Press("id:track-0")
	app.WaitUntil(5*time.Second, func() bool {
		track := m.playing.GetUntracked().track
		return track != nil && track.Source == id && m.player.Ready()
	})
	app.Press("id:nav-plugin-market")
	signOut := "id:market-sign-out-" + id
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(signOut) && app.Enabled(signOut) })
	app.Press(signOut)
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(signIn) && app.Enabled(signIn) })
}
