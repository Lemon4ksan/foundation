// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit

import "io"

// MmapFile represents a memory-mapped file supporting random-access reads and zero-copy slicing.
type MmapFile interface {
	io.ReaderAt
	io.Closer
	Bytes() []byte
	Len() int64
}
