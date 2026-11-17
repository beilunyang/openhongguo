package sign

// encryptPlaybackV5 使用既有 AES 轮函数和 V5 的 S 盒、轮密钥、位映射。
func encryptPlaybackV5(data, key, iv []byte, timestamp uint32) []byte {
	word := timestamp & 3
	box := playbackV5Sbox[word*256 : (word+1)*256]
	initial := [4]uint32{0xbbb5de72, 0xd4638d5b, 0x92a5f73b, 0x89f8a1c9}[word]
	cipher := aesV3{
		box:  box,
		con2: [4][4]int{{2, 0, 3, 1}, {1, 3, 2, 0}, {3, 2, 1, 0}, {2, 3, 1, 0}}[word],
		order: [4][16]int{
			{0, 9, 14, 15, 4, 13, 2, 7, 8, 1, 6, 3, 12, 5, 10, 11},
			{0, 9, 14, 11, 4, 13, 2, 3, 8, 1, 6, 15, 12, 5, 10, 7},
			{0, 9, 14, 3, 4, 13, 2, 15, 8, 1, 6, 11, 12, 5, 10, 7},
			{0, 9, 14, 15, 4, 13, 2, 3, 8, 1, 6, 11, 12, 5, 10, 7},
		}[word],
		keys: expandAESWith(key, initial, box),
	}
	wire := [8]uint{6, 5, 3, 4, 2, 7, 1, 0}
	compact := [8]uint{1, 7, 3, 6, 2, 0, 4, 5}
	var plain [32]byte
	for i := 0; i < 31; i++ {
		for j := range wire {
			at := i*8 + j
			if at < len(data) {
				plain[i] |= ((data[at] >> wire[j]) & 1) << compact[j]
			}
		}
	}
	plain[31] = 1
	first := cipher.block(xorBytes(plain[:16], iv))
	bits := append(first, cipher.block(xorBytes(plain[16:], first))...)
	out := append([]byte(nil), data...)
	for i := 0; i < 31; i++ {
		for j := range wire {
			at := i*8 + j
			if at < len(out) {
				out[at] = (out[at] &^ (1 << wire[j])) | (((bits[i] >> compact[j]) & 1) << wire[j])
			}
		}
	}
	return append([]byte{bits[31]}, out...)
}
