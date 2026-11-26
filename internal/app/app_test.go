package app

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
	"yoho-music/internal/player"
)

//go:embed testdata/seek.flac
var testAudio []byte

func TestMain(m *testing.M) { nsgotest.Main(m) }

func TestEmptyLibraryAndSettings(t *testing.T) {
	dir := t.TempDir()
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node { return newModel(cx, dir).root(cx) }, nsgotest.Size(1180, 780))
	app.Find("id:add-folder")
	app.Screenshot("yoho-empty")
	app.Press("id:source-menu-trigger")
	app.Press("id:settings")
	app.Find("id:appearance")
	app.Find("id:settings-add-folder")
	for _, width := range []core.Points{1180, 880} {
		app.Resize(width, 780)
		app.Settle()
		back := assertBackOnlyHeader(t, app).Bounds()
		content := app.Find("id:settings-content").Bounds()
		insetDifference := width - 2*content.Origin.X - content.Size.Width
		if content.Origin.X != back.Origin.X || insetDifference < -1 || insetDifference > 1 {
			t.Fatalf("settings content and back button use different columns at width %v: %v, %v", width, content, back)
		}
	}
	app.Press("id:back")
	app.Find("id:add-folder")
	app.Resize(880, 580)
	app.Screenshot("yoho-empty-minimum")
}

func TestLibraryPlayback(t *testing.T) {
	dir := testLibrary(t)
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node { return newModel(cx, dir).root(cx) }, nsgotest.Size(1180, 780))
	app.Press("id:nav-songs")
	app.WaitFor("id:track-0", 5*time.Second)
	app.Press("id:track-0")
	app.Press("id:now-playing")
	app.WaitFor("text:Follow the signal", 5*time.Second)
	app.Advance(350 * time.Millisecond)
	app.Screenshot("yoho-lyrics")
	app.Press("id:stage-animation")
	app.WaitFor("id:lyric-stage", time.Second)
	app.Press("id:lyric-stage-style")
	app.Press("id:lyric-stage-style-1")
	app.Press("id:lyric-stage-sparks")
	app.WaitFor("id:lyric-stage", time.Second)
	app.Screenshot("yoho-stage-steps")
	app.Press("id:stage-wall")
	app.WaitFor("id:cover-wall", time.Second)
	app.Screenshot("yoho-cover-wall")
	app.Press("id:stage-queue")
	app.WaitFor("id:track-0", time.Second)
	app.Press("id:close-stage")
	app.Advance(250 * time.Millisecond)
	app.WaitFor("id:search-trigger", time.Second)
	app.Press("id:search-trigger")
	app.WaitFor("id:search", time.Second)
	app.Type("id:search", "Signal")
	app.Submit("id:search")
	app.WaitFor("id:track-0", time.Second)
	assertBackOnlyHeader(t, app)
	app.Key("mod+k")
	app.WaitFor("id:search", time.Second)
	app.Key("escape")
	app.Press("id:back")
	app.Press("id:nav-playlists")
	app.Press("id:new-playlist")
	app.Type("id:playlist-name", "Test mix")
	app.Press("id:save-playlist")
	app.WaitFor("id:edit-playlist", time.Second)
}

