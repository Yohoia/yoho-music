package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/tsx"
	"github.com/nsgo-dev/nsgo/tsx/pkg"
	"yoho-music/internal/music"
)

const NSGoSourcePrefix = "nsgo:"

const (
	callTimeout         = 30 * time.Second
	signInTimeout       = 5 * time.Minute
	signInPromptHandler = "yoho.signInPrompt"
)

var errNSGoClosed = errors.New("the NSGo music plugin has closed")

// NSGoSource translates calls from worker goroutines into an NSGo function plugin.
// Its scope lasts as long as the installed source, including across page changes.
type NSGoSource struct {
	info    music.Info
	scope   *core.Scope
	ctx     context.Context
	cancel  context.CancelFunc
	runtime *tsx.Plugin
	exports map[string]bool
	grant   pkg.Capabilities
	// Sign-in prompts are scoped to one user-initiated call on the engine.
	signInSequence uint64
	signInPrompt   func(*core.HostCall)
	build          uint64
}

// LoadNSGo starts a verified package on the engine goroutine. The host grants
// only the capabilities it displayed when installing this package.
func LoadNSGo(cx *core.Scope, verified *NSGoPackage) (*NSGoSource, error) {
	p := verified.verified
	s := &NSGoSource{
		info:    music.Info{ID: NSGoSourcePrefix + p.App.ID, Name: p.App.Name, Version: p.App.Version, Playback: music.PlaybackStream},
		scope:   cx,
		build:   p.App.Build,
		exports: make(map[string]bool),
		grant:   pkg.Capabilities{Network: p.Capabilities.Network, Secrets: p.Capabilities.Secrets},
	}
	host := core.NewHost()
	if p.App.ID == netEasePackageID && p.Capabilities.Allows(core.AccessHost, netEaseEncodeHandler, "") {
		s.grant.Host = append(s.grant.Host, netEaseEncodeHandler)
		host.Handle(netEaseEncodeHandler, encodeNetEaseCall)
	}
	if p.Capabilities.Allows(core.AccessHost, signInPromptHandler, "") {
		s.grant.Host = append(s.grant.Host, signInPromptHandler)
		host.Handle(signInPromptHandler, func(call *core.HostCall) {
			if s.signInPrompt == nil {
				call.Reject(errors.New("start sign-in from the music source"))
				return
			}
			s.signInPrompt(call)
		})
	}
	rt, err := tsx.Load(cx, p, tsx.Sandbox{Grant: &s.grant, Storage: s.info.ID, Host: host})
	if err != nil {
		return nil, err
	}
	for _, name := range rt.Exports() {
		s.exports[name] = true
	}
	if !s.Can("search") || !s.Can("stream") {
		rt.Close()
		return nil, errors.New("an NSGo music plugin must export search and stream")
	}
	if s.Can("navigation") {
		// Navigation is synchronous metadata, like the list of exports.
		var navigation []music.NavItem
		rt.Call("navigation", nil, func(value any, err error) {
			if err != nil {
				return
			}
			data, err := json.Marshal(value)
			if err == nil {
				var items []music.NavItem
				if json.Unmarshal(data, &items) == nil {
					navigation = items
				}
			}
		})
		s.info.Navigation = navigation
	}
	s.runtime = rt
	s.ctx, s.cancel = context.WithCancel(cx.Context())
	return s, nil
}

func (s *NSGoSource) Info() music.Info { return s.info }
func (s *NSGoSource) Build() uint64    { return s.build }
func (s *NSGoSource) Can(name string) bool {
	return s.exports[name] && !strings.HasPrefix(name, "remote")
}

// Close runs on the engine goroutine and wakes workers with outstanding calls.
func (s *NSGoSource) Close() {
	if s.ctx.Err() == nil {
		s.cancel()
		s.runtime.Close()
	}
}

func (s *NSGoSource) call(method string, params, out any) error {
	return s.invoke(s.ctx, callTimeout, method, params, out, nil)
}

// SignIn invokes the account contract without a prompt consumer.
func (s *NSGoSource) SignIn() (*music.Account, error) {
	return s.SignInWithPrompt(s.ctx, nil)
}

