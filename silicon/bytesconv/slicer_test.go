// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestPatternSlicer_Slice(t *testing.T) {
	t.Parallel()

	slicer := bytesconv.NewPatternSlicer([]byte("KEY:"), 4)
	assert.NotNil(t, slicer)

	t.Run("pattern_found", func(t *testing.T) {
		t.Parallel()

		data := []byte("PREFIX_SECTION\r\nKEY: TARGET_PAYLOAD\r\n\r\n")
		chunks, matched := slicer.Slice(data)

		assert.True(t, matched)

		requireLen := 2
		assert.Len(t, chunks, requireLen)
		assert.Equal(t, "PREFIX_SECTION\r\nKEY:", string(chunks[0]))
		assert.Equal(t, " TARGET_PAYLOAD\r\n\r\n", string(chunks[1]))
	})

	t.Run("pattern_not_found", func(t *testing.T) {
		t.Parallel()

		data := []byte("PREFIX_SECTION\r\nOTHER_FIELD: test\r\n\r\n")
		chunks, matched := slicer.Slice(data)

		assert.False(t, matched)
		assert.Len(t, chunks, 1)
		assert.Equal(t, string(data), string(chunks[0]))
	})

	t.Run("slice_into", func(t *testing.T) {
		t.Parallel()

		dst := make([][]byte, 0, 4)
		data := []byte("PREFIX_SECTION\r\nKEY: TARGET_PAYLOAD\r\n\r\n")
		chunks, matched := slicer.SliceInto(data, dst)

		assert.True(t, matched)
		assert.Len(t, chunks, 2)
		assert.Equal(t, "PREFIX_SECTION\r\nKEY:", string(chunks[0]))
	})

	t.Run("slice_all_into", func(t *testing.T) {
		t.Parallel()

		splitComma := bytesconv.NewPatternSlicer([]byte(","), 1)
		data := []byte("a,b,c,d")
		all := splitComma.SliceAll(data)
		assert.Equal(
			t,
			[]string{"a,", "b,", "c,", "d"},
			[]string{string(all[0]), string(all[1]), string(all[2]), string(all[3])},
		)

		dst := make([][]byte, 0, 8)
		allInto := splitComma.SliceAllInto(data, dst)
		assert.Len(t, allInto, 4)
	})

	t.Run("boundary_split_at_zero", func(t *testing.T) {
		t.Parallel()

		slicerZero := bytesconv.NewPatternSlicer([]byte("KEY:"), 0)
		data := []byte("KEY:value")

		chunks, matched := slicerZero.Slice(data)
		assert.True(t, matched)
		assert.Len(t, chunks, 2)
		assert.Empty(t, chunks[0])
		assert.Equal(t, "KEY:value", string(chunks[1]))

		dst := make([][]byte, 0, 4)
		chunksInto, matchedInto := slicerZero.SliceInto(data, dst)
		assert.True(t, matchedInto)
		assert.Len(t, chunksInto, 2)
		assert.Empty(t, chunksInto[0])
		assert.Equal(t, "KEY:value", string(chunksInto[1]))
	})

	t.Run("boundary_split_at_len", func(t *testing.T) {
		t.Parallel()

		slicerEnd := bytesconv.NewPatternSlicer([]byte("KEY:"), 4)
		data := []byte("valueKEY:")

		chunks, matched := slicerEnd.Slice(data)
		assert.True(t, matched)
		assert.Len(t, chunks, 2)
		assert.Equal(t, "valueKEY:", string(chunks[0]))
		assert.Empty(t, chunks[1])

		dst := make([][]byte, 0, 4)
		chunksInto, matchedInto := slicerEnd.SliceInto(data, dst)
		assert.True(t, matchedInto)
		assert.Len(t, chunksInto, 2)
		assert.Equal(t, "valueKEY:", string(chunksInto[0]))
		assert.Empty(t, chunksInto[1])
	})

	t.Run("out_of_bounds_offsets", func(t *testing.T) {
		t.Parallel()

		data := []byte("KEY:value")

		slicerNeg := bytesconv.NewPatternSlicer([]byte("KEY:"), -1)
		chunksNeg, matchedNeg := slicerNeg.Slice(data)
		assert.False(t, matchedNeg)
		assert.Len(t, chunksNeg, 1)

		dst := make([][]byte, 0, 4)
		chunksIntoNeg, matchedIntoNeg := slicerNeg.SliceInto(data, dst)
		assert.False(t, matchedIntoNeg)
		assert.Len(t, chunksIntoNeg, 1)

		slicerOver := bytesconv.NewPatternSlicer([]byte("KEY:"), 100)
		chunksOver, matchedOver := slicerOver.Slice(data)
		assert.False(t, matchedOver)
		assert.Len(t, chunksOver, 1)

		chunksIntoOver, matchedIntoOver := slicerOver.SliceInto(data, dst)
		assert.False(t, matchedIntoOver)
		assert.Len(t, chunksIntoOver, 1)
	})
}
