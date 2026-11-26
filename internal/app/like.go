package app

import (
	"errors"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
)

type likeKey struct {
	source       music.Source
	id, listener string
	episode      bool
}

type trackLike struct {
	key      likeKey
	liked    *core.Signal[bool]
	busy     *core.Signal[bool]
	known    bool
	loading  bool
	version  uint64
	motionAt time.Time
}

func (m *model) trackLike(track music.Track) *trackLike {
	src := m.sourceByID(track.Source)
	if !canLike(src, track) {
		return nil
	}
	listener := ""
	if src != m.lib {
		if account := m.accountOf(src).account; account != nil {
			listener = account.ID
		}
	}
	key := likeKey{source: src, id: track.ID, listener: listener, episode: track.ShowID != ""}
	if state := m.likes[key]; state != nil {
		if !state.known && !state.loading && !state.busy.GetUntracked() {
			m.queueLikeRead(state)
		}
		return state
	}
	state := &trackLike{key: key, liked: core.NewSignal(m.cx, false), busy: core.NewSignal(m.cx, false)}
	m.likes[key] = state
	if src == m.lib {
		liked, _ := src.IsLiked([]string{track.ID})
		state.liked.Set(liked[track.ID])
		state.known = true
	} else {
		m.queueLikeRead(state)
	}
	return state
}

// Read the visible hearts in one request per source and listener.
func (m *model) queueLikeRead(state *trackLike) {
	if state.loading || state.known || state.busy.GetUntracked() {
		return
	}
	state.loading = true
	m.pendingLikes = append(m.pendingLikes, state)
	if len(m.pendingLikes) != 1 {
		return
	}
	m.cx.Post(func() {
		pending := m.pendingLikes
		m.pendingLikes = nil
		groups := make(map[likeKey][]*trackLike)
		for _, item := range pending {
			if item.busy.GetUntracked() {
				item.loading = false
				continue
			}
			key := item.key
			key.id, key.episode = "", false
			groups[key] = append(groups[key], item)
		}
		for key, states := range groups {
			versions := make(map[*trackLike]uint64, len(states))
			ids := make([]string, 0, len(states))
			for _, item := range states {
				ids = append(ids, item.key.id)
				versions[item] = item.version
			}
			go func() {
				liked, err := key.source.IsLiked(ids)
				m.cx.Post(func() {
					for _, item := range states {
						item.loading = false
						if item.version == versions[item] && !item.busy.GetUntracked() && err == nil {
							item.liked.Set(liked[item.key.id])
							item.known = true
						}
					}
				})
			}()
		}
	})
}

func (m *model) toggleLike(track music.Track) {
	m.toggleTrackLike(track, nil)
}

func (m *model) toggleTrackLike(track music.Track, animate func(bool)) {
	state := m.trackLike(track)
	if state == nil || state.busy.GetUntracked() {
		return
	}
	src := state.key.source
	if src != m.lib && src.Can("signIn") && m.accountOf(src).account == nil {
		m.changeAccount(src, true)
		return
	}
	previous := state.liked.GetUntracked()
	known := state.known
	state.version++
	state.busy.Set(true)
	if known {
		state.motionAt = m.cx.Now()
		state.liked.Set(!previous)
		if animate != nil {
			animate(!previous)
		}
	}
	go func() {
		var err error
		if !known {
			var liked map[string]bool
			liked, err = src.IsLiked([]string{track.ID})
			if errors.Is(err, music.ErrUnsupported) {
				err = nil
			}
			previous = liked[track.ID]
		}
		if err == nil {
			send := src.Like
			if track.ShowID != "" {
				send = src.LikeEpisode
			}
			err = send(track.ID, !previous)
		}
		m.cx.Post(func() {
			state.busy.Set(false)
			state.liked.Set(previous)
			if m.sourceByID(track.Source) != src || m.trackLike(track) != state {
				return
			}
			if errors.Is(err, music.ErrSignIn) && src.Can("signIn") {
				m.changeAccount(src, true)
				return
			}
			m.report(err)
			if err != nil {
				return
			}
			state.known = true
			state.liked.Set(!previous)
			if !known {
				state.motionAt = m.cx.Now()
			}
			if !known && animate != nil {
				animate(!previous)
			}
			m.invalidate()
			if src != m.lib && m.page.GetUntracked().page == "liked" {
				m.sourceVersion.Update(func(n int) int { return n + 1 })
			}
		})
	}()
}

func (m *model) likeButton(cx *core.Scope, track music.Track, id string, size, iconSize core.Points) core.Node {
	p := colors(cx)
	current := core.NewSignal(cx, m.trackLike(track))
	cx.Effect(func() {
		state := m.trackLike(track)
		current.Set(state)
		if state != nil && state.key.source == m.lib {
			m.updated.Get()
			if !state.busy.GetUntracked() {
				liked, _ := m.lib.IsLiked([]string{track.ID})
				state.liked.Set(liked[track.ID])
			}
		}
	})
	liked := func() bool { state := current.Get(); return state != nil && state.liked.Get() }
	busy := func() bool { state := current.Get(); return state == nil || state.busy.Get() }
	scale, animate, particles := favoriteMotion(cx, id, size, iconSize)
	// A local library change can remount its rows while the burst is running.
	if state := current.GetUntracked(); state != nil && !state.motionAt.IsZero() && cx.Now().Sub(state.motionAt) < 750*time.Millisecond {
		on := state.liked.GetUntracked()
		cx.Post(func() { animate(on) })
	}
	icon := core.Canvas{
		TestID: id + "-heart",
		Style:  core.Style{Width: core.Pt(iconSize), Height: core.Pt(iconSize), Animated: &core.AnimatedStyle{Scale: scale}},
		Draw: func(d *core.Drawing, bounds core.Size) {
			d.Scale(float32(bounds.Width)/24, float32(bounds.Height)/24)
			path := (&core.Path{}).MoveTo(19, 14).CubicTo(20.5, 12.5, 22, 10.8, 22, 8.5).
				CubicTo(22, 5.46, 19.54, 3, 16.5, 3).CubicTo(14.7, 3, 13.5, 3.5, 12, 5).
				CubicTo(10.5, 3.5, 9.3, 3, 7.5, 3).CubicTo(4.46, 3, 2, 5.46, 2, 8.5).
				CubicTo(2, 10.8, 3.5, 12.5, 5, 14).LineTo(12, 21).ClosePath()
			if liked() {
				d.SetFillColor(core.RGB(0xff4d6d)).FillPath(path, core.NonZero)
			} else {
				d.SetStrokeColor(p.muted).SetLineWidth(2).StrokePath(path)
			}
		},
	}
	button := base.Button{TestID: id, Selected: liked, Busy: busy, Disabled: busy,
		LabelFunc: func() string {
			if liked() {
				return i18n.T("Unlike")
			}
			return i18n.T("Like Song")
		},
		OnPress: func(core.PressEvent) { m.toggleTrackLike(track, animate) },
		Style: core.Style{Width: core.Pt(size), Height: core.Pt(size), FlexShrink: 0, CornerRadius: size / 2,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Hover: &core.Style{Background: p.hover}},
		Children: []core.Node{icon, particles},
	}
	return button
}
