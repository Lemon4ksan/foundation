// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aead_test

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/lemon4ksan/foundation/crypto/aead"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func newTestAEAD(t *testing.T) (cipher.AEAD, []byte) {
	t.Helper()
	key := make([]byte, aead.KeySize)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)

	a, err := aead.New(aead.AES256GCM, key)
	require.NoError(t, err)

	nonceBase := make([]byte, a.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonceBase)
	require.NoError(t, err)

	return a, nonceBase
}

func TestStream_Chunks_Roundtrip(t *testing.T) {
	t.Parallel()

	testSizes := []int{0, 10, 1024, 4096, 70000}

	for _, size := range testSizes {
		a, nonceBase := newTestAEAD(t)
		streamAAD := []byte("stream-metadata-context")

		plaintext := make([]byte, size)
		for i := range plaintext {
			plaintext[i] = byte(i % 251)
		}

		// Write using StreamWriter
		var encBuf bytes.Buffer
		sw, err := aead.NewStreamWriter(&encBuf, a, nonceBase, 2048, streamAAD)
		require.NoError(t, err)

		_, err = sw.Write(plaintext)
		require.NoError(t, err)
		require.NoError(t, sw.Close())

		// Read using StreamReader.Chunks()
		sr, err := aead.NewStreamReader(&encBuf, a, nonceBase, streamAAD)
		require.NoError(t, err)

		var recovered []byte
		for chunk, err := range sr.Chunks() {
			require.NoError(t, err)
			recovered = append(recovered, chunk...)
		}

		assert.Equal(t, plaintext, recovered)
	}
}

func TestEncryptChunks_DecryptChunks_Roundtrip(t *testing.T) {
	t.Parallel()

	payloads := [][]byte{
		{},
		[]byte("hello streaming aead iterators"),
		bytes.Repeat([]byte("chunked-data-payload-"), 500),
	}

	for _, payload := range payloads {
		a, nonceBase := newTestAEAD(t)
		streamAAD := []byte("auth-aad-data")

		encSeq, err := aead.EncryptChunks(bytes.NewReader(payload), a, nonceBase, streamAAD, 1024)
		require.NoError(t, err)

		var wire bytes.Buffer
		for frame, err := range encSeq {
			require.NoError(t, err)
			wire.Write(frame)
		}

		decSeq, err := aead.DecryptChunks(&wire, a, nonceBase, streamAAD)
		require.NoError(t, err)

		var recovered []byte
		for chunk, err := range decSeq {
			require.NoError(t, err)
			recovered = append(recovered, chunk...)
		}

		assert.Equal(t, payload, recovered)
	}
}

func TestStreamReader_Chunks_EarlyExit(t *testing.T) {
	t.Parallel()

	a, nonceBase := newTestAEAD(t)
	streamAAD := []byte("aad")
	payload := bytes.Repeat([]byte("A"), 10000)

	encSeq, err := aead.EncryptChunks(bytes.NewReader(payload), a, nonceBase, streamAAD, 1024)
	require.NoError(t, err)

	var wire bytes.Buffer
	for frame, err := range encSeq {
		require.NoError(t, err)
		wire.Write(frame)
	}

	sr, err := aead.NewStreamReader(&wire, a, nonceBase, streamAAD)
	require.NoError(t, err)

	count := 0
	for chunk, err := range sr.Chunks() {
		require.NoError(t, err)
		assert.True(t, len(chunk) > 0)
		count++
		if count == 2 {
			break
		}
	}
	assert.Equal(t, 2, count)
}

func TestStream_Chunks_Errors(t *testing.T) {
	t.Parallel()

	t.Run("invalid arguments", func(t *testing.T) {
		a, nonceBase := newTestAEAD(t)

		_, err := aead.EncryptChunks(bytes.NewReader(nil), nil, nonceBase, nil, 1024)
		assert.Error(t, err)

		_, err = aead.EncryptChunks(bytes.NewReader(nil), a, nonceBase[:4], nil, 1024)
		assert.True(t, errors.Is(err, aead.ErrInvalidNonceLength))

		_, err = aead.DecryptChunks(bytes.NewReader(nil), nil, nonceBase, nil)
		assert.Error(t, err)

		_, err = aead.DecryptChunks(bytes.NewReader(nil), a, nonceBase[:4], nil)
		assert.True(t, errors.Is(err, aead.ErrInvalidNonceLength))
	})

	t.Run("truncated stream", func(t *testing.T) {
		a, nonceBase := newTestAEAD(t)
		encSeq, err := aead.EncryptChunks(bytes.NewReader([]byte("test payload")), a, nonceBase, nil, 1024)
		require.NoError(t, err)

		var wire []byte
		for frame, err := range encSeq {
			require.NoError(t, err)
			wire = append(wire, frame...)
		}

		// Truncate
		sr, err := aead.NewStreamReader(bytes.NewReader(wire[:len(wire)-5]), a, nonceBase, nil)
		require.NoError(t, err)

		hadError := false
		for _, err := range sr.Chunks() {
			if err != nil {
				hadError = true
				break
			}
		}
		assert.True(t, hadError)
	})

	t.Run("tampered chunk", func(t *testing.T) {
		a, nonceBase := newTestAEAD(t)
		encSeq, err := aead.EncryptChunks(bytes.NewReader([]byte("test payload")), a, nonceBase, nil, 1024)
		require.NoError(t, err)

		var wire []byte
		for frame, err := range encSeq {
			require.NoError(t, err)
			wire = append(wire, frame...)
		}

		// Tamper payload byte
		wire[len(wire)-1] ^= 0x55

		decSeq, err := aead.DecryptChunks(bytes.NewReader(wire), a, nonceBase, nil)
		require.NoError(t, err)

		hadError := false
		for _, err := range decSeq {
			if err != nil {
				hadError = true
				break
			}
		}
		assert.True(t, hadError)
	})
}

