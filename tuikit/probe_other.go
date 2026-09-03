// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows && !linux && !darwin && !freebsd && !openbsd && !netbsd

package tuikit

func probeFd(_ uintptr) bool {
	return false
}
