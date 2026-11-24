package player

import (
	"os"
	"path/filepath"
	"testing"

	"yoho-music/internal/music"
)

type signedOutSource struct{ nothing }

func (signedOutSource) Stream(string, string) (*music.Stream, error) {
	return nil, music.ErrSignIn
}

func TestCachedPlaybackAfterSignOut(t *testing.T) {
	data, err := os.ReadFile("testdata/seek.mp3")
	if err != nil {
		t.Fatal(err)
	}
	ch := make(chan func(), 16)
	p := New(func(fn func()) { ch <- fn })
	t.Cleanup(p.Close)
	p.Cache = NewCache(t.TempDir(), 1<<20)
	src := signedOutSource{}
	p.Source = func(string) music.Source { return src }
	p.Quality = func() string { return "high" }
	p.SetVolume(0)
	track := music.Track{ID: "42", Source: "signed-out"}
	part := filepath.Join(p.Cache.dir, "cached.part")
	if err := os.WriteFile(part, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := p.Cache.keep(track.Source, track.ID, p.Quality(), &music.Stream{Type: "mp3"}, part, int64(len(data))); !ok {
		t.Fatal("could not seed cached audio")
	}

	p.PlayQueue([]music.Track{track}, 0)
	settle(t, ch, p)
	if !p.Playing() || !p.Ready() {
		t.Fatalf("could not play cached song after sign-out: %s", p.Err())
	}

	track.ID = "uncached"
	p.PlayQueue([]music.Track{track}, 0)
	settle(t, ch, p)
	if p.Playing() || p.Ready() || p.Err() != errorText(music.ErrSignIn) {
		t.Fatalf("uncached song did not require sign-in: %s", p.Err())
	}
}
