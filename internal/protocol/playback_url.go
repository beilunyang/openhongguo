package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"strings"
)

// playbackURL 统一处理明文、Base64 和模型 key_seed 保护的地址。
func playbackURL(value, seed string) (string, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return value, nil
	}
	raw, err := decodePlaybackBase64(value)
	if err != nil || len(raw) == 0 {
		return "", fmt.Errorf("playback URL is not encoded data")
	}
	if raw[0] == 0xa8 {
		if len(raw) < 20 || raw[2] != 1 || raw[3] != 0 || (len(raw)-4)%aes.BlockSize != 0 {
			return "", fmt.Errorf("playback URL envelope is invalid")
		}
		seedBytes, err := decodePlaybackBase64(seed)
		if err != nil || len(seedBytes) == 0 {
			return "", fmt.Errorf("playback URL seed is missing or invalid")
		}
		constants := []byte{
			0x4d, 0xd4, 0xc2, 0xe6, 0xb8, 0x31, 0x62, 0x09, 0x0e, 0x52, 0xb3, 0xc7, 0xa6, 0x73, 0x3b, 0xa4,
			0x1c, 0xb2, 0x46, 0x2b, 0x82, 0x9a, 0xb5, 0x8a, 0x19, 0x6b, 0x39, 0xdb, 0x57, 0x17, 0x75, 0x24,
			0xf4, 0x9b, 0xaf, 0x7f, 0x08, 0xe8, 0xd6, 0x8d, 0x26, 0xa7, 0x2e, 0x37, 0xc1, 0xa9, 0x5a, 0x2f,
			0x1f, 0x05, 0xa5, 0x18, 0x92, 0xae, 0xf2, 0x94, 0x97, 0x32, 0xb6, 0x2a, 0x38, 0xaa, 0xdd, 0x58,
		}
		first := sha512.Sum512(seedBytes)
		key := sha512.Sum512(append(first[:], constants...))
		block, err := aes.NewCipher(key[:16])
		if err != nil {
			return "", err
		}
		plaintext := make([]byte, len(raw)-4)
		cipher.NewCBCDecrypter(block, key[16:32]).CryptBlocks(plaintext, raw[4:])
		padding := int(plaintext[len(plaintext)-1])
		if padding > 0 && padding <= aes.BlockSize {
			plaintext = plaintext[:len(plaintext)-padding]
		}
		raw = plaintext
	}
	decoded := strings.TrimSpace(strings.TrimRight(string(raw), "\x00"))
	if !strings.HasPrefix(decoded, "https://") && !strings.HasPrefix(decoded, "http://") {
		return "", fmt.Errorf("playback URL has no HTTP(S) address")
	}
	return decoded, nil
}

func decodePlaybackBase64(value string) ([]byte, error) {
	value = strings.NewReplacer("+", "-", "/", "_").Replace(strings.TrimSpace(value))
	return base64.RawURLEncoding.DecodeString(strings.TrimRight(value, "="))
}
