// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package varint_test

import (
	"bytes"
	"math"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/net/quic/varint"
)

func TestStressVarint_AllBoundariesRoundTrip(t *testing.T) {
	boundaries := []uint64{
		0, 1, 2,
		varint.Max1Byte - 1, varint.Max1Byte, varint.Max1Byte + 1, varint.Max1Byte + 2,
		varint.Max2Byte - 1, varint.Max2Byte, varint.Max2Byte + 1, varint.Max2Byte + 2,
		varint.Max4Byte - 1, varint.Max4Byte, varint.Max4Byte + 1, varint.Max4Byte + 2,
		varint.Max - 2, varint.Max - 1, varint.Max,
	}

	for _, v := range boundaries {
		enc := varint.EncodeVarint(v)
		var buf [8]byte
		n := varint.EncodeVarintSlice(v, buf[:])
		if !bytes.Equal(enc, buf[:n]) {
			t.Fatalf("v=%d: EncodeVarint and EncodeVarintSlice diverge", v)
		}

		dec, readLen, err := varint.DecodeVarint(enc)
		if err != nil {
			t.Fatalf("v=%d: DecodeVarint failed: %v", v, err)
		}
		if dec != v {
			t.Fatalf("v=%d: DecodeVarint returned %d", v, dec)
		}
		if readLen != len(enc) {
			t.Fatalf("v=%d: readLen=%d != len(enc)=%d", v, readLen, len(enc))
		}
	}
}

func TestStressVarint_OverflowExhaustive(t *testing.T) {
	overflows := []uint64{
		1 << 62,
		(1 << 62) + 1,
		(1 << 62) + 0x123456,
		1 << 63,
		(1 << 63) | (1 << 62),
		math.MaxUint64 - 1,
		math.MaxUint64,
	}

	for _, v := range overflows {
		assertPanic(t, "EncodeVarint", func() {
			_ = varint.EncodeVarint(v)
		})
		var buf [8]byte
		assertPanic(t, "EncodeVarintSlice", func() {
			_ = varint.EncodeVarintSlice(v, buf[:])
		})
	}
}

func TestStressVarint_IteratorsEarlyBreakAndTruncation(t *testing.T) {
	values := []uint64{
		0, 63, 64, 16383, 16384, 1073741823, 1073741824, varint.Max,
	}

	var stream []byte
	for _, v := range values {
		stream = append(stream, varint.EncodeVarint(v)...)
	}

	// 1. Break at every possible step for DecodeSeq
	for stopAt := 0; stopAt <= len(values); stopAt++ {
		count := 0
		for val, n := range varint.DecodeSeq(stream) {
			if count == stopAt {
				break
			}
			if val != values[count] {
				t.Fatalf("stopAt %d: step %d val=%d, want %d", stopAt, count, val, values[count])
			}
			_ = n
			count++
		}
		if count != stopAt {
			t.Fatalf("stopAt %d: iterated %d times", stopAt, count)
		}
	}

	// 2. Break at every possible step for Values
	for stopAt := 0; stopAt <= len(values); stopAt++ {
		count := 0
		for val := range varint.Values(stream) {
			if count == stopAt {
				break
			}
			if val != values[count] {
				t.Fatalf("stopAt %d: step %d val=%d, want %d", stopAt, count, val, values[count])
			}
			count++
		}
		if count != stopAt {
			t.Fatalf("stopAt %d: iterated %d times", stopAt, count)
		}
	}

	// 3. Truncated stream in the middle of an 8-byte varint
	truncated := append([]byte(nil), stream...)
	truncated = append(truncated, 0xc0, 0x01, 0x02) // 8-byte tag with only 3 bytes
	var decodedTrunc []uint64
	for v := range varint.Values(truncated) {
		decodedTrunc = append(decodedTrunc, v)
	}
	if len(decodedTrunc) != len(values) {
		t.Fatalf("Values with truncated tail yielded %d, want %d", len(decodedTrunc), len(values))
	}
}

func TestStressVarint_Parallel(t *testing.T) {
	const goroutines = 16
	const iters = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			var buf [8]byte
			for i := 0; i < iters; i++ {
				val := uint64((id*1000 + i) % 100000)
				n := varint.EncodeVarintSlice(val, buf[:])
				dec, rLen, err := varint.DecodeVarint(buf[:n])
				if err != nil || dec != val || rLen != n {
					t.Errorf("parallel failed for id %d i %d", id, i)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}
