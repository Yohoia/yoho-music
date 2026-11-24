package app

import (
	"structs"

	"github.com/ebitengine/purego/objc"
	"github.com/nsgo-dev/nsgo/core"
)

type screenInsets struct {
	_                        structs.HostLayout
	Top, Left, Bottom, Right float64
}

func sampleNotch(cx *core.Scope, done func(notchScreen)) {
	if err := loadAppKit(); err != nil {
		done(notchScreen{})
		return
	}
	appKitThread.post(func() {
		sample := readNotchScreen()
		cx.Post(func() { done(sample) })
	})
}

func readNotchScreen() notchScreen {
	screens := send(cls("NSScreen"), "screens")
	if objc.Send[uint](screens, objc.RegisterName("count")) == 0 {
		return notchScreen{}
	}
	screen := send(screens, "firstObject")
	frame := objc.Send[appKitRect](screen, objc.RegisterName("frame"))
	visible := objc.Send[appKitRect](screen, objc.RegisterName("visibleFrame"))
	top := frame.Origin.Y + frame.Size.Height
	center := frame.Origin.X + frame.Size.Width/2
	cursor := objc.Send[appKitPoint](cls("NSEvent"), objc.RegisterName("mouseLocation"))
	info := notchScreen{
		cursor: core.Point{X: core.Points(cursor.X - center), Y: core.Points(top - cursor.Y)},
		width:  200, height: core.Points(max(22, min(40, top-visible.Origin.Y-visible.Size.Height))),
	}
	if objc.Send[bool](screen, objc.RegisterName("respondsToSelector:"), objc.RegisterName("safeAreaInsets")) {
		insets := objc.Send[screenInsets](screen, objc.RegisterName("safeAreaInsets"))
		if insets.Top > 0 {
			left := objc.Send[appKitRect](screen, objc.RegisterName("auxiliaryTopLeftArea"))
			right := objc.Send[appKitRect](screen, objc.RegisterName("auxiliaryTopRightArea"))
			info.real = true
			info.height = core.Points(insets.Top)
			info.width = core.Points(max(140, min(300, frame.Size.Width-left.Size.Width-right.Size.Width)))
		}
	}
	app := send(cls("NSApplication"), "sharedApplication")
	windows := send(app, "windows")
	for i := range objc.Send[uint](windows, objc.RegisterName("count")) {
		window := send(windows, "objectAtIndex:", i)
		if cocoaString(send(window, "title")) != notchTitle {
			continue
		}
		const aboveMenu = 27
		if objc.Send[int64](window, objc.RegisterName("level")) != aboveMenu {
			send(window, "setLevel:", int64(aboveMenu))
		}
		want := appKitRect{Origin: appKitPoint{X: center - notchWidth/2, Y: top - notchHeight},
			Size: appKitSize{Width: notchWidth, Height: notchHeight}}
		if objc.Send[appKitRect](window, objc.RegisterName("frame")) != want {
			send(window, "setFrame:display:", want, true)
		}
		info.positioned = true
		break
	}
	return info
}
