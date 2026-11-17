package sign

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// PlaybackV5Signer 保存客户端生命周期内稳定的进程信息；签名时只读取，可并发使用。
type PlaybackV5Signer struct {
	started int64
	nonce   string
}

func NewPlaybackV5Signer(nowMillis int64) *PlaybackV5Signer {
	var nonce [18]byte
	_, _ = rand.Read(nonce[:])
	return &PlaybackV5Signer{started: nowMillis, nonce: base64.RawURLEncoding.EncodeToString(nonce[:])}
}

// Sign 对实际发送的 gzip 请求体签名。查询串不添加 ts，选择分支时只调整 _rticket。
// pairs 里已有 _rticket 时就地填写（保持客户端的参数顺序），否则追加在末尾。
func (s *PlaybackV5Signer) Sign(pairs [][2]string, nowMillis int64, body []byte, entropy *Entropy) (string, map[string]string, error) {
	ts := uint32(nowMillis / 1000)
	bodyMD5 := [16]byte{}
	if body != nil {
		bodyMD5 = md5Raw(body)
	}
	withTicket := append([][2]string(nil), pairs...)
	slot := -1
	for i, kv := range withTicket {
		if kv[0] == "_rticket" {
			slot = i
			break
		}
	}
	if slot < 0 {
		withTicket = append(withTicket, [2]string{"_rticket", ""})
		slot = len(withTicket) - 1
	}
	for offset := int64(0); offset < 1024; offset++ {
		withTicket[slot][1] = strconv.FormatInt(nowMillis+offset, 10)
		query := encodeQuery(withTicket)
		queryHash := sm3Sum([]byte(query))
		iv := getIV(getIV(getIV(0x20230928, queryHash[:]), bodyMD5[:]), le32(ts))
		if iv&15 != 12 {
			continue
		}
		ent := Entropy{}
		if entropy == nil {
			ent = drawEntropy(nowMillis)
		} else {
			ent = *entropy
		}
		headers := map[string]string{
			"x-khronos": strconv.FormatUint(uint64(ts), 10),
			"x-helios":  playbackV5Helios(ts, ent.HeliosRandom),
			"x-medusa":  s.medusa(query, queryHash, bodyMD5, ts, ent),
		}
		if body != nil {
			headers["Content-Type"] = "application/json; charset=utf-8"
			headers["x-ss-stub"] = strings.ToUpper(hex.EncodeToString(bodyMD5[:]))
		}
		return query, headers, nil
	}
	return "", nil, fmt.Errorf("playback V5 signing branch unavailable")
}

func playbackV5Hash(query [32]byte, body [16]byte, ts uint32) []byte {
	const seed = uint32(0x9d4a97c2)
	count := seed + ts
	table := make([]uint32, len(branch0TT))
	for i, v := range branch0TT {
		table[i] = ror32(v, count+1)
	}
	pad := []byte{0xfa, 0x45, 0x61, 0xd7}
	for i, v := range pad {
		pad[i] = byte((uint16(v) | uint16(v)<<8) >> ((count + 2) & 7))
	}
	words := expandWords(concat(query[:], body[:], le32(ts), pad, []byte{0, 0, 0, 0, 0, 0, 1, 0xa0}))
	initial := make([]uint32, len(branch0Init))
	for i, v := range branch0Init {
		initial[i] = ror32(v, count)
	}
	state := mixRounds(append([]uint32(nil), initial...), [11]uint32{96, 1, 3, 4, 0, 6, 5, 2, 0, 7, 7}, words, table, seed)
	var result [32]byte
	for i, v := range state {
		binary.BigEndian.PutUint32(result[i*4:], v+initial[i])
	}
	for i := 0; i < 16; i++ {
		result[i] ^= result[i+16]
	}
	return append(result[:16:16], le32(sumMD5(result[:16])|0x20)...)
}

