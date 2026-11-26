package app

import (
	"github.com/nsgo-dev/nsgo/base"
	"github.com/nsgo-dev/nsgo/core"
	"yoho-music/internal/i18n"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
)

func (m *model) accountAvatarButton(cx *core.Scope, p palette, ref *core.Ref) core.Node {
	type identity struct {
		source music.Source
		url    string
	}
	current := core.NewMemo(cx, func() identity {
		source := m.activeSource()
		if source == nil || source.Info().ID == local.ID {
			return identity{}
		}
		state := m.accountOf(source)
		if state.account == nil {
			return identity{}
		}
		return identity{source: source, url: state.account.ImageURL}
	})
	return base.Button{
		Ref: ref, Label: i18n.T("Sources"), TestID: "source-menu-trigger",
		Expanded: m.sourceMenuOpen.Get,
		OnPress:  func(core.PressEvent) { m.sourceMenuOpen.Update(func(open bool) bool { return !open }) },
		Style: core.Style{
			Width: core.Pt(32), Height: core.Pt(32), CornerRadius: 16,
			AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter,
			Hover: &core.Style{Background: p.hover},
		},
		StyleFunc: func() core.Style {
			if m.sourceMenuOpen.Get() {
				return core.Style{Background: p.selected}
			}
			return core.Style{}
		},
		Children: []core.Node{core.Dynamic(func(cx *core.Scope) core.Node {
			image := core.ImageSource{Data: yohoAvatar}
			avatar := core.Image{
				Source: image, Placeholder: image, TestID: "account-avatar", ElementsHidden: true,
				ResizeMode: core.Cover,
				Style:      core.Style{Width: core.Pt(28), Height: core.Pt(28), CornerRadius: 14, Background: p.sidebar},
			}
			account := current.Get()
			if account.url == "" {
				return avatar
			}
			load := m.coverSource(cx, account.url, 64, account.source.Info().ID)
			failed := core.NewSignal(cx, false)
			avatar.Source = core.ImageSource{}
			avatar.SourceFunc = func() core.ImageSource {
				picture := load()
				if failed.Get() || len(picture.Data) == 0 && picture.URI == "" && picture.Pixels == nil {
					return image
				}
				return picture
			}
			avatar.OnError = func(error) { failed.Set(true) }
			return avatar
		})},
	}
}
