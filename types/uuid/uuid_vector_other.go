// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (!amd64 && !arm64) || purego

package uuid

func formatVector(u *UUID, dst *[36]byte) {
	formatScalar(u, dst)
}

func parseVector(s string) (UUID, bool) {
	return parseScalar(s)
}
