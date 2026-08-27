// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hpack_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/net/hpack"
)

func FuzzHPACKDecode(f *testing.F) {
	f.Add([]byte("\x82\x86\x84\x41\x0fwww.example.com"))
	f.Add([]byte("\x40\x0acustom-key\x0dcustom-header"))
	f.Add([]byte("\x00\x85\xf2\xb2\x4a\x65\x49\x24"))
	f.Add([]byte(""))
	f.Add([]byte("\xff\xff\xff\xff"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dec := hpack.NewDecoder(4096, nil)
		_, _ = dec.DecodeFull(data)
	})
}

func FuzzHPACKEncodeDecodeRoundtrip(f *testing.F) {
	f.Add("content-type", "application/json")
	f.Add(":path", "/index.html")
	f.Add("user-agent", "aoni/1.0.0")
	f.Add("x-custom-token", "secret12345")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, name, val string) {
		if name == "" {
			return
		}

		var buf bytes.Buffer
		enc := hpack.NewEncoder(&buf)
		if err := enc.WriteField(hpack.HeaderField{Name: name, Value: val}); err != nil {
			return
		}

		dec := hpack.NewDecoder(4096, nil)
		fields, err := dec.DecodeFull(buf.Bytes())
		if err != nil {
			t.Fatalf("failed to decode encoded field %s: %s: %v", name, val, err)
		}

		if len(fields) != 1 {
			t.Fatalf("expected 1 decoded field, got %d", len(fields))
		}
		if fields[0].Name != name || fields[0].Value != val {
			t.Fatalf("roundtrip mismatch: got %s: %s, want %s: %s", fields[0].Name, fields[0].Value, name, val)
		}
	})
}
