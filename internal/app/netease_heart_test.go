package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
	"github.com/nsgo-dev/nsgo/tsx"
	"yoho-music/internal/music"
	"yoho-music/internal/plugin"
)

// Heart mode advances its server cursor and filters repeated or unplayable songs.
func TestNetEaseHeartContinuation(t *testing.T) {
	var requests atomic.Int32
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		if req.URL.Path != "/eapi/playmode/intelligence/list" {
			return "", nil
		}
		step := requests.Add(1)
		want := []string{"42", "43", "44"}
		if step > int32(len(want)) || fields["songId"] != want[step-1] {
			return "", fmt.Errorf("heart cursor at step %d: %v", step, fields["songId"])
		}
		switch step {
		case 1:
			return `{"code":200,"data":[null,{"songInfo":null},{"songInfo":{"id":42}},{"songInfo":{"id":43}},{"songInfo":{"id":43}}]}`, nil
		case 2:
			return `{"code":200,"data":[{"songInfo":{"id":42}},{"songInfo":{"id":43}},{"songInfo":{"id":44}}]}`, nil
		default:
			return `{"code":200,"data":[{"songInfo":{"id":45}},{"songInfo":{"id":46}}]}`, nil
		}
	})
	netEaseHeartCall(t, app, func() error {
		for _, want := range [][]string{{"42", "43"}, {"44"}, {"46"}} {
			tracks, err := source.StationNext("HEART")
			if err != nil {
				return err
			}
			if len(tracks) != len(want) {
				return fmt.Errorf("heart tracks: got %+v, want %v", tracks, want)
			}
			for i, id := range want {
				if tracks[i].ID != id || tracks[i].Unavailable || tracks[i].Name != "Heart track "+id || tracks[i].Source != plugin.NetEaseSourceID {
					return fmt.Errorf("heart track %d: %+v", i, tracks[i])
				}
			}
		}
		return nil
	})
	if requests.Load() != 3 {
		t.Fatalf("heart made %d recommendation requests", requests.Load())
	}
}

// Empty and failed intelligence responses must never produce substitute tracks.
func TestNetEaseHeartUnavailable(t *testing.T) {
	for _, sample := range []struct {
		name, body string
		failure    error
		signIn     bool
	}{
		{name: "empty", body: `{"code":200,"data":[]}`},
		{name: "missing list", body: `{"code":200}`},
		{name: "malformed list", body: `{"code":200,"data":{}}`},
		{name: "malformed song", body: `{"code":200,"data":[{"songInfo":{"id":"invalid"}}]}`},
		{name: "removed songs", body: `{"code":200,"data":[null,{"songInfo":null}]}`},
		{name: "only seed", body: `{"code":200,"data":[{"songInfo":{"id":42}}]}`},
		{name: "unplayable", body: `{"code":200,"data":[{"songInfo":{"id":45}}]}`},
		{name: "service error", body: `{"code":500,"message":"心动推荐暂不可用"}`},
		{name: "network error", failure: errors.New("fixture offline")},
		{name: "expired session", body: `{"code":301}`, signIn: true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			var requests atomic.Int32
			app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
				if req.URL.Path == "/eapi/playmode/intelligence/list" {
					requests.Add(1)
					return sample.body, sample.failure
				}
				return "", nil
			})
			netEaseHeartCall(t, app, func() error {
				tracks, err := source.StationNext("HEART")
				if err == nil || len(tracks) != 0 {
					return fmt.Errorf("unavailable heart returned tracks %+v, error %v", tracks, err)
				}
				if sample.signIn && !errors.Is(err, music.ErrSignIn) {
					return fmt.Errorf("expired heart session: %w", err)
				}
				return nil
			})
			if count := requests.Load(); count < 1 || count > 3 {
				t.Fatalf("heart request count %d should be bounded", count)
			}
		})
	}
}

