// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xxhash_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/zstd/xxhash"
)

func TestXXHashSum64(t *testing.T) {
	data := []byte("foundation-high-performance-runtime")
	h1 := xxhash.Sum64(data)

	if h1 == 0 {
		t.Fatalf("Sum64 returned 0")
	}

	d := xxhash.New()
	d.Write(data)
	if d.Sum64() != h1 {
		t.Fatalf("Digest.Sum64 (%d) != Sum64 (%d)", d.Sum64(), h1)
	}

	// Test Marshal/Unmarshal
	marshaled, err := d.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary failed: %v", err)
	}

	d2 := xxhash.New()
	if err := d2.UnmarshalBinary(marshaled); err != nil {
		t.Fatalf("UnmarshalBinary failed: %v", err)
	}
	if d2.Sum64() != h1 {
		t.Fatalf("Unmarshaled digest mismatch: got %d, want %d", d2.Sum64(), h1)
	}
}

func BenchmarkXXHash(b *testing.B) {
	data := make([]byte, 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = xxhash.Sum64(data)
	}
}
