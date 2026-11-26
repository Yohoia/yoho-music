package app

import (
	"fmt"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
)

type collectionSaveKey struct {
	source             music.Source
	id, listener, kind string
}

type collectionSaveState struct {
	saved *core.Signal[bool]
	busy  *core.Signal[bool]
	known bool
}

func (m *model) collectionSave(src music.Source, kind, id string, saved bool) *collectionSaveState {
	listener := ""
	if account := m.accountOf(src).account; account != nil {
		listener = account.ID
	}
	key := collectionSaveKey{source: src, id: id, listener: listener, kind: kind}
	if state := m.savedCollections[key]; state != nil {
		return state
	}
	known := true
	switch kind {
	case "album":
		known = saved || !src.Can("savedAlbums")
	case "playlist":
		known = saved || !src.Can("playlists")
	}
	state := &collectionSaveState{saved: core.NewSignal(m.cx, saved), busy: core.NewSignal(m.cx, false), known: known}
	m.savedCollections[key] = state
	if !state.known {
		state.busy.Set(true)
		go func() {
			saved, err := readCollectionSaved(src, kind, id)
			m.cx.Post(func() {
				state.busy.Set(false)
				if err == nil {
					state.saved.Set(saved)
					state.known = true
				}
			})
		}()
	}
	return state
}

func readCollectionSaved(src music.Source, kind, id string) (bool, error) {
	if kind == "album" {
		return readAlbumSaved(src, id)
	}
	playlists, err := src.Playlists()
	if err != nil {
		return false, err
	}
	for _, list := range playlists {
		if list.ID == id && !list.Mine && !list.Fixed {
			return true, nil
		}
	}
	return false, nil
}

func readAlbumSaved(src music.Source, id string) (bool, error) {
	seen := make(map[string]bool)
	for offset := 0; offset < 10000; {
		page, err := src.SavedAlbums(offset)
		if err != nil {
			return false, err
		}
		if page == nil {
			return false, fmt.Errorf("source returned no saved album list")
		}
		previous := len(seen)
		for _, album := range page.Albums {
			seen[album.ID] = true
			if album.ID == id {
				return true, nil
			}
		}
		if !page.More {
			return false, nil
		}
		if len(seen) == previous {
			return false, fmt.Errorf("source returned an incomplete saved album list")
		}
		offset += len(page.Albums)
	}
	return false, fmt.Errorf("source returned an incomplete saved album list")
}

func (m *model) albumSaveButton(cx *core.Scope, src music.Source, album music.Album, id string, compact bool) core.Node {
	return m.collectionSaveButton(cx, src, "album", album.ID, album.Saved, id, compact)
}

func (m *model) collectionSaveButton(cx *core.Scope, src music.Source, kind, itemID string, initial bool, id string, compact bool) core.Node {
	p := colors(cx)
	stateOf := func() *collectionSaveState { return m.collectionSave(src, kind, itemID, initial) }
	current := core.NewSignal(cx, stateOf())
	cx.Effect(func() { current.Set(stateOf()) })
	saved := func() bool { return current.Get().saved.Get() }
	busy := func() bool { return current.Get().busy.Get() }
	title := func() string {
		if kind == "artist" {
			if saved() {
				return i18n.T("Unfollow")
			}
			return i18n.T("Follow")
		}
		if kind == "show" {
			if saved() {
				return i18n.T("Unsubscribe")
			}
			return i18n.T("Subscribe")
		}
		if saved() {
			if compact {
				return i18n.T("Saved")
			}
			return i18n.T("Remove from Saved")
		}
		return i18n.TC("Save", "library")
	}
	size := core.Points(14)
	style := core.Style{Direction: core.Row, Height: core.Pt(34), Padding: core.Symmetric(0, 14), CornerRadius: 17,
		Background: p.button, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6}
	tint := p.onButton
	if compact {
		size = 12
		tint = p.text
		style.Height, style.Padding, style.CornerRadius = core.Pt(30), core.Symmetric(0, 12), 15
		style.Background, style.BorderWidth, style.BorderColor = core.Transparent, 1, p.line
		style.Hover = &core.Style{Background: p.hover}
	}
	scale, animate, particles := favoriteMotion(cx, id, size+10, size)
	press := func(core.PressEvent) {
		state := current.GetUntracked()
		if state.busy.GetUntracked() {
			return
		}
		previous, known := state.saved.GetUntracked(), state.known
		state.busy.Set(true)
		if known {
			state.saved.Set(!previous)
			animate(!previous)
		}
		go func() {
			var err error
			if !known {
				previous, err = readCollectionSaved(src, kind, itemID)
			}
			if err == nil {
				switch kind {
				case "album":
					err = src.SaveAlbum(itemID, !previous)
				case "playlist":
					err = src.SavePlaylist(itemID, !previous)
				case "artist":
					err = src.FollowArtist(itemID, !previous)
				case "show":
					err = src.SubscribeShow(itemID, !previous)
				}
			}
			m.cx.Post(func() {
				state.busy.Set(false)
				state.saved.Set(previous)
				if m.sourceByID(src.Info().ID) != src || stateOf() != state {
					return
				}
				if !m.sourceActionResult(src, err) {
					return
				}
				state.saved.Set(!previous)
				state.known = true
				if !known {
					animate(!previous)
				}
				if m.page.GetUntracked().page == "savedAlbums" {
					m.sourceVersion.Update(func(n int) int { return n + 1 })
				}
			})
		}()
	}
	icon := ui.IconOf(func() *core.SVG {
		if saved() {
			return lucide.BookmarkCheck()
		}
		if kind == "artist" || kind == "show" {
			return lucide.Plus()
		}
		return lucide.Bookmark()
	}, size, func() core.Color { return tint })
	iconWrap := core.View{Style: core.Style{Width: core.Pt(size + 10), Height: core.Pt(size + 10), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
		Children: []core.Node{core.View{TestID: id + "-icon", Style: core.Style{Animated: &core.AnimatedStyle{Scale: scale}}, Children: []core.Node{icon}}, particles}}
	text := core.Text{ContentFunc: title, Style: core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, Color: tint}}
	return base.Button{LabelFunc: title, TestID: id, Selected: saved, Busy: busy, Disabled: busy,
		OnPress: press, Style: style, Children: []core.Node{iconWrap, text}}
}
