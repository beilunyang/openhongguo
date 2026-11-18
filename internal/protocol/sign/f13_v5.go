package sign

import "encoding/binary"

// f13V5Variant picks which of the SDK's 16 field-13 digests a request uses.
func f13V5Variant(querySM3 [32]byte, bodyMD5 [16]byte, ts uint32) uint32 {
	return getIV(getIV(getIV(0x20230928, querySM3[:]), bodyMD5[:]), le32(ts)) & 15
}

// f13V5Supported reports whether f13V5 implements the variant a request
// selects. The SDK has 16; signers adjust _rticket until one of these comes up.
func f13V5Supported(variant uint32) bool { return variant == 5 || variant == 12 }

// f13V5 computes x-medusa field 13 for the v04.10.03 SDK. The SDK runs it
// in its bytecode VM (so+0x2aac34) and picks one of 16 digests by
// f13V5Variant; each yields 16 bytes, followed by a common 4-byte tail.
// It returns false for a variant this package does not implement.
//
// The inputs are the query's SM3, the body's MD5 (zero for a request without
// a body) and the timestamp.
func f13V5(querySM3 [32]byte, bodyMD5 [16]byte, ts uint32) ([20]byte, bool) {
	var out [20]byte
	var head [16]byte
	switch f13V5Variant(querySM3, bodyMD5, ts) {
	case 5:
		head = f13V5HeadMD5(querySM3, bodyMD5, ts)
	case 12:
		head = f13V5HeadSHA(querySM3, bodyMD5, ts)
	default:
		return out, false
	}
	copy(out[:16], head[:])
	copy(out[16:], f13V5Tail(head))
	return out, true
}

// f13V5Tail is the 4 bytes every variant appends: an APHash variant over
// the first 8 head bytes with two bit fixes.
func f13V5Tail(head [16]byte) []byte {
	h := uint32(0x20220420)
	for i, cb := range head[:8] {
		if i&1 == 0 {
			h = h<<7 ^ uint32(cb) ^ h>>3 ^ h
		} else {
			h = ^((h<<11 | uint32(cb)) ^ h>>5 ^ h)
		}
	}
	out := le32(h)
	out[0] &= 0xdb
	out[3] ^= 1
	return out
}

// f13V5HeadMD5 is variant 5: an MD5 compression of one 64-byte block with
// these changes:
//   - The IV, the 64 round constants, the two padding words and the final
//     whitening words are fixed values rotated right by an amount derived
//     from the timestamp.
//   - The message words are laid out and visited in a custom order.
//   - The digest is (IV + state) XOR whitening.
func f13V5HeadMD5(querySM3 [32]byte, bodyMD5 [16]byte, ts uint32) [16]byte {
	var in [52]byte
	copy(in[:32], querySM3[:])
	copy(in[32:48], bodyMD5[:])
	binary.LittleEndian.PutUint32(in[48:], ts)
	w := func(at int) uint32 { return binary.LittleEndian.Uint32(in[at:]) }

	var pad [8]byte
	for i, b := range f13V5Pad {
		pad[i] = ror8(b, (ts-1)&7)
	}
	m := [16]uint32{
		w(36), w(32), binary.LittleEndian.Uint32(pad[:4]), w(24),
		w(40), w(28), binary.LittleEndian.Uint32(pad[4:]), w(44),
		w(0), w(4), w(8), 0x1a0, // 0x1a0: the message length in bits (52*8)
		w(48), w(16), w(20), w(12),
	}

	var iv [4]uint32
	for i, v := range f13V5IV {
		iv[i] = ror32(v, (ts+2)&31)
	}
	a, b, c, d := iv[0], iv[1], iv[2], iv[3]
	for i := 0; i < 64; i++ {
		var f uint32
		switch i / 16 {
		case 0:
			f = d ^ (b & (c ^ d))
		case 1:
			f = c ^ (d & (b ^ c))
		case 2:
			f = b ^ c ^ d
		default:
			f = c ^ (b | ^d)
		}
		k := ror32(f13V5K[i], (ts+3+uint32(i/16))&31)
		x := a + f + m[f13V5Order[i]] + k
		a, b, c, d = d, b+rol32(x, uint32(f13V5Shift[i])), b, c
	}
	state := [4]uint32{a, b, c, d}

	var out [16]byte
	for i := range state {
		binary.LittleEndian.PutUint32(out[i*4:], (iv[i]+state[i])^ror32(f13V5Final[i], (ts+8)&31))
	}
	return out
}

