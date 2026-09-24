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
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/crypto/aead"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func newTestCipher(t *testing.T) cipher.AEAD {
	t.Helper()
	key := make([]byte, aead.KeySize)
	_, err := io.ReadFull(rand.Reader, key)
	require.NoError(t, err)

	c, err := aead.New(aead.AES256GCM, key)
	require.NoError(t, err)
	return c
}

func TestStream_Adversarial_EarlyExit(t *testing.T) {
	t.Parallel()

	c := newTestCipher(t)
	nonce := make([]byte, c.NonceSize())
	aad := []byte("stream-aad")

	// Create 5 chunks of 1024 bytes
	chunkSize := 1024
	totalChunks := 5
	plaintext := bytes.Repeat([]byte("A"), chunkSize*totalChunks)

	encSeq, err := aead.EncryptChunks(bytes.NewReader(plaintext), c, nonce, aad, chunkSize)
	require.NoError(t, err)

	var streamBuf bytes.Buffer
	for frame, encErr := range encSeq {
		require.NoError(t, encErr)
		streamBuf.Write(frame)
	}

	t.Run("EncryptChunks early exit after 0 (first yield returns false)", func(t *testing.T) {
		seq, err := aead.EncryptChunks(bytes.NewReader(plaintext), c, nonce, aad, chunkSize)
		require.NoError(t, err)

		count := 0
		seq(func(frame []byte, err error) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})

	t.Run("EncryptChunks early exit mid-sequence", func(t *testing.T) {
		seq, err := aead.EncryptChunks(bytes.NewReader(plaintext), c, nonce, aad, chunkSize)
		require.NoError(t, err)

		count := 0
		for _, err := range seq {
			require.NoError(t, err)
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
	})

	t.Run("DecryptChunks early exit after 0 (first yield returns false)", func(t *testing.T) {
		decSeq, err := aead.DecryptChunks(bytes.NewReader(streamBuf.Bytes()), c, nonce, aad)
		require.NoError(t, err)

		count := 0
		decSeq(func(pt []byte, err error) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})

	t.Run("DecryptChunks early exit mid-sequence", func(t *testing.T) {
		decSeq, err := aead.DecryptChunks(bytes.NewReader(streamBuf.Bytes()), c, nonce, aad)
		require.NoError(t, err)

		count := 0
		for _, err := range decSeq {
			require.NoError(t, err)
			count++
			if count == 2 {
				break
			}
		}
		assert.Equal(t, 2, count)
	})
}

func TestStream_Adversarial_MalformedFraming(t *testing.T) {
	t.Parallel()

	c := newTestCipher(t)
	nonce := make([]byte, c.NonceSize())
	aad := []byte("stream-aad")

	t.Run("empty reader yields ErrStreamTruncated", func(t *testing.T) {
		decSeq, err := aead.DecryptChunks(bytes.NewReader(nil), c, nonce, aad)
		require.NoError(t, err)

		var errs []error
		for _, err := range decSeq {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.True(t, errors.Is(errs[0], aead.ErrStreamTruncated))
	})

	t.Run("truncated chunk header 1 to 4 bytes", func(t *testing.T) {
		for hLen := 1; hLen <= 4; hLen++ {
			headerFragment := make([]byte, hLen)
			decSeq, err := aead.DecryptChunks(bytes.NewReader(headerFragment), c, nonce, aad)
			require.NoError(t, err)

			var errs []error
			for _, err := range decSeq {
				if err != nil {
					errs = append(errs, err)
				}
			}
			require.Equal(t, 1, len(errs))
			assert.True(t, errors.Is(errs[0], aead.ErrStreamTruncated))
		}
	})

	t.Run("chunk size smaller than tag overhead", func(t *testing.T) {
		var badHeader [5]byte
		binary.BigEndian.PutUint32(badHeader[0:4], uint32(c.Overhead()-1))
		badHeader[4] = 0x01

		decSeq, err := aead.DecryptChunks(bytes.NewReader(badHeader[:]), c, nonce, aad)
		require.NoError(t, err)

		var errs []error
		for _, err := range decSeq {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.Contains(t, errs[0].Error(), "smaller than tag overhead")
	})

	t.Run("chunk size exceeding MaxChunkSize", func(t *testing.T) {
		var badHeader [5]byte
		binary.BigEndian.PutUint32(badHeader[0:4], uint32(aead.MaxChunkSize+100))
		badHeader[4] = 0x01

		decSeq, err := aead.DecryptChunks(bytes.NewReader(badHeader[:]), c, nonce, aad)
		require.NoError(t, err)

		var errs []error
		for _, err := range decSeq {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.True(t, errors.Is(errs[0], aead.ErrChunkTooLarge))
	})

	t.Run("truncated ciphertext bytes", func(t *testing.T) {
		var header [5]byte
		binary.BigEndian.PutUint32(header[0:4], 64)
		header[4] = 0x01
		incomplete := append(header[:], make([]byte, 20)...) // 20 < 64

		decSeq, err := aead.DecryptChunks(bytes.NewReader(incomplete), c, nonce, aad)
		require.NoError(t, err)

		var errs []error
		for _, err := range decSeq {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.True(t, errors.Is(errs[0], aead.ErrStreamTruncated))
	})

	t.Run("missing terminal flag chunk", func(t *testing.T) {
		// Encrypt 1 chunk with intermediate flag (0x00)
		plaintext := []byte("intermediate payload")
		var out bytes.Buffer
		sw, err := aead.NewStreamWriter(&out, c, nonce, 1024, aad)
		require.NoError(t, err)

		_, err = sw.Write(plaintext)
		require.NoError(t, err)
		// We deliberately DO NOT call sw.Close(), so no terminal chunk exists!
		// However sw buffers until > chunkSize, so let's write 2000 bytes to force intermediate chunk flush
		bigPt := bytes.Repeat([]byte("B"), 2000)
		_, err = sw.Write(bigPt)
		require.NoError(t, err)
		// Now out has at least one intermediate chunk (flag 0x00), but never was closed

		decSeq, err := aead.DecryptChunks(bytes.NewReader(out.Bytes()), c, nonce, aad)
		require.NoError(t, err)

		var yieldedChunks int
		var streamErr error
		for pt, err := range decSeq {
			if err != nil {
				streamErr = err
			} else {
				yieldedChunks++
				assert.True(t, len(pt) > 0)
			}
		}
		assert.True(t, yieldedChunks >= 1)
		assert.True(
			t,
			errors.Is(streamErr, aead.ErrStreamTruncated),
			"unclosed stream must terminate with ErrStreamTruncated",
		)
	})
}

func TestStream_Adversarial_TamperDetection(t *testing.T) {
	t.Parallel()

	c := newTestCipher(t)
	nonce := make([]byte, c.NonceSize())
	aad := []byte("integrity-test-aad")

	plaintext := []byte("Top secret stream data to be authenticated")
	var streamBuf bytes.Buffer
	sw, err := aead.NewStreamWriter(&streamBuf, c, nonce, 1024, aad)
	require.NoError(t, err)
	_, err = sw.Write(plaintext)
	require.NoError(t, err)
	err = sw.Close()
	require.NoError(t, err)

	origBytes := streamBuf.Bytes()

	t.Run("tampered ciphertext byte", func(t *testing.T) {
		tampered := append([]byte(nil), origBytes...)
		tampered[10] ^= 0x42 // flip bit inside ciphertext

		decSeq, err := aead.DecryptChunks(bytes.NewReader(tampered), c, nonce, aad)
		require.NoError(t, err)

		var streamErr error
		for _, err := range decSeq {
			if err != nil {
				streamErr = err
			}
		}
		require.NotNil(t, streamErr)
		assert.Contains(t, streamErr.Error(), "authentication failed")
	})

	t.Run("tampered flag in header", func(t *testing.T) {
		tampered := append([]byte(nil), origBytes...)
		tampered[4] ^= 0x01 // flip flag byte

		decSeq, err := aead.DecryptChunks(bytes.NewReader(tampered), c, nonce, aad)
		require.NoError(t, err)

		var streamErr error
		for _, err := range decSeq {
			if err != nil {
				streamErr = err
			}
		}
		require.NotNil(t, streamErr)
		assert.Contains(t, streamErr.Error(), "authentication failed")
	})

	t.Run("mismatched AAD", func(t *testing.T) {
		wrongAAD := []byte("wrong-aad-metadata")
		decSeq, err := aead.DecryptChunks(bytes.NewReader(origBytes), c, nonce, wrongAAD)
		require.NoError(t, err)

		var streamErr error
		for _, err := range decSeq {
			if err != nil {
				streamErr = err
			}
		}
		require.NotNil(t, streamErr)
		assert.Contains(t, streamErr.Error(), "authentication failed")
	})

	t.Run("mismatched base nonce", func(t *testing.T) {
		wrongNonce := append([]byte(nil), nonce...)
		wrongNonce[0] ^= 0xff

		decSeq, err := aead.DecryptChunks(bytes.NewReader(origBytes), c, wrongNonce, aad)
		require.NoError(t, err)

		var streamErr error
		for _, err := range decSeq {
			if err != nil {
				streamErr = err
			}
		}
		require.NotNil(t, streamErr)
		assert.Contains(t, streamErr.Error(), "authentication failed")
	})
}

func TestStream_Adversarial_ExtremeInputsAndLimits(t *testing.T) {
	t.Parallel()

	c := newTestCipher(t)
	nonce := make([]byte, c.NonceSize())
	aad := []byte("extreme-aad")

	t.Run("zero byte plaintext roundtrip", func(t *testing.T) {
		encSeq, err := aead.EncryptChunks(bytes.NewReader([]byte{}), c, nonce, aad, 1024)
		require.NoError(t, err)

		var encBuf bytes.Buffer
		for frame, err := range encSeq {
			require.NoError(t, err)
			encBuf.Write(frame)
		}
		assert.True(t, encBuf.Len() > 0, "must emit terminal chunk even for 0-byte stream")

		decSeq, err := aead.DecryptChunks(bytes.NewReader(encBuf.Bytes()), c, nonce, aad)
		require.NoError(t, err)

		var totalPlaintext []byte
		for pt, err := range decSeq {
			require.NoError(t, err)
			totalPlaintext = append(totalPlaintext, pt...)
		}
		assert.Equal(t, 0, len(totalPlaintext))
	})

	t.Run("multi-megabyte 2MB stream with 64KB chunks", func(t *testing.T) {
		size := 2 * 1024 * 1024 // 2MB
		chunkSize := 64 * 1024
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 251)
		}

		encSeq, err := aead.EncryptChunks(bytes.NewReader(data), c, nonce, aad, chunkSize)
		require.NoError(t, err)

		var encBuf bytes.Buffer
		for frame, err := range encSeq {
			require.NoError(t, err)
			encBuf.Write(frame)
		}

		decSeq, err := aead.DecryptChunks(bytes.NewReader(encBuf.Bytes()), c, nonce, aad)
		require.NoError(t, err)

		var recovered bytes.Buffer
		for pt, err := range decSeq {
			require.NoError(t, err)
			recovered.Write(pt)
		}
		assert.Equal(t, size, recovered.Len())
		assert.Equal(t, data, recovered.Bytes())
	})

	t.Run("chunk size clamping", func(t *testing.T) {
		// chunkSize <= 0 clamped to DefaultChunkSize
		seq1, err := aead.EncryptChunks(bytes.NewReader([]byte("hello")), c, nonce, aad, -10)
		require.NoError(t, err)
		assert.NotNil(t, seq1)

		// chunkSize < MinChunkSize clamped to MinChunkSize
		seq2, err := aead.EncryptChunks(bytes.NewReader([]byte("hello")), c, nonce, aad, 100)
		require.NoError(t, err)
		assert.NotNil(t, seq2)

		// chunkSize > MaxChunkSize clamped to MaxChunkSize
		seq3, err := aead.EncryptChunks(bytes.NewReader([]byte("hello")), c, nonce, aad, 32*1024*1024)
		require.NoError(t, err)
		assert.NotNil(t, seq3)
	})

	t.Run("invalid cipher or nonce", func(t *testing.T) {
		_, err := aead.EncryptChunks(bytes.NewReader(nil), nil, nonce, aad, 1024)
		assert.NotNil(t, err)

		_, err = aead.EncryptChunks(bytes.NewReader(nil), c, []byte("short"), aad, 1024)
		assert.NotNil(t, err)

		_, err = aead.DecryptChunks(bytes.NewReader(nil), nil, nonce, aad)
		assert.NotNil(t, err)

		_, err = aead.DecryptChunks(bytes.NewReader(nil), c, []byte("short"), aad)
		assert.NotNil(t, err)
	})
}

func TestStream_Adversarial_Concurrency(t *testing.T) {
	t.Parallel()

	c := newTestCipher(t)
	aad := []byte("concurrency-aad")

	var wg sync.WaitGroup
	workers := 25

	for w := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			workerNonce := make([]byte, c.NonceSize())
			binary.BigEndian.PutUint64(workerNonce[0:8], uint64(workerID))

			payload := []byte(fmt.Sprintf("worker payload for id %d repeated across multiple chunks", workerID))
			fullData := bytes.Repeat(payload, 50) // ~3KB

			encSeq, err := aead.EncryptChunks(bytes.NewReader(fullData), c, workerNonce, aad, 1024)
			if err != nil {
				t.Errorf("worker %d EncryptChunks failed: %v", workerID, err)
				return
			}

			var streamBuf bytes.Buffer
			for frame, err := range encSeq {
				if err != nil {
					t.Errorf("worker %d frame error: %v", workerID, err)
					return
				}
				streamBuf.Write(frame)
			}

			decSeq, err := aead.DecryptChunks(bytes.NewReader(streamBuf.Bytes()), c, workerNonce, aad)
			if err != nil {
				t.Errorf("worker %d DecryptChunks failed: %v", workerID, err)
				return
			}

			var recovered bytes.Buffer
			for pt, err := range decSeq {
				if err != nil {
					t.Errorf("worker %d decrypt error: %v", workerID, err)
					return
				}
				recovered.Write(pt)
			}

			if !bytes.Equal(recovered.Bytes(), fullData) {
				t.Errorf("worker %d data mismatch", workerID)
			}
		}(w)
	}

	wg.Wait()
}
