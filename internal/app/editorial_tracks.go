package app

import (
	"fmt"
	"time"

	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
	"yoho-music/internal/i18n"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
)

func (m *model) editorialTrackPage(cx *core.Scope, src music.Source, d catalogData, tracks []music.Track, footer core.Node, topPad core.Points) core.Node {
	p := colors(cx)
	sourceID := src.Info().ID

	sub := d.subtitle
	if sub == "" {
		sub = i18n.Songs(len(tracks))
	}
	titleCol := core.View{
		Style: core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0), Gap: 4},
		Children: []core.Node{
			label(d.title, 24, core.FontWeightBold, p.text),
			label(sub, 12, core.FontWeightNormal, p.muted),
		},
	}

	playBtn := base.Button{
		Label:   i18n.T("Play"),
		TestID:  "play-all",
		OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayQueue(tracks, 0) })() },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(34), Padding: core.Symmetric(0, 18),
			CornerRadius: 17, Background: p.button,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Opacity: core.Alpha(.88)},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.97}},
		},
		Children: []core.Node{
			ui.Icon(lucide.Play, 13, func() core.Color { return p.onButton }),
			label(i18n.T("Play"), 12, core.FontWeightSemibold, p.onButton),
		},
	}

	shuffleBtn := base.Button{
		Label:   i18n.T("Shuffle"),
		TestID:  "shuffle-all",
		OnPress: func(core.PressEvent) { m.playback(func() { m.player.PlayShuffled(tracks) })() },
		Style: core.Style{
			Direction: core.Row, Height: core.Pt(34), Padding: core.Symmetric(0, 16),
			CornerRadius: 17, Background: core.Transparent,
			BorderWidth: 1, BorderColor: p.line,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter, Gap: 6,
			Transition: core.Transition{Duration: 150 * time.Millisecond, Easing: core.EaseOut},
			Hover:      &core.Style{Background: p.hover},
			Active:     &core.Style{Transform: core.Transform{Scale: 0.97}},
		},
		Children: []core.Node{
			ui.Icon(lucide.Shuffle, 13, func() core.Color { return p.text }),
			label(i18n.T("Shuffle"), 12, core.FontWeightMedium, p.text),
		},
	}

	actions := []core.Node{playBtn, shuffleBtn}
	if sourceID != local.ID {
		actions = append(actions, m.sourceCatalogActions(cx, src, d)...)
	}

	headerRow := core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignEnd, Gap: 16,
			Padding: core.Edges{Top: topPad},
		},
		Children: []core.Node{
			titleCol,
			core.View{
				Style:    core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, FlexShrink: 0},
				Children: actions,
			},
		},
	}

	headerChildren := []core.Node{headerRow}
	if d.description != "" {
		desc := label(d.description, 13, core.FontWeightNormal, p.muted)
		desc.Style.Padding = core.Edges{Top: 10}
		headerChildren = append(headerChildren, desc)
	}
	headerChildren = append(headerChildren, core.View{
		Style:    core.Style{Padding: core.Edges{Top: 18, Bottom: 8}},
		Children: []core.Node{dashedDivider(p.line)},
	})

	list := core.List{
		Count: func() int { return len(tracks) }, EstimatedSize: 58, Style: fill,
		Key: func(i int) any { return fmt.Sprintf("%d:%s:%s", i, tracks[i].Source, tracks[i].ID) },
		Row: func(cx *core.Scope, cell core.ListRow) core.Node {
			return core.Dynamic(func(cx *core.Scope) core.Node {
				i := cell.Index()
				if i < 0 || i >= len(tracks) {
					return nil
				}
				return m.editorialTrackRow(cx, tracks[i], i, d.list, func() {
					m.player.PlayTrack(tracks, i)
				})
			})
		},
	}
	list.Header = paperWrap(headerChildren...)

	var footerNodes []core.Node
	if footer != nil {
		footerNodes = append(footerNodes, paperWrap(footer))
	}
	footerNodes = append(footerNodes, core.View{Style: core.Style{Height: core.Pt(108)}})
	list.Footer = core.View{Children: footerNodes}
	return list
}