// Playlist lookup paginates and uses only the current account's liked playlist.
func TestNetEaseHeartSeed(t *testing.T) {
	for _, sample := range []struct {
		name, playlists, detail string
		unavailable             bool
	}{
		{name: "second page"},
		{name: "empty liked songs", detail: `{"code":200,"playlist":{"id":86,"specialType":5,"creator":{"userId":123},"trackIds":[],"tracks":[]}}`, unavailable: true},
		{name: "wrong owner", detail: `{"code":200,"playlist":{"id":86,"specialType":5,"creator":{"userId":321},"trackIds":[{"id":42}]}}`, unavailable: true},
		{name: "wrong playlist", detail: `{"code":200,"playlist":{"id":87,"specialType":5,"creator":{"userId":123},"trackIds":[{"id":42}]}}`, unavailable: true},
		{name: "ordinary playlist", detail: `{"code":200,"playlist":{"id":86,"specialType":0,"creator":{"userId":123},"trackIds":[{"id":42}]}}`, unavailable: true},
		{name: "unplayable seed", detail: `{"code":200,"playlist":{"id":86,"specialType":5,"creator":{"userId":123},"trackIds":[{"id":45}],"tracks":[{"id":45}]}}`, unavailable: true},
		{name: "missing liked playlist", playlists: `{"code":200,"playlist":[{"id":85,"specialType":5,"creator":{"userId":321}}],"more":false}`, unavailable: true},
		{name: "stalled playlist pages", playlists: `{"code":200,"playlist":[{"id":85,"specialType":5,"creator":{"userId":321}}],"more":true}`, unavailable: true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			var pages, recommendations atomic.Int32
			app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
				switch req.URL.Path {
				case "/api/user/playlist":
					page := pages.Add(1)
					if sample.playlists != "" {
						return sample.playlists, nil
					}
					if sample.name == "second page" && page == 1 {
						return `{"code":200,"playlist":[{"id":85,"specialType":5,"creator":{"userId":321}}],"more":true}`, nil
					}
					if sample.name == "second page" && req.URL.Query().Get("offset") != "1" {
						return "", errors.New("playlist pagination did not advance by the returned rows")
					}
				case "/api/v6/playlist/detail":
					return sample.detail, nil
				case "/eapi/playmode/intelligence/list":
					recommendations.Add(1)
					return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
				}
				return "", nil
			})
			netEaseHeartCall(t, app, func() error {
				tracks, err := source.StationNext("HEART")
				if sample.unavailable {
					if !errors.Is(err, music.ErrUnavailable) || len(tracks) != 0 || recommendations.Load() != 0 {
						return fmt.Errorf("invalid seed returned %+v, error %v, heart requests %d", tracks, err, recommendations.Load())
					}
					return nil
				}
				if err != nil || len(tracks) != 2 || tracks[0].ID != "42" || tracks[1].ID != "43" {
					return fmt.Errorf("paginated seed returned %+v, error %v", tracks, err)
				}
				return nil
			})
			if pages.Load() > 3 {
				t.Fatalf("playlist lookup did not stop at repeated pages: %d", pages.Load())
			}
		})
	}
}

// A failed batch remains retryable without losing its cursor or replaying the seed.
func TestNetEaseHeartRetry(t *testing.T) {
	var requests atomic.Int32
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		if req.URL.Path != "/eapi/playmode/intelligence/list" {
			return "", nil
		}
		step := requests.Add(1)
		if step == 1 {
			return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
		}
		if fields["songId"] != "43" {
			return "", errors.New("heart retry lost its last successful cursor")
		}
		if step == 2 {
			return `{"code":500,"message":"retry later"}`, nil
		}
		return `{"code":200,"data":[{"songInfo":{"id":44}}]}`, nil
	})
	netEaseHeartCall(t, app, func() error {
		if _, err := source.StationNext("HEART"); err != nil {
			return err
		}
		if tracks, err := source.StationNext("HEART"); err == nil || len(tracks) != 0 {
			return fmt.Errorf("failed continuation returned %+v, error %v", tracks, err)
		}
		tracks, err := source.StationNext("HEART")
		if err != nil || len(tracks) != 1 || tracks[0].ID != "44" {
			return fmt.Errorf("retried heart returned %+v, error %v", tracks, err)
		}
		return nil
	})
}

