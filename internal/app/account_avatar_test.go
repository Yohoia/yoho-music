package app

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx/pkg"
	"yoho-music/internal/i18n"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

// Source changes and account refreshes select the matching avatar, and row logout signs out only its source.
func TestAccountAvatarAndSourceSignOut(t *testing.T) {
	portrait := func(tint color.RGBA) []byte {
		picture := image.NewRGBA(image.Rect(0, 0, 64, 64))
		for y := range 64 {
			for x := range 64 {
				picture.SetRGBA(x, y, tint)
			}
		}
		var data bytes.Buffer
		if err := png.Encode(&data, picture); err != nil {
			t.Fatal(err)
		}
		return data.Bytes()
	}
	netEaseImage := portrait(color.RGBA{R: 191, G: 83, B: 71, A: 255})
	qqImage := portrait(color.RGBA{R: 56, G: 139, B: 111, A: 255})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		switch r.URL.Path {
		case "/netease.png":
			w.Write(netEaseImage)
		case "/qq.png":
			w.Write(qqImage)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	makePackage := func(id, name, avatar string) *plugin.NSGoPackage {
		dir := t.TempDir()
		path := signFixture(t, dir, id, fmt.Sprintf(`
import { Secrets } from "nsgo";
export async function account() { return await Secrets.get("session") ? {id: %q, name: %q, imageUrl: %q} : null; }
export async function signOut() { await new Promise(resolve => setTimeout(resolve, 100)); await Secrets.delete("session"); }
export function search() { return {}; }
export function stream() { return null; }
export function home() { return {title: %q, sections: []}; }
export function navigation() { return [{builtin: "home"}]; }
`, id, name, avatar, name), pkg.Capabilities{Secrets: true, Network: &pkg.Network{HTTP: []pkg.HTTPRule{{Host: "127.0.0.1", Methods: []string{"GET"}}}}})
		store, err := plugin.OpenNSGoStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		bundle, err := store.ReadInstall(path)
		if err != nil {
			t.Fatal(err)
		}
		return bundle
	}
	netEasePackage := makePackage("dev.yoho.avatar-netease", "NetEase Listener", server.URL+"/netease.png")
	qqPackage := makePackage("dev.yoho.avatar-qq", "QQ Listener", server.URL+"/qq.png")
	var m *model
	var sources []*plugin.NSGoSource
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, "")
		for _, bundle := range []*plugin.NSGoPackage{netEasePackage, qqPackage} {
			secrets := cx.Secrets()
			secrets.Service += "#" + plugin.NSGoSourcePrefix + bundle.App().ID
			if err := secrets.Set("session", "avatar-fixture"); err != nil {
				t.Fatal(err)
			}
			source, err := plugin.LoadNSGo(cx, bundle)
			if err != nil {
				t.Fatal(err)
			}
			sources = append(sources, source)
			cx.Cleanup(source.Close)
		}
		m.plugins.Set(sources)
		for _, source := range sources {
			m.refreshAccount(source)
		}
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	waitAvatar := func(want []byte) {
		t.Helper()
		app.WaitUntil(5*time.Second, func() bool { return bytes.Equal(app.Find("id:account-avatar").Image.Source.Data, want) })
	}
	assertNoStandaloneSignOut := func() {
		t.Helper()
		if app.Exists("id:sign-out") || app.Exists("role:button /^"+i18n.T("Sign Out")+"$/") {
			t.Fatal("account menu retained a standalone sign-out action")
		}
		for _, username := range []string{"NetEase Listener", "QQ Listener"} {
			if app.Exists("id:source-menu >> text:" + username) {
				t.Fatal("account menu retained a username")
			}
		}
	}
	assertSelectedBackground := func(selected, other string) {
		t.Helper()
		active := app.Find("id:" + selected)
		if active.Style.Background == 0 || active.Style.Background == app.Find("id:"+other).Style.Background {
			t.Fatal("selected source does not have a distinct background")
		}
		background := active.Style.Background
		app.Hover("id:" + selected)
		if app.Find("id:"+selected).Style.Background != background {
			t.Fatal("selected source lost its background on hover")
		}
		if len(active.Children) != 2 {
			t.Fatal("source selector retained an extra selection indicator")
		}
	}
	assertMenuAnchor := func() {
		t.Helper()
		trigger := app.Find("id:source-menu-trigger").Bounds()
		menu := app.Find("id:source-menu").Bounds()
		if menu.Origin.X > trigger.Origin.X || menu.Origin.X+menu.Size.Width < trigger.Origin.X || menu.Origin.Y < trigger.Origin.Y+trigger.Size.Height {
			t.Fatalf("account menu lost its avatar anchor: %v, %v", trigger, menu)
		}
	}
	waitAvatar(yohoAvatar)
	app.Screenshot("account-avatar-default")
	app.Press("id:source-menu-trigger")
	for _, source := range sources {
		app.WaitFor("id:source-sign-out-"+source.Info().ID, 5*time.Second)
	}
	assertNoStandaloneSignOut()
	assertSelectedBackground("source-local", "source-"+sources[0].Info().ID)
	assertMenuAnchor()
	app.Press("id:source-" + sources[0].Info().ID)
	waitAvatar(netEaseImage)
	app.Press("id:source-menu-trigger")
	app.WaitUntil(5*time.Second, func() bool { return !m.accountOf(sources[0]).busy })
	waitAvatar(netEaseImage)
	assertNoStandaloneSignOut()
	assertSelectedBackground("source-"+sources[0].Info().ID, "source-local")
	app.Screenshot("account-avatar-netease-menu")
	app.Press("id:source-" + sources[1].Info().ID)
	waitAvatar(qqImage)
	app.Press("id:source-menu-trigger")
	app.WaitUntil(5*time.Second, func() bool { return app.Enabled("id:source-sign-out-" + sources[0].Info().ID) })
	app.Press("id:source-sign-out-" + sources[0].Info().ID)
	if m.source.GetUntracked() != sources[1].Info().ID {
		t.Fatal("signing out another source switched the active source")
	}
	app.Advance(100 * time.Millisecond)
	app.WaitUntil(5*time.Second, func() bool { return !m.accountOf(sources[0]).busy })
	waitAvatar(qqImage)
	app.Press("id:source-menu-trigger")
	app.WaitUntil(5*time.Second, func() bool { return !m.accountOf(sources[0]).busy && !m.accountOf(sources[1]).busy })
	if app.Exists("id:source-sign-out-" + sources[0].Info().ID) {
		t.Fatal("signed-out source retained its logout icon")
	}
	assertNoStandaloneSignOut()
	assertSelectedBackground("source-"+sources[1].Info().ID, "source-"+sources[0].Info().ID)
	logout := app.Find("id:source-sign-out-" + sources[1].Info().ID).Bounds()
	entry := app.Find("id:source-" + sources[1].Info().ID).Bounds()
	if logout.Origin.X < entry.Origin.X+entry.Size.Width {
		t.Fatalf("logout overlaps its source selector: %v, %v", logout, entry)
	}
	app.Screenshot("account-avatar-qq-menu")
	app.Resize(880, 580)
	app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = "dark" }) })
	app.Settle()
	assertNoStandaloneSignOut()
	assertSelectedBackground("source-"+sources[1].Info().ID, "source-local")
	assertMenuAnchor()
	app.Screenshot("account-avatar-qq-dark-minimum")
	app.Press("id:source-sign-out-" + sources[1].Info().ID)
	app.Advance(100 * time.Millisecond)
	app.WaitUntil(5*time.Second, func() bool { return !m.accountOf(sources[1]).busy })
	waitAvatar(yohoAvatar)
	if m.accountOf(sources[0]).account != nil || m.accountOf(sources[1]).account != nil {
		t.Fatal("logout did not clear its source account")
	}
	if len(app.Secrets.Items()) != 0 {
		t.Fatal("logout retained synthetic account credentials")
	}
	app.Press("id:source-menu-trigger")
	app.Press("id:source-local")
	waitAvatar(yohoAvatar)
	if m.source.GetUntracked() != local.ID {
		t.Fatal("avatar menu no longer selects local music")
	}
	app.Screenshot("account-avatar-signed-out")
}

