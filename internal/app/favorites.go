package app

import (
	"errors"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"yoho-music/internal/i18n"
	"yoho-music/internal/music"
)

// Keep tab selection in the route so navigation and refreshing preserve it.
func (m *model) favoritesPage(cx *core.Scope, src music.Source) core.Node {
	p := colors(cx)
	playlistTab := m.page.Get().id == "playlists" || !src.Can("savedAlbums")
	tab := func(title, id, target string, selected bool, enabled bool) core.Node {
		text := label(title, 13, core.FontWeightMedium, p.muted)
		if selected {
			text.Style.Color = p.onButton
		}
		return base.Button{Label: title, TestID: id, Selected: func() bool { return selected }, Disabled: func() bool { return !enabled },
			OnPress:   func(core.PressEvent) { m.navigate(route{page: "savedAlbums", id: target}) },
			Style:     core.Style{Height: core.Pt(32), Padding: core.Symmetric(0, 14), CornerRadius: 16, JustifyContent: core.JustifyCenter},
			StyleFunc: func() core.Style { return p.selectableButtonStyle(selected) }, Children: []core.Node{text}}
	}
	tabs := row(tab(i18n.T("Albums"), "favorites-albums", "", !playlistTab, src.Can("savedAlbums")),
		tab(i18n.T("Playlists"), "favorites-playlists", "playlists", playlistTab, src.Can("playlists")))
	tabs.Style.Gap = 6
	result := core.NewSignalFunc(cx, pageResult{}, nil)
	ctx := cx.Context()
	load := func(more bool) {
		state := result.GetUntracked()
		if state.loading {
			return
		}
		state.loading = true
		result.Set(state)
		go func() {
			data := state.data
			var err error
			if playlistTab {
				var playlists []music.Playlist
				playlists, err = src.Playlists()
				section := music.Section{}
				for _, list := range playlists {
					// Playlists returns the listener's own and saved lists.
					if !list.Mine && !list.Fixed {
						section.Playlists = append(section.Playlists, list)
					}
				}
				data = catalogData{sections: []music.Section{section}}
			} else {
				if !more {
					data = catalogData{sections: []music.Section{{}}}
				}
				data, err = loadMoreCatalog(src, route{page: "savedAlbums"}, data)
			}
			if ctx.Err() != nil {
				return
			}
			cx.Post(func() {
				if err != nil {
					data = state.data
				}
				result.Set(pageResult{data: data, err: err, done: true})
			})
		}()
	}
	load(false)
	content := core.Dynamic(func(cx *core.Scope) core.Node {
		state := result.Get()
		if !state.done {
			return paperWrap(marketLoadingSkeleton(cx, p))
		}
		var nodes []core.Node
		if state.err != nil {
			action := button(i18n.T("Retry"), "favorites-retry", lucide.RefreshCw, func() { load(false) })
			if src.Can("signIn") && errors.Is(state.err, music.ErrSignIn) {
				action = button(i18n.T("Sign In"), "catalog-sign-in", lucide.LogIn, func() { m.changeAccount(src, true) })
			}
			nodes = append(nodes, paperWrap(row(label(state.err.Error(), 13, core.FontWeightNormal, p.muted), spacer(), action)))
		}
		for _, section := range state.data.sections {
			nodes = append(nodes, m.section(cx, src, section, nil, false)...)
		}
		if len(nodes) == 0 {
			nodes = append(nodes, m.empty(cx, lucide.Bookmark, i18n.T("Nothing here yet"), "", nil))
		}
		if state.data.hasMore() {
			more := button(i18n.T("Load More"), "catalog-more", lucide.ChevronsDown, func() { load(true) })
			more.Disabled = func() bool { return result.Get().loading }
			nodes = append(nodes, paperWrap(more))
		}
		return core.View{Style: core.Style{Gap: 16}, Children: nodes}
	})
	return core.Scroll{Style: fill, Children: []core.Node{core.View{
		Style:    core.Style{Padding: core.Edges{Top: 88, Bottom: 108}, Gap: 14},
		Children: []core.Node{paperWrap(tabs), paperWrap(dashedDivider(p.line)), content},
	}}}
}
