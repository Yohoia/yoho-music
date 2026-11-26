package app

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
)

func (m *model) pageView(cx *core.Scope, marketSearchRef *core.Ref) core.Node {
	r := m.page.Get()
	if r.page == "settings" {
		return m.withDetailHeader(cx, m.settingsPage(cx), paperWrap)
	}
	if r.page == "plugin-market" {
		return m.withDetailHeader(cx, m.pluginMarket(cx, marketSearchRef), paperWrap)
	}
	if r.page == "playlist-edit" {
		return m.editPlaylist(cx, m.lib, r.id)
	}
	statusWrap := paperWrap
	switch r.page {
	case "album", "playlist", "show":
		statusWrap = collectionWrap
	}
	src := m.activeSource()
	if src == nil {
		return m.withDetailHeader(cx, core.View{Style: fill}, statusWrap)
	}
	if r.page == "source-playlist-edit" {
		m.sourceVersion.Get()
		return m.editPlaylist(cx, src, r.id)
	}
	if src.Info().ID == local.ID {
		m.updated.Get()
		body, err := m.catalogPage(cx, src, r)
		if err != nil {
			return m.withDetailHeader(cx, m.empty(cx, lucide.CircleAlert, err.Error(), "", nil), statusWrap)
		}
		return body
	}
	// Query plugins off the UI goroutine. The owner prevents a stale response
	// from replacing a newer page after navigation.
	m.sourceVersion.Get()
	result := core.NewSignalFunc(cx, pageResult{}, nil)
	load := func(more bool) {
		current := result.GetUntracked()
		if current.loading {
			return
		}
		current.loading, current.err = true, nil
		result.Set(current)
		go func() {
			var data catalogData
			var err error
			if more {
				data, err = loadMoreCatalog(src, r, current.data)
			} else {
				data, err = loadCatalog(src, r)
			}
			cx.Post(func() {
				if more && err != nil {
					data = current.data
				}
				result.Set(pageResult{data: data, err: err, done: true})
			})
		}()
	}
	load(false)
	return core.Dynamic(func(cx *core.Scope) core.Node {
		r := result.Get()
		if !r.done {
			return m.withDetailHeader(cx, m.empty(cx, lucide.Radio, i18n.T("Loading…"), "", nil), statusWrap)
		}
		if r.err != nil && !r.data.hasMore() {
			if errors.Is(r.err, music.ErrSignIn) && src.Can("signIn") {
				action := button(i18n.T("Sign In"), "catalog-sign-in", lucide.LogIn, func() { m.changeAccount(src, true) })
				action.Disabled = func() bool { return m.accountOf(src).busy }
				return m.withDetailHeader(cx, m.emptyWithAction(cx, lucide.LogIn, i18n.T("Sign in to continue"), "", action), statusWrap)
			}
			return m.withDetailHeader(cx, m.empty(cx, lucide.CircleAlert, r.err.Error(), "", nil), statusWrap)
		}
		var footer core.Node
		if r.data.hasMore() {
			more := button(i18n.T("Load More"), "catalog-more", lucide.ChevronsDown, func() { load(true) })
			more.Disabled = func() bool { return result.Get().loading }
			var controls []core.Node
			if r.err != nil {
				controls = append(controls, label(r.err.Error(), 13, core.FontWeightNormal, colors(cx).muted))
			}
			controls = append(controls, more)
			footer = core.View{Style: core.Style{Padding: core.Symmetric(12, 28), Gap: 6}, Children: controls}
		}
		return m.displayCatalog(cx, src, r.data, footer)
	})
}

type catalogData struct {
	title, subtitle, image, description string
	sections                            []music.Section
	list                                *music.Playlist
	album                               *music.Album
	artist                              *music.Artist
	show                                *music.Show
	next                                int
	searchNext                          map[music.SearchKind]int
	home                                bool
}

type pageResult struct {
	data          catalogData
	err           error
	done, loading bool
}

func (d catalogData) hasMore() bool { return d.next > 0 || len(d.searchNext) > 0 }

func (m *model) catalogPage(cx *core.Scope, src music.Source, r route) (core.Node, error) {
	if r.page == "home" && m.lib.TrackCount() == 0 {
		if m.lib.Scanning() {
			return m.empty(cx, lucide.Folder, i18n.T("Looking for music…"), "", nil), nil
		}
		return m.localEmptyHome(cx), nil
	}
	data, err := loadCatalog(src, r)
	if err != nil {
		return nil, err
	}
	return m.displayCatalog(cx, src, data, nil), nil
}

func (m *model) localEmptyHome(cx *core.Scope) core.Node {
	p := colors(cx)
	title := i18n.T("Add a music folder")
	if len(m.lib.Folders()) > 0 {
		title = i18n.T("No music found")
	}

	// [untitled]-style pink-magenta album sleeve with a peeking disc on the right.
	disc := core.View{
		Style: core.Style{
			Position: core.Absolute, Right: core.Pt(-16), Top: core.Pt(10),
			Width: core.Pt(88), Height: core.Pt(88), CornerRadius: 44,
			Gradients:   []core.Gradient{core.LinearGradient(135, core.RGB(0xf3d4fa), core.RGB(0x9a55d8))},
			BorderWidth: 1, BorderColor: core.RGBA(0xffffff, .35),
		},
	}
	sleeve := core.View{
		Style: core.Style{
			Width: core.Pt(108), Height: core.Pt(108), CornerRadius: 14,
			Gradients:    []core.Gradient{core.LinearGradient(135, core.RGB(0xffa3ee), core.RGB(0xe544d8))},
			ShadowColor:  core.RGBA(0xe544d8, .22),
			ShadowOffset: core.Point{Y: 10}, ShadowRadius: 24,
		},
	}
	artwork := core.View{
		Style:    core.Style{Width: core.Pt(108), Height: core.Pt(108), Margin: core.Edges{Bottom: 8}},
		Children: []core.Node{disc, sleeve},
	}

	addBtn := base.Button{
		Label:   i18n.T("Add a Music Folder"),
		TestID:  "add-folder",
		OnPress: func(core.PressEvent) { m.pickMusic(cx) },
		Style: core.Style{
			Height: core.Pt(38), Padding: core.Symmetric(0, 22), CornerRadius: 19,
			Background: p.button, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Margin:     core.Edges{Top: 6},
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Opacity: core.Alpha(.88)},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.96}},
		},
		Children: []core.Node{label(i18n.T("Add a Music Folder"), 13, core.FontWeightSemibold, p.onButton)},
	}

	// Secondary soft-gray discovery card pointing to Plugin Market.
	marketBtn := base.Button{
		Label:   i18n.T("Plugin Market"),
		OnPress: func(core.PressEvent) { m.navigate(route{page: "plugin-market"}) },
		Style: core.Style{
			Height: core.Pt(34), Padding: core.Symmetric(0, 18), CornerRadius: 17,
			Background: p.hover, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Margin:     core.Edges{Top: 4},
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: p.selected},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.96}},
		},
		Children: []core.Node{label(i18n.T("Plugin Market"), 12, core.FontWeightSemibold, p.text)},
	}
	discoverCard := core.View{
		Style: core.Style{
			Width: core.Pt(360), MaxWidth: core.Pct(90), CornerRadius: 18,
			Background: p.sidebar, Padding: core.Symmetric(22, 24),
			AlignItems: core.AlignCenter, Gap: 6, Margin: core.Edges{Top: 28},
		},
		Children: []core.Node{
			label(i18n.T("Plugin Market"), 14, core.FontWeightSemibold, p.text),
			core.Text{
				Content: i18n.T("Browse all plugins or import a signed Yoho package."),
				Style:   core.Style{FontSize: 12, Color: p.muted, TextAlign: core.TextAlignCenter},
			},
			marketBtn,
		},
	}

	return core.View{
		Style: core.Style{
			FlexGrow: 1, FlexShrink: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Gap: 6, Padding: core.All(24),
		},
		Children: []core.Node{
			artwork,
			label(title, 17, core.FontWeightSemibold, p.text),
			label(i18n.T("Add the folders where your music lives"), 13, core.FontWeightNormal, p.muted),
			addBtn,
			discoverCard,
		},
	}
}

