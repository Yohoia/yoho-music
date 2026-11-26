package app

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
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
	app.Press("id:source-menu-trigger")
	app.Press("id:source-local")
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
	app.Press("id:source-menu-trigger")
	app.Press("id:source-local")
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

	// Wait for local library scan to finish and check songs exist.
	app.Press("id:nav-songs")
	app.WaitFor("id:track-0", 5*time.Second)

	// Create a new playlist and verify the centered detail page renders with edit action even when empty.
	app.Press("id:nav-playlists")
	app.Press("id:new-playlist")
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
	app.Press("id:track-0")
	app.WaitFor("id:current-track", 2*time.Second)
}