// SignInWithPrompt starts a user-initiated login. The plugin echoes the request
// ID with each prompt, so a late response cannot change a subsequent dialog.
func (s *NSGoSource) SignInWithPrompt(parent context.Context, show func(music.SignInPrompt)) (*music.Account, error) {
	var account *music.Account
	err := s.invoke(parent, signInTimeout, "signIn", nil, &account, show)
	if err == nil && account == nil {
		err = context.Canceled
	}
	return account, err
}

type answer struct {
	json []byte
	err  error
}

func (s *NSGoSource) invoke(parent context.Context, timeout time.Duration, method string, params, out any, show func(music.SignInPrompt)) error {
	if !s.Can(method) {
		return music.ErrUnsupported
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	var args []any
	if params != nil {
		// Decode arguments into plain JSON values before giving them to JS.
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		var value any
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		args = []any{value}
	}
	result := make(chan answer, 1)
	s.scope.Post(func() {
		if ctx.Err() != nil || s.ctx.Err() != nil {
			return
		}
		var signInSequence uint64
		if method == "signIn" {
			s.signInSequence++
			signInSequence = s.signInSequence
			requestID := strconv.FormatUint(signInSequence, 10)
			args = []any{map[string]any{"requestId": requestID}}
			s.signInPrompt = func(call *core.HostCall) {
				data, _ := call.Data.(map[string]any)
				if ctx.Err() != nil || s.ctx.Err() != nil || data["requestId"] != requestID {
					call.Reject(context.Canceled)
					return
				}
				text, _ := data["text"].(string)
				qr, _ := data["qr"].(string)
				if len(text) > 4096 || len(qr) > 2048 {
					call.Reject(errors.New("sign-in prompt is too long"))
					return
				}
				if show != nil {
					show(music.SignInPrompt{Text: text, QR: qr})
				}
				call.Resolve(nil)
			}
		}
		s.runtime.Call(method, args, func(value any, err error) {
			if signInSequence != 0 && signInSequence == s.signInSequence {
				s.signInPrompt = nil
			}
			if ctx.Err() != nil {
				return
			}
			var data []byte
			if err == nil && out != nil {
				data, err = json.Marshal(value)
			}
			result <- answer{json: data, err: err}
		})
	})
	select {
	case reply := <-result:
		if s.ctx.Err() != nil {
			return errNSGoClosed
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if reply.err != nil {
			return musicPluginError(reply.err)
		}
		if out != nil {
			return json.Unmarshal(reply.json, out)
		}
		return nil
	case <-s.ctx.Done():
		return errNSGoClosed
	case <-ctx.Done():
		if s.ctx.Err() != nil {
			return errNSGoClosed
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return context.Canceled
		}
		return fmt.Errorf("NSGo plugin did not answer %s: %w", method, ctx.Err())
	}
}

func (s *NSGoSource) Stream(id, quality string) (*music.Stream, error) {
	var stream music.Stream
	if err := s.call("stream", map[string]string{"id": id, "quality": quality}, &stream); err != nil {
		return nil, err
	}
	if stream.URL == "" {
		return nil, music.ErrUnavailable
	}
	if stream.URL == demoAudioURL && s.info.ID == DemoSourceID {
		stream.URL = ""
		stream.Data = demoAudio
		return &stream, nil
	}
	u, err := url.Parse(stream.URL)
	if err != nil {
		return nil, errors.New("invalid plugin stream URL")
	}
	if err := s.CheckMediaURL(u); err != nil {
		return nil, err
	}
	stream.CheckURL = s.CheckMediaURL
	return &stream, nil
}

// CheckMediaURL applies to requests made by the Go host on behalf of the plugin.
func (s *NSGoSource) CheckMediaURL(u *url.URL) error {
	if s.ctx.Err() != nil {
		return errNSGoClosed
	}
	if u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || !s.grant.Allows(core.AccessNetwork, u.String(), "GET") {
		return fmt.Errorf("plugin %s has no media access to %q", s.info.Name, u.Host)
	}
	return nil
}

func musicPluginError(err error) error {
	var kind error
	switch core.ErrorCode(err) {
	case "signIn":
		kind = music.ErrSignIn
	case "setup":
		kind = music.ErrSetup
	case "unavailable":
		kind = music.ErrUnavailable
	case "unsupported":
		kind = music.ErrUnsupported
	default:
		return err
	}
	return music.Fail(kind, err.Error())
}

var _ music.Source = (*NSGoSource)(nil)
var _ music.Recorder = (*NSGoSource)(nil)
