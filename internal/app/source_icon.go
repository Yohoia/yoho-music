package app

import (
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/ui"
	musicservices "yoho-music/assets/music-services"
	"yoho-music/internal/plugin"
)

func sourceIcon(id string, fallback glyph, size core.Points, tint core.Color) core.Node {
	if id == plugin.NetEaseSourceID {
		return core.SVGView{SVG: musicservices.NetEase(), TestID: "netease-brand-icon",
			Style: core.Style{Width: core.Pt(size), Height: core.Pt(size), FlexShrink: 0,
				Background: core.RGB(0xffffff), CornerRadius: size / 2}}
	}
	return ui.Icon(fallback, size, func() core.Color { return tint })
}

func marketPluginArtwork(p palette, entry marketPlugin, size core.Points) core.Node {
	if entry.ID == plugin.NetEaseSourceID {
		return sourceIcon(entry.ID, nil, size, p.accent)
	}
	icon, _ := entry.identity()
	gradient := core.LinearGradient(135, core.RGB(0x475569), core.RGB(0x1e293b))
	if entry.ID == plugin.DemoSourceID {
		gradient = core.LinearGradient(135, core.RGB(0x6366f1), core.RGB(0x312e81))
	}
	return core.View{Style: core.Style{Width: core.Pt(size), Height: core.Pt(size), CornerRadius: 12, FlexShrink: 0,
		Gradients: []core.Gradient{gradient}, AlignItems: core.AlignCenter, JustifyContent: core.JustifyCenter},
		Children: []core.Node{ui.Icon(icon, size*.47, func() core.Color { return core.RGB(0xffffff) })}}
}
