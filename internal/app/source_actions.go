package app

import (
	"errors"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
)

func (m *model) sourceActionResult(src music.Source, err error) bool {
	if errors.Is(err, music.ErrSignIn) && src.Can("signIn") {
		m.changeAccount(src, true)
	} else {
		m.report(err)
	}
	return err == nil
}

func (m *model) sourceCatalogActions(cx *core.Scope, src music.Source, data catalogData) []core.Node {
	if m.page.Get().page == "playlists" && src.Can("createPlaylist") {
		return []core.Node{button(i18n.T("New Playlist"), "source-new-playlist", lucide.Plus, func() {
			m.navigate(route{page: "source-playlist-edit"})
		})}
	}
	if list := data.list; list != nil {
		if list.Mine && !list.Fixed && src.Can("renamePlaylist") {
			return []core.Node{iconButton(lucide.Pencil, i18n.T("Edit Playlist"), "edit-playlist", func() {
				m.navigate(route{page: "source-playlist-edit", id: list.ID})
			})}
		}
		if !list.Mine && src.Can("savePlaylist") {
			return []core.Node{m.sourceSaveButton(cx, src, "save-playlist-to-library", list.Saved, func() error {
				return src.SavePlaylist(list.ID, !list.Saved)
			})}
		}
	}
	if album := data.album; album != nil && src.Can("saveAlbum") {
		return []core.Node{m.sourceSaveButton(cx, src, "save-album", album.Saved, func() error {
			return src.SaveAlbum(album.ID, !album.Saved)
		})}
	}
	if artist := data.artist; artist != nil && src.Can("followArtist") {
		title, icon := i18n.T("Follow"), lucide.Plus
		if artist.Following {
			title, icon = i18n.T("Unfollow"), lucide.Check
		}
		return []core.Node{button(title, "follow-artist", icon, func() {
			m.changeSource(cx, src, func() error { return src.FollowArtist(artist.ID, !artist.Following) })
		})}
	}
	return nil
}

func (m *model) sourceSaveButton(cx *core.Scope, src music.Source, id string, saved bool, apply func() error) ui.Button {
	title, icon := i18n.TC("Save", "library"), lucide.Bookmark
	if saved {
		title, icon = i18n.T("Remove from Saved"), lucide.BookmarkCheck
	}
	return button(title, id, icon, func() { m.changeSource(cx, src, apply) })
}

func (m *model) sourceTrackActions(cx *core.Scope, src music.Source, track music.Track, index int, list *music.Playlist, at core.Point) []desktopItem {
	var actions []desktopItem
	if track.ShowID == "" && src.Can("addToPlaylist") && src.Can("playlists") {
		actions = append(actions, desktopItem{label: i18n.T("Add to Playlist…"), action: func() {
			go func() {
				playlists, err := src.Playlists()
				cx.Post(func() {
					if !m.sourceActionResult(src, err) {
						return
					}
					var choices []desktopItem
					for _, target := range playlists {
						if target.Mine && !target.Fixed {
							choices = append(choices, desktopItem{label: target.Name, action: func() {
								m.changeSource(cx, src, func() error { return src.AddToPlaylist(target.ID, []string{track.ID}) })
							}})
						}
					}
					if len(choices) == 0 {
						base.ShowAlertDialog(cx, base.AlertDialog{Title: i18n.T("No editable playlists"), Description: i18n.T("Create a playlist in this source first.")}, nil)
						return
					}
					trackMenu(cx, choices, at)
				})
			}()
		}})
	}
	if list == nil || !list.Mine || list.Fixed {
		return actions
	}
	if src.Can("removeFromPlaylist") {
		actions = append(actions, desktopItem{label: i18n.T("Remove from Playlist"), action: func() {
			m.changeSource(cx, src, func() error { return src.RemoveFromPlaylist(list.ID, []string{track.ID}) })
		}})
	}
	if src.Can("movePlaylistTrack") {
		move := func(title string, to int) desktopItem {
			return desktopItem{label: title, action: func() {
				m.changeSource(cx, src, func() error { return src.MovePlaylistTrack(list.ID, index, to) })
			}}
		}
		if index > 0 {
			actions = append(actions, move(i18n.T("Move Up"), index-1))
		}
		if index+1 < len(list.Tracks) {
			actions = append(actions, move(i18n.T("Move Down"), index+1))
		}
	}
	return actions
}
