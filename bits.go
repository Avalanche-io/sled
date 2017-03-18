package sled

const (
	// w controls the number of branches at a node (2^w branches).
	w = 6

	// exp2 is 2^w, which is the hashcode space.
	exp2 = 64
)

func flagPos(hashcode uint64, lev uint, bmp uint64) (uint64, uint64) {
	idx := (hashcode >> lev) & 0x3f
	flag := uint64(1) << uint64(idx)
	mask := uint64(flag - 1)
	pos := bitCount64(bmp & mask)
	return flag, pos
}

func bitCount64(x uint64) uint64 {
	x -= (x >> 1) & 0x5555555555555555
	x = ((x >> 2) & 0x3333333333333333) + (x & 0x3333333333333333)
	x = ((x >> 4) + x) & 0x0F0F0F0F0F0F0F0F
	x *= 0x0101010101010101
	return x >> 56
}

func bitCount32(x uint64) uint64 {
	x -= (x >> 1) & 0x55555555
	x = ((x >> 2) & 0x33333333) + (x & 0x33333333)
	x = ((x >> 4) + x) & 0x0f0f0f0f
	x *= 0x01010101
	return x >> 24
}
