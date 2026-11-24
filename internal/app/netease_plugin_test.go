package app

import (
	"context"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

func TestNetEaseQRLoginAndMedia(t *testing.T) {
	store, err := plugin.OpenNSGoStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := store.ReadInstall(netEasePackageFixture)
	if err != nil {
		t.Fatal(err)
	}

	const cookie = "MUSIC_U=test-session; __csrf=test-csrf"
	var polls atomic.Int32
	provider := func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "music.163.com" && req.URL.Host != "interfacepc.music.163.com" {
			return nil, errors.New("session left the NetEase API")
		}
		params := req.URL.Query()
		fields := make(map[string]any)
		if strings.HasPrefix(req.URL.Path, "/eapi/") {
			var err error
			fields, err = netEaseDesktopFixture(req)
			if err != nil {
				return nil, err
			}
		}
		headers := make(http.Header)
		var body string
		switch req.URL.Path {
		case "/eapi/login/qrcode/unikey":
			if req.Header.Get("Cookie") == "" || fields["type"] != float64(3) {
				return nil, errors.New("the QR request is missing its desktop login parameters")
			}
			body = `{"code":200,"unikey":"fixture-qr"}`
		case "/eapi/login/qrcode/client/login":
			if fields["key"] != "fixture-qr" {
				return nil, errors.New("the QR confirmation polled the wrong request")
			}
			if polls.Add(1) == 1 {
				body = `{"code":802}`
			} else {
				body = `{"code":803}`
				headers.Add("Set-Cookie", "MUSIC_U=test-session; Domain=.music.163.com; Path=/; HttpOnly")
				headers.Add("Set-Cookie", "__csrf=test-csrf; Domain=.music.163.com; Path=/")
			}
		case "/api/nuser/account/get":
			if req.Header.Get("Cookie") != cookie {
				return nil, errors.New("the confirmed QR session did not reach the account check")
			}
			body = `{"code":200,"profile":{"userId":123,"nickname":"NetEase Test Account"}}`
		case "/api/cloudsearch/pc":
			if req.Header.Get("Cookie") != cookie || params.Get("s") != "Yoho test" || params.Get("offset") != "30" {
				return nil, errors.New("incorrect authenticated search parameters")
			}
			body = `{"code":200,"result":{"songCount":80,"songs":[{"id":42,"name":"Fixture track","dt":12000,"ar":[{"id":5,"name":"Artist"}],"al":{"id":6,"name":"Album","picUrl":"http://p1.music.126.net/art.jpg"}}]}}`
		case "/eapi/song/enhance/player/url/v1":
			switch fields["ids"] {
			case "[42]":
				body = `{"code":200,"data":[{"id":42,"code":200,"url":"http://m701.music.126.net/demo.mp3","type":"mp3","br":128000}]}`
			case "[43]":
				body = `{"code":200,"data":[{"id":43,"code":-110,"url":null}]}`
			default:
				return nil, errors.New("unexpected track ID")
			}
		case "/eapi/song/lyric/v1":
			body = `{"code":200,"lrc":{"lyric":"[00:01.00]Fixture"},"yrc":{"lyric":"[1000,500](1000,500,0)Fixture"}}`
		case "/eapi/playlist/create":
			if fields["name"] != "Test playlist" || fields["csrf_token"] != "test-csrf" {
				return nil, errors.New("playlist creation lacks the account form fields")
			}
			body = `{"code":200,"playlist":{"id":888,"name":"Test playlist","userId":123}}`
		default:
			return nil, fmt.Errorf("unexpected NetEase path %q", req.URL.Path)
		}
		return &http.Response{StatusCode: http.StatusOK, Header: headers,
			Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
	}

	var cx *core.Scope
	load := func() *plugin.NSGoSource {
		source, err := plugin.LoadNSGo(cx, bundle)
		if err != nil {
			t.Fatal(err)
		}
		return source
	}
	var source *plugin.NSGoSource
	var m *model
	app := nsgotest.Start(t, tsx.WithFetcher(func(scope *core.Scope) core.Node {
		cx = scope
		source = load()
		cx.Cleanup(func() { source.Close() })
		m = newModel(scope, "")
		m.plugins.Set([]*plugin.NSGoSource{source})
		return core.View{}
	}, provider))

	results := make(chan error, 1)
	wait := func() {
		t.Helper()
		app.WaitUntil(5*time.Second, func() bool { return len(results) > 0 })
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	go func() {
		_, err := source.Search("Yoho test", music.SearchTracks, 30)
		if !errors.Is(err, music.ErrSignIn) {
			results <- fmt.Errorf("before sign-in, search returned %v", err)
			return
		}
		results <- nil
	}()
	wait()

	prompts := make(chan music.SignInPrompt, 12)
	go func() {
		account, err := source.SignInWithPrompt(context.Background(), func(p music.SignInPrompt) { prompts <- p })
		if err == nil && (account == nil || account.ID != "123") {
			err = errors.New("login did not return the verified account")
		}
		results <- err
	}()
	app.WaitUntil(5*time.Second, func() bool { return len(prompts) >= 2 || len(results) > 0 })
	if len(results) > 0 {
		wait()
		t.Fatal("login did not show a QR code")
	}
	<-prompts
	if p := <-prompts; p.QR != "https://music.163.com/login?codekey=fixture-qr" {
		t.Fatalf("wrong QR prompt: %v", p)
	}
	app.Advance(2 * time.Second)
	app.WaitUntil(5*time.Second, func() bool { return len(prompts) >= 2 || len(results) > 0 })
	if len(results) > 0 {
		wait()
		t.Fatal("login skipped the phone confirmation state")
	}
	<-prompts
	if p := <-prompts; p.QR == "" || !strings.Contains(p.Text, "确认登录") {
		t.Fatalf("missing phone confirmation prompt: %v", p)
	}
	app.Advance(2 * time.Second)
	wait()

	if len(app.Secrets.Items()) != 1 {
		t.Fatal("the plugin did not save its session in the secret store")
	}

	app.Post(func() {
		source.Close()
		source = load()
		m.plugins.Set([]*plugin.NSGoSource{source})
	})
	app.Settle()
	go func() {
		account, err := source.Account()
		if err != nil || account == nil || account.ID != "123" {
			results <- fmt.Errorf("restored account is missing: %v", err)
			return
		}
		search, err := source.Search("Yoho test", music.SearchTracks, 30)
		if err != nil || search == nil || !search.More || len(search.Tracks) != 1 ||
			search.Tracks[0].Source != plugin.NetEaseSourceID || search.Tracks[0].Album.ImageURL != "https://p1.music.126.net/art.jpg" {
			results <- fmt.Errorf("unexpected NetEase search result: %v, %v", search, err)
			return
		}
		stream, err := source.Stream("42", "high")
		if err != nil || stream == nil || stream.URL != "https://m701.music.126.net/demo.mp3" || stream.BitRate != 128000 {
			results <- fmt.Errorf("unexpected NetEase stream result: %v", err)
			return
		}
		if _, err := source.Stream("43", "standard"); !errors.Is(err, music.ErrUnavailable) {
			results <- fmt.Errorf("track without playback rights returned %v", err)
			return
		}
		lyric, err := source.Lyrics("42")
		if err != nil || lyric == nil || len(lyric.Lines) != 1 || lyric.Lines[0].Words[0].Text != "Fixture" {
			results <- fmt.Errorf("unexpected lyric result: %v, %v", lyric, err)
			return
		}
		list, err := source.CreatePlaylist("Test playlist")
		if err != nil || list == nil || !list.Mine || list.ID != "888" {
			results <- fmt.Errorf("unexpected playlist: %v, %v", list, err)
			return
		}
		results <- nil
	}()
	wait()

	const position = 3 * time.Second
	track := music.Track{ID: "42", Source: plugin.NetEaseSourceID, Duration: 12000}
	app.Post(func() {
		m.setAccount(track.Source, pluginAccount{source: source, account: &music.Account{ID: "123"}})
		m.player.Cue([]music.Track{track})
		m.player.Hold(position)
		m.refreshPlayback()
		m.changeAccount(source, false)
	})
	app.Settle()
	app.WaitUntil(5*time.Second, func() bool { return !m.accountOf(source).busy })
	if len(app.Secrets.Items()) != 0 {
		t.Fatal("the NetEase session remained in the secret store after sign-out")
	}
	if m.accountOf(source).account != nil {
		t.Fatal("sign-out did not clear the account")
	}
	if !m.player.Playing() || !m.playing.Get().playing {
		t.Fatal("sign-out interrupted the current song")
	}
	if current := m.player.Current(); current == nil || current.ID != track.ID || m.player.Position() != position {
		t.Fatal("sign-out did not retain the current track and position")
	}
	go func() {
		if _, err := source.Stream(track.ID, "high"); !errors.Is(err, music.ErrSignIn) {
			results <- fmt.Errorf("requesting a new stream after sign-out returned %v", err)
			return
		}
		results <- nil
	}()
	wait()
}

// Read the provider's request format to check the payload, instead of accepting
// any encrypted form. The test uses only synthetic session credentials.
func netEaseDesktopFixture(req *http.Request) (map[string]any, error) {
	if req.Method != http.MethodPost {
		return nil, errors.New("the desktop API expects a POST")
	}
	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	ciphertext, err := hex.DecodeString(req.Form.Get("params"))
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("invalid desktop API payload")
	}
	block, err := aes.NewCipher([]byte("e82ckenh8dichen8"))
	if err != nil {
		return nil, err
	}
	for offset := 0; offset < len(ciphertext); offset += aes.BlockSize {
		block.Decrypt(ciphertext[offset:offset+aes.BlockSize], ciphertext[offset:offset+aes.BlockSize])
	}
	parts := strings.SplitN(string(ciphertext), "-36cd479b6b5-", 3)
	if len(parts) != 3 || parts[0] != "/api/"+strings.TrimPrefix(req.URL.Path, "/eapi/") {
		return nil, errors.New("desktop API payload does not match the requested route")
	}
	var fields map[string]any
	err = json.Unmarshal([]byte(parts[1]), &fields)
	return fields, err
}
