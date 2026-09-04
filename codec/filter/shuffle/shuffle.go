// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package shuffle implements a high-performance byte transposition (shuffle) filter
// for multi-byte numeric arrays (e.g. FP16, BF16, FP32, FP64, ML tensors, safetensors,
// time-series, and geometric vertices).
//
// Shuffling groups identical significance bytes (such as IEEE-754 sign/exponent bytes)
// contiguously, drastically improving entropy and LZ-based compression ratios.
package shuffle

import "errors"

// Standard element widths in bytes.
const (
	WidthFP16 = 2
	WidthFP32 = 4
	WidthFP64 = 8
)

// ErrInvalidWidth is returned when element width is not positive.
var ErrInvalidWidth = errors.New("shuffle: element width must be positive")

// Encode transposes multi-byte elements of src into dst.
// If dst has insufficient capacity, a new slice is allocated.
// Any trailing bytes that do not form a complete word are copied un-transposed.
func Encode(src, dst []byte, width int) ([]byte, error) {
	if width <= 0 {
		return nil, ErrInvalidWidth
	}
	n := len(src)
	if n == 0 || width == 1 {
		if cap(dst) < n {
			dst = make([]byte, n)
		} else {
			dst = dst[:n]
		}
		copy(dst, src)
		return dst, nil
	}

	if cap(dst) < n {
		dst = make([]byte, n)
	} else {
		dst = dst[:n]
	}

	elements := n / width
	remainder := n % width

	switch width {
	case 2:
		// Specialized 2-byte word (FP16 / BF16 / INT16)
		p0 := 0
		p1 := elements
		for i := 0; i < elements; i++ {
			idx := i << 1
			dst[p0+i] = src[idx]
			dst[p1+i] = src[idx+1]
		}
	case 4:
		// Specialized 4-byte word (FP32 / INT32)
		p0 := 0
		p1 := elements
		p2 := elements * 2
		p3 := elements * 3
		for i := 0; i < elements; i++ {
			idx := i << 2
			dst[p0+i] = src[idx]
			dst[p1+i] = src[idx+1]
			dst[p2+i] = src[idx+2]
			dst[p3+i] = src[idx+3]
		}
	case 8:
		// Specialized 8-byte word (FP64 / INT64)
		p0 := 0
		p1 := elements
		p2 := elements * 2
		p3 := elements * 3
		p4 := elements * 4
		p5 := elements * 5
		p6 := elements * 6
		p7 := elements * 7
		for i := 0; i < elements; i++ {
			idx := i << 3
			dst[p0+i] = src[idx]
			dst[p1+i] = src[idx+1]
			dst[p2+i] = src[idx+2]
			dst[p3+i] = src[idx+3]
			dst[p4+i] = src[idx+4]
			dst[p5+i] = src[idx+5]
			dst[p6+i] = src[idx+6]
			dst[p7+i] = src[idx+7]
		}
	default:
		// Generic N-byte word
		for b := 0; b < width; b++ {
			offset := b * elements
			for i := 0; i < elements; i++ {
				dst[offset+i] = src[i*width+b]
			}
		}
	}

	if remainder > 0 {
		tailStart := elements * width
		copy(dst[tailStart:], src[tailStart:])
	}

	return dst, nil
}

// Decode reverses the byte transposition, restoring the original multi-byte words from src into dst.
func Decode(src, dst []byte, width int) ([]byte, error) {
	if width <= 0 {
		return nil, ErrInvalidWidth
	}
	n := len(src)
	if n == 0 || width == 1 {
		if cap(dst) < n {
			dst = make([]byte, n)
		} else {
			dst = dst[:n]
		}
		copy(dst, src)
		return dst, nil
	}

	if cap(dst) < n {
		dst = make([]byte, n)
	} else {
		dst = dst[:n]
	}

	elements := n / width
	remainder := n % width

	switch width {
	case 2:
		p0 := 0
		p1 := elements
		for i := 0; i < elements; i++ {
			idx := i << 1
			dst[idx] = src[p0+i]
			dst[idx+1] = src[p1+i]
		}
	case 4:
		p0 := 0
		p1 := elements
		p2 := elements * 2
		p3 := elements * 3
		for i := 0; i < elements; i++ {
			idx := i << 2
			dst[idx] = src[p0+i]
			dst[idx+1] = src[p1+i]
			dst[idx+2] = src[p2+i]
			dst[idx+3] = src[p3+i]
		}
	case 8:
		p0 := 0
		p1 := elements
		p2 := elements * 2
		p3 := elements * 3
		p4 := elements * 4
		p5 := elements * 5
		p6 := elements * 6
		p7 := elements * 7
		for i := 0; i < elements; i++ {
			idx := i << 3
			dst[idx] = src[p0+i]
			dst[idx+1] = src[p1+i]
			dst[idx+2] = src[p2+i]
			dst[idx+3] = src[p3+i]
			dst[idx+4] = src[p4+i]
			dst[idx+5] = src[p5+i]
			dst[idx+6] = src[p6+i]
			dst[idx+7] = src[p7+i]
		}
	default:
		for b := 0; b < width; b++ {
			offset := b * elements
			for i := 0; i < elements; i++ {
				dst[i*width+b] = src[offset+i]
			}
		}
	}

	if remainder > 0 {
		tailStart := elements * width
		copy(dst[tailStart:], src[tailStart:])
	}

	return dst, nil
}
