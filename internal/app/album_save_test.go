package app

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx/pkg"
	"yoho-music/internal/i18n"
	"yoho-music/internal/plugin"
)

func TestAlbumSaveAcrossDetailAndPeek(t *testing.T) {
	dir := t.TempDir()
	path := signFixture(t, dir, "dev.yoho.album-save-fixture", `
import { Storage } from "nsgo";
export function search() { return {}; }
export function stream() { return null; }
export function album({id}) { return {album: {id, name: "Album " + id, saved: false}, tracks: []}; }
export async function savedAlbums({offset}) {
  const saved = JSON.parse(await Storage.getItem("saved") || '{"known":true}');
  const albums = [{id: "other", name: "Other"}, ...Object.keys(saved).filter(id => saved[id]).map(id => ({id, name: "Album " + id, saved: true}))];
  return {albums: albums.slice(offset, offset+1), more: offset+1 < albums.length};
}
export async function playlists() {
  const saved = JSON.parse(await Storage.getItem("playlists") || '{"saved":true}');
  return [{id:"mine", name:"My own playlist", mine:true, saved:true}, {id:"liked", name:"Liked songs", mine:true, fixed:true}, ...Object.keys(saved).filter(id => saved[id]).map(id => ({id, name:"Saved playlist"}))];
}
export async function playlist({id}) {
 const saved = JSON.parse(await Storage.getItem("playlists") || '{"saved":true}');
 return {id, name:"Saved playlist", saved: !!saved[id], tracks:[]};
}
export async function savePlaylist({id,on}) {
 const saved = JSON.parse(await Storage.getItem("playlists") || '{"saved":true}');
 saved[id] = on;
 await Storage.setItem("playlists", JSON.stringify(saved));
}
export async function saveAlbum({id, on}) {
  if (id === "refused") throw new Error("Album save refused");
  const saved = JSON.parse(await Storage.getItem("saved") || '{"known":true}');
  saved[id] = on;
  await Storage.setItem("saved", JSON.stringify(saved));
}
`, pkg.Capabilities{})
	store, err := plugin.OpenNSGoStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := store.ReadInstall(path)
	if err != nil {
		t.Fatal(err)
	}
	var m *model
	var source *plugin.NSGoSource
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, "")
		var err error
		source, err = plugin.LoadNSGo(cx, bundle)
		if err != nil {
			panic(err)
		}
		cx.Cleanup(source.Close)
		m.plugins.Set([]*plugin.NSGoSource{source})
		m.source.Set(source.Info().ID)
		m.page.Set(route{page: "album", id: "known"})
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.WaitUntil(time.Second, func() bool { return app.Exists("id:save-album") && app.Enabled("id:save-album") })
	if !app.Find("id:save-album").Props.Selected {
		t.Fatal("album detail ignored the saved album on the second page")
	}
	app.Post(func() { m.openPeek(source.Info().ID, "album", "known", "Album known", "") })
	app.WaitUntil(time.Second, func() bool { return app.Exists("id:peek-save-album") && app.Enabled("id:peek-save-album") })
	if !app.Find("id:peek-save-album").Props.Selected {
		t.Fatal("album peek ignored the saved state")
	}
	app.Press("id:peek-save-album")
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:peek-save-album") })
	if app.Find("id:peek-save-album").Props.Selected || app.Find("id:save-album").Props.Selected {
		t.Fatal("removing an album did not update detail and peek")
	}
	app.Screenshot("album-unsaved-peek")
	app.Press("id:peek-close")
	app.Press("id:save-album")
	app.Advance(500 * time.Millisecond)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:save-album") })
	if !app.Find("id:save-album").Props.Selected {
		t.Fatal("saving the album did not update its button")
	}
	app.Post(func() { m.navigate(route{page: "savedAlbums"}) })
	app.WaitFor("text:Other", time.Second)
	app.Find("id:favorites-albums")
	app.Press("id:catalog-more")
	app.WaitFor("text:Album known", time.Second)
	app.Screenshot("favorites-albums")
	app.Press("id:favorites-playlists")
	app.WaitFor("text:Saved playlist", time.Second)
	for _, absent := range []string{"text:My own playlist", "text:Liked songs", "text:Album known"} {
		if app.Exists(absent) {
			t.Fatalf("favorites retained %s", absent)
		}
	}
	app.Screenshot("favorites-playlists")
	app.Resize(880, 580)
	app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = "dark" }) })
	app.Settle()
	app.Screenshot("favorites-playlists-dark")
	app.Resize(1180, 780)
	app.Post(func() { m.navigate(route{page: "playlist", id: "saved"}) })
	app.WaitFor("id:save-playlist-to-library", time.Second)
	app.Press("id:save-playlist-to-library")
	app.Advance(500 * time.Millisecond)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:save-playlist-to-library") })
	if app.Find("id:save-playlist-to-library").Props.Selected {
		t.Fatal("playlist was not removed from saved")
	}
	app.Press("id:back")
	app.WaitFor("id:favorites-playlists", time.Second)
	app.WaitFor("text:"+i18n.T("Nothing here yet"), time.Second)
	if !app.Find("id:favorites-playlists").Props.Selected {
		t.Fatal("returning to favorites lost the playlist tab")
	}
	app.Press("id:favorites-albums")
	app.WaitFor("text:Other", time.Second)
	app.Post(func() { m.navigate(route{page: "album", id: "known"}) })
	app.WaitUntil(time.Second, func() bool { return app.Exists("id:save-album") && app.Enabled("id:save-album") })
	if !app.Find("id:save-album").Props.Selected {
		t.Fatal("reopening an album lost its saved state")
	}
	app.Post(func() { m.navigate(route{page: "album", id: "refused"}) })
	app.WaitUntil(time.Second, func() bool { return app.Exists("id:save-album") && app.Enabled("id:save-album") })
	app.Press("id:save-album")
	app.Advance(500 * time.Millisecond)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:save-album") })
	if app.Find("id:save-album").Props.Selected || m.error.GetUntracked() == "" {
		t.Fatal("failed album save did not restore state and report its error")
	}
	netEaseHeartCall(t, app, func() error {
		saved, err := readAlbumSaved(source, "known")
		if err == nil && !saved {
			t.Error("album was not saved in the source")
		}
		return err
	})
}

func TestNetEaseLikeAndAlbumSaveRequests(t *testing.T) {
	var liked, saved bool
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		switch req.URL.Path {
		case "/eapi/radio/like":
			liked = fields["like"] == true
			return `{"code":200}`, nil
		case "/eapi/album/sub":
			saved = true
			return `{"code":200}`, nil
		case "/eapi/album/unsub":
			saved = false
			return `{"code":200}`, nil
		}
		return "", nil
	})
	netEaseHeartCall(t, app, func() error {
		if err := source.Like("42", true); err != nil {
			return err
		}
		if !liked {
			return fmt.Errorf("NetEase like request did not set like=true")
		}
		if err := source.SaveAlbum("86", true); err != nil {
			return err
		}
		if !saved {
			return fmt.Errorf("NetEase save did not reach album/sub")
		}
		if err := source.SaveAlbum("86", false); err != nil {
			return err
		}
		if saved {
			return fmt.Errorf("NetEase remove did not reach album/unsub")
		}
		return nil
	})
}
