package sign

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SignGET builds the query string and portable signature headers for one GET.
// pairs are encoded in the given order. ts and _rticket are appended.
// A nil entropy draws fresh random fields. Header values are not logged here.
func SignGET(pairs [][2]string, nowMillis int64, deviceID, versionName string, entropy *Entropy) (string, map[string]string) {
	return signRequest("/reading/bookapi/search/tab/v", pairs, nowMillis, deviceID, versionName, nil, entropy)
}

// SignPOST signs one JSON body. The returned headers include the body stub.
// The caller must send the same body bytes that were signed.
func SignPOST(path string, pairs [][2]string, nowMillis int64, deviceID, versionName string, body []byte, entropy *Entropy) (string, map[string]string) {
	return signRequest(path, pairs, nowMillis, deviceID, versionName, body, entropy)
}

func signRequest(path string, pairs [][2]string, nowMillis int64, deviceID, versionName string, body []byte, entropy *Entropy) (string, map[string]string) {
	ent := Entropy{}
	if entropy != nil {
		ent = *entropy
	} else {
		ent = drawEntropy(nowMillis)
	}
	hasBody := body != nil
	base := uint64(nowMillis)
	var query string
	var ticket uint64
	var khronos uint32
	var medusaIn medusaV5Input
	// x-medusa field 13 has 16 variants picked by a hash of the request; this
	// package implements some of them, so step _rticket by a millisecond
	// until the request picks one. A request finds one within a few tries.
	for offset := uint64(0); offset < 256; offset++ {
		ticket = base + offset
		khronos = uint32(ticket / 1000)
		withTicket := append([][2]string{}, pairs...)
		withTicket = append(withTicket,
			[2]string{"ts", strconv.FormatUint(uint64(khronos), 10)},
			[2]string{"_rticket", strconv.FormatUint(ticket, 10)},
		)
		query = encodeQuery(withTicket)
		var ok bool
		if medusaIn, ok = newMedusaV5Input(query, body, khronos); ok {
			break
		}
	}
	if ent.NowSecs == 0 {
		ent.NowSecs = int64(khronos)
	}
	headers := map[string]string{
		"Accept":                    "application/json; charset=utf-8,application/x-protobuf",
		"x-xs-from-web":             "0",
		"x-ss-req-ticket":           strconv.FormatUint(ticket, 10),
		"x-tt-request-tag":          "t=0;n=0",
		"sdk-version":               "2",
		"passport-sdk-version":      "50561",
		"x-vc-bdturing-sdk-version": "3.7.2.cn",
		"x-khronos":                 strconv.FormatUint(uint64(khronos), 10),
		"x-ladon":                   base64.StdEncoding.EncodeToString(be32(khronos)),
		"x-argus":                   base64.StdEncoding.EncodeToString(le32(khronos)),
		"x-gorgon":                  xGorgon(query, body, hasBody, khronos, ent.GorgonRandom),
		"x-helios":                  heliosV5(khronos, ent.HeliosRandom),
		"x-medusa":                  assembleMedusaV5(medusaIn, deviceID, processMedusaEnv(), ent.medusaRandom()),
		"x-tt-dt":                   "",
	}
	if hasBody {
		headers["Content-Type"] = "application/json; charset=UTF-8"
		sum := md5Raw(body)
		headers["x-ss-stub"] = strings.ToUpper(hex.EncodeToString(sum[:]))
	}
	return query, headers
}

func encodeQuery(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, pair := range pairs {
		parts[i] = encodeComponent(pair[0]) + "=" + encodeComponent(pair[1])
	}
	return strings.Join(parts, "&")
}

func encodeComponent(s string) string {
	const unreserved = "-_.!~*'()"
	var b strings.Builder
	for _, c := range []byte(s) {
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || strings.ContainsRune(unreserved, rune(c)) {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

// medusaRandom maps the drawn entropy onto the three rand() values the SDK
// uses. rand() returns 31-bit values.
func (e Entropy) medusaRandom() MedusaRandom {
	return MedusaRandom{Message: e.MessageRandom & 0x7fffffff, Key: e.Random & 0x7fffffff, Packed: e.PackedRandom & 0x7fffffff}
}

var (
	medusaEnvOnce sync.Once
	medusaEnv     MedusaEnv
)

// processMedusaEnv is the environment this process reports, fixed at first use.
func processMedusaEnv() MedusaEnv {
	medusaEnvOnce.Do(func() { medusaEnv = DefaultMedusaEnv(time.Now().UnixMilli()) })
	return medusaEnv
}

func drawEntropy(nowMillis int64) Entropy {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	n := binary.LittleEndian.Uint64(buf[:8])
	return Entropy{
		EnvLaunch:     100 + int64(n%21),
		EnvPID:        10001 + int64(binary.LittleEndian.Uint16(buf[8:10])%2000),
		NowSecs:       nowMillis / 1000,
		MessageRandom: binary.LittleEndian.Uint32(buf[0:4]),
		Random:        binary.LittleEndian.Uint32(buf[4:8]),
		PackedRandom:  binary.LittleEndian.Uint32(buf[8:12]),
		GorgonRandom:  uint32(binary.LittleEndian.Uint16(buf[12:14])),
		HeliosRandom:  binary.LittleEndian.Uint32(buf[12:16]),
	}
}
