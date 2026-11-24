package app

import (
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"yoho-music/internal/player"
)

func cls(name string) objc.ID { return objc.ID(objc.GetClass(name)) }

func send(target objc.ID, name string, args ...any) objc.ID {
	return target.Send(objc.RegisterName(name), args...)
}

func exportedString(symbol uintptr) objc.ID { return **(**objc.ID)(unsafe.Pointer(&symbol)) }

func (m *model) installMediaControls() {
	type command struct {
		name string
		run  func(objc.ID)
	}
	play := func(want bool) func(objc.ID) {
		return func(objc.ID) {
			if m.player.Playing() != want {
				m.player.TogglePlay()
			}
		}
	}
	commands := []command{
		{"playCommand", play(true)}, {"pauseCommand", play(false)},
		{"togglePlayPauseCommand", func(objc.ID) { m.player.TogglePlay() }},
		{"nextTrackCommand", func(objc.ID) { m.player.Next() }},
		{"previousTrackCommand", func(objc.ID) { m.player.Prev() }},
		{"changePlaybackPositionCommand", nil},
	}
	var center, remote objc.ID
	var targets [][2]objc.ID
	var queue, lib uintptr
	var initOnce sync.Once
	var dispatch func(uintptr, uintptr, uintptr)
	var pending sync.Mutex
	var work []func()
	onMain := func(f func()) {
		pending.Lock()
		work = append(work, f)
		pending.Unlock()
	}
	system, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY)
	if err != nil {
		return
	}
	queue, err = purego.Dlsym(system, "_dispatch_main_q")
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&dispatch, system, "dispatch_async_f")
	drain := purego.NewCallback(func(uintptr) {
		pending.Lock()
		jobs := work
		work = nil
		pending.Unlock()
		pool := send(send(cls("NSAutoreleasePool"), "alloc"), "init")
		defer send(pool, "drain")
		for _, run := range jobs {
			run()
		}
	})
	post := func(f func()) { onMain(f); dispatch(queue, 0, drain) }
	initialize := func() {
		lib, _ = purego.Dlopen("/System/Library/Frameworks/MediaPlayer.framework/MediaPlayer", purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if lib == 0 {
			return
		}
		center = send(cls("MPNowPlayingInfoCenter"), "defaultCenter")
		remote = send(cls("MPRemoteCommandCenter"), "sharedCommandCenter")
		for _, cmd := range commands {
			obj := send(remote, cmd.name)
			handler := objc.NewBlock(func(_ objc.Block, event objc.ID) int64 {
				var position time.Duration
				if cmd.run == nil {
					seconds := objc.Send[float64](event, objc.RegisterName("positionTime"))
					position = time.Duration(seconds * float64(time.Second))
				}
				m.cx.Post(func() {
					if cmd.run == nil {
						m.seek(position)
						return
					} else {
						cmd.run(0)
					}
					m.refreshPlayback()
				})
				return 0
			})
			target := send(obj, "addTargetWithHandler:", uintptr(handler))
			handler.Release()
			targets = append(targets, [2]objc.ID{obj, target})
		}
	}
	m.player.PublishMedia = func(state player.MediaState) {
		post(func() {
			if state.Track == nil && center == 0 {
				return
			}
			initOnce.Do(initialize)
			if center == 0 {
				return
			}
			for i, pair := range targets {
				send(pair[0], "setEnabled:", state.Track != nil && (i != len(targets)-1 || state.Track.Duration > 0))
			}
			if state.Track == nil {
				send(center, "setNowPlayingInfo:", objc.ID(0))
				send(center, "setPlaybackState:", 3)
				return
			}
			dict := send(cls("NSMutableDictionary"), "dictionary")
			put := func(name string, value objc.ID) {
				if key, err := purego.Dlsym(lib, name); err == nil && value != 0 {
					// The symbol stores a pointer to an NSString constant.
					send(dict, "setObject:forKey:", value, exportedString(key))
				}
			}
			text := func(text string) objc.ID { return send(cls("NSString"), "stringWithUTF8String:", text) }
			number := func(n float64) objc.ID { return send(cls("NSNumber"), "numberWithDouble:", n) }
			put("MPMediaItemPropertyTitle", text(state.Track.Name))
			put("MPMediaItemPropertyArtist", text(state.Track.ArtistNames()))
			put("MPMediaItemPropertyAlbumTitle", text(state.Track.Album.Name))
			put("MPMediaItemPropertyPlaybackDuration", number(state.Track.Length().Seconds()))
			put("MPNowPlayingInfoPropertyElapsedPlaybackTime", number(state.Position.Seconds()))
			playbackState, rate := 2, 0.0
			if state.Playing {
				playbackState, rate = 1, 1
			}
			put("MPNowPlayingInfoPropertyPlaybackRate", number(rate))
			send(center, "setNowPlayingInfo:", dict)
			send(center, "setPlaybackState:", playbackState)
		})
	}
	m.cx.Cleanup(func() {
		post(func() {
			if center == 0 {
				return
			}
			for _, pair := range targets {
				send(pair[0], "removeTarget:", pair[1])
			}
			send(center, "setNowPlayingInfo:", objc.ID(0))
		})
	})
	m.player.PublishMedia(player.MediaState{Track: m.player.Current(), Position: m.player.Position(), Playing: m.player.Playing()})
}
