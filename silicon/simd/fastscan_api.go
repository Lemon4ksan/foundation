// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build amd64 && !purego

package simd

import (
	"bytes"
	"math/bits"
	"simd/archsimd"
	"unsafe"
)

// ScanByteVector scans data for target byte using 256-bit AVX2 SIMD instructions with 4-way loop unrolling.
func ScanByteVector(data []byte, target byte) int {
	n := len(data)
	if n == 0 {
		return -1
	}

	if n >= 32 && hasAVX2 {
		targetVec := archsimd.BroadcastUint8x32(target)
		i := 0

		// 4-way loop unrolling (128 bytes per iteration) for high throughput on large buffers
		for i+128 <= n {
			c0 := archsimd.LoadUint8x32(data[i : i+32])
			c1 := archsimd.LoadUint8x32(data[i+32 : i+64])
			c2 := archsimd.LoadUint8x32(data[i+64 : i+96])
			c3 := archsimd.LoadUint8x32(data[i+96 : i+128])

			m0 := c0.Equal(targetVec).ToBits()
			m1 := c1.Equal(targetVec).ToBits()
			m2 := c2.Equal(targetVec).ToBits()
			m3 := c3.Equal(targetVec).ToBits()

			if (m0 | m1 | m2 | m3) != 0 {
				if m0 != 0 {
					return i + bits.TrailingZeros32(m0)
				}
				if m1 != 0 {
					return i + 32 + bits.TrailingZeros32(m1)
				}
				if m2 != 0 {
					return i + 64 + bits.TrailingZeros32(m2)
				}
				return i + 96 + bits.TrailingZeros32(m3)
			}
			i += 128
		}

		// 32-byte chunk processing
		for i+32 <= n {
			chunk := archsimd.LoadUint8x32(data[i : i+32])
			mask := chunk.Equal(targetVec).ToBits()
			if mask != 0 {
				return i + bits.TrailingZeros32(mask)
			}
			i += 32
		}

		// Tail scalar processing
		if i < n {
			if idx := bytes.IndexByte(data[i:], target); idx >= 0 {
				return i + idx
			}
		}
		return -1
	}

	return bytes.IndexByte(data, target)
}

// IndexCRLFCRLFVector searches for "\r\n\r\n" in data and returns the index of the first byte after the sequence.
func IndexCRLFCRLFVector(data []byte) int {
	n := len(data)
	if n < 4 {
		return -1
	}

	if n >= 32 && hasAVX2 {
		crVec := archsimd.BroadcastUint8x32('\r')
		i := 0

		for i+32 <= n {
			chunk := archsimd.LoadUint8x32(data[i : i+32])
			mask := chunk.Equal(crVec).ToBits()

			for mask != 0 {
				bit := bits.TrailingZeros32(mask)
				pos := i + bit
				if pos+3 < n {
					val := *(*uint32)(unsafe.Pointer(&data[pos]))
					if val == CRLFCRLFUint32 {
						return pos + 4
					}
				}
				mask &= mask - 1
			}
			i += 32
		}

		for ; i < n; i++ {
			if data[i] == '\r' && i+3 < n {
				val := *(*uint32)(unsafe.Pointer(&data[i]))
				if val == CRLFCRLFUint32 {
					return i + 4
				}
			}
		}
		return -1
	}

	return IndexDoubleCRLF(data)
}
