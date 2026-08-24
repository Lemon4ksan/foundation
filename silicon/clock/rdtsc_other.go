// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build (!amd64 && !arm64) || purego

package clock

import "time"

func rdtsc() uint64 {
	return uint64(time.Now().UnixNano())
}