func testLibrary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	music := filepath.Join(dir, "Music")
	if err := os.Mkdir(music, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(music, "Signal.flac"), testAudio, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(music, "Signal.lrc"), []byte("[00:00.00]Follow the signal\n[00:01.00]Wait for the chorus"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeJSON(filepath.Join(dir, "library.json"), map[string][]string{"folders": {music}}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDesktopRestoresSession(t *testing.T) {
	dir := testLibrary(t)
	t.Setenv("YOHO_MUSIC_DATA_DIR", dir)
	library := local.New(dir)
	library.Rescan()
	deadline := time.Now().Add(5 * time.Second)
	for library.Scanning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	result, err := library.Search("Signal", music.SearchTracks, 0)
	if err != nil || result == nil || len(result.Tracks) != 1 {
		t.Fatalf("scan returned %+v, %v", result, err)
	}
	if err := writeJSON(filepath.Join(dir, "session.json"), player.Session{Queue: result.Tracks, Order: []int{0}, At: 0, Position: 1000}); err != nil {
		t.Fatal(err)
	}
	app := nsgotest.StartApp(t, App, nsgotest.Size(1180, 780))
	app.WaitFor("id:current-track", 5*time.Second)
	app.WaitUntil(5*time.Second, func() bool { return app.Find("id:current-track").Text == "Signal" })
	app.Press("id:play-toggle")
	app.Press("id:now-playing")
	app.WaitFor("text:Follow the signal", 5*time.Second)
}

func TestCollectionDetailCenteredLayout(t *testing.T) {
	dir := testLibrary(t)
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	assertDetailHeader := func(content string) *nsgotest.View {
		t.Helper()
		back := assertBackOnlyHeader(t, app)
		left := app.Find(content).Bounds().Origin.X
		if x := back.Bounds().Origin.X; x != left {
			t.Fatalf("back button starts at %v, content at %v", x, left)
		}
		return back
	}

	// Wait for local library scan to finish and check songs exist.
	app.Press("id:nav-songs")
	app.WaitFor("id:track-0", 5*time.Second)

	// Create a new playlist and verify the centered detail page renders with edit action even when empty.
	app.Press("id:nav-playlists")
	app.Press("id:new-playlist")
	assertDetailHeader("id:playlist-name")
	app.Type("id:playlist-name", "Centered Mix")
	app.Press("id:save-playlist")
	app.WaitFor("id:edit-playlist", 2*time.Second)
	app.Find("text:Centered Mix")

	// Add the local track to the playlist and verify Play, Shuffle, and Track 0 render and play.
	tracks, err := m.lib.Search("Signal", music.SearchTracks, 0)
	if err != nil || tracks == nil || len(tracks.Tracks) == 0 {
		t.Fatalf("expected scanned track, got %+v, %v", tracks, err)
	}
	playlists, err := m.lib.Playlists()
	if err != nil || len(playlists) == 0 {
		t.Fatalf("expected created playlist, got %+v, %v", playlists, err)
	}
	if err := m.lib.AddToPlaylist(playlists[0].ID, []string{tracks.Tracks[0].ID}); err != nil {
		t.Fatalf("AddToPlaylist: %v", err)
	}
	app.WaitFor("id:play-all", 2*time.Second)
	app.Find("id:shuffle-all")
	app.WaitFor("id:track-0", 2*time.Second)
	for _, width := range []core.Points{1180, 880, 1360} {
		app.Resize(width, 780)
		app.Settle()
		back := assertDetailHeader("id:track-0").Bounds()
		if width >= 1180 && back.Origin.Y*2+back.Size.Height != detailHeaderHeight {
			t.Fatalf("back button is not centered vertically: %v", back)
		}
	}
	app.Press("id:track-0")
	app.WaitFor("id:current-track", 2*time.Second)
	app.Press("id:back")
	app.WaitFor("id:playlist-name", time.Second)
}

func assertBackOnlyHeader(t *testing.T, app *nsgotest.App) *nsgotest.View {
	t.Helper()
	back := app.WaitFor("id:back", time.Second)
	for _, id := range []string{"id:source-menu-trigger", "id:search-trigger", "id:nav-home"} {
		if app.Exists(id) {
			t.Fatalf("detail header still has %s", id)
		}
	}
	return back
}

func TestSelectedButtonsKeepTheirFillOnHover(t *testing.T) {
	for _, theme := range []struct {
		name       string
		appearance core.Appearance
	}{
		{"light", core.AppearanceLight},
		{"dark", core.AppearanceDark},
	} {
		t.Run(theme.name, func(t *testing.T) {
			app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
				cx.SetSubtreeAppearance(theme.appearance)
				p := colors(cx)
				m := newModel(cx, "")
				installed := core.NewSignal(cx, false)
				return core.View{Style: core.Style{FlexGrow: 1, AlignItems: core.AlignStart, Gap: 8, Background: p.background}, Children: []core.Node{
					m.subNavPill(cx, "Selected tab", "text-tab", true, func() {}, func() bool { return true }),
					m.subNavIconTo(cx, lucide.Heart, "Selected icon tab", route{page: "home"}),
					marketFilter(cx, p, "Available", "active-filter", 1, installed, false),
					marketFilter(cx, p, "Installed", "inactive-filter", 0, installed, true),
					settingsSegment(p, "appearance", []settingsSegOption{{value: "system", label: "System appearance"}}, func() string { return "system" }, func(string) {}),
				}}
			})
			for _, selector := range []string{"id:text-tab", "id:nav-home", "id:active-filter", "label:System appearance"} {
				rest := app.Find(selector).Style.Background
				app.Hover(selector)
				if hover := app.Find(selector).Style.Background; hover != rest {
					t.Errorf("%s selected fill changed on hover from %08x to %08x", selector, rest, hover)
				}
			}
			rest := app.Find("id:inactive-filter").Style.Background
			app.Hover("id:inactive-filter")
			if app.Find("id:inactive-filter").Style.Background == rest {
				t.Error("unselected filter did not highlight on hover")
			}
			selected := app.Find("id:active-filter")
			fill, text := selected.Style.Background, selected.Children[0].TextStyle.Color
			app.Press("id:inactive-filter")
			app.Hover("id:inactive-filter")
			if button := app.Find("id:inactive-filter"); button.Style.Background != fill || button.Children[0].TextStyle.Color != text {
				t.Error("filter selected under the pointer did not keep its selected colors")
			}
		})
	}
}
