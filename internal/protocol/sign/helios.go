package sign

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// heliosV5 builds x-helios as the v04.10.03 SDK does: a 34-round ARX cipher
// over "<ts>-<license>-<aid>", keyed from md5(random || aid). The random is
// sent in the clear as the first 4 bytes.
func heliosV5(ts, random uint32) string {
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

func diffuse(words []uint64) []uint64 {
	table := []uint64{words[0]}
	b0 := words[0]
	b8 := words[1]
	queue := append([]uint64{}, words[2:]...)
	for i := uint64(0); i < 34; i++ {
		x8 := ror64(b8, 8) + b0
		x8 ^= i
		queue = append(queue, x8)
		x8 ^= ror64(b0, 61)
		table = append(table, x8)
		b0 = x8
		b8 = queue[0]
		queue = queue[1:]
	}
	return table
}
