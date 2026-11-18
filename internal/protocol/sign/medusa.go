package sign

import ()

type Entropy struct {
	EnvLaunch     int64
	EnvPID        int64
	NowSecs       int64
	MessageRandom uint32
	Random        uint32
	PackedRandom  uint32
	GorgonRandom  uint32
	HeliosRandom  uint32
}

func xmxor(data, key []byte) []byte {
	n := len(data)
	encoded := make([]byte, n)
	for i := 0; i < n; i++ {
		at := (i * 4) & 28
		d0 := key[at]
		d1 := key[at+1]
		d2 := byte((uint32((data[i]<<4)|(data[i]>>4)) + uint32(d0)))
		d2 = ^d2 ^ d1
		d2 = (d2 << 3) | (d2 >> 5)
		d2 += d1
		d2 ^= d0
		encoded[n-i-1] = ^d2
	}
	last := encoded[n-1] ^ encoded[n-2]
	first := encoded[0]
	encoded[0] = (^last) + first
	encoded[1] = (encoded[0] ^ encoded[n-1] ^ 254) + encoded[1]
	shifted := (encoded[1] << 3) | (encoded[1] >> 5)
	fix2 := (last - first) ^ (shifted ^ 2)
	encoded[2] += fix2
	for i := 0; i < n-4; i++ {
		sh := (encoded[i+2] << 3) | (encoded[i+2] >> 5)
		temp := sh ^ encoded[i+1] ^ byte(i+3)
		encoded[i+3] = (^temp) + encoded[i+3]
	}
	encoded[n-1] ^= encoded[n-2]
	var sum uint32
	for _, item := range encoded[1:] {
		sum += uint32(item)
	}
	encoded[0] = (encoded[0] ^ encoded[1]) + byte(sum)
	return encoded
}

func keyHash(signKey []byte, random uint32) ([32]byte, [4]byte) {
	buf := append([]byte{}, signKey...)
	buf = append(buf, le32(random)...)
	buf = append(buf, signKey...)
	hash := sm3Sum(buf)
	d1 := byte((random >> 16) & 255)
	d2 := (uint32(d1) << 11) | (random >> 24)
	d2 ^= uint32(d1) >> 5
	d2 ^= uint32(d1)
	d2 = ^d2
	var seed [4]byte
	copy(seed[:], le32(d2))
	return hash, seed
}
