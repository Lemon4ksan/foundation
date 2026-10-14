// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package varint_test

import (
	"io"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/varint"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type shortPeeker struct {
	data []byte
}

func (s *shortPeeker) Peek(b []byte) (int, error) {
	// Returns fewer bytes than requested without error
	if len(s.data) == 0 {
		return 0, nil
	}
	n := copy(b, s.data[:1])
	return n, nil
}

func TestRepro_Peek_ShortReadOrNil(t *testing.T) {
	t.Run("nil peeker", func(t *testing.T) {
		_, err := varint.Peek(nil)
		require.Error(t, err)
	})

	t.Run("short read peeker without error returns ErrUnexpectedEOF", func(t *testing.T) {
		// Encodes 4-byte varint, but shortPeeker returns only 1 byte without error
		p := &shortPeeker{data: []byte{0x80, 0x01, 0x02, 0x03}}
		_, err := varint.Peek(p)
		assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	})
}

func TestRepro_Read_NilReader(t *testing.T) {
	t.Run("nil ByteReader in Read returns error instead of panic", func(t *testing.T) {
		_, err := varint.Read(nil)
		require.Error(t, err)
	})

	t.Run("nil ByteReader in ReadSeq terminates cleanly", func(t *testing.T) {
		for _, err := range varint.ReadSeq(nil) {
			require.Error(t, err)
			break
		}
	})

	t.Run("nil reader in byteReader ReadByte", func(t *testing.T) {
		br := varint.NewReader(nil)
		require.NotNil(t, br)
		_, err := br.ReadByte()
		require.Error(t, err)
	})

	t.Run("nil writer in byteWriter WriteByte", func(t *testing.T) {
		bw := varint.NewWriter(nil)
		require.NotNil(t, bw)
		err := bw.WriteByte(0)
		require.Error(t, err)
	})
}
