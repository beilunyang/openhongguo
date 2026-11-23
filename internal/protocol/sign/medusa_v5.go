package sign

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

// MedusaEnv is the device environment the v04.10.03 SDK reports inside
// x-medusa. One device keeps the same values across requests; PID, PPID and
// LaunchMillis describe the app process, and the clock offsets are added to
// the request timestamp.
//
// DefaultMedusaEnv returns the values the SDK reports under the unidbg
// emulator (openhongguo-signer), except Status, which the service only
// accepts with a real device's value.
type MedusaEnv struct {
	PID, PPID    int64
	LaunchMillis int64
	// Clock18, Clock19 and Clock20 are environment fields 18-20 minus the
	// request timestamp (three clocks the SDK samples).
	Clock18, Clock19, Clock20 int64
	Channel                   string
	// Fields of the JSON blob (message field 24).
	UnH, Fkd, Pd int64
	Sts          int64
	Lp, Fl       string
	// Status is message field 21. The SDK reports 369 on a real device and
	// 377 under the emulator; since 2026-11-23 the service answers playback
	// requests that carry 377 with an empty body.
	Status int64
}

// Message field 21 as the SDK reports it on a real device (captured from
// the 7.4.5.32 app on 2026-11-23) and under the unidbg emulator.
const (
	medusaDeviceStatus   = 369
	medusaEmulatorStatus = 377
)

// DefaultMedusaEnv returns an environment shaped like the emulator's, with
// the process fields set from the given launch time and the status of a
// real device.
func DefaultMedusaEnv(launchMillis int64) MedusaEnv {
	return MedusaEnv{
		PID: 4321, PPID: 1234,
		LaunchMillis: launchMillis,
		Clock18:      766353820, Clock19: 16, Clock20: -1072050540,
		Channel: "xiaomi_8662_64",
		UnH:     2398682137, Sts: 27, Fkd: 1963693275, Pd: 1442983616,
		Lp:     "2|520830913092|521433182354|2078052786776|1081701896",
		Fl:     "0|0|2523408538|2586339478|997724116|0|0|0|0|606600138",
		Status: medusaDeviceStatus,
	}
}

// MedusaRandom holds the three rand() values the SDK draws per request.
type MedusaRandom struct {
	Message uint32 // message field 3
	Key     uint32 // key hash seed; its bytes also go in the header and trailer
	Packed  uint32 // the word after the packed header byte
}

const (
	medusaV5Version    = 5
	medusaV5SDKVersion = "v04.10.03-ml-android"
	medusaV5SDKCode    = 0x040a0300
	medusaV5EnvVersion = "v04.10.03.04-bugfix"
	medusaV5PackedHead = 0xca
)

// medusaV5Flags is message field 10; the same bytes lead the transformed message.
var medusaV5Flags = []byte{0x08, 0, 0, 0, 0, 0, 0, 0}

// medusaV5Input is what a request contributes to x-medusa.
type medusaV5Input struct {
	querySM3 [32]byte
	bodyMD5  [16]byte
	digest   [20]byte // message field 13
	ts       uint32
}

// newMedusaV5Input hashes the request. query is the query string that is
// sent (without '?'); body is nil for a request without one. It returns
// false when the request selects a field-13 variant this package lacks;
// the caller then changes _rticket and tries again.
func newMedusaV5Input(query string, body []byte, ts uint32) (medusaV5Input, bool) {
	in := medusaV5Input{querySM3: sm3Sum([]byte(query)), ts: ts}
	if body != nil {
		in.bodyMD5 = md5Raw(body)
	}
	digest, ok := f13V5(in.querySM3, in.bodyMD5, ts)
	in.digest = digest
	return in, ok
}

// assembleMedusaV5 builds x-medusa as the v04.10.03 SDK does.
func assembleMedusaV5(in medusaV5Input, deviceID string, env MedusaEnv, rnd MedusaRandom) string {
	return sealMedusaV5(medusaV5Message(in, deviceID, env, rnd), in, rnd)
}

