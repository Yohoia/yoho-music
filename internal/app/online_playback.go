package app

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/music"
)

func (m *model) episodePane(cx *core.Scope, track music.Track) core.Node {
	p := colors(cx)
	children := []core.Node{label(track.Name, 23, core.FontWeightBold, p.text)}
	if track.Published != "" {
		children = append(children, label(track.Published, 13, core.FontWeightMedium, p.muted))
	}
	if track.Description != "" {
		children = append(children, label(track.Description, 15, core.FontWeightNormal, p.text))
	}
	children = append(children, row(
		button(i18n.T("Open Podcast"), "episode-show", lucide.Radio, func() {
			m.revealSourceTrack(track.Source, route{page: "show", id: track.ShowID})
		}),
		button(i18n.T("Captions"), "episode-captions", lucide.Mic, func() { m.stage.Set("captions") }),
	))
	return core.Scroll{Style: fill, Children: []core.Node{core.View{Style: core.Style{Gap: 12, Padding: core.All(18)}, Children: children}}}
}

type trackComments struct {
	items              []music.Comment
	next, total        int
	loaded, more, busy bool
	problem            string
}

func (m *model) commentsPane(cx *core.Scope, track music.Track) core.Node {
	src := m.sourceByID(track.Source)
	if src == nil || !src.Can("comments") {
		return m.empty(cx, lucide.MessageCircle, i18n.T("No Comments"), "", nil)
	}
	state := core.NewSignalFunc(cx, trackComments{}, nil)
	draft := core.NewSignal(cx, "")
	run := func(work func(*trackComments) error, done func()) {
		current := state.GetUntracked()
		if current.busy {
			return
		}
		current.busy, current.problem = true, ""
		state.Set(current)
		go func() {
			next := current
			next.items = slices.Clone(current.items)
			err := work(&next)
			cx.Post(func() {
				if err != nil {
					next = current
					next.problem = err.Error()
					if errors.Is(err, music.ErrSignIn) && src.Can("signIn") {
						next.problem = i18n.T("Sign in to continue.")
						m.changeAccount(src, true)
					}
				} else if done != nil {
					done()
				}
				next.busy = false
				state.Set(next)
			})
		}()
	}
	load := func() {
		run(func(next *trackComments) error {
			page, err := src.Comments(track.ID, next.next)
			if err != nil {
				return err
			}
			next.loaded = true
			next.more = false
			if page == nil {
				return nil
			}
			for _, comment := range page.Comments {
				if !slices.ContainsFunc(next.items, func(existing music.Comment) bool { return existing.ID == comment.ID }) {
					next.items = append(next.items, comment)
				}
			}
			next.next += len(page.Comments)
			next.total = max(page.Total, len(next.items))
			next.more = page.More && len(page.Comments) > 0
			return nil
		}, nil)
	}
	load()
	feed := core.Dynamic(func(cx *core.Scope) core.Node {
		current := state.Get()
		p := colors(cx)
		items := make([]core.Node, 0, len(current.items)+3)
		if current.problem != "" {
			items = append(items, label(current.problem, 13, core.FontWeightNormal, p.accent))
		}
		if current.loaded {
			items = append(items, label(fmt.Sprintf("%s · %d", i18n.T("Comments"), current.total), 19, core.FontWeightBold, p.text))
		}
		if current.loaded && len(current.items) == 0 {
			items = append(items, label(i18n.T("No comments yet."), 14, core.FontWeightNormal, p.muted))
		}
		for i, comment := range current.items {
			name := label(comment.Author, 13, core.FontWeightBold, p.text)
			meta := row(m.artwork(cx, comment.AvatarURL, 28, track.Source), name, spacer())
			if src.Can("likeComment") {
				like := button(fmt.Sprint(comment.Likes), fmt.Sprintf("comment-like-%d", i), lucide.Heart, func() {
					run(func(next *trackComments) error {
						if err := src.LikeComment(track.ID, comment.ID, !comment.Liked); err != nil {
							return err
						}
						next.items[i].Liked = !comment.Liked
						if comment.Liked {
							next.items[i].Likes = max(0, comment.Likes-1)
						} else {
							next.items[i].Likes++
						}
						return nil
					}, nil)
				})
				like.Disabled = func() bool { return state.Get().busy }
				if comment.Liked {
					like.Variant = ui.ButtonPrimary
				}
				meta.Children = append(meta.Children, like)
			}
			items = append(items, core.View{Style: core.Style{Gap: 4, Padding: core.Symmetric(8, 0), Borders: core.Edges{Bottom: 1}, BorderColor: p.line},
				Children: []core.Node{meta, label(comment.Text, 14, core.FontWeightNormal, p.text)}})
		}
		if current.busy {
			items = append(items, label(i18n.T("Loading…"), 13, core.FontWeightNormal, p.muted))
		} else if current.more || !current.loaded {
			items = append(items, button(i18n.T("Load More"), "comments-more", lucide.ChevronsDown, load))
		}
		return core.View{Style: core.Style{Gap: 10, Padding: core.All(14)}, Children: items}
	})
	children := []core.Node{core.Scroll{Style: fill, Children: []core.Node{feed}}}
	if src.Can("postComment") {
		submit := func() {
			message := strings.TrimSpace(draft.Get())
			if message == "" {
				return
			}
			run(func(next *trackComments) error {
				made, err := src.PostComment(track.ID, message)
				if err != nil {
					return err
				}
				if made != nil && made.ID != "" && !slices.ContainsFunc(next.items, func(existing music.Comment) bool { return existing.ID == made.ID }) {
					next.items = append([]music.Comment{*made}, next.items...)
					next.total++
					next.next++
				}
				return nil
			}, func() { draft.Set("") })
		}
		send := button(i18n.T("Post"), "comment-post", lucide.Send, submit)
		send.Disabled = func() bool { return state.Get().busy || strings.TrimSpace(draft.Get()) == "" }
		input := ui.Input{Label: i18n.T("Write a Comment"), TestID: "comment-draft", Placeholder: i18n.T("Write a comment…"),
			Value: draft.Get, OnChange: draft.Set, OnSubmit: func(string) { submit() }, Style: fill, Disabled: func() bool { return state.Get().busy }}
		form := row(input, send)
		form.Style.Padding = core.Symmetric(8, 14)
		children = append(children, form)
	}
	return core.View{Style: fill, Children: children}
}

