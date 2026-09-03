// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec

import "math"

// ShannonEntropy calculates the Shannon entropy of a byte sample in bits per byte (range: 0.0 to 8.0).
func ShannonEntropy(sample []byte) float64 {
	if len(sample) == 0 {
		return 0
	}
	var counts [256]int
	for _, b := range sample {
		counts[b]++
	}
	n := float64(len(sample))
	var entropy float64
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / n
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// IsIncompressibleSample returns true if the sample has entropy >= 7.92 bits/byte (indicative of dense compressed/encrypted data).
func IsIncompressibleSample(sample []byte) bool {
	if len(sample) < 512 {
		return false
	}
	return ShannonEntropy(sample) >= 7.92
}