func loadCatalog(src music.Source, r route) (catalogData, error) {
	var d catalogData
	switch r.page {
	case "home", "browse", "songs", "albums", "artists":
		var browse *music.Browse
		var err error
		if r.page == "home" {
			d.title, d.home = i18n.T("Home"), true
			if src.Can("home") {
				browse, err = src.Home()
			}
		} else if r.page == "browse" {
			browse, err = src.Browse(r.id)
		} else {
			browse, err = src.Browse(r.page)
		}
		if browse != nil {
			if browse.Title != "" {
				d.title = browse.Title
			}
			d.subtitle, d.image, d.description, d.sections = browse.Subtitle, browse.ImageURL, browse.Description, browse.Sections
		}
		return d, err
	case "album":
		p, err := src.Album(r.id)
		if p != nil {
			d.album = &p.Album
			d.title, d.subtitle, d.image, d.sections = p.Album.Name, albumSubtitle(p.Album), p.Album.ImageURL, []music.Section{{Tracks: p.Tracks}}
		}
		return d, err
	case "artist":
		p, err := src.Artist(r.id)
		if p != nil {
			d.artist = &p.Artist
			d.title, d.image, d.sections = p.Artist.Name, p.Artist.ImageURL, []music.Section{{Tracks: p.Tracks, Albums: p.Albums}}
		}
		return d, err
	case "playlist":
		p, err := src.Playlist(r.id)
		if p != nil {
			d.title, d.subtitle, d.image, d.description, d.sections, d.list = p.Name, p.Owner, p.ImageURL, p.Description, []music.Section{{Tracks: p.Tracks}}, p
		}
		return d, err
	case "playlists":
		lists, err := src.Playlists()
		d.title, d.sections = i18n.T("Playlists"), []music.Section{{Playlists: lists}}
		return d, err
	case "recent":
		tracks, err := src.Recent()
		d.title, d.sections = i18n.T("Recently Played"), []music.Section{{Tracks: tracks}}
		return d, err
	case "savedAlbums":
		d.title, d.sections = i18n.T("Saved Albums"), []music.Section{{}}
		return loadMoreCatalog(src, r, d)
	case "search":
		d.title, d.subtitle = i18n.T("Search"), r.id
		d.searchNext = make(map[music.SearchKind]int, len(catalogSearchKinds))
		for _, kind := range catalogSearchKinds {
			d.searchNext[kind] = 0
		}
		d.sections = []music.Section{{}}
		return loadMoreCatalog(src, r, d)
	case "liked":
		d.title = i18n.T("Liked Songs")
		d.sections = []music.Section{{}}
		return loadMoreCatalog(src, r, d)
	case "shows":
		shows, err := src.Shows()
		d.title, d.sections = i18n.T("Podcasts"), []music.Section{{Shows: shows}}
		return d, err
	case "show":
		show, err := src.Show(r.id, 0)
		if show != nil {
			d.show = &show.Show
			d.title, d.subtitle, d.image, d.description = show.Show.Name, show.Show.Publisher, show.Show.ImageURL, show.Show.Description
			d.sections = []music.Section{{Tracks: show.Episodes}}
			if show.More {
				d.next = len(show.Episodes)
			}
		}
		return d, err
	case "stations":
		stations, err := src.Stations()
		d.title, d.sections = i18n.T("Stations"), []music.Section{{Stations: stations}}
		return d, err
	}
	return d, nil
}

var catalogSearchKinds = []music.SearchKind{
	music.SearchTracks, music.SearchAlbums, music.SearchArtists, music.SearchPlaylists, music.SearchShows,
}

func loadMoreCatalog(src music.Source, r route, d catalogData) (catalogData, error) {
	section := d.sections[0]
	localSource := src.Info().ID == local.ID
	switch r.page {
	case "search":
		next := make(map[music.SearchKind]int)
		for _, kind := range catalogSearchKinds {
			offset, ok := d.searchNext[kind]
			if !ok {
				continue
			}
			for {
				p, err := src.Search(r.id, kind, offset)
				if err != nil {
					if offset > 0 || kind == music.SearchTracks && !errors.Is(err, music.ErrUnsupported) {
						return catalogData{}, err
					}
					break
				}
				if p == nil {
					break
				}
				section.Tracks = append(section.Tracks, p.Tracks...)
				section.Albums = append(section.Albums, p.Albums...)
				section.Artists = append(section.Artists, p.Artists...)
				section.Playlists = append(section.Playlists, p.Playlists...)
				section.Shows = append(section.Shows, p.Shows...)
				n := p.Len()
				if !p.More || n == 0 {
					break
				}
				offset += n
				if !localSource {
					next[kind] = offset
					break
				}
			}
		}
		d.searchNext = next
	case "liked":
		offset := d.next
		d.next = 0
		for {
			p, err := src.Liked(offset)
			if err != nil {
				return catalogData{}, err
			}
			if p == nil {
				break
			}
			section.Tracks = append(section.Tracks, p.Tracks...)
			if !p.More || len(p.Tracks) == 0 {
				break
			}
			offset += len(p.Tracks)
			if !localSource {
				d.next = offset
				break
			}
		}
	case "show":
		p, err := src.Show(r.id, d.next)
		if err != nil {
			return catalogData{}, err
		}
		d.next = 0
		if p != nil {
			section.Tracks = append(section.Tracks, p.Episodes...)
			if p.More && len(p.Episodes) > 0 {
				d.next = len(section.Tracks)
			}
		}
	case "savedAlbums":
		page, err := src.SavedAlbums(d.next)
		if err != nil {
			return catalogData{}, err
		}
		d.next = 0
		if page != nil {
			section.Albums = append(section.Albums, page.Albums...)
			if page.More && len(page.Albums) > 0 {
				d.next = len(section.Albums)
			}
		}
	}
	d.sections = []music.Section{section}
	return d, nil
}

func (m *model) pageHeader(cx *core.Scope, title, sub, image string, actions ...core.Node) core.Node {
	p := colors(cx)
	var headings = []core.Node{label(title, 20, core.FontWeightMedium, p.text)}
	if sub != "" {
		headings = append(headings, label(sub, 13, core.FontWeightNormal, p.muted))
	}
	var items []core.Node
	if image != "" {
		items = append(items, m.artwork(cx, image, 72, m.source.GetUntracked()))
	}
	items = append(items, core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, Gap: 4}, Children: headings})
	items = append(items, actions...)
	view := row(items...)
	view.Style.Padding = core.Symmetric(14, 0)
	view.Style.Gap = 16
	return paperWrap(view)
}

func (m *model) empty(cx *core.Scope, icon glyph, title, detail string, pick func()) core.Node {
	var action core.Node
	if pick != nil {
		action = button(i18n.T("Add a Music Folder"), "add-folder", lucide.Plus, pick)
	}
	return m.emptyWithAction(cx, icon, title, detail, action)
}

func (m *model) emptyWithAction(cx *core.Scope, icon glyph, title, detail string, action core.Node) core.Node {
	p := colors(cx)
	children := []core.Node{
		core.View{Style: core.Style{Width: core.Pt(72), Height: core.Pt(72), CornerRadius: 20, Background: p.hover, JustifyContent: core.JustifyCenter, AlignItems: core.AlignCenter}, Children: []core.Node{
			ui.Icon(icon, 30, func() core.Color { return p.muted }),
		}},
		label(title, 22, core.FontWeightBold, p.text),
	}
	if detail != "" {
		children = append(children, label(detail, 13, core.FontWeightNormal, p.muted))
	}
	if action != nil {
		if b, ok := action.(ui.Button); ok {
			b.Color = func() core.Color { return p.accent }
			b.Style = core.Style{Height: core.Pt(36), CornerRadius: 18, Margin: core.Symmetric(6, 0)}
			action = b
		}
		children = append(children, action)
	}
	return core.View{Style: core.Style{FlexGrow: 1, FlexShrink: 1, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 10, Padding: core.All(24)}, Children: children}
}

func (m *model) artwork(cx *core.Scope, url string, size core.Points, source string) core.Node {
	p := colors(cx)
	face := core.Style{Width: core.Pt(size), Height: core.Pt(size), CornerRadius: 2, Background: p.chip, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}
	if url == "" {
		return core.View{Style: face, Children: []core.Node{ui.Icon(lucide.Music, min(size/2, 48), func() core.Color { return p.muted })}}
	}
	return core.Image{SourceFunc: m.coverSource(cx, url, int(size*2), source), Style: face, ResizeMode: core.Cover}
}

func (m *model) circleArtwork(cx *core.Scope, url string, size core.Points, source string) core.Node {
	p := colors(cx)
	face := core.Style{Width: core.Pt(size), Height: core.Pt(size), CornerRadius: size / 2, Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter}
	if url == "" {
		return core.View{Style: face, Children: []core.Node{ui.Icon(lucide.Music, min(size/2, 24), func() core.Color { return p.muted })}}
	}
	return core.Image{SourceFunc: m.coverSource(cx, url, int(size*2), source), Style: face, ResizeMode: core.Cover}
}

var playFillSVG = func() *core.SVG {
	svg, _ := core.ParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor"><path d="M7 4.4c0-1.1 1.2-1.8 2.2-1.2l11.2 7.6c.9.6.9 1.8 0 2.4L9.2 20.8c-1 .6-2.2-.1-2.2-1.2V4.4z"/></svg>`))
	return svg
}()

var coverGradients = [][2]uint32{
	{0xFA2CD1, 0x802BE4}, // [untitled] signature diagonal magenta-purple
	{0x1B2838, 0xC9A27C},
	{0x2D3A3A, 0x6B8F71},
	{0x2B1D19, 0x8C4A32},
	{0x1E3C72, 0xF6B93B},
	{0x4A3F55, 0x9B72AA},
}

