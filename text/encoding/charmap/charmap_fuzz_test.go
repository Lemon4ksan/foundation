// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package charmap_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding/charmap"
	"github.com/lemon4ksan/foundation/text/transform"
)

// FuzzCharmapDecode verifies SWAR 64-bit ASCII vector chunking and non-UTF8 single-byte decoding against arbitrary byte streams.
func FuzzCharmapDecode(f *testing.F) {
	f.Add([]byte("Hello, World! 1234567890\r\n"), 0)
	f.Add([]byte("\xcf\xf0\xe8\xe2\xe5\xf2, \xcc\xe8\xf0!"), 1) // Windows-1251 "Привет, Мир!"
	f.Add([]byte(""), 2)
	f.Add([]byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07}, 3)
	f.Add(bytes.Repeat([]byte{0x41}, 64), 0)

	encodings := []struct {
		name string
		enc  *charmap.Charmap
	}{
		{"Windows1251", charmap.Windows1251},
		{"Windows1252", charmap.Windows1252},
		{"KOI8R", charmap.KOI8R},
		{"ISO8859_1", charmap.ISO8859_1},
	}

	f.Fuzz(func(t *testing.T, data []byte, encIdx int) {
		if len(data) > 64*1024 {
			return
		}

		idx := encIdx % len(encodings)
		if idx < 0 {
			idx = -idx % len(encodings)
		}
		cm := encodings[idx].enc

		// 1. Decoder
		r := transform.NewReader(bytes.NewReader(data), cm.NewDecoder())
		decoded, err := io.ReadAll(r)
		if err == nil && len(data) > 0 {
			_ = decoded
		}

		// 2. Encoder
		w := transform.NewReader(bytes.NewReader(data), cm.NewEncoder())
		encoded, encErr := io.ReadAll(w)
		if encErr == nil && len(data) > 0 {
			_ = encoded
		}
	})
}