// Duplicate batches advance through server cursors within a bounded request count.
func TestNetEaseHeartRepeatedBatches(t *testing.T) {
	var requests atomic.Int32
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		if req.URL.Path != "/eapi/playmode/intelligence/list" {
			return "", nil
		}
		switch requests.Add(1) {
		case 1:
			return `{"code":200,"data":[{"songInfo":{"id":43}},{"songInfo":{"id":44}}]}`, nil
		case 2:
			if fields["songId"] != "44" {
				return "", errors.New("heart did not retain the previous batch cursor")
			}
			return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
		case 3:
			if fields["songId"] != "43" {
				return "", errors.New("heart did not advance over its repeated batch")
			}
			return `{"code":200,"data":[{"songInfo":{"id":46}}]}`, nil
		default:
			return `{"code":200,"data":[{"songInfo":{"id":46}}]}`, nil
		}
	})
	netEaseHeartCall(t, app, func() error {
		if _, err := source.StationNext("HEART"); err != nil {
			return err
		}
		tracks, err := source.StationNext("HEART")
		if err != nil || len(tracks) != 1 || tracks[0].ID != "46" {
			return fmt.Errorf("heart failed to advance beyond repeated songs: %+v, %v", tracks, err)
		}
		if tracks, err := source.StationNext("HEART"); !errors.Is(err, music.ErrUnavailable) || len(tracks) != 0 {
			return fmt.Errorf("stalled heart returned %+v, %v", tracks, err)
		}
		return nil
	})
	if requests.Load() != 4 {
		t.Fatalf("stalled cursor kept requesting songs: %d", requests.Load())
	}
}

// Concurrent calls share the same in-flight heart recommendation batch.
func TestNetEaseHeartConcurrentRequests(t *testing.T) {
	var requests atomic.Int32
	release := make(chan struct{})
	defer close(release)
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		if req.URL.Path != "/eapi/playmode/intelligence/list" {
			return "", nil
		}
		requests.Add(1)
		select {
		case <-release:
		case <-req.Context().Done():
			return "", req.Context().Err()
		}
		return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
	})
	type result struct {
		tracks []music.Track
		err    error
	}
	results := make(chan result, 2)
	for range 2 {
		go func() {
			tracks, err := source.StationNext("HEART")
			results <- result{tracks, err}
		}()
	}
	app.WaitUntil(5*time.Second, func() bool { return requests.Load() > 0 })
	app.Settle()
	release <- struct{}{}
	app.WaitUntil(5*time.Second, func() bool { return len(results) == 2 })
	for range 2 {
		result := <-results
		if result.err != nil || len(result.tracks) != 2 || result.tracks[0].ID != "42" || result.tracks[1].ID != "43" {
			t.Fatalf("concurrent heart result: %+v, %v", result.tracks, result.err)
		}
	}
	if requests.Load() != 1 {
		t.Fatalf("concurrent heart calls issued %d requests", requests.Load())
	}
}

// Signing in again starts a fresh heart cursor, including for the same account.
func TestNetEaseHeartSessionReset(t *testing.T) {
	var requests atomic.Int32
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		switch req.URL.Path {
		case "/eapi/login/qrcode/unikey":
			return `{"code":200,"unikey":"heart-qr"}`, nil
		case "/eapi/login/qrcode/client/login":
			return `{"code":803,"cookie":"MUSIC_U=heart-session"}`, nil
		}
		if req.URL.Path != "/eapi/playmode/intelligence/list" {
			return "", nil
		}
		requests.Add(1)
		if fields["songId"] != "42" {
			return "", errors.New("heart retained a signed-out session's cursor")
		}
		return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
	})
	netEaseHeartCall(t, app, func() error {
		if _, err := source.StationNext("HEART"); err != nil {
			return err
		}
		if err := source.SignOut(); err != nil {
			return err
		}
		if tracks, err := source.StationNext("HEART"); !errors.Is(err, music.ErrSignIn) || len(tracks) != 0 {
			return fmt.Errorf("signed-out heart returned %+v, %v", tracks, err)
		}
		return nil
	})
	for item := range app.Secrets.Items() {
		t.Fatalf("sign-out retained a secret for %s", item.Account)
	}
	if requests.Load() != 1 {
		t.Fatalf("signed-out heart requested another batch: %d", requests.Load())
	}
	prompts := make(chan music.SignInPrompt, 8)
	result := make(chan error, 1)
	go func() {
		_, err := source.SignInWithPrompt(context.Background(), func(prompt music.SignInPrompt) { prompts <- prompt })
		result <- err
	}()
	app.WaitUntil(5*time.Second, func() bool { return len(prompts) >= 2 || len(result) > 0 })
	app.Advance(2 * time.Second)
	app.WaitUntil(5*time.Second, func() bool { return len(result) > 0 })
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	netEaseHeartCall(t, app, func() error {
		tracks, err := source.StationNext("HEART")
		if err != nil || len(tracks) != 2 || tracks[0].ID != "42" || tracks[1].ID != "43" {
			return fmt.Errorf("new login retained the old heart progress: %+v, %v", tracks, err)
		}
		return nil
	})
}