func coverGradientFor(key string) core.Gradient {
	var sum uint32
	for i := 0; i < len(key); i++ {
		sum = sum*31 + uint32(key[i])
	}
	pair := coverGradients[int(sum)%len(coverGradients)]
	return core.LinearGradient(135, core.RGB(pair[0]), core.RGB(pair[1]))
}

func (m *model) tileArtwork(cx *core.Scope, name, url string, source string) core.Node {
	face := core.Style{
		Width: core.Pct(100), Height: core.Pct(100), CornerRadius: 15,
		AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	if url == "" {
		face.Gradients = []core.Gradient{coverGradientFor(name)}
		return core.View{Style: face}
	}
	return core.Image{SourceFunc: m.coverSource(cx, url, 360, source), Style: face, ResizeMode: core.Cover}
}

func (m *model) playCollection(src music.Source, r route, shuffle bool) {
	if src == nil {
		return
	}
	sourceID := src.Info().ID
	go func() {
		var tracks []music.Track
		var err error
		switch r.page {
		case "album":
			var p *music.AlbumPage
			p, err = src.Album(r.id)
			if p != nil {
				tracks = p.Tracks
			}
		case "playlist":
			var p *music.Playlist
			p, err = src.Playlist(r.id)
			if p != nil {
				tracks = p.Tracks
			}
		}
		m.cx.Post(func() {
			if err != nil {
				m.report(err)
				return
			}
			tracks = withSource(tracks, sourceID)
			if len(tracks) == 0 {
				return
			}
			if shuffle {
				m.player.PlayShuffled(tracks)
			} else {
				m.player.PlayQueue(tracks, 0)
			}
			m.refreshPlayback()
		})
	}()
}

func (m *model) displayCatalog(cx *core.Scope, src music.Source, d catalogData, footer core.Node) core.Node {
	sourceID := src.Info().ID
	var tracks []music.Track
	for _, section := range d.sections {
		tracks = append(tracks, section.Tracks...)
	}
	tracks = withSource(tracks, sourceID)
	trackOnly := len(d.sections) == 1 && len(tracks) > 0 &&
		len(d.sections[0].Albums)+len(d.sections[0].Artists)+len(d.sections[0].Playlists)+len(d.sections[0].Shows)+len(d.sections[0].Stations)+len(d.sections[0].Links) == 0
	if d.album != nil || d.list != nil || d.show != nil || (!d.home && !m.hasSubNav() && trackOnly) {
		return m.withDetailHeader(cx, m.collectionDetail(cx, src, d, tracks, footer), collectionWrap)
	}
	return m.withDetailHeader(cx, m.catalogSections(cx, src, d, tracks, footer), paperWrap)
}

func (m *model) catalogSections(cx *core.Scope, src music.Source, d catalogData, tracks []music.Track, footer core.Node) core.Node {
	sourceID := src.Info().ID
	var actions []core.Node
	if len(tracks) > 0 && !d.home {
		actions = append(actions, button(i18n.T("Play"), "play-all", lucide.Play, m.playback(func() { m.player.PlayQueue(tracks, 0) })),
			button(i18n.T("Shuffle"), "shuffle-all", lucide.Shuffle, m.playback(func() { m.player.PlayShuffled(tracks) })))
	}
	if m.page.Get().page == "playlists" && sourceID == local.ID {
		actions = append(actions, button(i18n.T("New Playlist"), "new-playlist", lucide.Plus, func() { m.navigate(route{page: "playlist-edit"}) }))
	}
	if sourceID != local.ID {
		actions = append(actions, m.sourceCatalogActions(cx, src, d)...)
	}
	var header core.Node
	if !d.home || len(actions) > 0 || d.subtitle != "" || d.image != "" {
		header = m.pageHeader(cx, d.title, d.subtitle, d.image, actions...)
	}
	if d.description != "" {
		description := label(d.description, 13, core.FontWeightNormal, colors(cx).muted)
		description.Style.Padding = core.Edges{Bottom: 12}
		descWrap := paperWrap(description)
		if header != nil {
			header = core.View{Children: []core.Node{header, descWrap}}
		} else {
			header = descWrap
		}
	}
	categoryPage := m.hasSubNav()
	topPad := detailHeaderHeight
	if categoryPage {
		topPad = 74
	}
	if len(d.sections) == 1 {
		section := d.sections[0]
		if len(tracks) > 0 && len(section.Albums)+len(section.Artists)+len(section.Playlists)+len(section.Shows)+len(section.Stations)+len(section.Links) == 0 {
			list := m.trackList(cx, tracks, d.list, false)
			if header != nil {
				list.Header = core.View{Style: core.Style{Padding: core.Edges{Top: topPad}}, Children: []core.Node{header}}
			} else {
				list.Header = core.View{Style: core.Style{Height: core.Pt(topPad)}}
			}
			var footerNodes []core.Node
			if footer != nil {
				footerNodes = append(footerNodes, paperWrap(footer))
			}
			footerNodes = append(footerNodes, core.View{Style: core.Style{Height: core.Pt(108)}})
			list.Footer = core.View{Children: footerNodes}
			return list
		}
	}
	var body []core.Node
	if header != nil {
		body = append(body, header)
	}
	var count int
	for i := 0; i < len(d.sections); i++ {
		section := d.sections[i]
		var items []core.Node
		if d.home && isHomeTrackColumn(section) && i+1 < len(d.sections) && isHomeTrackColumn(d.sections[i+1]) {
			items = []core.Node{m.homePairedTrackColumns(cx, src, section, d.sections[i+1], d.list)}
			i++
		} else {
			items = m.section(cx, src, section, d.list, d.home)
		}
		if len(items) == 0 {
			continue
		}
		if count > 0 || (i > 0 && len(body) > 0) {
			body = append(body, paperWrap(dashedDivider(colors(cx).line)))
		}
		count += len(items)
		body = append(body, items...)
	}
	if count == 0 {
		style := fill
		if !categoryPage {
			style.Padding = core.Edges{Top: topPad}
		}
		return core.View{Style: style, Children: append(body, m.empty(cx, lucide.Music, i18n.T("Nothing here yet"), "", nil))}
	}
	if footer != nil {
		body = append(body, paperWrap(footer))
	}
	return core.Scroll{Style: fill, Children: []core.Node{core.View{Style: core.Style{Gap: 16, Padding: core.Edges{Top: topPad, Bottom: 108}}, Children: body}}}
}

func dashedDivider(lineColor core.Color) core.Node {
	return core.Canvas{
		Style: core.Style{
			Height: core.Pt(1),
			Margin: core.Symmetric(8, 0),
		},
		Draw: func(d *core.Drawing, size core.Size) {
			d.FillFromView(core.PaintColor(lineColor))
			for x := float32(0); x < float32(size.Width); x += 8 {
				w := min(float32(4), float32(size.Width)-x)
				d.FillRect(x, 0, w, 1)
			}
		},
	}
}

func collectionWrap(children ...core.Node) core.View {
	return core.View{
		Style: core.Style{
			Width: core.Pct(84), MaxWidth: core.Pt(920), AlignSelf: core.AlignCenter,
		},
		Children: children,
	}
}

func (m *model) collectionBackdrop(cx *core.Scope, title, coverURL, sourceID string) core.Node {
	return core.Dynamic(func(cx *core.Scope) core.Node {
		frame := cx.Env().Window()
		dark := cx.Dark()
		var art core.Node
		if coverURL != "" {
			art = core.Image{
				SourceFunc:     m.coverSource(cx, coverURL, 512, sourceID),
				ResizeMode:     core.Cover,
				BlurRadius:     22,
				ElementsHidden: true,
				Style: core.Style{
					Position: core.Absolute,
					Left:     core.Pt(-24), Top: core.Pt(-24),
					Width:  core.Pt(frame.Width + 48),
					Height: core.Pt(frame.Height + 48),
				},
			}
		} else {
			art = core.View{
				Style: core.Style{
					Position: core.Absolute,
					Top:      core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0),
					Gradients: []core.Gradient{coverGradientFor(title)},
					Opacity:   core.Alpha(.34),
				},
			}
		}
		veilHex := uint32(0xffffff)
		topAlpha, bottomAlpha := float32(.66), float32(.88)
		if dark {
			veilHex = 0x151517
			topAlpha, bottomAlpha = .62, .84
		}
		veil := core.View{
			Style: core.Style{
				Position: core.Absolute,
				Top:      core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0),
				Gradients: []core.Gradient{{
					Kind:  core.GradientLinear,
					Angle: 180,
					Stops: []core.ColorStop{
						{Color: core.RGBA(veilHex, topAlpha), At: core.Pct(0)},
						{Color: core.RGBA(veilHex, bottomAlpha), At: core.Pct(100)},
					},
				}},
			},
		}
		return core.View{
			PointerEvents: core.PointerEventsNone,
			Style: core.Style{
				Position: core.Absolute, Top: core.Pt(0), Right: core.Pt(0), Bottom: core.Pt(0), Left: core.Pt(0),
				Overflow: core.Hidden,
			},
			Children: []core.Node{art, veil},
		}
	})
}

