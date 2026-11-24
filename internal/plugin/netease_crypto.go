package plugin

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/nsgo-dev/nsgo/core"
)

const netEaseEncodeHandler = "yoho.neteaseEncode"

// The encoding is part of NetEase's desktop request format. Session
// confidentiality comes from HTTPS and the NSGo secret store.
func encodeNetEaseCall(call *core.HostCall) {
	data, _ := call.Data.(map[string]any)
	path, _ := data["path"].(string)
	payload, _ := data["payload"].(string)
	if !strings.HasPrefix(path, "/api/") || len(path) > 256 || len(payload) == 0 || len(payload) > 512<<10 {
		call.Reject(errors.New("invalid NetEase request"))
		return
	}
	sum := md5.Sum([]byte("nobody" + path + "use" + payload + "md5forencrypt"))
	input := strings.Join([]string{path, payload, hex.EncodeToString(sum[:])}, "-36cd479b6b5-")
	encoded, err := aesECB([]byte("e82ckenh8dichen8"), []byte(input))
	if err != nil {
		call.Reject(err)
		return
	}
	call.Resolve(strings.ToUpper(hex.EncodeToString(encoded)))
}

// aesECB is adapted from MyGo Music v0.1.25, internal/plugin/host.go (AGPL-3.0).
func aesECB(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	pad := aes.BlockSize - len(text)%aes.BlockSize
	text = append(append([]byte(nil), text...), bytes.Repeat([]byte{byte(pad)}, pad)...)
	out := make([]byte, len(text))
	for i := 0; i < len(text); i += aes.BlockSize {
		block.Encrypt(out[i:i+aes.BlockSize], text[i:i+aes.BlockSize])
	}
	return out, nil
}
