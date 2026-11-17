package sign

import "encoding/binary"

var aesCon2 = [4][4]int{{1, 0, 2, 3}, {2, 0, 3, 1}, {0, 1, 3, 2}, {1, 0, 2, 3}}

var aesOrder = [4][16]int{
	{0, 9, 14, 11, 4, 13, 2, 7, 8, 1, 6, 15, 12, 5, 10, 3},
	{0, 9, 14, 15, 4, 13, 2, 7, 8, 1, 6, 3, 12, 5, 10, 11},
	{0, 9, 14, 7, 4, 13, 2, 11, 8, 1, 6, 3, 12, 5, 10, 15},
	{0, 9, 14, 11, 4, 13, 2, 7, 8, 1, 6, 15, 12, 5, 10, 3},
}

var aesInitWord = [4]uint32{0xca025ddc, 0x823dc546, 0xc9420583, 0xc298225f}

type aesV3 struct {
	box   []byte
	con2  [4]int
	order [16]int
	keys  [12][4]byte
}

func newAESV3(key []byte, khronos uint32) aesV3 {
	word := int(khronos & 3)
	all := sbox()
	start := word * 256
	return aesV3{
		box:   append([]byte{}, all[start:start+256]...),
		con2:  aesCon2[word],
		order: aesOrder[word],
		keys:  expandAES(key, word),
	}
}

func expandAES(key []byte, word int) [12][4]byte {
	all := sbox()
	return expandAESWith(key, aesInitWord[word], all[word*256:word*256+256])
}

func expandAESWith(key []byte, init uint32, box []byte) [12][4]byte {
	initial := make([]byte, 0, 16)
	var le [4]byte
	binary.LittleEndian.PutUint32(le[:], init)
	for i := 0; i < 4; i++ {
		initial = append(initial, le[:]...)
	}
	expanded := xorBytes(initial, key)
	expanded = append(expanded, make([]byte, 32)...)
	rounds := uint32(8)
	for i := 4; i < 12; i++ {
		at := 4 * (i - 1)
		a, b, c, d := expanded[at], expanded[at+1], expanded[at+2], expanded[at+3]
		if i&3 == 0 {
			t := ((init >> (rounds & 24)) ^ uint32(box[b])) & 255
			b = box[c]
			c = box[d]
			d = box[a]
			a = byte(t)
		}
		rounds += 2
		expanded[at+4] = a ^ expanded[at-12]
		expanded[at+5] = b ^ expanded[at-11]
		expanded[at+6] = c ^ expanded[at-10]
		expanded[at+7] = d ^ expanded[at-9]
	}
	var keys [12][4]byte
	for i := range keys {
		copy(keys[i][:], expanded[i*4:i*4+4])
	}
	return keys
}

func xorBytes(a, b []byte) []byte {
	out := append([]byte{}, a...)
	for i := range out {
		out[i] ^= b[i%len(b)]
	}
	return out
}

func addCon(state *[4][4]byte, keys [][4]byte, con2 [4]int) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			state[i][j] ^= keys[i][con2[j]]
		}
	}
}

func (a aesV3) sub(state *[4][4]byte) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			state[i][j] = a.box[state[i][j]]
		}
	}
	old := *state
	for i := 0; i < 4; i++ {
		state[i] = old[a.con2[i]]
	}
}

func (a aesV3) shift(state *[4][4]byte) {
	var flat [16]byte
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			flat[i*4+j] = state[i][j]
		}
	}
	for i := 0; i < 16; i++ {
		state[i/4][i%4] = flat[a.order[i]]
	}
}

func shiftCon(state *[4][4]byte, con2 [4]int) {
	old := *state
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			state[i][j] = old[i][con2[j]]
		}
	}
}

func mixColumns(state *[4][4]byte) {
	xt := func(x byte) byte {
		v := x << 1
		if x&128 != 0 {
			v ^= 0x1b
		}
		return v
	}
	for i := 0; i < 4; i++ {
		t := state[0][i] ^ state[1][i] ^ state[2][i] ^ state[3][i]
		u := state[0][i]
		state[0][i] ^= t ^ xt(state[0][i]^state[1][i])
		state[1][i] ^= t ^ xt(state[1][i]^state[2][i])
		state[2][i] ^= t ^ xt(state[2][i]^state[3][i])
		state[3][i] ^= t ^ xt(state[3][i]^u)
	}
}

func (a aesV3) block(value []byte) []byte {
	var state [4][4]byte
	for i := 0; i < 4; i++ {
		copy(state[i][:], value[i*4:i*4+4])
	}
	keys := a.keys[:]
	addCon(&state, keys[0:4], a.con2)
	for i := 1; i < 3; i++ {
		a.sub(&state)
		a.shift(&state)
		if i == 1 {
			shiftCon(&state, a.con2)
			mixColumns(&state)
		}
		addCon(&state, keys[i*4:i*4+4], a.con2)
	}
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			state[i][j] ^= keys[4+i][j]
		}
	}
	out := make([]byte, 0, 16)
	for i := 0; i < 4; i++ {
		out = append(out, state[i][:]...)
	}
	return out
}

func (a aesV3) encrypt(data, iv []byte) []byte {
	source := append([]byte{}, data...)
	var plaintext [32]byte
	for i := 0; i < 31; i++ {
		at := i * 8
		n0 := (source[at] >> 4) & 2
		n1 := n0 | (source[at+1] & 64)
		n2 := n1 | ((source[at+2] >> 2) & 1)
		n3 := n2 | ((source[at+3] << 3) & 128)
		n4 := n3 | ((source[at+4] >> 1) & 4)
		n5 := n4 | ((source[at+5] << 3) & 16)
		n6 := n5 | ((source[at+6] << 5) & 32)
		plaintext[i] = n6 | ((source[at+7] >> 4) & 8)
	}
	plaintext[31] = 1
	var blocks []byte
	previous := append([]byte{}, iv...)
	for off := 0; off < 32; off += 16 {
		value := a.block(xorBytes(plaintext[off:off+16], previous))
		blocks = append(blocks, value...)
		previous = value
	}
	out := source
	for i := 0; i < 31; i++ {
		at := i * 8
		k := blocks[i]
		out[at] &= 0xdf
		out[at] |= (k << 4) & 32
		out[at+1] &= 0xbf
		out[at+1] |= k & 64
		out[at+2] &= 0xfb
		out[at+2] |= (k << 2) & 4
		out[at+3] &= 0xef
		out[at+3] |= (k >> 3) & 16
		out[at+4] &= 0xf7
		out[at+4] |= (k + k) & 8
		out[at+5] &= 0xfd
		out[at+5] |= (k >> 3) & 2
		out[at+6] &= 0xfe
		out[at+6] |= (k >> 5) & 1
		out[at+7] &= 0x7f
		out[at+7] |= (k << 4) & 128
	}
	result := []byte{blocks[31]}
	return append(result, out...)
}