func (m *model) collectionDetail(cx *core.Scope, src music.Source, d catalogData, tracks []music.Track, footer core.Node) core.Node {
	p := colors(cx)
	sourceID := src.Info().ID
	dark := cx.Dark()

	coverURL := d.image
	if coverURL == "" && len(tracks) > 0 {
		coverURL = tracks[0].Album.ImageURL
	}

	shadowAlpha := float32(.16)
	neutralBg := core.RGBA(0xffffff, .62)
	borderTint := core.RGBA(0x09090b, .06)
	if dark {
		shadowAlpha = .42
		neutralBg = core.RGBA(0xffffff, .10)
		borderTint = core.RGBA(0xffffff, .10)
	}

	var coverContent core.Node
	if coverURL == "" {
		coverContent = core.View{
			Style: core.Style{
				Width: core.Pct(100), Height: core.Pct(100), CornerRadius: 14,
				Gradients: []core.Gradient{coverGradientFor(d.title)},
			},
		}
	} else {
		coverContent = core.Image{
			SourceFunc: m.coverSource(cx, coverURL, 384, sourceID),
			ResizeMode: core.Cover,
			Style:      core.Style{Width: core.Pct(100), Height: core.Pct(100), CornerRadius: 14},
		}
	}
	heroCover := core.View{
		Style: core.Style{
			Width: core.Pt(186), Height: core.Pt(186), CornerRadius: 14, FlexShrink: 0,
			BorderWidth: 1, BorderColor: borderTint,
			ShadowColor: core.RGBA(0x000000, shadowAlpha), ShadowOffset: core.Point{Y: 10}, ShadowRadius: 24,
		},
		Children: []core.Node{coverContent},
	}

	var kindText string
	switch {
	case d.album != nil:
		kindText = i18n.T("Album")
	case d.list != nil:
		kindText = i18n.T("Playlist")
	case d.show != nil:
		kindText = i18n.T("Podcast")
	}

	var rightChildren []core.Node
	if kindText != "" {
		kind := label(kindText, 12, core.FontWeightSemibold, p.muted)
		kind.Style.Margin = core.Edges{Bottom: 6}
		rightChildren = append(rightChildren, kind)
	}
	title := label(d.title, 28, core.FontWeightBold, p.text)
	title.MaxLines = 2
	rightChildren = append(rightChildren, title)

	var metaParts []string
	if d.subtitle != "" {
		metaParts = append(metaParts, d.subtitle)
	}
	if len(tracks) > 0 {
		metaParts = append(metaParts, i18n.Songs(len(tracks)))
		var total time.Duration
		for _, t := range tracks {
			total += t.Length()
		}
		if total > 0 {
			metaParts = append(metaParts, formatTime(total))
		}
	}
	if len(metaParts) > 0 {
		meta := label(strings.Join(metaParts, " · "), 13, core.FontWeightNormal, p.muted)
		meta.MaxLines = 1
		meta.Style.FontVariant = core.FontTabularNums
		meta.Style.Margin = core.Edges{Top: 6}
		rightChildren = append(rightChildren, meta)
	}
	if d.description != "" {
		desc := label(d.description, 12, core.FontWeightNormal, p.muted)
		desc.MaxLines = 2
		desc.Style.Margin = core.Edges{Top: 4}
		rightChildren = append(rightChildren, desc)
	}

	var actionNodes []core.Node
	if len(tracks) > 0 {
		playBtn := base.Button{
			Label:   i18n.T("Play"),
			TestID:  "play-all",
			OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayQueue(tracks, 0) })() },
			Style: core.Style{
				Direction: core.Row, Height: core.Pt(36), Padding: core.Symmetric(0, 24),
				CornerRadius: 18, Background: p.button, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
				Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
				Hover:      &core.Style{Opacity: core.Alpha(.88)},
				Active:     &core.Style{Transform: core.Transform{Scale: 0.97}},
			},
			Children: []core.Node{
				ui.Icon(func() *core.SVG { return playFillSVG }, 12, func() core.Color { return p.onButton }),
				label(i18n.T("Play"), 13, core.FontWeightSemibold, p.onButton),
			},
		}
		shuffleBtn := base.Button{
			Label:   i18n.T("Shuffle"),
			TestID:  "shuffle-all",
			OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayShuffled(tracks) })() },
			Style: core.Style{
				Direction: core.Row, Height: core.Pt(36), Padding: core.Symmetric(0, 20),
				CornerRadius: 18, Background: neutralBg,
				AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
				Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
				Hover:      &core.Style{Background: p.hover},
				Active:     &core.Style{Transform: core.Transform{Scale: 0.97}},
			},
			Children: []core.Node{
				ui.Icon(lucide.Shuffle, 13, func() core.Color { return p.text }),
				label(i18n.T("Shuffle"), 13, core.FontWeightMedium, p.text),
			},
		}
		actionNodes = append(actionNodes, playBtn, shuffleBtn)
	}
	if d.list != nil && d.list.Mine && !d.list.Fixed && sourceID == local.ID {
		actionNodes = append(actionNodes, iconButton(lucide.Pencil, i18n.T("Edit Playlist"), "edit-playlist", func() {
			m.navigate(route{page: "playlist-edit", id: d.list.ID})
		}))
	}
	if sourceID != local.ID {
		actionNodes = append(actionNodes, m.sourceCatalogActions(cx, src, d)...)
	}
	if d.show != nil {
		show := *d.show
		if src.Can("subscribeShow") {
			subTitle, icon := i18n.T("Subscribe"), lucide.Plus
			if show.Subscribed {
				subTitle, icon = i18n.T("Unsubscribe"), lucide.Check
			}
			actionNodes = append(actionNodes, button(subTitle, "subscribe-show", icon, func() {
				m.changeSource(cx, src, func() error { return src.SubscribeShow(show.ID, !show.Subscribed) })
			}))
		}
		if show.PublisherPage != "" {
			pubTitle := show.Publisher
			if pubTitle == "" {
				pubTitle = i18n.T("Publisher")
			}
			actionNodes = append(actionNodes, button(pubTitle, "show-publisher", lucide.ArrowUpRight, func() {
				m.navigate(route{page: "browse", id: show.PublisherPage})
			}))
		}
	}
	if len(actionNodes) > 0 {
		rightChildren = append(rightChildren, core.View{
			Style: core.Style{
				Direction: core.Row, Wrap: core.Wrap, AlignItems: core.AlignCenter,
				Gap: 10, Margin: core.Edges{Top: 16},
			},
			Children: actionNodes,
		})
	}

	hero := collectionWrap(core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignEnd, Gap: 28,
			Padding: core.Edges{Top: 68, Bottom: 28},
		},
		Children: []core.Node{
			heroCover,
			core.View{
				Style:    core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), JustifyContent: core.JustifyEnd, Padding: core.Edges{Bottom: 4}},
				Children: rightChildren,
			},
		},
	})

	var content core.Node
	if len(tracks) == 0 {
		content = core.View{
			Style:    fill,
			Children: []core.Node{hero, m.empty(cx, lucide.Music, i18n.T("Nothing here yet"), "", nil)},
		}
	} else {
		list := core.List{
			Count: func() int { return len(tracks) }, EstimatedSize: 56, Style: fill,
			Key: func(i int) any { return fmt.Sprintf("%d:%s:%s", i, tracks[i].Source, tracks[i].ID) },
			Row: func(cx *core.Scope, cell core.ListRow) core.Node {
				return core.Dynamic(func(cx *core.Scope) core.Node {
					i := cell.Index()
					if i < 0 || i >= len(tracks) {
						return nil
					}
					return m.collectionSongRow(cx, tracks[i], i, d.list, func() {
						m.player.PlayTrack(tracks, i)
					})
				})
			},
		}
		list.Header = hero
		var footerNodes []core.Node
		if footer != nil {
			footerNodes = append(footerNodes, collectionWrap(footer))
		}
		footerNodes = append(footerNodes, core.View{Style: core.Style{Height: core.Pt(108)}})
		list.Footer = core.View{Children: footerNodes}
		content = list
	}

	return core.View{
		Style:    fill,
		Children: []core.Node{m.collectionBackdrop(cx, d.title, coverURL, sourceID), content},
	}
}