// Refreshing a previously loaded account keeps its avatar until the request completes.
func TestAccountAvatarRefreshRetainsAccount(t *testing.T) {
	var m *model
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) { return "", nil },
		func(cx *core.Scope, source *plugin.NSGoSource) core.Node {
			m = newModel(cx, "")
			m.plugins.Set([]*plugin.NSGoSource{source})
			m.source.Set(plugin.NetEaseSourceID)
			m.setAccount(plugin.NetEaseSourceID, pluginAccount{source: source, account: &music.Account{ID: "123", ImageURL: "https://p1.music.126.net/avatar.png"}})
			return core.View{}
		})
	app.Post(func() {
		m.refreshAccount(source)
		if account := m.accountOf(source); account.account == nil || !strings.Contains(account.account.ImageURL, "avatar.png") || !account.busy {
			t.Error("refresh cleared the account before checking it")
		}
	})
	app.Settle()
}

// Opening the avatar menu changes neither playback visibility nor the player state.
func TestAvatarMenuDoesNotChangePlayback(t *testing.T) {
	for _, trackSource := range []string{"", local.ID, plugin.NetEaseSourceID} {
		t.Run("track="+trackSource, func(t *testing.T) {
			queued := trackSource != ""
			var m *model
			app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) { return "", nil },
				func(cx *core.Scope, source *plugin.NSGoSource) core.Node {
					m = newModel(cx, "")
					m.plugins.Set([]*plugin.NSGoSource{source})
					if queued {
						m.player.Cue([]music.Track{{ID: "42", Source: trackSource, Name: "Restored track"}})
						m.refreshPlayback()
					}
					return m.root(cx)
				})
			dock := func() *nsgotest.View { return app.Find("id:playbar-surface").Parent }
			before := dock().Style.Opacity
			if m.source.GetUntracked() != local.ID || m.lib.Can("signIn") || m.lib.Can("account") {
				t.Fatal("local music unexpectedly uses a login account")
			}
			app.Press("id:source-menu-trigger")
			if m.accountOf(source).busy {
				t.Fatal("opening the avatar menu started an account refresh")
			}
			if dock().Style.Opacity != before {
				t.Fatal("opening the avatar menu changed playback visibility")
			}
			if queued {
				if before != 1 || m.player.Current() == nil || m.player.Current().ID != "42" {
					t.Fatal("restored playback visibility depends on account refresh")
				}
			} else if before != 0 || m.player.Current() != nil {
				t.Fatal("empty player became visible")
			}
			app.Press("id:source-menu-backdrop")
			if m.stage.GetUntracked() != "" || m.player.Playing() || dock().Style.Opacity != before {
				t.Fatal("closing the avatar menu changed playback")
			}
			app.Screenshot("avatar-menu-playback-" + strings.ReplaceAll(trackSource, ":", "-"))
		})
	}
}
