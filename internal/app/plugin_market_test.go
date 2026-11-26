package app

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx"
	"yoho-music/internal/i18n"
	"yoho-music/internal/plugin"
)

const netEasePackageFixture = "../plugin/bundled/netease/netease.nsgopkg"

func marketFixture(t *testing.T, netEaseDownload func()) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("market fixture source path is unavailable")
	}
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/plugin-market.json":
			http.ServeFile(w, r, filepath.Join(root, "plugin-market.json"))
		case "/internal/plugin/bundled/netease/netease.nsgopkg":
			if netEaseDownload != nil {
				netEaseDownload()
			}
			http.ServeFile(w, r, filepath.Join(root, "internal/plugin/bundled/netease/netease.nsgopkg"))
		case "/internal/plugin/bundled/demo.nsgopkg":
			http.ServeFile(w, r, filepath.Join(root, "internal/plugin/bundled/demo.nsgopkg"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server.URL + "/plugin-market.json"
}

func TestMarketDownloadsNetEaseOnRequest(t *testing.T) {
	var downloads atomic.Int32
	marketURL := marketFixture(t, func() { downloads.Add(1) })
	dir := t.TempDir()
	path := filepath.Join(dir, "nsgo-plugins", "dev.yoho.music.netease.nsgopkg")
	start := func(t *testing.T, marketURL string) *nsgotest.App {
		root := tsx.WithFetcher(func(cx *core.Scope) core.Node {
			m := newModel(cx, dir)
			m.marketURL = marketURL
			m.appMenu(cx)
			return m.root(cx)
		}, func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":200,"result":[]}`)), Request: req}, nil
		})
		return nsgotest.Start(t, root, nsgotest.Size(1180, 780))
	}
	if !t.Run("browse, download and install", func(t *testing.T) {
		app := start(t, marketURL)
		button := "id:market-install-" + plugin.NetEaseSourceID
		source := "id:source-" + plugin.NetEaseSourceID
		app.Press("id:source-menu-trigger")
		app.Press("id:nav-plugin-market")
		app.WaitUntil(5*time.Second, func() bool { return app.Exists(button) && app.Enabled(button) })
		assertBackOnlyHeader(t, app)
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) || app.Exists(source) || downloads.Load() != 0 {
			t.Fatal("browsing downloaded or loaded the plugin")
		}
		denied := false
		app.OnAlert = func(core.AlertConfig) int {
			denied = true
			return 1
		}
		app.Press(button)
		app.WaitUntil(5*time.Second, func() bool { return denied && app.Enabled(button) })
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) || app.Exists(source) || downloads.Load() != 1 {
			t.Fatal("canceling installation left the downloaded plugin installed")
		}
		app.OnAlert = func(core.AlertConfig) int { return 0 }
		app.Press(button)
		app.WaitFor("id:market-remove-"+plugin.NetEaseSourceID, 5*time.Second)
		app.Press("id:back")
		app.Press("id:source-menu-trigger")
		app.WaitFor(source, 5*time.Second)
		if _, err := os.Stat(path); err != nil || downloads.Load() != 2 {
			t.Fatalf("the downloaded plugin was not installed: %v", err)
		}
	}) {
		return
	}
	t.Run("reload from disk without the market", func(t *testing.T) {
		app := start(t, "")
		source := "id:source-" + plugin.NetEaseSourceID
		app.Press("id:source-menu-trigger")
		app.WaitFor(source, 5*time.Second)
		if downloads.Load() != 2 {
			t.Fatal("restarting downloaded the installed plugin again")
		}
		app.Press(source)
		for _, nav := range []string{"nav-home", "nav-browseplaylists-hub", "nav-recent", "nav-savedAlbums", "nav-liked"} {
			app.WaitFor("id:"+nav, time.Second)
		}
		app.WaitFor("id:catalog-sign-in", time.Second)
		app.Press("id:source-menu-trigger")
		app.WaitFor(source, time.Second)
		if app.Exists("id:sign-in") {
			t.Fatal("the source menu should not offer sign-in")
		}
		app.Key("escape")
		app.Find("id:catalog-sign-in")
	})
}

