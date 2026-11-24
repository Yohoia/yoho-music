package app

import (
	"github.com/nsgo-dev/nsgo/core"
	"yoho-music/internal/local"
	"yoho-music/internal/music"
)

func (m *model) coverSource(cx *core.Scope, raw string, size int, source string) func() core.ImageSource {
	address := music.SizedImage(raw, size)
	if source == "" || source == local.ID {
		return func() core.ImageSource { return core.Src(address) }
	}
	result := core.NewSignalFunc(cx, core.ImageSource{}, nil)
	cx.Effect(func() {
		check := music.MediaPolicy(m.sourceByID(source))
		result.Set(core.ImageSource{})
		if check == nil || address == "" {
			return
		}
		ctx := cx.Context()
		go func() {
			data, err := music.FetchMediaImage(ctx, address, nil, check)
			cx.Post(func() {
				if err == nil && ctx.Err() == nil {
					result.Set(core.ImageSource{Data: data})
				}
			})
		}()
	})
	return result.Get
}