func (m *model) collectionSongRow(cx *core.Scope, track music.Track, index int, list *music.Playlist, play func()) core.Node {
	p := colors(cx)
	dark := cx.Dark()
	activeBg := core.RGBA(0xffffff, .58)
	if dark {
		activeBg = core.RGBA(0xffffff, .10)
	}
	isPlaying := func() bool {
		cur := m.playing.Get().track
		return cur != nil && cur.ID == track.ID && cur.Source == track.Source
	}

	num := core.Text{
		ContentFunc: func() string {
			if isPlaying() && m.playing.Get().playing {
				return "▶"
			}
			return fmt.Sprint(index + 1)
		},
		Style: core.Style{Width: core.Pt(24), FontSize: 13, FontVariant: core.FontTabularNums, TextAlign: core.TextAlignCenter, Color: p.muted, FlexShrink: 0},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Color: p.text, FontWeight: core.FontWeightSemibold}
			}
			return core.Style{}
		},
	}

	var thumb core.Node
	thumbFace := core.Style{
		Width: core.Pt(40), Height: core.Pt(40), CornerRadius: 6, FlexShrink: 0,
		Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	if track.Album.ImageURL == "" {
		thumb = core.View{Style: thumbFace, Children: []core.Node{ui.Icon(lucide.Music, 15, func() core.Color { return p.muted })}}
	} else {
		thumb = core.Image{SourceFunc: m.coverSource(cx, track.Album.ImageURL, 80, track.Source), Style: thumbFace, ResizeMode: core.Cover}
	}

	title := core.Text{
		Content:  track.Name,
		MaxLines: 1,
		Style:    core.Style{FontSize: 14, FontWeight: core.FontWeightMedium, Color: p.text},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{FontWeight: core.FontWeightSemibold}
			}
			return core.Style{}
		},
	}
	titleStackChildren := []core.Node{title}
	if artists := track.ArtistNames(); artists != "" {
		artist := label(artists, 12, core.FontWeightNormal, p.muted)
		artist.MaxLines = 1
		titleStackChildren = append(titleStackChildren, artist)
	}
	titleStack := core.View{
		Style:    core.Style{FlexGrow: 1.35, FlexShrink: 1, FlexBasis: core.Pt(0), MinWidth: core.Pt(0), Gap: 2, Padding: core.Edges{Right: 12}},
		Children: titleStackChildren,
	}

	albumCol := core.View{
		Style: core.Style{FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0), MinWidth: core.Pt(0), Padding: core.Edges{Right: 12}},
		Children: []core.Node{func() core.Node {
			al := label(track.Album.Name, 13, core.FontWeightNormal, p.muted)
			al.MaxLines = 1
			return al
		}()},
	}

	dur := label(formatTime(track.Length()), 13, core.FontWeightNormal, p.muted)
	dur.Style.FontVariant = core.FontTabularNums
	dur.Style.Width = core.Pt(44)
	dur.Style.TextAlign = core.TextAlignEnd
	dur.Style.FlexShrink = 0

	mainBtn := base.Button{
		Label:   track.Name,
		TestID:  "track-" + fmt.Sprint(index),
		OnPress: func(core.PressEvent) { m.playback(play)() },
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 14,
			FlexGrow: 1, FlexShrink: 1, Height: core.Pt(54),
			Padding: core.Symmetric(0, 10), CornerRadius: 8,
			Hover: &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Background: activeBg}
			}
			return core.Style{}
		},
		Children: []core.Node{num, thumb, titleStack, albumCol, dur},
	}

	menuRef := &core.Ref{}
	entry := row(mainBtn)
	source := m.sourceByID(track.Source)
	if canLike(source, track) {
		like := iconButton(lucide.Heart, i18n.T("Like Song"), "like-"+fmt.Sprint(index), func() { m.toggleLike(track) })
		like.Size, like.IconSize = 28, 14
		if source == m.lib {
			like.Tint = func() core.Color {
				m.updated.Get()
				liked, _ := m.lib.IsLiked([]string{track.ID})
				if liked[track.ID] {
					return p.text
				}
				return p.muted
			}
		}
		entry.Children = append(entry.Children, like)
	}
	entry.Children = append(entry.Children, base.Button{
		Ref: menuRef, Label: i18n.T("Track Options"), TestID: "track-menu-" + fmt.Sprint(index),
		Style:    core.Style{Width: core.Pt(28), Height: core.Pt(28), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, CornerRadius: 6, Hover: &core.Style{Background: p.hover}},
		Children: []core.Node{ui.Icon(lucide.Ellipsis, 14, func() core.Color { return p.muted })},
		OnPress: func(event core.PressEvent) {
			at := event.PagePosition
			if bounds, ok := menuRef.Bounds(); ok {
				at = core.Point{X: bounds.Origin.X, Y: bounds.Origin.Y + bounds.Size.Height}
			}
			m.trackOptions(cx, track, index, list, at)
		},
	})
	entry.OnContextMenu = func(event core.ContextMenuEvent) { m.trackOptions(cx, track, index, list, event.PagePosition) }
	entry.Style.Gap = 2
	entry.Style.Padding = core.Symmetric(1, 0)
	return collectionWrap(entry)
}