func medusaV5Message(in medusaV5Input, deviceID string, env MedusaEnv, rnd MedusaRandom) []byte {
	ts := in.ts
	const counter = 694367 // message field 15; the same in every capture
	return concat(
		protoBytes(1, playbackV5Magic),
		protoSInt(2, medusaV5Version),
		protoSInt(3, int64(rnd.Message)),
		protoString(4, appID),
		protoString(5, deviceID),
		protoString(6, playbackV5License),
		protoString(8, medusaV5SDKVersion),
		protoSInt(9, medusaV5SDKCode),
		protoBytes(10, medusaV5Flags),
		protoSInt(12, int64(ts)),
		protoBytes(13, in.digest[:]),
		protoBytes(14, in.querySM3[:6]),
		protoBytes(15, concat(protoSInt(1, counter), protoSInt(2, counter), protoSInt(3, counter), protoSInt(5, counter))),
		protoSInt(17, int64(ts)),
		protoString(20, "none"),
		protoSInt(21, env.Status),
		protoBytes(23, medusaV5Env(ts, deviceID, env)),
		protoString(24, medusaV5JSON(env)),
	)
}

func medusaV5Env(ts uint32, deviceID string, env MedusaEnv) []byte {
	return concat(
		protoString(6, medusaV5EnvVersion),
		protoSInt(7, env.PID),
		protoString(8, env.Channel),
		protoBytes(12, medusaV5Device(deviceID, env)),
		protoEmpty(13),
		protoString(14, notSet),
		protoSInt(15, env.PPID),
		protoSInt(18, int64(ts)+env.Clock18),
		protoSInt(19, int64(ts)+env.Clock19),
		protoSInt(20, int64(ts)+env.Clock20),
	)
}

const (
	notSet   = "!notset!"
	unsetInt = -999999
)

// medusaV5Device is environment field 12. Fields the emulator cannot answer
// carry the SDK's placeholders: "!notset!" and -999999.
func medusaV5Device(deviceID string, env MedusaEnv) []byte {
	out := concat(protoSInt(1, 1), protoString(3, appID), protoString(4, deviceID))
	for tag := uint32(5); tag <= 40; tag++ {
		switch {
		case tag == 5 || tag == 6 || (tag >= 11 && tag <= 13) || tag == 15 || tag == 22 || (tag >= 30 && tag <= 37):
			out = append(out, protoString(tag, notSet)...)
		case (tag >= 7 && tag <= 10) || tag == 14 || (tag >= 23 && tag <= 25) || tag == 27 || tag == 29 || tag == 38:
			out = append(out, protoSInt(tag, unsetInt)...)
		case tag >= 16 && tag <= 21:
			out = append(out, protoFloat(tag, unsetInt)...)
		case tag == 26 || tag == 28 || tag == 40:
			out = append(out, protoSInt(tag, env.LaunchMillis)...)
		}
	}
	return out
}

func medusaV5JSON(env MedusaEnv) string {
	return fmt.Sprintf(`{"cmr":16777216,"cmr2":16777216,"un_h":%d,"vpn":0,"sts":%d,"kd":0,"fkd":%d,"pd":%d,"lp":%s,"fl":%s,"dyn":"","do":0,"tk":true}`,
		env.UnH, env.Sts, env.Fkd, env.Pd, strconv.Quote(env.Lp), strconv.Quote(env.Fl))
}

// sealMedusaV5 encrypts a message into the x-medusa wire format.
func sealMedusaV5(message []byte, in medusaV5Input, rnd MedusaRandom) string {
	ts := in.ts
	hash, seed := keyHash(playbackV5SignKey, rnd.Key)
	transformed := append(append([]byte(nil), medusaV5Flags...), xmxor(message, hash[:])...)
	for i, j := 0, len(transformed)-1; i < j; i, j = i+1, j-1 {
		transformed[i], transformed[j] = transformed[j], transformed[i]
	}
	for i := range transformed {
		transformed[i] ^= seed[(^i)&3]
	}
	check := uint32(0x18000001) | uint32(in.querySM3[0]&63)<<14 | uint32(in.digest[0]&63)<<8
	packed := concat([]byte{medusaV5PackedHead}, le32(rnd.Packed), le32(check), transformed, []byte{byte(rnd.Key >> 16), byte(rnd.Key >> 24)})
	key := md5Raw(playbackV5SignKey[:16])
	iv := md5Raw(playbackV5SignKey[16:])
	encrypted := encryptPlaybackV5(packed, key[:], iv[:], ts)
	prefix := concat(le32(medusaV5Version), playbackV5Magic)
	for i := range prefix {
		prefix[i] ^= byte(ts >> (uint(i&3) * 8))
	}
	return base64.StdEncoding.EncodeToString(concat(prefix, []byte{byte(rnd.Key), byte(rnd.Key >> 8), 0, 1}, encrypted))
}

func protoEmpty(tag uint32) []byte {
	return append(protoKey(tag, 2), 0)
}