func (m *model) editorialTrackRow(cx *core.Scope, track music.Track, index int, list *music.Playlist, play func()) core.Node {
	p := colors(cx)
	hovered := core.NewSignal(cx, false)
	isPlaying := func() bool {
		cur := m.playing.Get().track
		return cur != nil && cur.ID == track.ID && cur.Source == track.Source
	}

	num := core.Text{
		ContentFunc: func() string {
			if (isPlaying() && m.playing.Get().playing) || hovered.Get() {
				return "▶"
			}
			return fmt.Sprintf("%02d", index+1)
		},
		Style: core.Style{
			Width: core.Pt(26), FontSize: 12, FontVariant: core.FontTabularNums,
			TextAlign: core.TextAlignCenter, Color: p.muted, FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			if isPlaying() || hovered.Get() {
				return core.Style{Color: p.text, FontWeight: core.FontWeightSemibold}
			}
			return core.Style{}
		},
	}

	thumbFace := core.Style{
		Width: core.Pt(40), Height: core.Pt(40), CornerRadius: 8, FlexShrink: 0,
		Background: p.sidebar, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
	}
	var thumb core.Node
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
	titleStackChildren := []core.Node{
		base.Button{
			Label:   track.Name,
			OnPress: func(core.PressEvent) { m.playback(play)() },
			Style:   core.Style{AlignItems: core.AlignStart},
			Children: []core.Node{title},
		},
	}
	if artists := track.ArtistNames(); artists != "" {
		artistText := core.Text{
			Content:  artists,
			MaxLines: 1,
			Style: core.Style{
				FontSize: 12, FontWeight: core.FontWeightNormal, Color: p.muted,
				Hover: &core.Style{Color: p.text},
			},
		}
		if len(track.Artists) > 0 && track.Artists[0].ID != "" {
			a := track.Artists[0]
			titleStackChildren = append(titleStackChildren, base.Button{
				Label:   a.Name,
				TestID:  "track-artist-" + fmt.Sprint(index),
				OnPress: func(core.PressEvent) { m.openPeek(track.Source, "artist", a.ID, a.Name, a.ImageURL) },
				Style:   core.Style{AlignSelf: core.AlignStart},
				Children: []core.Node{artistText},
			})
		} else {
			titleStackChildren = append(titleStackChildren, artistText)
		}
	}
	titleStack := core.View{
		Style:    core.Style{FlexGrow: 1.4, FlexShrink: 1, FlexBasis: core.Pt(0), MinWidth: core.Pt(0), Gap: 2, Padding: core.Edges{Right: 12}},
		Children: titleStackChildren,
	}

	var albumNode core.Node
	al := core.Text{
		Content:  track.Album.Name,
		MaxLines: 1,
		Style: core.Style{
			FontSize: 12, FontWeight: core.FontWeightNormal, Color: p.muted,
			Hover: &core.Style{Color: p.text},
		},
	}
	if track.Album.ID != "" && track.Album.Name != "" {
		albumNode = base.Button{
			Label:   track.Album.Name,
			TestID:  "track-album-" + fmt.Sprint(index),
			OnPress: func(core.PressEvent) { m.openPeek(track.Source, "album", track.Album.ID, track.Album.Name, track.Album.ImageURL) },
			Style:   core.Style{AlignSelf: core.AlignStart, MaxWidth: core.Pct(100)},
			Children: []core.Node{al},
		}
	} else {
		albumNode = al
	}
	albumCol := core.View{
		Style:    core.Style{FlexGrow: 1, FlexShrink: 1, FlexBasis: core.Pt(0), MinWidth: core.Pt(0), Padding: core.Edges{Right: 12}},
		Children: []core.Node{albumNode},
	}

	dur := label(formatTime(track.Length()), 12, core.FontWeightNormal, p.muted)
	dur.Style.FontVariant = core.FontTabularNums
	dur.Style.Width = core.Pt(44)
	dur.Style.TextAlign = core.TextAlignEnd
	dur.Style.FlexShrink = 0

	leadPlayBtn := base.Button{
		Label:    track.Name,
		TestID:   "track-" + fmt.Sprint(index),
		OnPress:  func(core.PressEvent) { m.playback(play)() },
		Style:    core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12, FlexShrink: 0, Padding: core.Edges{Left: 10}},
		Children: []core.Node{num, thumb},
	}

	mainRow := core.View{
		Style: core.Style{
			Direction: core.Row, AlignItems: core.AlignCenter, Gap: 12,
			FlexGrow: 1, FlexShrink: 1, Height: core.Pt(56),
			Padding: core.Edges{Right: 10}, CornerRadius: 12,
			Hover: &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if isPlaying() {
				return core.Style{Background: p.selected}
			}
			return core.Style{}
		},
		Children: []core.Node{leadPlayBtn, titleStack, albumCol, dur},
	}

	menuRef := &core.Ref{}
	entry := row(mainRow)
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
		Style: core.Style{
			Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Hover: &core.Style{Background: p.hover},
		},
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
	entry.OnMouseEnter = func(core.MouseEvent) { hovered.Set(true) }
	entry.OnMouseLeave = func(core.MouseEvent) { hovered.Set(false) }
	entry.Style.Gap = 2
	entry.Style.Padding = core.Symmetric(1, 0)
	return paperWrap(entry)
}
