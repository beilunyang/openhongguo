package sign

import "encoding/hex"

func rc4Gorgon(data, key []byte) []byte {
	var s [256]byte
	for i := range s {
		s[i] = byte(i)
	}
	j := 0
	for i := 0; i < 256; i++ {
		j = (j + int(s[i]) + int(key[i%len(key)])) & 255
		s[i] = s[j]
	}
	out := make([]byte, 0, len(data))
	i, j := 0, 0
	for _, value := range data {
		i++
		x := s[i&255]
		j += int(x)
		y := s[j&255]
		s[i&255] = y
		out = append(out, value^s[(int(y)+int(y))&255])
	}
	return out
}

// xGorgon builds x-gorgon with the emulator's flags.
func xGorgon(query string, body []byte, hasBody bool, khronos, random uint32) string {
	return xGorgonFlags(query, body, hasBody, khronos, random, medusaEmulatorFlags)
}

// xGorgonFlags builds x-gorgon. flags is x-medusa's message field 10: its
// first two bytes key the cipher and are sent as bytes 4 and 5 (08 00 under
// the emulator, 04 05 on a device; v04.06.04 used 40 01).
func xGorgonFlags(query string, body []byte, hasBody bool, khronos, random uint32, flags []byte) string {
	queryMD5 := md5Raw([]byte(query))
	bodyMD5 := [16]byte{}
	if hasBody {
		bodyMD5 = md5Raw(body)
	}
	input := make([]byte, 0, 20)
	input = append(input, queryMD5[:4]...)
	input = append(input, bodyMD5[:4]...)
	input = append(input, 0, 0, 0, 0)
	input = append(input, le32(medusaV5SDKCode)...)
	input = append(input, be32(khronos)...)
	key := []byte{0x4a, flags[0], 0x16, byte((random >> 8) & 255), 0x47, 0x6c, flags[1], byte(random & 255)}
	out := rc4Gorgon(input, key)
	n := len(out)
	for i := 0; i < n; i++ {
		value := out[i]
		swapped := (value >> 4) | (value << 4)
		next := out[0]
		if i+1 < n {
			next = out[i+1]
		}
		out[i] = byte(^(reverseBits(next^swapped) ^ 20))
	}
	result := []byte{0x84, 0x04, byte(random & 255), byte((random >> 8) & 255), flags[0], flags[1]}
	result = append(result, out...)
	return hex.EncodeToString(result)
}