// A recommendation completed after sign-out cannot return the old account's songs.
func TestNetEaseHeartSignOutDuringRequest(t *testing.T) {
	var requests atomic.Int32
	release := make(chan struct{})
	defer close(release)
	app, source := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		if req.URL.Path != "/eapi/playmode/intelligence/list" {
			return "", nil
		}
		requests.Add(1)
		select {
		case <-release:
		case <-req.Context().Done():
			return "", req.Context().Err()
		}
		return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
	})
	result := make(chan error, 1)
	go func() {
		tracks, err := source.StationNext("HEART")
		if !errors.Is(err, music.ErrSignIn) || len(tracks) != 0 {
			err = fmt.Errorf("stale heart returned %+v, %v", tracks, err)
		} else {
			err = nil
		}
		result <- err
	}()
	app.WaitUntil(5*time.Second, func() bool { return requests.Load() > 0 })
	netEaseHeartCall(t, app, source.SignOut)
	release <- struct{}{}
	app.WaitUntil(5*time.Second, func() bool { return len(result) > 0 })
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

// Clicking the discovery card requests heart music and keeps the current page.
func TestNetEaseHeartCardPlayback(t *testing.T) {
	var requests atomic.Int32
	var m *model
	app, _ := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		if req.URL.Path == "/eapi/playmode/intelligence/list" {
			requests.Add(1)
			return `{"code":200,"data":[]}`, nil
		}
		return "", nil
	}, func(cx *core.Scope, source *plugin.NSGoSource) core.Node {
		m = newModel(cx, "")
		m.plugins.Set([]*plugin.NSGoSource{source})
		m.source.Set(plugin.NetEaseSourceID)
		m.setAccount(plugin.NetEaseSourceID, pluginAccount{source: source, account: &music.Account{ID: "123"}})
		return m.root(cx)
	})
	app.WaitFor("label:心动模式", 5*time.Second)
	if requests.Load() != 0 {
		t.Fatal("showing the heart card consumed recommendations")
	}
	app.Press("label:心动模式")
	if current := m.page.GetUntracked(); current != (route{page: "home"}) {
		t.Fatalf("heart card opened an intermediate page: %+v", current)
	}
	app.WaitUntil(5*time.Second, func() bool { return m.error.GetUntracked() != "" })
	if requests.Load() != 1 || m.player.Radio() != "" || len(m.player.Queue()) != 0 {
		t.Fatalf("empty heart started playback or made extra requests: %d, %+v", requests.Load(), m.player.Queue())
	}
	app.Screenshot("netease-heart-unavailable")
	app.Post(func() { m.navigate(route{page: "browse", id: "for-you"}) })
	app.WaitUntil(5*time.Second, func() bool { return len(app.Query("label:心动模式")) == 2 })
	for i := range 2 {
		app.Post(func() { m.error.Set("") })
		app.Settle()
		app.Press(fmt.Sprintf("label:心动模式 >> nth=%d", i))
		if current := m.page.GetUntracked(); current != (route{page: "browse", id: "for-you"}) {
			t.Fatalf("for-you heart card opened an intermediate page: %+v", current)
		}
		app.WaitUntil(5*time.Second, func() bool { return m.error.GetUntracked() != "" })
	}
	if requests.Load() != 3 {
		t.Fatalf("heart cover and title did not both request playback: %d", requests.Load())
	}
}