func TestPluginMarket(t *testing.T) {
	dir := t.TempDir()
	marketURL := marketFixture(t, nil)
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		m.marketURL = marketURL
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.OnAlert = func(core.AlertConfig) int { return 0 }
	app.Press("id:source-menu-trigger")
	app.Press("id:nav-plugin-market")
	app.WaitUntil(5*time.Second, func() bool { return app.Enabled("id:market-install-demo") })
	app.Screenshot("plugin-market")
	id := plugin.DemoSourceID
	card := "id:market-card-" + id
	dialog := "id:market-detail"
	app.Press(card)
	app.WaitFor(dialog, time.Second)
	app.WaitFor("id:market-detail-install-demo", time.Second)
	app.Screenshot("plugin-market-detail")
	app.Key("escape")
	app.WaitForGone(dialog, time.Second)
	app.Press(card)
	app.PressAt("id:market-detail-backdrop", core.Point{X: 20, Y: 150})
	app.WaitForGone(dialog, time.Second)

	app.Press("id:market-installed")
	app.WaitForGone("id:market-install-demo", time.Second)
	app.Press("id:market-browse-all")
	app.Type("id:market-search", "no matching source")
	app.WaitForGone("id:market-install-demo", time.Second)
	app.Press("id:market-clear-search")
	app.Press(card)
	app.Press("id:market-detail-install-demo")

	detailSignIn := "id:market-detail-sign-in-" + id
	detailSignOut := "id:market-detail-sign-out-" + id
	app.WaitForGone("id:market-detail-install-demo", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(detailSignIn) && app.Enabled(detailSignIn) })
	app.Press(detailSignIn)
	app.WaitFor(detailSignOut, 5*time.Second)
	app.Press("id:market-detail-close")
	app.WaitForGone(dialog, time.Second)
	app.Resize(880, 550)
	app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = "dark" }) })
	app.Settle()
	app.Screenshot("plugin-market-installed-minimum")
	app.Press(card)
	app.WaitFor(dialog, time.Second)
	app.Screenshot("plugin-market-detail-installed-minimum")
	app.Press("id:market-detail-open-" + id)
	app.WaitFor("id:track-0", 5*time.Second)
	app.Press("id:track-0")
	app.WaitUntil(5*time.Second, func() bool {
		track := m.playing.GetUntracked().track
		return track != nil && track.Source == id && m.player.Ready()
	})
	app.Press("id:source-menu-trigger")
	app.Press("id:nav-plugin-market")
	app.WaitFor(card, 5*time.Second)
	app.Press(card)
	signOut := "id:market-detail-sign-out-" + id
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(signOut) && app.Enabled(signOut) })
	app.Press(signOut)
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(detailSignIn) && app.Enabled(detailSignIn) })
}

// Market edits preserve the card layout while updating its content and source icon.
func TestPluginMarketCardContent(t *testing.T) {
	dir := t.TempDir()
	store, err := plugin.OpenNSGoStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("market fixture source path is unavailable")
	}
	for _, relative := range []string{netEasePackageFixture, "../plugin/bundled/demo.nsgopkg"} {
		bundle, err := store.ReadInstall(filepath.Join(filepath.Dir(filename), relative))
		if err != nil {
			t.Fatal(err)
		}
		if err := store.Install(bundle); err != nil {
			t.Fatal(err)
		}
	}
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.Press("id:source-menu-trigger")
	app.Press("id:nav-plugin-market")
	card := "id:market-card-" + plugin.NetEaseSourceID
	remove := "id:market-remove-" + plugin.NetEaseSourceID
	app.WaitFor(card, 5*time.Second)
	app.WaitFor(remove, time.Second)
	for _, theme := range []struct {
		name, appearance string
		width            core.Points
	}{{"light", "light", 1180}, {"dark-minimum", "dark", 880}} {
		app.Resize(theme.width, 580)
		app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = theme.appearance }) })
		app.Settle()
		search := app.Find("id:market-search").Bounds()
		importButton := app.Find("id:market-import").Bounds()
		if search.Origin.Y+search.Size.Height < importButton.Origin.Y || importButton.Origin.Y+importButton.Size.Height < search.Origin.Y || search.Origin.X+search.Size.Width > importButton.Origin.X {
			t.Fatalf("search and import do not share a row at width %v: %v, %v", theme.width, search, importButton)
		}
		for _, absent := range []string{"id:market-manage", "text:" + i18n.T("Installed Plugins"), "text:" + i18n.T("Browse all plugins or import a signed Yoho package."), "id:market-sign-out-" + plugin.NetEaseSourceID} {
			if app.Exists(absent) {
				t.Fatalf("simplified market retained %s", absent)
			}
		}
		app.Find(card + " >> text:网易云音乐")
		app.Find("id:market-tile-" + plugin.NetEaseSourceID + " >> text:v1.0.20")
		tile := app.Find("id:market-tile-" + plugin.NetEaseSourceID).Bounds()
		demoTile := app.Find("id:market-tile-" + plugin.DemoSourceID).Bounds()
		if tile.Size.Height < 120 || tile.Origin.Y != demoTile.Origin.Y || tile.Origin.X <= demoTile.Origin.X {
			t.Fatalf("market card layout changed: %v, %v", demoTile, tile)
		}
		app.Find(card + " >> id:netease-brand-icon")
		app.Screenshot("plugin-market-cards-" + theme.name)
	}
	app.Type("id:market-search", "no matching source")
	app.WaitForGone(card, time.Second)
	app.Press("id:market-clear-search")
	app.WaitFor(card, time.Second)
	app.Press(card)
	app.WaitFor("id:market-detail >> id:netease-brand-icon", time.Second)
	app.Screenshot("plugin-market-netease-detail")
	app.Press("id:market-detail-open-" + plugin.NetEaseSourceID)
	app.WaitUntil(5*time.Second, func() bool {
		return m.source.GetUntracked() == plugin.NetEaseSourceID && m.page.GetUntracked().page == "home"
	})
	app.Press("id:source-menu-trigger")
	app.Find("id:source-" + plugin.NetEaseSourceID + " >> id:netease-brand-icon")
	app.Screenshot("plugin-market-netease-source-menu")
	if m.accountOf(m.activeSource()).account != nil {
		t.Fatal("opening an installed plugin unexpectedly signed in")
	}
}
