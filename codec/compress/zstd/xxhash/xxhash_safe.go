// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xxhash

import "unsafe"

// Sum64String computes the 64-bit xxHash digest of s.
func Sum64String(s string) uint64 {
	if len(s) == 0 {
		return Sum64(nil)
	}
	return Sum64(unsafe.Slice(unsafe.StringData(s), len(s)))
}

// WriteString adds more data to d. It always returns len(s), nil.
func (d *Digest) WriteString(s string) (n int, err error) {
	if len(s) == 0 {
		return 0, nil
	}
	return d.Write(unsafe.Slice(unsafe.StringData(s), len(s)))
}