// Starting heart playback attaches the station feed and refills as the user skips.
func TestNetEaseHeartContinuousPlayback(t *testing.T) {
	previousClient := http.DefaultClient
	http.DefaultClient = &http.Client{Transport: netEaseHeartTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "m701.music.126.net" {
			return nil, fmt.Errorf("unexpected heart media host %s", req.URL.Host)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(testAudio)), Request: req}, nil
	})}
	t.Cleanup(func() { http.DefaultClient = previousClient })
	var requests atomic.Int32
	var m *model
	app, _ := netEaseHeartSource(t, func(req *http.Request, fields map[string]any) (string, error) {
		switch req.URL.Path {
		case "/eapi/playmode/intelligence/list":
			switch requests.Add(1) {
			case 1:
				return `{"code":200,"data":[{"songInfo":{"id":43}}]}`, nil
			case 2:
				return `{"code":200,"data":[{"songInfo":{"id":44}}]}`, nil
			case 3:
				return `{"code":200,"data":[{"songInfo":{"id":46}}]}`, nil
			default:
				return `{"code":200,"data":[]}`, nil
			}
		case "/eapi/song/enhance/player/url/v1":
			var ids []json.Number
			if err := json.Unmarshal([]byte(fmt.Sprint(fields["ids"])), &ids); err != nil || len(ids) != 1 {
				return "", errors.New("invalid heart stream request")
			}
			return fmt.Sprintf(`{"code":200,"data":[{"id":%s,"url":"https://m701.music.126.net/%s.flac","type":"flac","level":"exhigh"}]}`, ids[0], ids[0]), nil
		case "/eapi/song/lyric/v1":
			return `{"code":200,"lrc":{"lyric":"[00:01.00]Heart fixture"}}`, nil
		case "/eapi/song/like/get":
			return `{"code":200,"ids":[42]}`, nil
		}
		return "", nil
	}, func(cx *core.Scope, source *plugin.NSGoSource) core.Node {
		m = newModel(cx, "")
		m.plugins.Set([]*plugin.NSGoSource{source})
		m.source.Set(plugin.NetEaseSourceID)
		m.setAccount(plugin.NetEaseSourceID, pluginAccount{source: source, account: &music.Account{ID: "123"}})
		return m.root(cx)
	})
	app.WaitFor("label:心动模式", 5*time.Second)
	app.Screenshot("netease-heart-card")
	app.Press("label:心动模式")
	if current := m.page.GetUntracked(); current != (route{page: "home"}) {
		t.Fatalf("heart card opened an intermediate page: %+v", current)
	}
	app.WaitUntil(5*time.Second, func() bool { return m.player.Ready() && len(m.player.Queue()) == 3 })
	if m.player.Radio() == "" || m.player.Current().ID != "42" {
		t.Fatalf("heart did not start its radio queue: %+v", m.player.Queue())
	}
	app.Screenshot("netease-heart-playing")
	app.Press("id:next")
	app.WaitUntil(5*time.Second, func() bool { return m.player.Current().ID == "43" && len(m.player.Queue()) == 4 })
	app.Press("id:next")
	app.WaitUntil(5*time.Second, func() bool { return m.player.Current().ID == "44" && m.error.GetUntracked() != "" })
	if requests.Load() != 4 || len(m.player.Queue()) != 4 {
		t.Fatalf("empty continuation added substitute tracks: %d requests, %+v", requests.Load(), m.player.Queue())
	}
}

type netEaseHeartTransport func(*http.Request) (*http.Response, error)