func albumSubtitle(album music.Album) string {
	var names []string
	for _, a := range album.Artists {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	artist := strings.Join(names, ", ")
	year := ""
	if len(album.Released) >= 4 {
		year = album.Released[:4]
	}
	if artist != "" && year != "" {
		return artist + " · " + year
	}
	if artist != "" {
		return artist
	}
	return year
}

func (m *model) changeSource(cx *core.Scope, src music.Source, fn func() error) {
	go func() {
		err := fn()
		cx.Post(func() {
			if m.sourceActionResult(src, err) {
				m.sourceVersion.Update(func(n int) int { return n + 1 })
			}
		})
	}()
}

func (m *model) section(cx *core.Scope, src music.Source, section music.Section, list *music.Playlist, home bool) []core.Node {
	p := colors(cx)
	var items []core.Node
	if section.Title != "" {
		var header core.Node
		if section.MorePage != "" {
			titleBtn := base.Button{
				Label:   section.Title,
				TestID:  "show-" + section.MorePage,
				OnPress: func(core.PressEvent) { m.navigate(route{page: "browse", id: section.MorePage}) },
				Style: core.Style{
					Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4,
					Padding:    core.Symmetric(2, 0),
					Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
					Hover:      &core.Style{Opacity: core.Alpha(.72)},
				},
				Children: []core.Node{
					label(section.Title, 16, core.FontWeightSemibold, p.text),
					ui.Icon(lucide.ChevronRight, 16, func() core.Color { return p.muted }),
				},
			}
			header = row(titleBtn, spacer())
		} else if home {
			header = row(label(section.Title, 16, core.FontWeightSemibold, p.text), spacer())
		} else {
			header = row(label(section.Title, 14, core.FontWeightNormal, p.muted), spacer())
		}
		items = append(items, paperWrap(header))
	}
	var cards []core.Node
	for _, album := range section.Albums {
		cards = append(cards, m.tile(cx, src, album.Name, albumSubtitle(album), album.ImageURL, route{page: "album", id: album.ID}, true))
	}
	for _, artist := range section.Artists {
		cards = append(cards, m.tile(cx, src, artist.Name, "", artist.ImageURL, route{page: "artist", id: artist.ID}, false))
	}
	for _, list := range section.Playlists {
		sub := list.Owner
		if sub == "" && list.TrackCount > 0 {
			sub = i18n.Songs(list.TrackCount)
		} else if sub == "" && len(list.Tracks) > 0 {
			sub = i18n.Songs(len(list.Tracks))
		}
		cards = append(cards, m.tile(cx, src, list.Name, sub, list.ImageURL, route{page: "playlist", id: list.ID}, true))
	}
	for _, show := range section.Shows {
		cards = append(cards, m.tile(cx, src, show.Name, show.Publisher, show.ImageURL, route{page: "show", id: show.ID}, false))
	}
	heroFeatureRow := home && section.Title == "" && len(section.Albums) == 0 && len(section.Artists) == 0 && len(section.Playlists) == 0 && len(section.Shows) == 0
	for _, link := range section.Links {
		if heroFeatureRow {
			l := link
			cards = append(cards, m.featureTile(cx, l.Title, l.Subtitle, l.ImageURL, l.Page, func() {
				m.navigate(route{page: "browse", id: l.Page})
			}))
		} else {
			cards = append(cards, m.tile(cx, src, link.Title, link.Subtitle, link.ImageURL, route{page: "browse", id: link.Page}, false))
		}
	}
	for _, station := range section.Stations {
		if heroFeatureRow {
			st := station
			cards = append(cards, m.featureTile(cx, st.Name, st.Description, st.ImageURL, "station:"+st.ID, func() {
				m.playStation(src, st)
			}))
		} else {
			cards = append(cards, button(station.Name, "station-"+station.ID, lucide.Radio, func() { m.playStation(src, station) }))
		}
	}
	if len(cards) > 0 {
		grid := core.View{
			Style: core.Style{
				Display:             core.DisplayGrid,
				GridTemplateColumns: core.GridTracks("repeat(5, minmax(0, 1fr))"),
				ColumnGap:           20,
				RowGap:              24,
			},
			Children: cards,
		}
		items = append(items, paperWrap(grid))
	}
	if len(section.Tracks) > 0 {
		tracks := withSource(section.Tracks, src.Info().ID)
		if home {
			limit := min(len(tracks), 6)
			gridItems := make([]core.Node, 0, limit)
			for i := 0; i < limit; i++ {
				gridItems = append(gridItems, m.homeTrackCard(cx, tracks, i, list))
			}
			trackGrid := core.View{
				Style: core.Style{
					Display:             core.DisplayGrid,
					GridTemplateColumns: core.GridTracks("repeat(2, minmax(0, 1fr))"),
					ColumnGap:           28,
					RowGap:              14,
				},
				Children: gridItems,
			}
			items = append(items, paperWrap(trackGrid))
		} else if len(tracks) > 12 {
			compact := m.trackList(cx, tracks, list, false)
			compact.Style = core.Style{Height: core.Pt(378)}
			items = append(items, compact)
		} else {
			var rows []core.Node
			for i, t := range tracks {
				rows = append(rows, m.songRow(cx, t, i, list, false, func() { m.player.PlayTrack(tracks, i) }))
			}
			items = append(items, core.View{Children: rows})
		}
	}
	if section.Empty != "" && len(cards)+len(section.Tracks) == 0 {
		emptyText := label(section.Empty, 13, core.FontWeightNormal, p.muted)
		items = append(items, paperWrap(emptyText))
	}
	return items
}

type featureCardStyle struct {
	topA, topB uint32
	band       uint32
	icon       glyph
}

func featureStyleFor(key string) featureCardStyle {
	switch {
	case key == "daily":
		return featureCardStyle{topA: 0x4b5563, topB: 0x1f2937, band: 0x3f4145, icon: lucide.Calendar}
	case key == "radar":
		return featureCardStyle{topA: 0x99e2d0, topB: 0x489b84, band: 0x67b398, icon: lucide.Radio}
	case key == "recommendations":
		return featureCardStyle{topA: 0x4b749f, topB: 0x1d3f66, band: 0x1e4369, icon: lucide.Heart}
	case strings.HasPrefix(key, "station:"):
		return featureCardStyle{topA: 0xd9b99b, topB: 0x8f6b4b, band: 0xb18f6c, icon: lucide.Radio}
	case key == "charts":
		return featureCardStyle{topA: 0x6b6d62, topB: 0x353730, band: 0x44453d, icon: lucide.ListMusic}
	default:
		return featureCardStyle{topA: 0x4f46e5, topB: 0x1e1b4b, band: 0x312e81, icon: lucide.Music}
	}
}

func (m *model) featureTile(cx *core.Scope, title, subtitle, image, key string, action func()) core.Node {
	theme := featureStyleFor(key)
	smooth := core.Curve{X1: 0.22, Y1: 1, X2: 0.36, Y2: 1}
	shadowAlpha := float32(.16)
	if cx.Dark() {
		shadowAlpha = .40
	}

	var bg core.Node
	if image != "" {
		bg = core.Image{
			SourceFunc: m.coverSource(cx, image, 360, m.source.GetUntracked()),
			ResizeMode: core.Cover,
			Style:      core.Style{Width: core.Pct(100), Height: core.Pct(100)},
		}
	} else {
		bg = core.View{
			Style: core.Style{
				Width: core.Pct(100), Height: core.Pct(100),
				Gradients: []core.Gradient{core.LinearGradient(145, core.RGB(theme.topA), core.RGB(theme.topB))},
			},
		}
	}

	topScrim := core.View{
		PointerEvents: core.PointerEventsNone,
		Style: core.Style{
			Position: core.Absolute, Top: core.Pt(0), Left: core.Pt(0), Right: core.Pt(0), Height: core.Pct(55),
			Gradients: []core.Gradient{{
				Kind:  core.GradientLinear,
				Angle: 180,
				Stops: []core.ColorStop{
					{Color: core.RGBA(0x000000, .44), At: core.Pct(0)},
					{Color: core.RGBA(0x000000, 0), At: core.Pct(100)},
				},
			}},
		},
	}

	titleText := core.Text{
		Content:  title,
		MaxLines: 1,
		Style:    core.Style{FontSize: 16, FontWeight: core.FontWeightBold, Color: core.RGB(0xffffff), FlexShrink: 1},
	}
	titleOverlay := core.View{
		PointerEvents: core.PointerEventsNone,
		Style: core.Style{
			Position: core.Absolute, Top: core.Pt(11), Left: core.Pt(12), Right: core.Pt(12),
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6,
		},
		Children: []core.Node{
			ui.Icon(theme.icon, 15, func() core.Color { return core.RGB(0xffffff) }),
			titleText,
		},
	}

	upper := core.View{
		Style:    core.Style{Width: core.Pct(100), AspectRatio: 1, Overflow: core.Hidden},
		Children: []core.Node{bg, topScrim, titleOverlay},
	}

	caption := subtitle
	if caption == "" {
		caption = title
	}
	captionText := core.Text{
		Content:  caption,
		MaxLines: 2,
		Style:    core.Style{FontSize: 12, FontWeight: core.FontWeightMedium, LineHeight: 16, Color: core.RGBA(0xffffff, .94)},
	}
	lower := core.View{
		Style: core.Style{
			Height: core.Pt(54), Padding: core.Symmetric(8, 12),
			Background: core.RGB(theme.band), JustifyContent: core.JustifyCenter,
		},
		Children: []core.Node{captionText},
	}

	return base.Button{
		Label:   title,
		OnPress: func(core.PressEvent) { action() },
		Style: core.Style{
			Width: core.Pct(100), CornerRadius: 13, Overflow: core.Hidden,
			Transition: core.Transition{Duration: 240 * time.Millisecond, Easing: smooth},
			Hover: &core.Style{
				Transform:    core.Transform{Scale: 1.02},
				ShadowColor:  core.RGBA(0x000000, shadowAlpha),
				ShadowOffset: core.Point{Y: 8},
				ShadowRadius: 18,
			},
			Active: &core.Style{Transform: core.Transform{Scale: 0.98}},
		},
		Children: []core.Node{upper, lower},
	}
}

func isHomeTrackColumn(s music.Section) bool {
	return s.Title != "" && s.MorePage != "" &&
		len(s.Albums)+len(s.Artists)+len(s.Playlists)+len(s.Shows)+len(s.Stations)+len(s.Links) == 0 &&
		(len(s.Tracks) > 0 || s.Empty != "")
}

func (m *model) homePairedTrackColumns(cx *core.Scope, src music.Source, left, right music.Section, list *music.Playlist) core.Node {
	p := colors(cx)
	buildCol := func(sec music.Section) core.Node {
		titleBtn := base.Button{
			Label:   sec.Title,
			TestID:  "show-" + sec.MorePage,
			OnPress: func(core.PressEvent) { m.navigate(route{page: "browse", id: sec.MorePage}) },
			Style: core.Style{
				Direction: core.Row, AlignItems: core.AlignCenter, Gap: 4,
				Padding:    core.Symmetric(2, 0),
				Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
				Hover:      &core.Style{Opacity: core.Alpha(.72)},
			},
			Children: []core.Node{
				label(sec.Title, 16, core.FontWeightSemibold, p.text),
				ui.Icon(lucide.ChevronRight, 16, func() core.Color { return p.muted }),
			},
		}
		colChildren := []core.Node{row(titleBtn, spacer())}
		if len(sec.Tracks) > 0 {
			tracks := withSource(sec.Tracks, src.Info().ID)
			limit := min(len(tracks), 3)
			rows := make([]core.Node, 0, limit)
			for i := 0; i < limit; i++ {
				rows = append(rows, m.homeTrackCard(cx, tracks, i, list))
			}
			colChildren = append(colChildren, core.View{Style: core.Style{Gap: 10}, Children: rows})
		} else if sec.Empty != "" {
			empty := label(sec.Empty, 13, core.FontWeightNormal, p.muted)
			empty.Style.Padding = core.Edges{Top: 6}
			colChildren = append(colChildren, empty)
		}
		return core.View{
			Style:    core.Style{MinWidth: core.Pt(0), Gap: 12},
			Children: colChildren,
		}
	}
	return paperWrap(core.View{
		Style: core.Style{
			Display:             core.DisplayGrid,
			GridTemplateColumns: core.GridTracks("repeat(2, minmax(0, 1fr))"),
			ColumnGap:           28,
			AlignItems:          core.AlignStart,
		},
		Children: []core.Node{buildCol(left), buildCol(right)},
	})
}

func (m *model) homeTrackCard(cx *core.Scope, tracks []music.Track, index int, list *music.Playlist) core.Node {
	p := colors(cx)
	track := tracks[index]
	isPlaying := func() bool {
		cur := m.playing.Get().track
		return cur != nil && cur.ID == track.ID && cur.Source == track.Source
	}

	thumbFace := core.Style{
		Width: core.Pt(56), Height: core.Pt(56), CornerRadius: 8, FlexShrink: 0,
		Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	var thumb core.Node
	if track.Album.ImageURL == "" {
		thumb = core.View{Style: thumbFace, Children: []core.Node{ui.Icon(lucide.Music, 18, func() core.Color { return p.muted })}}
	} else {
		thumb = core.Image{SourceFunc: m.coverSource(cx, track.Album.ImageURL, 112, track.Source), Style: thumbFace, ResizeMode: core.Cover}
	}

	title := core.Text{
		Content:  track.Name,
		MaxLines: 1,
		Style:    core.Style{FontSize: 15, FontWeight: core.FontWeightMedium, Color: p.text},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{FontWeight: core.FontWeightSemibold}
			}
			return core.Style{}
		},
	}
	textChildren := []core.Node{title}
	if artists := track.ArtistNames(); artists != "" {
		artist := label(artists, 12, core.FontWeightNormal, p.muted)
		artist.MaxLines = 1
		textChildren = append(textChildren, artist)
	}
	info := core.View{
		Style:    core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 4},
		Children: textChildren,
	}

	card := base.Button{
		Label:   track.Name,
		TestID:  "home-track-" + fmt.Sprint(index),
		OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayTrack(tracks, index) })() },
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12,
			Padding: core.All(6), CornerRadius: 10,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Background: p.chip}
			}
			return core.Style{}
		},
		Children: []core.Node{thumb, info},
	}
	return core.View{
		OnContextMenu: func(event core.ContextMenuEvent) { m.trackOptions(cx, track, index, list, event.PagePosition) },
		Children:      []core.Node{card},
	}
}

