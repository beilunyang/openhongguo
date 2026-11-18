package sign

import (
	"encoding/binary"
)

var branch0TT = [64]uint32{
	0xebb64faf, 0x07aadcc2, 0xcf3187bf, 0xe01138ff, 0x6d0bfcff, 0x5a30a3be, 0xb41ad638, 0x34180eb8,
	0xf233eb6f, 0xb1a584cc, 0xccc30dc7, 0x47d1db51, 0xd55653de, 0x70a84fa1, 0x57473c12, 0xf76f0288,
	0x2c077f0a, 0xda0dcad0, 0xfbb86f6c, 0xfdc4cf00, 0x688a020d, 0xe676c6a6, 0x8cd6338b, 0x1a3c8d0e,
	0xcce8b06b, 0x6ad0ed0b, 0xa0522717, 0xdc71ac83, 0x2285db71, 0xd5b4dda6, 0x736f8650, 0x6560306c,
	0x617ce2a6, 0xe423417e, 0x0a40e143, 0x544e4032, 0x88dffb2a, 0x716c1ae0, 0x4c467a88, 0x05b23bb3,
	0xe1d0b866, 0xbaa3dcb8, 0xae3374d3, 0xc3381a50, 0x1702f75b, 0xfe6da368, 0xf0b4cf48, 0x4e0ffbb8,
	0x72aad10d, 0x26c53a3d, 0xf2bce0f6, 0xb4557581, 0x4a257fdd, 0x8c3182a2, 0xab0b3b86, 0x3d5dfb14,
	0x4f103634, 0xd37b52d7, 0x444eff16, 0xeb0a33d1, 0x6ca86f6e, 0x00284ba7, 0x08387cfa, 0x5fb37586,
}

var branch0Init = [8]uint32{
	0x7aba4fc8, 0x67166507, 0x6403fa00, 0x340f512f,
	984304912, 3005047866, 2874125293, 2152413264,
}

func expandWords(data []byte) []uint32 {
	di := make([]uint32, 0, 16+112)
	for off := 0; off+4 <= len(data); off += 4 {
		di = append(di, binary.BigEndian.Uint32(data[off:off+4]))
	}
	di0 := di[0]
	for i := 0; i < 112; i++ {
		di1 := di[i+1]
		di14 := di[i+14]
		r1 := rol32(di1, 14) ^ rol32(di1, 25) ^ (di1 >> 3)
		r2 := rol32(di14, 13) ^ rol32(di14, 15) ^ (di14 >> 10)
		di = append(di, di0+di[i+9]+r1+r2)
		di0 = di1
	}
	return di
}

func mixRounds(state []uint32, rounds [11]uint32, di, tt []uint32, ivV1 uint32) []uint32 {
	for i := uint32(0); i < rounds[0]; i++ {
		base := ivV1 + i
		w := di[base&127]
		idx := func(k int) int { return int(rounds[k]) }
		n1 := ((state[idx(3)] ^ state[idx(4)]) & state[idx(1)]) ^ state[idx(3)]
		n2 := rol32(state[idx(1)], 26) ^ rol32(state[idx(1)], 21) ^ rol32(state[idx(1)], 7)
		n4 := w + n1 + n2 + tt[base&63] + state[idx(5)]
		n5 := rol32(state[idx(2)], 30) ^ rol32(state[idx(2)], 19) ^ rol32(state[idx(2)], 10)
		n6 := (state[idx(2)] & state[idx(6)]) | ((state[idx(2)] | state[idx(6)]) & state[idx(7)])
		n7 := n5 + n6
		old := state[idx(9)]
		last := state[len(state)-1]
		state = append([]uint32{last}, state[:len(state)-1]...)
		state[idx(10)] = n7 + n4
		state[idx(8)] = old + n4
	}
	return state
}
