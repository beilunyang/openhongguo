package sign

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"math/bits"
	"net/url"
	"strconv"
)

// The playback model endpoint (/novel/player/video_model/v1/) began rejecting
// the four-byte timestamp stubs this package otherwise sends for x-argus and
// x-ladon: from 2026-11-10 it answers a new anonymous device with HTTP 200 and
// an empty body unless those two headers carry real signatures. Search,
// video_detail and every other read still accept the stubs, so only the
// playback path is signed here; SignPOST / SignGET are left unchanged.
//
// X-Argus is the public "Argus 551" envelope: a protobuf bean is padded and
// run through the Simon 64/128 block cipher, wrapped in a masked and reversed
// frame, then encrypted with standard AES-128-CBC. X-Ladon is a 34-round ARX
// cipher over "<ts>-<licenseID>-<aid>" keyed by md5(nonce || aid). Each request
// draws fresh random nonces, and both headers bind the exact query and body
// that are sent.
//
// This is an independent, clean-room implementation written from the public
// wire protocol and checked only against the fixed golden vectors in
// playback_test.go. No third-party signing code was copied, and no external
// signing service or device identity is introduced. It reuses this package's
// own SM3 (sm3Sum), protobuf varint (protoKey/varint) and MD5 (md5Raw)
// primitives; AES-128-CBC comes from the Go standard library because the Argus
// envelope uses textbook AES, not the modified block cipher in aes.go.

// playbackLicenseID is the video_model license id the official player presents.
const playbackLicenseID = "1611921764"

// Public Argus 551 constants for this app family. The sign key seeds Simon's
// key schedule; the AES key and IV encrypt the final frame.
var (
	argusSignKey = mustHex("fc78e0a9657a0c748ce51559903ccf03510e51d3cff232d71343e88a321c5304")
	argusAESKey  = mustHex("d31e3718288a1027baab59f146a09a9c")
	argusAESIV   = mustHex("ea180a0336ed352fcd24e4d50018ae54")
	argusMask    = []byte{0xf2, 0xf7, 0xfc, 0xff, 0xf2, 0xf7, 0xfc, 0xff}
	argusTail    = []byte{0xea, 0xfb, 0x2c, 0xfe, 0x85, 0x68, 0x51, 0x91, 0x54}
)

// SignPlaybackPOST signs the video_model request. It is SignPOST plus real
// x-argus and x-ladon headers, and it drops x-medusa: when the account-style
// Medusa header rides along with a valid 551 envelope the live service answers
// playback with an empty 200 body, while removing it returns the full model
// (observed 2026-11-10). Account and non-playback requests keep their Medusa.
func SignPlaybackPOST(path string, pairs [][2]string, nowMillis int64, deviceID, versionName string, body []byte, entropy *Entropy) (string, map[string]string) {
	query, headers := signRequest(path, pairs, nowMillis, deviceID, versionName, body, entropy)
	// signRequest may bump khronos by a few milliseconds to pick an F13 branch,
	// so read back the timestamp it actually put in the query.
	khronos, _ := strconv.ParseUint(headers["x-khronos"], 10, 64)
	ts := uint32(khronos)
	bodyMD5 := [16]byte{}
	if len(body) > 0 {
		bodyMD5 = md5Raw(body)
	}
	aid := playbackAID(query)
	headers["x-argus"] = argusHeader(query, bodyMD5, ts, randUint32(), aid, queryValue(query, "device_id"), queryValue(query, "iid"))
	headers["x-ladon"] = ladonHeader(ts, randUint32(), aid)
	delete(headers, "x-medusa")
	return query, headers
}

// argusHeader builds the x-argus value. query is the exact signed query sent on
// the wire (ts and _rticket already appended); bodyMD5 is the raw MD5 of the
// body (all zero for an empty body); ts is the x-khronos second; nonce is a
// fresh random field.
func argusHeader(query string, bodyMD5 [16]byte, ts, nonce uint32, aid, deviceID, installID string) string {
	bodyHash := sm3Sum(bodyMD5[:])
	queryBytes := []byte(query)
	if query == "" {
		queryBytes = make([]byte, 16)
	}
	queryHash := sm3Sum(queryBytes)

	counters := concat(
		pbVarint(1, 1), pbVarint(2, 1), pbVarint(3, 1), pbVarint(7, 0xc792eccc),
	)
	device := concat(
		pbString(1, "NX551J"), pbVarint(2, 8196), pbVarint(4, 0x80e0d800),
	)
	bean := concat(
		pbVarint(1, 1077940818),
		pbVarint(2, 2),
		pbVarint(3, uint64(nonce)),
		pbString(4, aid),
		pbString(5, deviceID),
		pbString(6, playbackLicenseID),
		pbString(7, queryValue(query, "version_name")),
		pbString(8, "v04.04.05-ov-android"),
		pbVarint(9, 134744640),
		pbBytes(10, make([]byte, 8)),
		pbVarint(11, 0),
		pbVarint(12, uint64(ts<<1)),
		pbBytes(13, bodyHash[:6]),
		pbBytes(14, queryHash[:6]),
		pbBytes(15, counters),
		pbString(16, installID),
		pbString(20, "none"),
		pbVarint(21, 738),
		pbBytes(23, device),
		pbVarint(25, 2),
	)

	plain := pkcs7(bean)
	simonEncrypt(plain)

	frame := concat(argusMask, plain, argusTail)
	for i := len(argusMask); i < len(frame); i++ {
		frame[i] ^= argusMask[i%len(argusMask)]
	}
	for i, j := 0, len(frame)-1; i < j; i, j = i+1, j-1 {
		frame[i], frame[j] = frame[j], frame[i]
	}
	frame = pkcs7(append(frame, 0x61, 0x6f)) // "ao" marker, then PKCS#7

	block, err := aes.NewCipher(argusAESKey)
	if err != nil {
		return ""
	}
	cipher.NewCBCEncrypter(block, argusAESIV).CryptBlocks(frame, frame)
	return base64.StdEncoding.EncodeToString(append([]byte{0xf2, 0x81}, frame...))
}