func (transport netEaseHeartTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func netEaseHeartSource(t *testing.T, reply func(*http.Request, map[string]any) (string, error), view ...func(*core.Scope, *plugin.NSGoSource) core.Node) (*nsgotest.App, *plugin.NSGoSource) {
	t.Helper()
	store, err := plugin.OpenNSGoStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("heart fixture source path is unavailable")
	}
	bundle, err := store.ReadInstall(filepath.Join(filepath.Dir(filename), netEasePackageFixture))
	if err != nil {
		t.Fatal(err)
	}
	provider := func(req *http.Request) (*http.Response, error) {
		fields := make(map[string]any)
		if strings.HasPrefix(req.URL.Path, "/eapi/") {
			var err error
			fields, err = netEaseDesktopFixture(req)
			if err != nil {
				return nil, err
			}
		}
		if req.URL.Path == "/eapi/playmode/intelligence/list" {
			if fields["startMusicId"] != "42" || fields["playlistId"] != "86" || fields["type"] != "fromPlayOne" || fields["count"] != float64(1) ||
				fields["id"] != nil || fields["pid"] != nil || fields["sid"] != nil || !strings.Contains(req.Header.Get("Cookie"), "MUSIC_U=heart-session") {
				return nil, errors.New("incorrect heart EAPI parameters")
			}
		}
		body, err := reply(req, fields)
		if err != nil {
			return nil, err
		}
		if body == "" {
			switch req.URL.Path {
			case "/api/v3/discovery/recommend/songs":
				body = `{"code":200,"data":{"dailySongs":[]}}`
			case "/api/toplist/detail":
				body = `{"code":200,"list":[]}`
			case "/api/personalized/playlist":
				body = `{"code":200,"result":[]}`
			case "/api/v1/play/record":
				body = `{"code":200,"weekData":[],"allData":[]}`
			case "/api/nuser/account/get":
				body = `{"code":200,"profile":{"userId":123,"nickname":"Heart Fixture"}}`
			case "/api/user/playlist":
				body = `{"code":200,"playlist":[{"id":85,"specialType":5,"creator":{"userId":321}},{"id":86,"specialType":5,"creator":{"userId":123}}],"more":false}`
			case "/api/v6/playlist/detail":
				body = `{"code":200,"playlist":{"id":86,"specialType":5,"creator":{"userId":123},"trackIds":[{"id":42}],"tracks":[{"id":42}]}}`
			case "/eapi/v3/song/detail":
				var ids []struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal([]byte(fmt.Sprint(fields["c"])), &ids); err != nil {
					return nil, err
				}
				songs, privileges := []map[string]any{}, []map[string]any{}
				for _, song := range ids {
					songs = append(songs, map[string]any{"id": song.ID, "name": "Heart track " + song.ID})
					status := 0
					if song.ID == "45" {
						status = -1
					}
					privileges = append(privileges, map[string]any{"id": song.ID, "st": status})
				}
				encoded, err := json.Marshal(map[string]any{"code": 200, "songs": songs, "privileges": privileges})
				if err != nil {
					return nil, err
				}
				body = string(encoded)
			default:
				t.Errorf("unexpected heart request %s", req.URL.Path)
				return nil, fmt.Errorf("unexpected heart request %s", req.URL.Path)
			}
		}
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	}
	var source *plugin.NSGoSource
	app := nsgotest.Start(t, tsx.WithFetcher(func(cx *core.Scope) core.Node {
		secrets := cx.Secrets()
		secrets.Service += "#" + plugin.NetEaseSourceID
		if err := secrets.Set("netease-session", "MUSIC_U=heart-session"); err != nil {
			t.Fatal(err)
		}
		var err error
		source, err = plugin.LoadNSGo(cx, bundle)
		if err != nil {
			t.Fatal(err)
		}
		cx.Cleanup(source.Close)
		if len(view) > 0 {
			return view[0](cx, source)
		}
		return core.View{}
	}, provider), nsgotest.Size(1180, 780))
	return app, source
}

func netEaseHeartCall(t *testing.T, app *nsgotest.App, call func() error) {
	t.Helper()
	result := make(chan error, 1)
	go func() { result <- call() }()
	app.WaitUntil(5*time.Second, func() bool { return len(result) > 0 })
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}
