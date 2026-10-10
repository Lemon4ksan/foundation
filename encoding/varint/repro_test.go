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