func (s *PlaybackV5Signer) medusa(query string, queryHash [32]byte, bodyMD5 [16]byte, ts uint32, entropy Entropy) string {
	deviceID := queryValue(query, "device_id")
	version := queryValue(query, "version_name")
	device := concat(
		protoSInt(1, 1), protoSInt(2, 2), protoString(3, appID), protoString(4, deviceID),
		protoString(5, s.nonce), protoString(6, "!noperm!"), protoString(12, "Asia/Shanghai,8"),
		protoString(13, "zh_CN"), protoString(22, queryValue(query, "os_version")), protoSInt(25, s.started),
		protoString(30, queryValue(query, "device_type")), protoString(31, "HONOR"),
		protoString(32, queryValue(query, "device_type")), protoString(34, "HONOR"),
	)
	env := concat(protoString(6, "v04.10.03.04-bugfix"), protoString(8, queryValue(query, "channel")), protoBytes(12, device), protoString(14, version))
	digest := playbackV5Hash(queryHash, bodyMD5, ts)
	flags := make([]byte, 8)
	message := concat(
		protoBytes(1, playbackV5Magic), protoSInt(2, 5), protoSInt(3, int64(entropy.MessageRandom)),
		protoString(4, appID), protoString(5, deviceID), protoString(6, playbackV5License),
		protoString(7, version), protoString(8, "v04.10.03-ml-android"), protoSInt(9, 0x040a0300),
		protoBytes(10, flags), protoSInt(12, int64(ts)), protoBytes(13, digest), protoBytes(14, queryHash[:6]),
		protoBytes(15, concat(protoSInt(1, 1), protoSInt(2, 1), protoSInt(3, 1), protoSInt(5, 1))),
		protoSInt(17, int64(ts)), protoString(20, "none"), protoSInt(21, 377), protoBytes(23, env),
		protoString(24, "{\"cmr\":16777216,\"cmr2\":16777216,\"un_h\":11,\"vpn\":0,\"kd\":0,\"dyn\":\"\",\"do\":1,\"tk\":true}"),
	)
	hash, mask := keyHash(playbackV5SignKey, entropy.Random)
	transformed := append(flags, xmxor(message, hash[:])...)
	for i, j := 0, len(transformed)-1; i < j; i, j = i+1, j-1 {
		transformed[i], transformed[j] = transformed[j], transformed[i]
	}
	for i := range transformed {
		transformed[i] ^= mask[(^i)&3]
	}
	check := uint32(0x18000001) | uint32(queryHash[0]&63)<<14 | uint32(digest[0]&63)<<8
	packed := concat([]byte{0xca}, le32(entropy.PackedRandom), le32(check), transformed, []byte{byte(entropy.Random >> 16), byte(entropy.Random >> 24)})
	key := md5Raw(playbackV5SignKey[:16])
	iv := md5Raw(playbackV5SignKey[16:])
	encrypted := encryptPlaybackV5(packed, key[:], iv[:], ts)
	prefix := concat(le32(5), playbackV5Magic)
	for i := range prefix {
		prefix[i] ^= byte(ts >> (uint(i&3) * 8))
	}
	result := concat(prefix, []byte{byte(entropy.Random), byte(entropy.Random >> 8), 0, 1}, encrypted)
	return base64.StdEncoding.EncodeToString(result)
}

func playbackV5Helios(ts, random uint32) string {
	seed := md5Raw(append(le32(random), []byte(appID)...))
	ascii := []byte(hex.EncodeToString(seed[:]))
	words := make([]uint64, 4)
	for i := range words {
		words[i] = binary.LittleEndian.Uint64(ascii[i*8:])
	}
	table := diffuse(words)
	text := []byte(fmt.Sprintf("%d-%s-%s", ts, playbackV5License, appID))
	padding := byte(16 - len(text)%16)
	for range int(padding) {
		text = append(text, padding)
	}
	out := le32(random)
	for at := 0; at < len(text); at += 16 {
		a := binary.LittleEndian.Uint64(text[at:])
		b := binary.LittleEndian.Uint64(text[at+8:])
		for _, key := range table[:34] {
			b = key ^ (a + ror64(b, 8))
			a = b ^ ror64(a, 61)
		}
		var block [16]byte
		binary.LittleEndian.PutUint64(block[:8], a)
		binary.LittleEndian.PutUint64(block[8:], b)
		out = append(out, block[:]...)
	}
	return base64.StdEncoding.EncodeToString(out)
}
