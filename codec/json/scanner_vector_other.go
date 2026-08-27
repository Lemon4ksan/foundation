// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !amd64 || purego

package json

func skipWhitespaceVector(data []byte, cursor int) int {
	return skipWhitespaceScalar(data, cursor)
}

func scanStringSpecialVector(data []byte, cursor int) int {
	return scanStringSpecialScalar(data, cursor)
}