// f13V5HeadSHA is variant 12: a SHA-256 style compression with 96 rounds
// over the expanded block, with tables rotated by the timestamp.
func f13V5HeadSHA(querySM3 [32]byte, bodyMD5 [16]byte, ts uint32) [16]byte {
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
	words := expandWords(concat(querySM3[:], bodyMD5[:], le32(ts), pad, []byte{0, 0, 0, 0, 0, 0, 1, 0xa0}))
	initial := make([]uint32, len(branch0Init))
	for i, v := range branch0Init {
		initial[i] = ror32(v, count)
	}
	state := mixRounds(append([]uint32(nil), initial...), [11]uint32{96, 1, 3, 4, 0, 6, 5, 2, 0, 7, 7}, words, table, seed)
	var result [32]byte
	for i, v := range state {
		binary.BigEndian.PutUint32(result[i*4:], v+initial[i])
	}
	var out [16]byte
	for i := range out {
		out[i] = result[i] ^ result[i+16]
	}
	return out
}

func ror8(value byte, count uint32) byte {
	count &= 7
	return value>>count | value<<(8-count)
}

var (
	f13V5IV    = [4]uint32{0x79e0f2fb, 0xc8b52570, 0xebc2f8cd, 0x7c104d93}
	f13V5Final = [4]uint32{0x19be4866, 0xe85986b4, 0xe19b326e, 0x71d1d7d4}
	f13V5Pad   = [8]byte{0x84, 0x96, 0x77, 0x9d, 0xd4, 0x15, 0x0b, 0xf8}
	f13V5K     = [64]uint32{
		0xa7aefe20, 0x7149f1d6, 0x47e4ca07, 0xe9b58f67, 0x93b924de, 0xc614d0f5, 0x38afe0ef, 0xb2bbad73,
		0xe24444c3, 0x9d3aec9b, 0xdf7b37e4, 0xd8b16d40, 0xf8ac31b8, 0x76b9a90b, 0x31d833ee, 0x953fce64,
		0x6a6b2b48, 0x8c138276, 0x6d24a010, 0x18da124b, 0xbbeb82ee, 0x39f7ea56, 0x149f4fe1, 0x229946bc,
		0x0327f309, 0xf4f50e66, 0x62d569aa, 0x78419f92, 0x4db088a7, 0x7430a018, 0xf31d88f4, 0x2f4ed651,
		0x9b13fe59, 0x45c5910d, 0x374bf0ec, 0xd5a5b69e, 0xefaceb16, 0x714f4811, 0x4c98233b, 0xc9e6e7b1,
		0xd0623b3f, 0xdf5e4f6f, 0xccb31244, 0xaddab856, 0x6faf9c9e, 0x62804e10, 0x6fa7e29d, 0x7e429d51,
		0xcdb31e01, 0xb339e419, 0xe497a8a3, 0xebeca6bc, 0x1746cd57, 0xfbf4f54a, 0xd5ea2a8b, 0xb4f02ed2,
		0x0512b192, 0xefad4d29, 0xd59bec05, 0x2a1436e2, 0xfaea73c1, 0x1ebdbae8, 0x8ff257bd, 0x2d59351d,
	}
	f13V5Order = [64]uint8{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
		0, 10, 5, 14, 3, 4, 15, 13, 6, 1, 12, 8, 9, 7, 11, 2,
		8, 10, 3, 1, 14, 3, 7, 2, 5, 0, 13, 9, 11, 4, 15, 12,
		8, 9, 15, 10, 2, 6, 5, 13, 13, 1, 7, 3, 11, 14, 4, 12,
	}
	f13V5Shift = [64]uint8{
		7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22,
		5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20,
		4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23, 4, 11, 16, 23,
		6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21,
	}
)
