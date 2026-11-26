package app

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"yoho-music/internal/local"
	"yoho-music/internal/plugin"
)

func TestPluginMarketRemove(t *testing.T) {
	for _, active := range []bool{false, true} {
		name := "inactive source"
		if active {
			name = "active source"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			store, err := plugin.OpenNSGoStore(dir)
			if err != nil {
				t.Fatal(err)
			}
			_, filename, _, ok := runtime.Caller(0)
			if !ok {
				t.Fatal("market fixture source path is unavailable")
			}
			bundle, err := store.ReadInstall(filepath.Join(filepath.Dir(filename), "../plugin/bundled/demo.nsgopkg"))
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Install(bundle); err != nil {
				t.Fatal(err)
			}
			marketURL := marketFixture(t, nil)
			id := plugin.DemoSourceID
			path := filepath.Join(dir, "nsgo-plugins", bundle.App().ID+".nsgopkg")
			if !t.Run("remove through market", func(t *testing.T) {
				var m *model
				app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
					m = newModel(cx, dir)
					m.marketURL = marketURL
					return m.root(cx)
				}, nsgotest.Size(1180, 780))
				app.Press("id:source-menu-trigger")
				app.WaitFor("id:source-"+id, 5*time.Second)
				if active {
					app.Press("id:source-" + id)
					app.Press("id:source-menu-trigger")
				}
				app.Press("id:nav-plugin-market")
				remove := "id:market-remove-" + id
				app.WaitFor(remove, 5*time.Second)
				if active {
					app.Resize(880, 580)
					app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = "dark" }) })
					app.Settle()
				}
				app.WaitFor("id:market-install-"+plugin.NetEaseSourceID, 5*time.Second)
				if app.Exists("id:market-remove-" + plugin.NetEaseSourceID) {
					t.Fatal("an uninstalled plugin has a remove button")
				}
				app.Hover("id:market-tile-" + id)
				app.WaitUntil(time.Second, func() bool { return app.Enabled(remove) })
				app.Screenshot("plugin-market-remove-" + name)
				canceled := false
				app.OnAlert = func(core.AlertConfig) int {
					canceled = true
					return 1
				}
				app.Press(remove)
				app.WaitUntil(time.Second, func() bool { return canceled })
				app.Find(remove)
				if app.Exists("id:market-detail") {
					t.Fatal("pressing remove also opened plugin details")
				}
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("canceling removal changed the installed package: %v", err)
				}
				app.Press("id:market-card-" + id)
				detailRemove := "id:market-detail-remove-" + id
				app.WaitFor(detailRemove, time.Second)
				app.Screenshot("plugin-market-detail-remove-" + name)
				app.OnAlert = func(core.AlertConfig) int { return 0 }
				if active {
					app.Press(detailRemove)
				} else {
					app.Press("id:market-detail-close")
					app.Hover("id:market-tile-" + id)
					app.WaitUntil(time.Second, func() bool { return app.Enabled(remove) })
					app.Press(remove)
				}
				app.WaitForGone(remove, 5*time.Second)
				app.WaitFor("id:market-install-demo", time.Second)
				if active {
					app.WaitForGone(detailRemove, time.Second)
					if app.Exists("id:market-detail") {
						app.Press("id:market-detail-close")
					}
				}
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("removed package still exists: %v", err)
				}
				if m.source.GetUntracked() != local.ID || m.page.GetUntracked().page != "plugin-market" {
					t.Fatal("removing a plugin did not retain the market with local music selected")
				}
				app.Press("id:back")
				app.Press("id:source-menu-trigger")
				if app.Exists("id:source-" + id) {
					t.Fatal("removed plugin remains in the source menu")
				}
			}) {
				return
			}
			t.Run("restart", func(t *testing.T) {
				app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
					m := newModel(cx, dir)
					m.marketURL = marketURL
					return m.root(cx)
				}, nsgotest.Size(1180, 780))
				app.Press("id:source-menu-trigger")
				app.Press("id:nav-plugin-market")
				app.WaitFor("id:market-install-demo", 5*time.Second)
				if app.Exists("id:market-remove-"+id) || app.Exists("id:market-open-"+id) {
					t.Fatal("removed plugin was reloaded after restart")
				}
			})
		})
	}
}