// ladonHeader builds the x-ladon value for timestamp ts and app id aid, keyed
// by md5(nonce || aid) and prefixed with the little-endian nonce.
func ladonHeader(ts, nonce uint32, aid string) string {
	seed := le32(nonce)
	digest := md5.Sum(append(append([]byte{}, seed...), aid...))
	hexKey := []byte(hex.EncodeToString(digest[:]))

	var keys [34]uint64
	k := binary.LittleEndian.Uint64(hexKey)
	pending := [3]uint64{
		binary.LittleEndian.Uint64(hexKey[8:]),
		binary.LittleEndian.Uint64(hexKey[16:]),
		binary.LittleEndian.Uint64(hexKey[24:]),
	}
	for i := range keys {
		keys[i] = k
		x := (bits.RotateLeft64(pending[0], -8) + k) ^ uint64(i)
		pending = [3]uint64{pending[1], pending[2], x}
		k = x ^ bits.RotateLeft64(k, 3)
	}

	plain := pkcs7([]byte(strconv.FormatUint(uint64(ts), 10) + "-" + playbackLicenseID + "-" + aid))
	for at := 0; at < len(plain); at += 16 {
		a := binary.LittleEndian.Uint64(plain[at:])
		b := binary.LittleEndian.Uint64(plain[at+8:])
		for _, rk := range keys {
			b = rk ^ (a + bits.RotateLeft64(b, -8))
			a = b ^ bits.RotateLeft64(a, 3)
		}
		binary.LittleEndian.PutUint64(plain[at:], a)
		binary.LittleEndian.PutUint64(plain[at+8:], b)
	}
	return base64.StdEncoding.EncodeToString(append(seed, plain...))
}

// simonEncrypt encrypts each 16-byte block of plain in place with Simon 64/128.
func simonEncrypt(plain []byte) {
	keys := simonKeySchedule()
	for at := 0; at < len(plain); at += 16 {
		a := binary.LittleEndian.Uint64(plain[at:])
		b := binary.LittleEndian.Uint64(plain[at+8:])
		for _, rk := range keys {
			a, b = b, a^(bits.RotateLeft64(b, 1)&bits.RotateLeft64(b, 8))^bits.RotateLeft64(b, 2)^rk
		}
		binary.LittleEndian.PutUint64(plain[at:], a)
		binary.LittleEndian.PutUint64(plain[at+8:], b)
	}
}

// simonKeySchedule expands the 256-bit sign key into 72 Simon 64/128 round keys.
func simonKeySchedule() [72]uint64 {
	var k [72]uint64
	for i := 0; i < 4; i++ {
		k[i] = binary.LittleEndian.Uint64(argusSignKey[i*8:])
	}
	const z uint64 = 0x3dc94c3a046d678b
	for i := 4; i < len(k); i++ {
		x := bits.RotateLeft64(k[i-1], -3) ^ k[i-3]
		x ^= bits.RotateLeft64(x, -1)
		k[i] = ^k[i-4] ^ x ^ ((z >> uint((i-4)%62)) & 1) ^ 3
	}
	return k
}

// pkcs7 pads to the next 16-byte multiple, adding a whole block when already
// aligned, as the Argus and Ladon envelopes require.
func pkcs7(data []byte) []byte {
	pad := 16 - len(data)%16
	out := append([]byte{}, data...)
	for i := 0; i < pad; i++ {
		out = append(out, byte(pad))
	}
	return out
}

// pbVarint encodes one wire-type-0 protobuf field. Unlike protoSInt it does not
// zig-zag; the Argus bean uses plain varints.
func pbVarint(tag uint32, value uint64) []byte {
	return append(protoKey(tag, 0), varint(value)...)
}

// pbBytes encodes one length-delimited field and always emits it, even when the
// body is empty (protoBytes drops empty bodies, which the Argus bean must not).
func pbBytes(tag uint32, body []byte) []byte {
	out := append(protoKey(tag, 2), varint(uint64(len(body)))...)
	return append(out, body...)
}

func pbString(tag uint32, s string) []byte {
	return pbBytes(tag, []byte(s))
}

func concat(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func queryValue(query, name string) string {
	values, err := url.ParseQuery(query)
	if err != nil {
		return ""
	}
	return values.Get(name)
}

// playbackAID reads the app id from the signed query, defaulting to the player's
// own id when absent.
func playbackAID(query string) string {
	if aid := queryValue(query, "aid"); aid != "" {
		return aid
	}
	return "1967"
}

func randUint32() uint32 {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return binary.LittleEndian.Uint32(b[:])
}
