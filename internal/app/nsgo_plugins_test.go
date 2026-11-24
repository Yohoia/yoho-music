package app

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx/bundler"
	"github.com/nsgo-dev/nsgo/tsx/pkg"
	"yoho-music/internal/plugin"
)

func TestNSGoPluginSearchAndPlayback(t *testing.T) {
	var played atomic.Bool
	var forbidden atomic.Bool
	blocked := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forbidden.Store(true)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer blocked.Close()
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Fixture") != "yes" {
			http.Error(w, "missing playback header", http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/redirect.flac" {
			http.Redirect(w, r, blocked.URL, http.StatusFound)
			return
		}
		played.Store(true)
		http.ServeContent(w, r, "audio.flac", time.Time{}, bytes.NewReader(testAudio))
	}))
	defer service.Close()

	const id = "dev.yoho.fixture"
	script := fmt.Sprintf(`
export async function search({query, kind, offset}) {
  if (query === "Redirect") return {tracks: [{id: "redirect", name: "Redirect test", duration: 5000}]};
  if (query === "Sign in") throw Object.assign(new Error("Please sign in"), {code: "signIn"});
  return query === "Signal" && kind === "tracks" && offset === 0
    ? {tracks: [{id: "track", name: "Plugin Signal", duration: 5000}]}
    : {};
}
export function stream({id, quality}) {
  if (id === "track" && quality) return {url: %q, type: "flac", headers: {"X-Fixture": "yes"}};
  if (id === "redirect") return {url: %q, type: "flac", headers: {"X-Fixture": "yes"}};
}
`, service.URL+"/audio.flac", service.URL+"/redirect.flac")
	dir := t.TempDir()
	packagePath := signFixture(t, dir, id, script, pkg.Capabilities{Network: &pkg.Network{HTTP: []pkg.HTTPRule{{Host: "127.0.0.1", Paths: []string{"/audio.flac", "/redirect.flac"}}}}})
	untrusted, err := plugin.OpenNSGoStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := untrusted.ReadInstall(packagePath); !errors.Is(err, pkg.ErrUntrusted) {
		t.Fatalf("package signed by an unknown root: %v", err)
	}
	var m *model
	start := func(t *testing.T) *nsgotest.App {
		return nsgotest.Start(t, func(cx *core.Scope) core.Node {
			m = newModel(cx, dir)
			return m.root(cx)
		}, nsgotest.Size(1180, 780))
	}
	nativeID := plugin.NSGoSourcePrefix + id
	if !t.Run("install and play", func(t *testing.T) {
		app := start(t)
		app.PickFiles = func(core.FilesRequest) []core.File { return []core.File{{Path: packagePath}} }
		app.OnAlert = func(core.AlertConfig) int { return 0 }
		app.Press("id:settings")
		app.WaitUntil(time.Second, func() bool { return app.Enabled("id:plugin-nsgo-install") })
		app.Press("id:plugin-nsgo-install")
		app.WaitFor("id:source-"+nativeID, 5*time.Second)
		app.Press("id:source-" + nativeID)
		app.Type("id:search", "Signal")
		app.Submit("id:search")
		app.WaitFor("id:track-0", time.Second)
		app.Press("id:track-0")
		app.WaitUntil(5*time.Second, func() bool {
			track := m.player.Current()
			return played.Load() && m.player.Ready() && track != nil && track.Source == nativeID
		})
	}) {
		return
	}
	t.Run("reload, reject redirect and remove", func(t *testing.T) {
		app := start(t)
		app.OnAlert = func(core.AlertConfig) int { return 0 }
		app.WaitFor("id:source-"+nativeID, 5*time.Second)
		app.Press("id:source-" + nativeID)
		app.Type("id:search", "Sign in")
		app.Submit("id:search")
		app.WaitFor("text:Please sign in", time.Second)
		app.Post(func() { m.query.Set("") })
		app.Settle()
		app.Type("id:search", "Redirect")
		app.Submit("id:search")
		app.WaitFor("id:track-0", time.Second)
		app.Press("id:track-0")
		app.WaitUntil(5*time.Second, func() bool { return strings.Contains(m.player.Err(), "no media access") })
		if forbidden.Load() {
			t.Fatal("player followed a redirect outside the plugin's network rules")
		}
		app.Press("id:settings")
		app.WaitFor("id:plugin-remove-"+nativeID, time.Second)
		app.Press("id:plugin-remove-" + nativeID)
		app.WaitForGone("id:source-"+nativeID, 5*time.Second)
	})
}

