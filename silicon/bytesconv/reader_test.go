package bytesconv

import (
	"strings"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestRepro_StreamReader_ReadSlice_Hang(t *testing.T) {
	// A stream with 5000 'a' bytes without newline, buffer size 4096.
	// Must return ErrBufferFull without hanging in an infinite loop.
	done := make(chan struct{})
	var err error
	go func() {
		r := NewStreamReader(strings.NewReader(strings.Repeat("a", 5000)), 4096)
		_, err = r.ReadSlice('\n')
		close(done)
	}()

	select {
	case <-done:
		assert.ErrorIs(t, err, ErrBufferFull)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("StreamReader.ReadSlice hung indefinitely on line exceeding buffer size")
	}
}

func TestRepro_StreamReader_Peek_Hang(t *testing.T) {
	// Peeking more bytes than buffer capacity (5000 > 4096) on a stream with data
	// Must return ErrBufferFull without hanging in an infinite loop.
	done := make(chan struct{})
	var err error
	go func() {
		r := NewStreamReader(strings.NewReader(strings.Repeat("a", 5000)), 4096)
		_, err = r.Peek(5000)
		close(done)
	}()

	select {
	case <-done:
		assert.ErrorIs(t, err, ErrBufferFull)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("StreamReader.Peek hung indefinitely on peek count exceeding buffer size")
	}
}

func TestRepro_StreamReader_Discard_UnderlyingData(t *testing.T) {
	// Stream has 10000 bytes. Buffer is 4096.
	// Discarding 5000 bytes should discard 5000 bytes, not return early EOF.
	r := NewStreamReader(strings.NewReader(strings.Repeat("a", 10000)), 4096)
	_, err := r.Peek(10)
	assert.NoError(t, err)

	n, err := r.Discard(5000)
	assert.NoError(t, err)
	assert.Equal(t, 5000, n)
}
