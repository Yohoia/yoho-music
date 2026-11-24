package app

import (
	"fmt"
	"yoho-music/internal/i18n"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

func (m *model) cacheSettings(cx *core.Scope) core.Node {
	cache := m.player.Cache
	if cache == nil {
		return nil
	}
	used := core.NewSignal(cx, int64(-1))
	busy := core.NewSignal(cx, false)
	measure := func(work func()) {
		busy.Set(true)
		go func() {
			if work != nil {
				work()
			}
			size := cache.Size()
			cx.Post(func() {
				used.Set(size)
				busy.Set(false)
			})
		}()
	}
	measure(nil)
	title := i18n.T("Audio Cache")
	limit := ui.Select[int]{Label: title, TestID: "audio-cache-limit", Disabled: busy.Get,
		Options: []ui.Option[int]{{Value: 0, Label: i18n.T("Off")}, {Value: 512, Label: "512 MB"},
			{Value: 1024, Label: "1 GB"}, {Value: 2048, Label: "2 GB"}, {Value: 5120, Label: "5 GB"}},
		Value: func() int { return m.prefs.Get().CacheMB },
		OnChange: func(mb int) {
			if busy.GetUntracked() {
				return
			}
			m.updatePreferences(cx, func(p *preferences) { p.CacheMB = mb })
			measure(func() { cache.SetLimit(int64(mb) << 20) })
		},
	}
	clear := button(i18n.T("Clear Cache"), "audio-cache-clear", lucide.Trash, func() {
		confirm(cx, i18n.T("Clear Audio Cache?"),
			i18n.T("Cached tracks will download again when played online."),
			func() {
				if !busy.GetUntracked() {
					measure(cache.Clear)
				}
			})
	})
	clear.Disabled = func() bool { return busy.Get() || used.Get() <= 0 }
	size := core.Text{ContentFunc: func() string {
		if used.Get() < 0 {
			return ""
		}
		return fmt.Sprintf(i18n.T("%.0f MB used"), float64(used.Get())/(1<<20))
	}, Style: core.Style{FontSize: 12, Color: colors(cx).muted}}
	return settingsRow(cx, title, row(size, limit, clear))
}
