// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package varint_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/varint"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestValues_Adversarial_EarlyExit(t *testing.T) {
	t.Parallel()

	var buf []byte
	for i := range 10 {
		buf = varint.Append(buf, uint64(i*100))
	}

	t.Run("exit after 0 (first yield returns false)", func(t *testing.T) {
		count := 0
		varint.Values(buf)(func(v uint64) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})

	t.Run("exit after 1 iteration", func(t *testing.T) {
		count := 0
		for v := range varint.Values(buf) {
			count++
			assert.Equal(t, uint64(0), v)
			if count == 1 {
				break
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("exit mid sequence", func(t *testing.T) {
		count := 0
		for range varint.Values(buf) {
			count++
			if count == 4 {
				break
			}
		}
		assert.Equal(t, 4, count)
	})
}

func TestValues_Adversarial_MalformedAndTruncated(t *testing.T) {
	t.Parallel()

	t.Run("empty or nil slice", func(t *testing.T) {
		count := 0
		for range varint.Values(nil) {
			count++
		}
		assert.Equal(t, 0, count)

		count = 0
		for range varint.Values([]byte{}) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("truncated varints across byte lengths", func(t *testing.T) {
		// 2-byte varint prefix (0b01xxxxxx) with only 1 byte
		trunc2 := []byte{0x40}
		count := 0
		for range varint.Values(trunc2) {
			count++
		}
		assert.Equal(t, 0, count)

		// 4-byte varint prefix (0b10xxxxxx) with 1, 2, 3 bytes
		for l := 1; l <= 3; l++ {
			trunc4 := make([]byte, l)
			trunc4[0] = 0x80
			count = 0
			for range varint.Values(trunc4) {
				count++
			}
			assert.Equal(t, 0, count)
		}

		// 8-byte varint prefix (0b11xxxxxx) with 1 to 7 bytes
		for l := 1; l <= 7; l++ {
			trunc8 := make([]byte, l)
			trunc8[0] = 0xc0
			count = 0
			for range varint.Values(trunc8) {
				count++
			}
			assert.Equal(t, 0, count)
		}
	})

	t.Run("valid varints followed by truncated bytes", func(t *testing.T) {
		var buf []byte
		buf = varint.Append(buf, 42)
		buf = varint.Append(buf, 1000)
		// Append truncated 4-byte varint (only 2 bytes)
		buf = append(buf, 0x80, 0x01)

		var vals []uint64
		for v := range varint.Values(buf) {
			vals = append(vals, v)
		}
		require.Equal(t, 2, len(vals))
		assert.Equal(t, uint64(42), vals[0])
		assert.Equal(t, uint64(1000), vals[1])
	})
}

type customErrByteReader struct {
	data []byte
	err  error
}

func (c *customErrByteReader) ReadByte() (byte, error) {
	if len(c.data) > 0 {
		b := c.data[0]
		c.data = c.data[1:]
		return b, nil
	}
	return 0, c.err
}

func TestReadSeq_Adversarial(t *testing.T) {
	t.Parallel()

	t.Run("empty reader", func(t *testing.T) {
		r := bytes.NewReader(nil)
		count := 0
		for _, err := range varint.ReadSeq(r) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("early exit after 0 (first yield returns false)", func(t *testing.T) {
		var buf []byte
		buf = varint.Append(buf, 10)
		buf = varint.Append(buf, 20)
		r := bytes.NewReader(buf)

		count := 0
		varint.ReadSeq(r)(func(v uint64, err error) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})

	t.Run("early exit mid-sequence", func(t *testing.T) {
		var buf []byte
		for i := range 5 {
			buf = varint.Append(buf, uint64(i+1))
		}
		r := bytes.NewReader(buf)

		count := 0
		for _, err := range varint.ReadSeq(r) {
			require.NoError(t, err)
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
	})

	t.Run("truncated mid-varint yields io.ErrUnexpectedEOF", func(t *testing.T) {
		// Valid varint 999, followed by truncated 4-byte varint (2 bytes total)
		var buf []byte
		buf = varint.Append(buf, 999)
		buf = append(buf, 0x80, 0x01)
		r := bytes.NewReader(buf)

		var vals []uint64
		var errs []error
		for v, err := range varint.ReadSeq(r) {
			if err != nil {
				errs = append(errs, err)
			} else {
				vals = append(vals, v)
			}
		}
		require.Equal(t, 1, len(vals))
		assert.Equal(t, uint64(999), vals[0])
		require.Equal(t, 1, len(errs))
		assert.True(t, errors.Is(errs[0], io.ErrUnexpectedEOF))
	})

	t.Run("custom read error yields error and terminates", func(t *testing.T) {
		customErr := errors.New("underlying network failure")
		r := &customErrByteReader{
			data: varint.Append(nil, 555),
			err:  customErr,
		}

		var vals []uint64
		var errs []error
		for v, err := range varint.ReadSeq(r) {
			if err != nil {
				errs = append(errs, err)
			} else {
				vals = append(vals, v)
			}
		}
		require.Equal(t, 1, len(vals))
		assert.Equal(t, uint64(555), vals[0])
		require.Equal(t, 1, len(errs))
		assert.True(t, errors.Is(errs[0], customErr))
	})
}

func TestVarint_Adversarial_ExtremeInputs(t *testing.T) {
	t.Parallel()

	t.Run("all boundary values roundtrip", func(t *testing.T) {
		boundaries := []uint64{
			0,
			1,
			63,                  // max 1-byte
			64,                  // min 2-byte
			16383,               // max 2-byte
			16384,               // min 4-byte
			1073741823,          // max 4-byte
			1073741824,          // min 8-byte
			4611686018427387903, // max 8-byte (2^62 - 1)
		}

		var buf []byte
		for _, b := range boundaries {
			buf = varint.Append(buf, b)
		}

		var recovered []uint64
		for v := range varint.Values(buf) {
			recovered = append(recovered, v)
		}
		require.Equal(t, len(boundaries), len(recovered))
		for i := range boundaries {
			assert.Equal(t, boundaries[i], recovered[i])
		}

		// Verify same with ReadSeq
		r := bytes.NewReader(buf)
		var fromSeq []uint64
		for v, err := range varint.ReadSeq(r) {
			require.NoError(t, err)
			fromSeq = append(fromSeq, v)
		}
		assert.Equal(t, boundaries, fromSeq)
	})

	t.Run("massive stream 50000 varints", func(t *testing.T) {
		n := 50000
		var buf []byte
		for i := range n {
			buf = varint.Append(buf, uint64(i))
		}

		count := 0
		expected := uint64(0)
		for v := range varint.Values(buf) {
			assert.Equal(t, expected, v)
			expected++
			count++
		}
		assert.Equal(t, n, count)
	})
}

func TestVarint_Adversarial_Concurrency(t *testing.T) {
	t.Parallel()

	var buf []byte
	for i := range 100 {
		buf = varint.Append(buf, uint64(i*7))
	}

	var wg sync.WaitGroup
	goroutines := 30

	for range goroutines {
		wg.Add(2)

		go func() {
			defer wg.Done()
			count := 0
			for range varint.Values(buf) {
				count++
			}
			assert.Equal(t, 100, count)
		}()

		go func() {
			defer wg.Done()
			r := bytes.NewReader(buf)
			count := 0
			for _, err := range varint.ReadSeq(r) {
				assert.Nil(t, err)
				count++
			}
			assert.Equal(t, 100, count)
		}()
	}

	wg.Wait()
}

func TestVarint_Adversarial_ZeroAlloc(t *testing.T) {
	var buf []byte
	for i := range 100 {
		buf = varint.Append(buf, uint64(i*13))
	}

	allocsValues := testing.AllocsPerRun(100, func() {
		for v := range varint.Values(buf) {
			if v > 1000000 {
				t.Fatal("unexpected")
			}
		}
	})
	assert.Equal(t, float64(0), allocsValues, "varint.Values must be 0 allocs/op")
}
