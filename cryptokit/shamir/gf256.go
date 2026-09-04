// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shamir

var (
	expTable [512]byte
	logTable [256]byte
)

func init() {
	x := byte(1)
	for i := 0; i < 255; i++ {
		expTable[i] = x
		expTable[i+255] = x
		logTable[x] = byte(i)

		// Multiply by 3 in GF(2^8) modulo 0x11B (AES polynomial)
		// x * 3 = (x * 2) ^ x
		x2 := byte(x << 1)
		if (x & 0x80) != 0 {
			x2 ^= 0x1B
		}
		x = x2 ^ x
	}
	logTable[0] = 0
}

// Add adds two elements in GF(2^8) (equivalent to XOR).
func Add(a, b byte) byte {
	return a ^ b
}

// Sub subtracts two elements in GF(2^8) (equivalent to XOR).
func Sub(a, b byte) byte {
	return a ^ b
}

// Mul multiplies two elements in GF(2^8) using lookup tables.
func Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return expTable[int(logTable[a])+int(logTable[b])]
}

// Div divides a by b in GF(2^8). Panics if b == 0.
func Div(a, b byte) byte {
	if b == 0 {
		panic("shamir: division by zero in GF(2^8)")
	}
	if a == 0 {
		return 0
	}
	return expTable[int(logTable[a])-int(logTable[b])+255]
}

// evalPoly evaluates polynomial coeffs at point x using Horner's method in GF(2^8).
func evalPoly(coeffs []byte, x byte) byte {
	if x == 0 {
		return coeffs[0]
	}
	deg := len(coeffs) - 1
	out := coeffs[deg]
	for i := deg - 1; i >= 0; i-- {
		out = Add(Mul(out, x), coeffs[i])
	}
	return out
}

// lagrange0 computes the Lagrange interpolation at x=0 given x and y sample points.
func lagrange0(xs, ys []byte) byte {
	var secret byte
	k := len(xs)
	for j := 0; j < k; j++ {
		num := byte(1)
		den := byte(1)
		for m := 0; m < k; m++ {
			if m == j {
				continue
			}
			num = Mul(num, xs[m])
			den = Mul(den, Add(xs[j], xs[m]))
		}
		basis := Div(num, den)
		secret = Add(secret, Mul(ys[j], basis))
	}
	return secret
}
