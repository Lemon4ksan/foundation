// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aead_test

import (
	"bytes"
	"crypto/cipher"
	"crypto/rand"
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