type mockShortNonceAEAD struct{}

func (m mockShortNonceAEAD) NonceSize() int { return 4 }
func (m mockShortNonceAEAD) Overhead() int  { return 16 }
func (m mockShortNonceAEAD) Seal(dst, nonce, plaintext, additionalData []byte) []byte {
	return append(dst, plaintext...)
}
func (m mockShortNonceAEAD) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	return append(dst, ciphertext...), nil
}

func TestRepro_StreamAEAD_ShortNoncePanic(t *testing.T) {
	mockCipher := mockShortNonceAEAD{}
	nonce4 := []byte{1, 2, 3, 4}

	// Must return ErrInvalidNonceLength instead of accepting short nonces and panicking
	_, err := aead.NewStreamWriter(io.Discard, mockCipher, nonce4, 1024, nil)
	if err == nil {
		t.Fatal("expected error for nonce size < 8")
	}
	if !errors.Is(err, aead.ErrInvalidNonceLength) {
		t.Fatalf("expected ErrInvalidNonceLength, got %v", err)
	}

	_, err = aead.NewStreamReader(bytes.NewReader(nil), mockCipher, nonce4, nil)
	if err == nil {
		t.Fatal("expected error for nonce size < 8")
	}
	if !errors.Is(err, aead.ErrInvalidNonceLength) {
		t.Fatalf("expected ErrInvalidNonceLength, got %v", err)
	}
}

func TestRepro_StreamAEAD_CorruptedHeaderFlag(t *testing.T) {
	c, nonce := newTestAEAD(t)
	// Frame with invalid flag 0x02
	var badFrame [5 + 16]byte
	binary.BigEndian.PutUint32(badFrame[0:4], 16)
	badFrame[4] = 0x02 // Corrupted header flag!

	sr, err := aead.NewStreamReader(bytes.NewReader(badFrame[:]), c, nonce, nil)
	require.NoError(t, err)

	buf := make([]byte, 32)
	_, err = sr.Read(buf)
	if err == nil {
		t.Fatal("expected error on corrupted header flag")
	}
	if !errors.Is(err, aead.ErrInvalidHeaderFlag) {
		t.Fatalf("expected ErrInvalidHeaderFlag, got %v", err)
	}
}

func TestRepro_StreamAEAD_ChunkTooSmall(t *testing.T) {
	c, nonce := newTestAEAD(t)
	var badFrame [5]byte
	binary.BigEndian.PutUint32(badFrame[0:4], uint32(c.Overhead()-1)) // smaller than overhead
	badFrame[4] = 0x01

	sr, err := aead.NewStreamReader(bytes.NewReader(badFrame[:]), c, nonce, nil)
	require.NoError(t, err)

	buf := make([]byte, 32)
	_, err = sr.Read(buf)
	if err == nil {
		t.Fatal("expected error on chunk smaller than overhead")
	}
	if !errors.Is(err, aead.ErrChunkTooSmall) {
		t.Fatalf("expected ErrChunkTooSmall, got %v", err)
	}
}

type oneByteWriter struct {
	buf bytes.Buffer
}

func (w *oneByteWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	w.buf.WriteByte(p[0])
	return 1, nil
}

func TestRepro_StreamAEAD_PartialWriter(t *testing.T) {
	c, nonce := newTestAEAD(t)
	pw := &oneByteWriter{}

	payload := []byte("stream encryption through a one-byte-at-a-time partial writer")
	sw, err := aead.NewStreamWriter(pw, c, nonce, 1024, []byte("aad"))
	require.NoError(t, err)

	n, err := sw.Write(payload)
	require.NoError(t, err)
	assert.Equal(t, len(payload), n)
	require.NoError(t, sw.Close())

	// Verify plaintext successfully decrypts
	sr, err := aead.NewStreamReader(bytes.NewReader(pw.buf.Bytes()), c, nonce, []byte("aad"))
	require.NoError(t, err)

	dec, err := io.ReadAll(sr)
	require.NoError(t, err)
	assert.Equal(t, payload, dec)
}



