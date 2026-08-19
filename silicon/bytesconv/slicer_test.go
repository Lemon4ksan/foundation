// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
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
		assert.Equal(t, []string{"a,", "b,", "c,", "d"}, []string{string(all[0]), string(all[1]), string(all[2]), string(all[3])})

		dst := make([][]byte, 0, 8)
		allInto := splitComma.SliceAllInto(data, dst)
		assert.Len(t, allInto, 4)
	})
}
