// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package delta

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
)

func TestDelta_Roundtrip(t *testing.T) {
	distances := []int{1, 2, 3, 4, 8, 16, 256}
	r := rand.New(rand.NewSource(42))

	original := make([]byte, 8192)
	r.Read(original)

	for _, dist := range distances {
		t.Run(t.Name(), func(t *testing.T) {
			buf := make([]byte, len(original))
			copy(buf, original)

			encFilter, err := NewFilter(dist)
			if err != nil {
				t.Fatalf("dist %d: NewFilter failed: %v", dist, err)
			}
			encFilter.Encode(buf)

			decFilter, err := NewFilter(dist)
			if err != nil {
				t.Fatalf("dist %d: NewFilter failed: %v", dist, err)
			}
			decFilter.Decode(buf)

			if !bytes.Equal(buf, original) {
				t.Fatalf("dist %d: roundtrip mismatch", dist)
			}
		})
	}
}

func TestDelta_Streaming(t *testing.T) {
	distances := []int{1, 4, 16}
	r := rand.New(rand.NewSource(99))

	original := make([]byte, 4096)
	r.Read(original)

	for _, dist := range distances {
		var encoded bytes.Buffer
		writer, err := NewWriter(&encoded, dist)
		if err != nil {
			t.Fatalf("NewWriter failed: %v", err)
		}

		// Write in uneven chunks
		for i := 0; i < len(original); {
			chunkSize := min(17, len(original)-i)
			n, err := writer.Write(original[i : i+chunkSize])
			if err != nil || n != chunkSize {
				t.Fatalf("Write failed: %v", err)
			}
			i += chunkSize
		}

		reader, err := NewReader(&encoded, dist)
		if err != nil {
			t.Fatalf("NewReader failed: %v", err)
		}

		decoded, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}

		if !bytes.Equal(decoded, original) {
			t.Fatalf("dist %d: streaming roundtrip mismatch", dist)
		}
	}
}
