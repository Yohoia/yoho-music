package app

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx"
	"yoho-music/internal/plugin"
)

const netEasePackageFixture = "../plugin/bundled/netease/netease.nsgopkg"

func marketFixture(t *testing.T, netEaseDownload func()) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/plugin-market.json":
			http.ServeFile(w, r, "../../plugin-market.json")
		case "/internal/plugin/bundled/netease/netease.nsgopkg":
			if netEaseDownload != nil {
				netEaseDownload()
			}
			http.ServeFile(w, r, netEasePackageFixture)
		case "/internal/plugin/bundled/demo.nsgopkg":
			http.ServeFile(w, r, "../plugin/bundled/demo.nsgopkg")
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
		app.Press("id:nav-plugin-market")
		app.WaitUntil(5*time.Second, func() bool { return app.Exists(button) && app.Enabled(button) })
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
		app.WaitFor(source, 5*time.Second)
		if downloads.Load() != 2 {
			t.Fatal("restarting downloaded the installed plugin again")
		}
		app.Press(source)
		for _, nav := range []string{"nav-home", "nav-liked", "nav-playlists", "nav-savedAlbums", "nav-recent", "nav-shows"} {
			app.WaitFor("id:"+nav, time.Second)
		}
		app.WaitFor("text:最近一周", time.Second)
		app.WaitFor("text:私人雷达", time.Second)
		app.WaitFor("id:station-FAMILIAR", time.Second)
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
	signIn := "id:market-sign-in-" + id
	app.WaitForGone("id:market-detail-install-demo", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(detailSignIn) && app.Enabled(detailSignIn) })
	app.Press(detailSignIn)
	app.Press("id:sign-in-confirm")
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
	app.Press("id:nav-plugin-market")
	signOut := "id:market-sign-out-" + id
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(signOut) && app.Enabled(signOut) })
	app.Press(signOut)
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(signIn) && app.Enabled(signIn) })
}
