package app

import (
	"math"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"yoho-music/internal/music"
)

func TestPlaybarKeepsArtworkOnPlaybackChange(t *testing.T) {
	dir := testLibrary(t)
	artwork, err := filepath.Abs("../../assets/Icon.png")
	if err != nil {
		t.Fatal(err)
	}
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.WaitFor("id:track-0", 5*time.Second)
	app.Post(func() {
		result, _ := m.lib.Search("Signal", music.SearchTracks, 0)
		track := result.Tracks[0]
		track.Album.ImageURL = (&url.URL{Scheme: "file", Path: artwork}).String()
		m.player.Cue([]music.Track{track})
		m.refreshPlayback()
	})
	app.WaitFor("id:now-playing", time.Second)
	app.Advance(400 * time.Millisecond)
	cover := func() *nsgotest.View {
		return app.Find("id:now-playing").Children[0].Children[0]
	}
	original := cover().ID
	app.Press("id:play-toggle")
	app.Advance(100 * time.Millisecond)
	if changed := cover().ID; changed != original {
		t.Fatalf("pressing play replaced the cover view %d with %d", original, changed)
	}
	app.Press("id:play-toggle")
	app.Advance(100 * time.Millisecond)
	if changed := cover().ID; changed != original {
		t.Fatalf("pressing pause replaced the cover view %d with %d", original, changed)
	}
	app.Press("id:playbar-collapse")
	app.Advance(400 * time.Millisecond)
	app.Press("id:playbar-compact-play")
	app.Advance(100 * time.Millisecond)
	if changed := cover().ID; changed != original {
		t.Fatalf("pressing play in compact mode replaced the cover view %d with %d", original, changed)
	}
	app.Find("id:playbar-expand")
}

func TestPlaybarCrossfadesDuringResize(t *testing.T) {
	dir := testLibrary(t)
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node { return newModel(cx, dir).root(cx) },
		nsgotest.Size(1180, 780), nsgotest.ManualFrames())
	app.WaitFor("id:track-0", 5*time.Second)
	app.Press("id:track-0")
	app.WaitFor("id:playbar-collapse", time.Second)
	app.Advance(400 * time.Millisecond)
	width := app.Find("id:playbar-surface").Bounds().Size.Width

	checkMidpoint := func() {
		t.Helper()
		if layout := app.LayoutAnimation; layout == nil || layout.Update == nil {
			t.Fatal("resizing the playbar did not request a native layout animation")
		}
		app.Advance(app.LayoutAnimation.Update.Duration / 2)
		expanded := app.Animated("id:playbar-expanded-controls").Opacity
		compact := app.Animated("id:playbar-compact-controls").Opacity
		if expanded < .15 || compact < .15 || math.Abs(float64(expanded+compact-1)) > .05 {
			t.Fatalf("controls did not crossfade at the midpoint: expanded=%.2f compact=%.2f", expanded, compact)
		}
	}

	app.Press("id:playbar-collapse")
	checkMidpoint()
	app.Advance(300 * time.Millisecond)
	if got := app.Find("id:playbar-surface").Bounds().Size.Width; got >= width {
		t.Fatalf("playbar did not shrink: %v >= %v", got, width)
	}
	app.Press("id:playbar-expand")
	checkMidpoint()
	app.Advance(300 * time.Millisecond)
	if got := app.Find("id:playbar-surface").Bounds().Size.Width; got != width {
		t.Fatalf("playbar did not return to its expanded width: %v != %v", got, width)
	}
}