func TestNSGoPluginAccount(t *testing.T) {
	const id = "dev.yoho.account-fixture"
	dir := t.TempDir()
	path := signFixture(t, dir, id, `
import { host, Secrets } from "nsgo";

export async function signIn({requestId}) {
  await host.call("yoho.signInPrompt", {requestId, text: "Scan the demo code", qr: "https://example.test/sign-in"});
  await new Promise(resolve => setTimeout(resolve, 1000));
  await host.call("yoho.signInPrompt", {requestId, text: "Completing sign-in"});
  await Secrets.set("session", "test-session");
  return account();
}
export async function account() {
  return await Secrets.get("session") ? {id: "listener", name: "Test Listener"} : null;
}
export async function signOut() { await Secrets.delete("session"); }
export function search() { return {}; }
export function stream() { return null; }
`, pkg.Capabilities{Secrets: true, Host: []string{"yoho.signInPrompt"}})
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		return newModel(cx, dir).root(cx)
	}, nsgotest.Size(1180, 780))
	app.PickFiles = func(core.FilesRequest) []core.File { return []core.File{{Path: path}} }
	app.OnAlert = func(core.AlertConfig) int { return 0 }
	app.Press("id:nav-plugin-market")
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:market-import") })
	app.Press("id:market-import")
	sourceID := plugin.NSGoSourcePrefix + id
	marketSignIn := "id:market-sign-in-" + sourceID
	start := func() {
		app.WaitUntil(5*time.Second, func() bool { return app.Exists(marketSignIn) && app.Enabled(marketSignIn) })
		app.Press(marketSignIn)
		app.Press("id:sign-in-confirm")
		app.WaitFor("text:Scan the demo code", time.Second)
		app.WaitFor("id:sign-in-qr", time.Second)
	}
	start()
	app.Press("id:sign-in-cancel")
	app.Advance(time.Second)
	app.WaitUntil(time.Second, func() bool { return app.Enabled(marketSignIn) })
	if len(app.Secrets.Items()) != 0 {
		t.Fatal("canceled sign-in saved a credential")
	}
	start()
	app.Advance(time.Second)
	app.WaitFor("text:Test Listener", time.Second)
	if len(app.Secrets.Items()) != 1 {
		t.Fatal("sign-in did not store the credential")
	}
	app.Press("id:market-install-demo")
	demoSignIn := "id:market-sign-in-" + plugin.DemoSourceID
	app.WaitUntil(5*time.Second, func() bool { return app.Exists(demoSignIn) && app.Enabled(demoSignIn) })
	app.Press(demoSignIn)
	app.Press("id:sign-in-confirm")
	demoSignOut := "id:market-sign-out-" + plugin.DemoSourceID
	app.WaitFor(demoSignOut, 5*time.Second)
	app.Press("id:market-open-" + plugin.DemoSourceID)
	app.WaitFor("id:track-0", 5*time.Second)
	app.Press("id:nav-plugin-market")
	app.WaitUntil(5*time.Second, func() bool { return app.Enabled("id:market-sign-out-" + sourceID) })
	app.Press("id:market-sign-out-" + sourceID)
	app.WaitUntil(time.Second, func() bool { return app.Enabled(marketSignIn) })
	app.Find(demoSignOut)
	if len(app.Secrets.Items()) != 0 {
		t.Fatal("sign-out did not clear the credential")
	}
}

func signFixture(t *testing.T, dir, id, script string, caps pkg.Capabilities) string {
	t.Helper()
	bundle, err := bundler.Compile(fstest.MapFS{"index.js": {Data: []byte(script)}}, "index.js")
	if err != nil {
		t.Fatal(err)
	}
	root, err := pkg.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	release, err := pkg.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	cert, err := pkg.Certify(root, release.Public().(ed25519.PublicKey), []string{id}, now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	data, err := pkg.Build(pkg.Contents{App: pkg.App{ID: id, Name: "NSGo Fixture", Version: "1.0.0", Build: 1}, Source: bundle, Capabilities: caps}, release, cert)
	if err != nil {
		t.Fatal(err)
	}
	roots := filepath.Join(dir, "nsgo-plugins", "roots")
	if err := os.MkdirAll(roots, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(roots, "test.pub"), []byte(pkg.EncodePublicKey(root.Public().(ed25519.PublicKey))), 0o600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "fixture.nsgopkg")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