func (m *model) tile(cx *core.Scope, src music.Source, name, sub, image string, r route, playable bool) core.Node {
	p := colors(cx)
	const coverRadius = core.Points(15)
	smooth := core.Curve{X1: 0.22, Y1: 1, X2: 0.36, Y2: 1}
	hovered := core.NewSignal(cx, false)

	coverBtn := base.Button{
		Label:   name,
		OnPress: func(core.PressEvent) { m.navigate(r) },
		Style: core.Style{
			Width: core.Pct(100), Height: core.Pct(100), CornerRadius: coverRadius,
			Transition: core.Transition{Duration: 240 * time.Millisecond, Easing: smooth},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.985}},
		},
		StyleFunc: func() core.Style {
			if hovered.Get() {
				shadowAlpha := float32(.14)
				if cx.Dark() {
					shadowAlpha = .38
				}
				return core.Style{
					Transform:    core.Transform{Scale: 1.02},
					ShadowColor:  core.RGBA(0x000000, shadowAlpha),
					ShadowOffset: core.Point{Y: 6},
					ShadowRadius: 16,
				}
			}
			return core.Style{Transform: core.Transform{Scale: 1}}
		},
		Children: []core.Node{m.tileArtwork(cx, name, image, m.source.GetUntracked())},
	}
	coverChildren := []core.Node{coverBtn}
	if playable {
		badgeFill, badgeInk := core.RGB(0x191919), core.RGB(0xffffff)
		if cx.Dark() {
			badgeFill, badgeInk = core.RGB(0xf4f4f5), core.RGB(0x141416)
		}
		// LibraryGridItem PlayBadge at ~0.5x: 42×42pt, r=13pt, overhang right -6pt / bottom -3pt.
		playBtn := base.Button{
			Label:    i18n.T("Play") + " " + name,
			Disabled: func() bool { return !hovered.Get() },
			OnPress:  func(core.PressEvent) { m.playCollection(src, r, false) },
			Style: core.Style{
				Position: core.Absolute, Right: core.Pt(-6), Bottom: core.Pt(-3),
				Width: core.Pt(42), Height: core.Pt(42), CornerRadius: 13,
				Background: badgeFill, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
				ShadowColor: core.RGBA(0x000000, .24), ShadowOffset: core.Point{Y: 4}, ShadowRadius: 12,
				Transition: core.Transition{Duration: 220 * time.Millisecond, Easing: smooth},
				Hover:      &core.Style{Transform: core.Transform{Scale: 1.07}},
				Active:     &core.Style{Transform: core.Transform{Scale: 0.93}},
			},
			StyleFunc: func() core.Style {
				if hovered.Get() {
					return core.Style{
						Opacity:   core.Alpha(1),
						Transform: core.Transform{Scale: 1, TranslateY: 0},
					}
				}
				return core.Style{
					Opacity:       core.Alpha(0),
					Transform:     core.Transform{Scale: 0.82, TranslateY: 5},
					PointerEvents: core.PointerEventsNone,
				}
			},
			Children: []core.Node{ui.Icon(func() *core.SVG { return playFillSVG }, 14, func() core.Color { return badgeInk })},
		}
		coverChildren = append(coverChildren, playBtn)
	}
	coverBox := core.View{
		Style:        core.Style{Width: core.Pct(100), AspectRatio: 1},
		OnMouseEnter: func(core.MouseEvent) { hovered.Set(true) },
		OnMouseLeave: func(core.MouseEvent) { hovered.Set(false) },
		Children:     coverChildren,
	}

	titleColor := core.RGB(0x191919)
	subColor := core.RGB(0xa6a6a6)
	if cx.Dark() {
		titleColor = p.text
		subColor = p.muted
	}
	title := label(name, 13, core.FontWeightSemibold, titleColor)
	title.MaxLines = 1
	titleBtn := base.Button{
		Label:   name,
		OnPress: func(core.PressEvent) { m.navigate(r) },
		Style: core.Style{
			AlignItems: core.AlignStart,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Opacity: core.Alpha(.72)},
		},
		Children: []core.Node{title},
	}

	var metaChildren []core.Node
	if sub != "" {
		subLabel := label(sub, 12, core.FontWeightNormal, subColor)
		subLabel.MaxLines = 1
		subLabel.Style.FlexGrow = 1
		metaChildren = append(metaChildren, subLabel)
	} else {
		metaChildren = append(metaChildren, spacer())
	}
	menuRef := &core.Ref{}
	openMenu := func(at core.Point) {
		var items []desktopItem
		if playable {
			items = append(items,
				desktopItem{label: i18n.T("Play"), action: func() { m.playCollection(src, r, false) }},
				desktopItem{label: i18n.T("Shuffle"), action: func() { m.playCollection(src, r, true) }},
			)
		}
		items = append(items, desktopItem{label: i18n.T("Open"), action: func() { m.navigate(r) }})
		trackMenu(cx, items, at)
	}
	moreBtn := base.Button{
		Ref:   menuRef,
		Label: i18n.T("More"),
		Style: core.Style{
			Width: core.Pt(22), Height: core.Pt(22), CornerRadius: 5,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: p.hover},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.92}},
		},
		Children: []core.Node{ui.Icon(lucide.Ellipsis, 14, func() core.Color { return subColor })},
		OnPress: func(event core.PressEvent) {
			at := event.PagePosition
			if bounds, ok := menuRef.Bounds(); ok {
				at = core.Point{X: bounds.Origin.X, Y: bounds.Origin.Y + bounds.Size.Height}
			}
			openMenu(at)
		},
	}
	metaChildren = append(metaChildren, moreBtn)
	metaRow := row(metaChildren...)
	metaRow.Style.Gap = 4

	return core.View{
		Style:         core.Style{Width: core.Pct(100), Gap: 10},
		OnContextMenu: func(event core.ContextMenuEvent) { openMenu(event.PagePosition) },
		Children:      []core.Node{coverBox, core.View{Style: core.Style{Gap: 3}, Children: []core.Node{titleBtn, metaRow}}},
	}
}

func (m *model) trackList(cx *core.Scope, tracks []music.Track, list *music.Playlist, queue bool) core.List {
	return core.List{Count: func() int { return len(tracks) }, EstimatedSize: 44, Style: fill,
		Key: func(i int) any { return fmt.Sprintf("%d:%s:%s", i, tracks[i].Source, tracks[i].ID) },
		Row: func(cx *core.Scope, cell core.ListRow) core.Node {
			return core.Dynamic(func(cx *core.Scope) core.Node {
				i := cell.Index()
				if i < 0 || i >= len(tracks) {
					return nil
				}
				return m.songRow(cx, tracks[i], i, list, queue, func() {
					if queue {
						m.player.Jump(i)
					} else {
						m.player.PlayTrack(tracks, i)
					}
				})
			})
		},
	}
}

