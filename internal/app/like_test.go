package app

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx/pkg"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

func TestLikeButtonPersistsAndShowsSelection(t *testing.T) {
	dir := testLibrary(t)
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.WaitFor("id:like-0", 5*time.Second)
	app.Press("id:like-0")
	app.WaitUntil(time.Second, func() bool {
		page, _ := m.lib.Liked(0)
		return len(page.Tracks) == 1
	})
	if !app.Find("id:like-0").Props.Selected {
		t.Fatal("the song was liked but its heart did not show the selected state")
	}
	app.Press("id:track-0")
	app.Press("id:now-playing")
	app.WaitFor("id:now-playing-like", time.Second)
	if !app.Find("id:now-playing-like").Props.Selected {
		t.Fatal("the player heart did not share the song's liked state")
	}
	app.Press("id:now-playing-like")
	app.WaitUntil(time.Second, func() bool {
		page, _ := m.lib.Liked(0)
		return len(page.Tracks) == 0
	})
	if app.Find("id:now-playing-like").Props.Selected {
		t.Fatal("unliking the song retained the selected heart")
	}
}

func TestOnlineLikeStateAndAnimation(t *testing.T) {
	dir := t.TempDir()
	path := signFixture(t, dir, "dev.yoho.like-fixture", `
import { Storage } from "nsgo";
export function search() { return {}; }
export function stream() { return null; }
export async function isLiked({ids}) {
  const liked = JSON.parse(await Storage.getItem("liked") || "{}");
  return Object.fromEntries(ids.map(id => [id, !!liked[id]]));
}
export async function like({id, on}) {
  await new Promise(resolve => setTimeout(resolve, 180));
  if (id === "refused") throw new Error("Like request refused");
  const liked = JSON.parse(await Storage.getItem("liked") || "{}");
  liked[id] = on;
  await Storage.setItem("liked", JSON.stringify(liked));
}
export async function likeEpisode({id, on}) { return like({id: "episode-" + id, on}); }
let commentLiked = false;
export function comments() { return {comments:[{id:"comment", author:"Listener", text:"A comment", likes:commentLiked ? 1 : 0, liked:commentLiked}], total:1}; }
export function likeComment({on}) { commentLiked=on; }

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
	var showComments *core.Signal[bool]
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, "")
		var loadErr error
		source, loadErr = plugin.LoadNSGo(cx, bundle)
		if loadErr != nil {
			panic(loadErr)
		}
		cx.Cleanup(source.Close)
		m.plugins.Set([]*plugin.NSGoSource{source})
		showComments = core.NewSignal(cx, false)
		track := music.Track{Source: source.Info().ID, ID: "song"}
		refused := music.Track{Source: source.Info().ID, ID: "refused"}
		episode := music.Track{Source: source.Info().ID, ID: "podcast", ShowID: "show"}
		return core.Dynamic(func(cx *core.Scope) core.Node {
			if showComments.Get() {
				return m.commentsPane(cx, track)
			}
			return core.View{Style: core.Style{Padding: core.All(80), Gap: 40}, Children: []core.Node{
				m.likeButton(cx, track, "first", 40, 22),
				m.likeButton(cx, track, "second", 40, 22),
				m.likeButton(cx, refused, "refused", 40, 22),
				m.likeButton(cx, episode, "episode", 40, 22),
			}}
		})
	}, nsgotest.Size(400, 420), nsgotest.ManualFrames())
	app.WaitUntil(time.Second, func() bool { return len(m.likes) == 3 && m.likes[likeKey{source: source, id: "song"}].known })
	app.Press("id:first")
	if !app.Find("id:first").Props.Selected || !app.Find("id:second").Props.Selected {
		t.Fatal("online song hearts did not share the optimistic selected state")
	}
	if app.Enabled("id:first") || app.Enabled("id:second") {
		t.Fatal("a pending like can be submitted again")
	}
	app.Advance(150 * time.Millisecond)
	if scale := app.Animated("id:first-heart").Transform.Scale; math.Abs(float64(scale)-1) < .01 {
		t.Fatal("liking did not animate the heart scale")
	}
	if len(app.Find("id:first-burst").Ops) == 0 {
		t.Fatal("liking did not draw the confetti burst")
	}
	app.Screenshot("like-burst")
	app.Advance(time.Second)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:first") })
	check := func(id string, want bool) {
		t.Helper()
		netEaseHeartCall(t, app, func() error {
			liked, err := source.IsLiked([]string{id})
			if err != nil {
				return err
			}
			if liked[id] != want {
				return fmt.Errorf("source like state for %s = %v, want %v", id, liked[id], want)
			}
			return nil
		})
	}
	check("song", true)
	app.Press("id:second")
	app.Advance(time.Second)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:second") })
	check("song", false)
	if app.Find("id:first").Props.Selected {
		t.Fatal("unliking did not update the other heart")
	}
	app.Press("id:refused")
	app.Advance(time.Second)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:refused") })
	if app.Find("id:refused").Props.Selected || m.error.GetUntracked() == "" {
		t.Fatal("a failed like did not restore the heart and report its error")
	}
	app.Press("id:episode")
	app.Advance(time.Second)
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:episode") })
	check("episode-podcast", true)
	app.Post(func() { showComments.Set(true) })
	app.WaitFor("id:comment-like-0", time.Second)
	app.Press("id:comment-like-0")
	app.WaitUntil(time.Second, func() bool { return app.Find("id:comment-like-0").Props.Selected })
	app.Advance(150 * time.Millisecond)
	if len(app.Find("id:comment-like-0-burst").Ops) == 0 {
		t.Fatal("comment like did not reuse the confetti effect")
	}
	app.Press("id:comment-like-0")
	app.WaitUntil(time.Second, func() bool { return !app.Find("id:comment-like-0").Props.Selected })
}

func TestLikeReducedMotion(t *testing.T) {
	dir := t.TempDir()
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return core.View{Style: core.Style{Padding: core.All(40)}, Children: []core.Node{
			m.likeButton(cx, music.Track{Source: "local", ID: "song"}, "like", 32, 16),
		}}
	}, nsgotest.Environment(core.Environment{ReduceMotion: true}), nsgotest.ManualFrames())
	app.Press("id:like")
	app.Advance(150 * time.Millisecond)
	if app.Animated("id:like-heart").Transform.Scale != 1 || len(app.Find("id:like-burst").Ops) != 0 {
		t.Fatal("reduced motion still played the spatial heart animation")
	}
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:like") })
	if !app.Find("id:like").Props.Selected {
		t.Fatal("reduced motion lost the selected state")
	}
}

// A fixed frame makes the upstream burst's ring and confetti visible in native screenshots.
func TestFavoriteMotionAppearance(t *testing.T) {
	dir := t.TempDir()
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		like := m.likeButton(cx, music.Track{Source: "local", ID: "preview"}, "preview", 40, 22)
		return core.View{Style: core.Style{Padding: core.All(80), Background: colors(cx).background}, Children: []core.Node{
			core.View{Style: core.Style{Width: core.Pt(100), Height: core.Pt(100), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}, Children: []core.Node{
				like,
				core.View{PointerEvents: core.PointerEventsNone, Style: core.Style{Position: core.Absolute, Left: core.Pt(6), Top: core.Pt(6), Width: core.Pt(88), Height: core.Pt(88)}, Children: []core.Node{
					core.Canvas{Style: core.Style{Width: core.Pct(100), Height: core.Pct(100)}, Draw: func(d *core.Drawing, size core.Size) { drawLikeBurst(d, size, 22, 180*time.Millisecond) }},
				}},
			}},
		}}
	}, nsgotest.Size(300, 300))
	app.Press("id:preview")
	app.WaitUntil(time.Second, func() bool { return app.Enabled("id:preview") })
	app.Screenshot("favorite-motion-light")
	app.Post(func() { m.updatePreferences(m.cx, func(p *preferences) { p.Appearance = "dark" }) })
	app.Settle()
	app.Screenshot("favorite-motion-dark")
}
