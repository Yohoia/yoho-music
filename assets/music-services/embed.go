package musicservices

import (
	_ "embed"

	"github.com/nsgo-dev/nsgo/core"
)

//go:embed netease-cloud-music.svg
var netEaseSource []byte

var netEase = core.MustParseSVG(netEaseSource)

func NetEase() *core.SVG { return netEase }
