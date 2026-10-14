// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bin_test

import (
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/bin"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestRepro_Reader_IntegerOverflow(t *testing.T) {
	t.Run("Bytes math.MaxInt", func(t *testing.T) {
		r := bin.NewReader([]byte{1, 2, 3})
		_ = r.U8() // pos is now 1
		b := r.Bytes(math.MaxInt)
		assert.Nil(t, b)
		assert.Equal(t, bin.ErrBufferTooShort, r.Err())
	})

	t.Run("Skip math.MaxInt then U8", func(t *testing.T) {
		r := bin.NewReader([]byte{1, 2, 3})
		_ = r.U8() // pos is now 1
		r.Skip(math.MaxInt)
		assert.Equal(t, bin.ErrBufferTooShort, r.Err())
		u := r.U8()
		assert.Equal(t, uint8(0), u)
		assert.Equal(t, bin.ErrBufferTooShort, r.Err())
	})
}

func TestRepro_NilPointerPanics(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}

	t.Run("UnmarshalLE nil interface", func(t *testing.T) {
		err := bin.UnmarshalLE(data, nil)
		require.Error(t, err)
	})

	t.Run("UnmarshalLE typed nil pointer", func(t *testing.T) {
		var h *TestHeader
		err := bin.UnmarshalLE(data, h)
		require.Error(t, err)
	})

	t.Run("MarshalLE nil interface", func(t *testing.T) {
		_, err := bin.MarshalLE(nil, nil)
		require.Error(t, err)
	})

	t.Run("MarshalLE typed nil pointer", func(t *testing.T) {
		var h *TestHeader
		_, err := bin.MarshalLE(h, nil)
		require.Error(t, err)
	})

	t.Run("UnmarshalBE nil interface", func(t *testing.T) {
		err := bin.UnmarshalBE(data, nil)
		require.Error(t, err)
	})

	t.Run("UnmarshalBE typed nil pointer", func(t *testing.T) {
		var h *TestHeader
		err := bin.UnmarshalBE(data, h)
		require.Error(t, err)
	})

	t.Run("MarshalBE nil interface", func(t *testing.T) {
		_, err := bin.MarshalBE(nil, nil)
		require.Error(t, err)
	})

	t.Run("MarshalBE typed nil pointer", func(t *testing.T) {
		var h *TestHeader
		_, err := bin.MarshalBE(h, nil)
		require.Error(t, err)
	})
}
