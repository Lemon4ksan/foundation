// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package varint_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/net/quic/varint"
)

func TestEncodeDecodeVarint(t *testing.T) {
	testCases := []uint64{
		0,
		1,
		37,
		63, // 1 byte max
		64, // 2 bytes min
		15293,
		16383, // 2 bytes max
		16384, // 4 bytes min
		494878333,
		1073741823, // 4 bytes max
		1073741824, // 8 bytes min
		15128880994,
		(1 << 62) - 1, // 8 bytes max
	}

	for _, tc := range testCases {
		enc := varint.EncodeVarint(tc)
		var buf [8]byte
		n := varint.EncodeVarintSlice(tc, buf[:])

		if !bytes.Equal(enc, buf[:n]) {
			t.Fatalf("EncodeVarint vs EncodeVarintSlice mismatch for %d: %x vs %x", tc, enc, buf[:n])
		}

		dec, readLen, err := varint.DecodeVarint(enc)
		if err != nil {
			t.Fatalf("DecodeVarint(%d) failed: %v", tc, err)
		}
		if dec != tc {
			t.Fatalf("DecodeVarint(%d) = %d", tc, dec)
		}
		if readLen != len(enc) {
			t.Fatalf("DecodeVarint(%d) readLen = %d, want %d", tc, readLen, len(enc))
		}
	}
}

func TestDecodeVarintTruncated(t *testing.T) {
	// 2-byte tag with 1 byte input
	_, _, err := varint.DecodeVarint([]byte{0x40})
	if err != varint.ErrTruncated {
		t.Fatalf("expected ErrTruncated, got %v", err)
	}

	// 4-byte tag with 2 bytes input
	_, _, err = varint.DecodeVarint([]byte{0x80, 0x01})
	if err != varint.ErrTruncated {
		t.Fatalf("expected ErrTruncated, got %v", err)
	}

	// 8-byte tag with 4 bytes input
	_, _, err = varint.DecodeVarint([]byte{0xc0, 0x00, 0x00, 0x01})
	if err != varint.ErrTruncated {
		t.Fatalf("expected ErrTruncated, got %v", err)
	}

	// Empty payload
	_, _, err = varint.DecodeVarint(nil)
	if err != varint.ErrTruncated {
		t.Fatalf("expected ErrTruncated, got %v", err)
	}
}

func BenchmarkEncodeVarintSlice(b *testing.B) {
	var buf [8]byte
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = varint.EncodeVarintSlice(494878333, buf[:])
	}
}

func BenchmarkDecodeVarint(b *testing.B) {
	encoded := varint.EncodeVarint(494878333)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = varint.DecodeVarint(encoded)
	}
}