func (m *model) remoteDevicePicker(cx *core.Scope) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		src := m.activeSource()
		if track := m.playing.Get().track; track != nil {
			if playingSource := m.sourceByID(track.Source); playingSource != nil && playingSource.Info().Playback == music.PlaybackRemote {
				src = playingSource
			}
		}
		if src == nil || src.Info().Playback != music.PlaybackRemote || !src.Can("remoteDevices") {
			return nil
		}
		return iconButton(lucide.MonitorSpeaker, i18n.T("Playback Devices"), "remote-devices", func() {
			m.showRemoteDevices(cx, src)
		})
	})
}

func (m *model) showRemoteDevices(cx *core.Scope, src music.Source) {
	go func() {
		devices, err := src.RemoteDevices()
		cx.Post(func() {
			if err != nil {
				if errors.Is(err, music.ErrSignIn) && src.Can("signIn") {
					m.changeAccount(src, true)
				} else {
					m.report(err)
				}
				return
			}
			title := i18n.T("Playback Devices")
			cancel := i18n.T("Cancel")
			actions := []core.AlertAction{{Label: cancel, Role: core.AlertCancel}}
			choices := make(map[string]string, len(devices))
			for _, device := range devices {
				name := device.Name
				if device.Active {
					name += i18n.T(" · Active")
				}
				if _, exists := choices[name]; exists || name == cancel || name == "" {
					name = fmt.Sprintf("%s (%d)", name, len(actions))
				}
				choices[name] = device.ID
				actions = append(actions, core.AlertAction{Label: name})
			}
			var detail string
			if len(devices) == 0 {
				detail = i18n.T("No playback devices found.")
			}
			base.ShowAlertDialog(cx, base.AlertDialog{Title: title, Description: detail, Actions: actions}, func(choice string) {
				if id, ok := choices[choice]; ok && src.Can("remoteUse") {
					m.changeSource(cx, src, func() error { return src.RemoteUse(id) })
				}
			})
		})
	}()
}
