package plugin

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"

	"github.com/nsgo-dev/nsgo/tsx/pkg"
)

const demoPackageID = "dev.yoho.music.market-demo"
const DemoSourceID = NSGoSourcePrefix + demoPackageID
const netEasePackageID = "dev.yoho.music.netease"
const NetEaseSourceID = NSGoSourcePrefix + netEasePackageID
const demoAudioURL = "yoho-demo://audio/tone.mp3"

const demoPackageSHA256 = "4cf7938d98b5d477ebbf32e65c89ae84e92782e06e5e87cb738b7dad47c24c7e"

//go:embed bundled/root.pub
var demoRoot string

//go:embed bundled/tone.mp3
var demoAudio []byte

//go:embed bundled/netease/root.pub
var netEaseRoot string

func (s *NSGoStore) verify(data []byte) (*pkg.Package, error) {
	p, err := s.trust.Verify(data)
	if err != nil {
		return nil, err
	}
	if p.App.ID == demoPackageID {
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != demoPackageSHA256 {
			return nil, errors.New("the Yoho Demo ID is reserved for the official demo package")
		}
	}
	certificate, err := p.Certificate.Info()
	if err != nil {
		return nil, err
	}
	if (p.App.ID == netEasePackageID) != (certificate.Signer == s.netEaseSigner) {
		return nil, errors.New("the bundled NetEase signing root is reserved for the NetEase plugin")
	}
	return p, nil
}
