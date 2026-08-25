// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (!amd64 && !arm64) || purego

package hex

func encodeVector(dst, src []byte) int {
	return encodeScalar(dst, src)
}

func decodeVector(dst, src []byte) (int, error) {
	return decodeScalar(dst, src)
}
