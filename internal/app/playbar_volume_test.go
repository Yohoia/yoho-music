package app

import (
	"math"
	"testing"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/nsgotest"
)

func TestPlaybarVolumeTrackClick(t *testing.T) {
	dir := testLibrary(t)
	var m *model
	app := nsgotest.Start(t, func(cx *core.Scope) core.Node {
		m = newModel(cx, dir)
		return m.root(cx)
	}, nsgotest.Size(1180, 780))
	app.WaitFor("id:track-0", 5*time.Second)
	app.Press("id:track-0")
	app.WaitFor("id:volume-toggle", time.Second)
	app.Advance(350 * time.Millisecond)

	app.Press("id:volume-toggle")
	app.Advance(250 * time.Millisecond)

	checkVolume := func(want float64) {
		t.Helper()
		app.Advance(250 * time.Millisecond)
		if track := app.Find("id:volume"); track.Parent.Style.Opacity != 1 {
			t.Fatal("adjusting the volume closed the popup")
		}
		var got float64
		app.Post(func() { got = m.prefs.GetUntracked().Volume })
		app.Settle()
		if math.Abs(got-want) > .01 {
			t.Fatalf("volume = %.2f, want %.2f", got, want)
		}
	}

	app.PressAt("id:volume", core.Point{X: 20, Y: 57})
	checkVolume(.5)
	app.PressAt("id:volume", core.Point{X: 20, Y: 82})
	checkVolume(.25)
	app.Pan("id:volume", core.Point{Y: -24.5})
	checkVolume(.5)
}