func (m *model) songRow(cx *core.Scope, track music.Track, index int, list *music.Playlist, queue bool, play func()) core.Node {
	p := colors(cx)
	isPlaying := func() bool {
		cur := m.playing.Get().track
		return cur != nil && cur.ID == track.ID && cur.Source == track.Source
	}
	num := core.Text{
		ContentFunc: func() string {
			if isPlaying() && m.playing.Get().playing {
				return "▶"
			}
			return fmt.Sprintf("%02d", index+1)
		},
		Style: core.Style{Width: core.Pt(24), FontSize: 12, FontVariant: core.FontTabularNums, Color: p.muted},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Color: p.text}
			}
			return core.Style{}
		},
	}
	title := core.Text{
		Content:  track.Name,
		MaxLines: 1,
		Style:    core.Style{FontSize: 14, FontWeight: core.FontWeightNormal, Color: p.muted, FlexShrink: 1},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Color: p.text, FontWeight: core.FontWeightMedium}
			}
			return core.Style{}
		},
	}
	rowItems := []core.Node{num, title}
	if artists := track.ArtistNames(); artists != "" {
		artistText := label(artists, 12, core.FontWeightNormal, p.text)
		artistText.MaxLines = 1
		artistChip := core.View{
			Style:    core.Style{Background: p.chip, Padding: core.Symmetric(2, 6), CornerRadius: 0, FlexShrink: 1},
			Children: []core.Node{artistText},
		}
		rowItems = append(rowItems, artistChip)
	}
	rowItems = append(rowItems, spacer())
	dur := label(formatTime(track.Length()), 12, core.FontWeightNormal, p.muted)
	dur.Style.FontVariant = core.FontTabularNums
	rowItems = append(rowItems, dur)

	button := base.Button{
		Label:   track.Name,
		TestID:  "track-" + fmt.Sprint(index),
		OnPress: func(core.PressEvent) { m.playback(play)() },
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10,
			FlexGrow: 1, FlexShrink: 1, Height: core.Pt(40),
			Padding: core.Symmetric(0, 8), CornerRadius: 2,
			Hover: &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Background: p.chip}
			}
			return core.Style{}
		},
		Children: rowItems,
	}
	menuRef := &core.Ref{}
	entry := row(button)
	source := m.sourceByID(track.Source)
	if canLike(source, track) {
		like := iconButton(lucide.Heart, i18n.T("Like Song"), "like-"+fmt.Sprint(index), func() { m.toggleLike(track) })
		like.Size, like.IconSize = 28, 14
		if source == m.lib {
			like.Tint = func() core.Color {
				m.updated.Get()
				liked, _ := m.lib.IsLiked([]string{track.ID})
				if liked[track.ID] {
					return p.text
				}
				return p.muted
			}
		}
		entry.Children = append(entry.Children, like)
	}
	entry.Children = append(entry.Children, base.Button{Ref: menuRef, Label: i18n.T("Track Options"), TestID: "track-menu-" + fmt.Sprint(index),
		Style:    core.Style{Width: core.Pt(28), Height: core.Pt(28), AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, CornerRadius: 2, Hover: &core.Style{Background: p.hover}},
		Children: []core.Node{ui.Icon(lucide.Ellipsis, 14, func() core.Color { return p.muted })},
		OnPress: func(event core.PressEvent) {
			at := event.PagePosition
			if bounds, ok := menuRef.Bounds(); ok {
				at = core.Point{X: bounds.Origin.X, Y: bounds.Origin.Y + bounds.Size.Height}
			}
			m.trackOptions(cx, track, index, list, at)
		}})
	entry.OnContextMenu = func(event core.ContextMenuEvent) { m.trackOptions(cx, track, index, list, event.PagePosition) }
	entry.Style.Gap = 2
	if queue {
		entry.Style.Padding = core.Symmetric(1, 20)
		return entry
	}
	entry.Style.Padding = core.Symmetric(1, 0)
	return paperWrap(entry)
}

func canLike(src music.Source, track music.Track) bool {
	if src == nil {
		return false
	}
	if track.ShowID != "" {
		return src.Can("likeEpisode")
	}
	return src.Can("like")
}

func (m *model) toggleLike(track music.Track) {
	src := m.sourceByID(track.Source)
	if !canLike(src, track) {
		return
	}
	go func() {
		liked, err := src.IsLiked([]string{track.ID})
		if errors.Is(err, music.ErrUnsupported) {
			err = nil
		}
		if err == nil {
			send := src.Like
			if track.ShowID != "" {
				send = src.LikeEpisode
			}
			err = send(track.ID, !liked[track.ID])
		}
		m.cx.Post(func() {
			if errors.Is(err, music.ErrSignIn) && src.Can("signIn") {
				m.changeAccount(src, true)
				return
			}
			m.report(err)
			if err == nil {
				m.invalidate()
				if src != m.lib && m.page.GetUntracked().page == "liked" {
					m.sourceVersion.Update(func(n int) int { return n + 1 })
				}
			}
		})
	}()
}

func (m *model) trackOptions(cx *core.Scope, track music.Track, index int, list *music.Playlist, at core.Point) {
	src := m.sourceByID(track.Source)
	if src == nil {
		return
	}
	var items []desktopItem
	if track.Album.ID != "" {
		items = append(items, desktopItem{label: i18n.T("Go to Album"), action: func() { m.revealSourceTrack(track.Source, route{page: "album", id: track.Album.ID}) }})
	}
	for _, artist := range track.Artists {
		if artist.ID != "" {
			items = append(items, desktopItem{label: artist.Name, action: func() { m.revealSourceTrack(track.Source, route{page: "artist", id: artist.ID}) }})
		}
	}
	if track.Source == local.ID || track.Source == "" {
		playlists, _ := m.lib.Playlists()
		for _, target := range playlists {
			items = append(items, desktopItem{label: i18n.T("Add to ") + target.Name, action: func() {
				m.run(func() error { return m.lib.AddToPlaylist(target.ID, []string{track.ID}) })
			}})
		}
		if list != nil && list.Mine && !list.Fixed {
			items = append(items, desktopItem{label: i18n.T("Remove from Playlist"), action: func() { m.run(func() error { return m.lib.RemoveFromPlaylist(list.ID, []string{track.ID}) }) }})
			if index > 0 {
				items = append(items, desktopItem{label: i18n.T("Move Up"), action: func() { m.run(func() error { return m.lib.MovePlaylistTrack(list.ID, index, index-1) }) }})
			}
			if index+1 < len(list.Tracks) {
				items = append(items, desktopItem{label: i18n.T("Move Down"), action: func() { m.run(func() error { return m.lib.MovePlaylistTrack(list.ID, index, index+1) }) }})
			}
		}
	}
	if src != m.lib {
		items = append(items, m.sourceTrackActions(cx, src, track, index, list, at)...)
	}
	trackMenu(cx, items, at)
}

func withSource(tracks []music.Track, source string) []music.Track {
	result := append([]music.Track(nil), tracks...)
	for i := range result {
		result[i].Source = source
	}
	return result
}

func formatTime(d time.Duration) string {
	n := max(int(d/time.Second), 0)
	if n >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", n/3600, n/60%60, n%60)
	}
	return fmt.Sprintf("%d:%02d", n/60, n%60)
}

func (m *model) editPlaylist(cx *core.Scope, src music.Source, id string) core.Node {
	var initial string
	if id != "" && src == m.lib {
		if p, _ := m.lib.Playlist(id); p != nil {
			initial = p.Name
		}
	}
	name := core.NewSignal(cx, initial)
	ready := core.NewSignal(cx, id == "" || src == m.lib)
	busy := core.NewSignal(cx, false)
	if !ready.GetUntracked() {
		go func() {
			p, err := src.Playlist(id)
			cx.Post(func() {
				if m.sourceActionResult(src, err) && p != nil && p.Mine && !p.Fixed {
					name.Set(p.Name)
					ready.Set(true)
				}
			})
		}()
	}
	title := i18n.T("New Playlist")
	if id != "" {
		title = i18n.T("Edit Playlist")
	}
	save := func() {
		value := strings.TrimSpace(name.Get())
		if value == "" || busy.Get() || !ready.Get() {
			return
		}
		busy.Set(true)
		go func() {
			target := id
			var err error
			if target == "" {
				var p *music.Playlist
				p, err = src.CreatePlaylist(value)
				if p != nil {
					target = p.ID
				}
			} else {
				err = src.RenamePlaylist(target, value)
			}
			cx.Post(func() {
				busy.Set(false)
				if m.sourceActionResult(src, err) && target != "" {
					m.navigate(route{page: "playlist", id: target})
				}
			})
		}()
	}
	saveButton := button(i18n.T("Save"), "save-playlist", lucide.Check, save)
	saveButton.Disabled = func() bool { return !ready.Get() || busy.Get() || strings.TrimSpace(name.Get()) == "" }
	actions := row(saveButton, button(i18n.T("Cancel"), "cancel-playlist", nil, m.goBack))
	if id != "" && src.Can("deletePlaylist") {
		remove := button(i18n.T("Delete Playlist"), "delete-playlist", lucide.Trash, func() {
			confirm(cx, i18n.T("Delete Playlist?"), name.Get(), func() {
				busy.Set(true)
				go func() {
					err := src.DeletePlaylist(id)
					cx.Post(func() {
						busy.Set(false)
						if m.sourceActionResult(src, err) {
							m.navigate(route{page: "home"})
						}
					})
				}()
			})
		})
		remove.Disabled = func() bool { return !ready.Get() || busy.Get() }
		actions.Children = append(actions.Children, remove)
	}
	form := core.View{Style: fill.Merge(core.Style{Padding: core.Edges{Top: detailHeaderHeight}}), Children: []core.Node{m.pageHeader(cx, title, "", ""),
		paperWrap(core.View{Style: core.Style{Gap: 14, MaxWidth: core.Pt(500)}, Children: []core.Node{
			ui.Input{Value: name.Get, OnChange: name.Set, OnSubmit: func(string) { save() }, Disabled: func() bool { return !ready.Get() || busy.Get() },
				Placeholder: i18n.T("Playlist Name"), TestID: "playlist-name"}, actions,
		}}),
	}}
	return m.withDetailHeader(cx, form, paperWrap)
}
