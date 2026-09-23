// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package varint_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/varint"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestValues(t *testing.T) {
	t.Parallel()

	expectedVals := []uint64{25, 494, 15293, 494872, 1073741823, 4611686018427387900}
	var encoded []byte
	for _, v := range expectedVals {
		encoded = varint.Append(encoded, v)
	}

	t.Run("all values", func(t *testing.T) {
		var recovered []uint64
		for val := range varint.Values(encoded) {
			recovered = append(recovered, val)
		}
		assert.Equal(t, expectedVals, recovered)
	})

	t.Run("empty slice", func(t *testing.T) {
		count := 0
		for range varint.Values(nil) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("early exit", func(t *testing.T) {
		count := 0
		for range varint.Values(encoded) {
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
	})

	t.Run("truncated input", func(t *testing.T) {
		truncated := append([]byte(nil), encoded[:len(encoded)-3]...)
		count := 0
		for range varint.Values(truncated) {
			count++
		}
		assert.Equal(t, len(expectedVals)-1, count)
	})
}

func TestReadSeq(t *testing.T) {
	t.Parallel()

	expectedVals := []uint64{1, 100, 10000, 1000000, 100000000}
	var buf []byte
	for _, v := range expectedVals {
		buf = varint.Append(buf, v)
	}

	t.Run("successive reads", func(t *testing.T) {
		reader := bytes.NewReader(buf)
		var recovered []uint64
		for val, err := range varint.ReadSeq(reader) {
			require.NoError(t, err)
			recovered = append(recovered, val)
		}
		assert.Equal(t, expectedVals, recovered)
	})

	t.Run("early exit", func(t *testing.T) {
		reader := bytes.NewReader(buf)
		count := 0
		for _, err := range varint.ReadSeq(reader) {
			require.NoError(t, err)
			count++
			if count == 2 {
				break
			}
		}
		assert.Equal(t, 2, count)
	})

	t.Run("error terminates", func(t *testing.T) {
		// Truncated multi-byte varint
		truncated := append(buf, 0x80) // 4-byte varint with only 1 byte available
		reader := bytes.NewReader(truncated)

		hadError := false
		count := 0
		for _, err := range varint.ReadSeq(reader) {
			if err != nil {
				hadError = true
				break
			}
			count++
		}
		assert.True(t, hadError)
		assert.Equal(t, len(expectedVals), count)
	})
}

func BenchmarkValues_ZeroAlloc(b *testing.B) {
	var encoded []byte
	for _, v := range []uint64{10, 200, 3000, 40000, 500000, 6000000, 70000000} {
		encoded = varint.Append(encoded, v)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for v := range varint.Values(encoded) {
			if v == 0 {
				b.Fatal("unexpected zero")
			}
		}
	}
}

func BenchmarkReadSeq_ZeroAlloc(b *testing.B) {
	var encoded []byte
	for _, v := range []uint64{10, 200, 3000, 40000, 500000, 6000000, 70000000} {
		encoded = varint.Append(encoded, v)
	}
	reader := bytes.NewReader(encoded)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		reader.Reset(encoded)
		for v, err := range varint.ReadSeq(reader) {
			if err != nil || v == 0 {
				b.Fatal("unexpected error or zero")
			}
		}
	}
}
