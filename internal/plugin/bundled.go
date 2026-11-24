package plugin

import (
	"bytes"
	_ "embed"
	"errors"

	"github.com/nsgo-dev/nsgo/tsx/pkg"
)

const demoPackageID = "dev.yoho.music.market-demo"
const DemoSourceID = NSGoSourcePrefix + demoPackageID
const demoAudioURL = "yoho-demo://audio/tone.mp3"

//go:embed bundled/demo.nsgopkg
var demoPackage []byte

//go:embed bundled/root.pub
var demoRoot string

//go:embed bundled/tone.mp3
var demoAudio []byte

func (s *NSGoStore) verify(data []byte) (*pkg.Package, error) {
	p, err := s.trust.Verify(data)
	if err != nil {
		return nil, err
	}
	if p.App.ID == demoPackageID && !bytes.Equal(data, demoPackage) {
		return nil, errors.New("the Yoho Demo ID is reserved for the bundled package")
	}
	return p, nil
}

// BundledDemo prepares the app's signed demo through the same installer as imports.
func (s *NSGoStore) BundledDemo() (*NSGoPackage, error) {
	return s.prepareInstall(demoPackage)
}
