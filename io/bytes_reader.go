// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package io

import (
	"bytes"
	"io"
	"slices"
)

var bomUTF8 = []byte{0xEF, 0xBB, 0xBF}

// BytesReader is implemented by response body streams or buffer wrappers
// that expose pre-buffered contiguous payload bytes to avoid io.ReadAll growing-buffer overhead.
type BytesReader interface {
	// Bytes returns the contiguous buffered payload and whether the backing store is volatile.
	// When volatile is true (e.g. off-heap memory, pooled fasthttp buffer, mmap page),
	// callers MUST NOT retain or mutate the returned slice beyond the reader's lifecycle (Close).
	Bytes() (data []byte, volatile bool)
}

// InspectBytes attempts to extract contiguous payload bytes from r without allocations.
// Returns the data slice, volatility flag, and a boolean indicating whether the reader supports contiguous byte inspection.
func InspectBytes(r io.Reader) (data []byte, volatile bool, ok bool) {
	if r == nil {
		return nil, false, false
	}

	if br, ok := r.(BytesReader); ok {
		b, vol := br.Bytes()
		return b, vol, true
	}

	if buf, ok := r.(*bytes.Buffer); ok {
		return buf.Bytes(), false, true
	}

	return nil, false, false
}

// ReadAllSafe returns the payload bytes, guaranteeing that the returned slice
// is safe to retain independently of the reader's lifecycle.
// If r provides contiguous non-volatile bytes, returns them without allocation;
// if volatile, returns a cloned slice; otherwise falls back to [io.ReadAll].
func ReadAllSafe(r io.Reader) ([]byte, error) {
	if data, volatile, ok := InspectBytes(r); ok {
		if len(data) == 0 {
			return nil, nil
		}

		if volatile {
			return slices.Clone(data), nil
		}

		return data, nil
	}

	return io.ReadAll(r)
}

// StripBOMBytes detects and strips UTF-8, UTF-16LE, and UTF-16BE Byte Order Marks (BOM) from a byte slice.
func StripBOMBytes(data []byte) []byte {
	if len(data) >= 3 && bytes.HasPrefix(data, bomUTF8) {
		return data[3:]
	}

	if len(data) >= 2 {
		if (data[0] == 0xFE && data[1] == 0xFF) || (data[0] == 0xFF && data[1] == 0xFE) {
			return data[2:]
		}
	}

	return data
}
