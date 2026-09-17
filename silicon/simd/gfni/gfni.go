package gfni

// MultiplyGF2P8 multiplies two 8-bit integers in the Galois Field GF(2^8).
// This is a software fallback/stub for the VGF2P8MULB instruction.
func MultiplyGF2P8(a, b byte) byte {
	var p byte = 0
	for range 8 {
		if (b & 1) == 1 {
			p ^= a
		}
		hiBitSet := (a & 0x80) != 0
		a <<= 1
		if hiBitSet {
			a ^= 0x1B // GF(2^8) reduction polynomial x^8 + x^4 + x^3 + x + 1
		}
		b >>= 1
	}
	return p
}

// MultiplyGF2P8Vector multiplies a vector of bytes by a scalar in GF(2^8).
func MultiplyGF2P8Vector(dst, src []byte, scalar byte) {
	for i := range src {
		dst[i] = MultiplyGF2P8(src[i], scalar)
	}
}
