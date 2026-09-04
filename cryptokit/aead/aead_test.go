// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aead_test

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"testing"

	"github.com/lemon4ksan/foundation/cryptokit/aead"
)

func TestAEAD_Algorithms(t *testing.T) {
	algorithms := []aead.Algorithm{
		aead.AES256GCM,
		aead.ChaCha20Poly1305,
		aead.XChaCha20Poly1305,
	}

	for _, algo := range algorithms {
		t.Run(algo.String(), func(t *testing.T) {
			key := make([]byte, aead.KeySize)
			if _, err := io.ReadFull(rand.Reader, key); err != nil {
				t.Fatal(err)
			}

			nonceSize, err := algo.NonceSize()
			if err != nil {
				t.Fatalf("NonceSize failed: %v", err)
			}

			nonce := make([]byte, nonceSize)
			if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
				t.Fatal(err)
			}

			plaintext := []byte("foundation-aead-payload-verification-2026")
			aad := []byte("associated-metadata-header")

			// Seal
			ciphertext, err := aead.Seal(algo, key, nonce, plaintext, aad)
			if err != nil {
				t.Fatalf("Seal failed: %v", err)
			}

			// Open
			decrypted, err := aead.Open(algo, key, nonce, ciphertext, aad)
			if err != nil {
				t.Fatalf("Open failed: %v", err)
			}
			if !bytes.Equal(decrypted, plaintext) {
				t.Fatalf("plaintext mismatch: %s != %s", decrypted, plaintext)
			}

			// Tamper AAD
			_, err = aead.Open(algo, key, nonce, ciphertext, []byte("tampered-aad"))
			if err == nil {
				t.Fatal("expected Open to fail on tampered AAD")
			}

			// Tamper ciphertext
			corruptCT := append([]byte(nil), ciphertext...)
			corruptCT[len(corruptCT)-1] ^= 0xFF
			_, err = aead.Open(algo, key, nonce, corruptCT, aad)
			if err == nil {
				t.Fatal("expected Open to fail on tampered ciphertext")
			}

			// Invalid key length
			_, err = aead.New(algo, key[:16])
			if !errors.Is(err, aead.ErrInvalidKeyLength) {
				t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
			}
		})
	}
}

func TestStreamAEAD_Roundtrip(t *testing.T) {
	algorithms := []aead.Algorithm{
		aead.AES256GCM,
		aead.ChaCha20Poly1305,
		aead.XChaCha20Poly1305,
	}

	payloadSizes := []int{
		0,      // Empty stream
		42,     // Small stream (< 1 chunk)
		1024,   // Exactly 1 chunk
		2048,   // Exactly 2 chunks
		10000,  // Multiple chunks with remainder
		100000, // Large payload
	}

	for _, algo := range algorithms {
		t.Run(algo.String(), func(t *testing.T) {
			key := make([]byte, aead.KeySize)
			_, _ = io.ReadFull(rand.Reader, key)

			cipherInst, err := aead.New(algo, key)
			if err != nil {
				t.Fatal(err)
			}

			nonceBase := make([]byte, cipherInst.NonceSize())
			_, _ = io.ReadFull(rand.Reader, nonceBase)

			streamAAD := []byte("stream-channel-binding-test")
			chunkSize := 1024 // 1 KiB chunks for testing

			for _, size := range payloadSizes {
				data := make([]byte, size)
				_, _ = io.ReadFull(rand.Reader, data)

				// Write stream
				var buf bytes.Buffer
				writer, err := aead.NewStreamWriter(&buf, cipherInst, nonceBase, chunkSize, streamAAD)
				if err != nil {
					t.Fatalf("NewStreamWriter failed: %v", err)
				}

				// Write in irregular chunk pieces to test internal buffering
				pieceSize := 333
				for offset := 0; offset < len(data); offset += pieceSize {
					end := offset + pieceSize
					if end > len(data) {
						end = len(data)
					}
					if _, err := writer.Write(data[offset:end]); err != nil {
						t.Fatalf("writer.Write failed: %v", err)
					}
				}

				if err := writer.Close(); err != nil {
					t.Fatalf("writer.Close failed: %v", err)
				}

				// Read stream
				reader, err := aead.NewStreamReader(&buf, cipherInst, nonceBase, streamAAD)
				if err != nil {
					t.Fatalf("NewStreamReader failed: %v", err)
				}

				recovered, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("ReadAll failed for size %d: %v", size, err)
				}

				if !bytes.Equal(recovered, data) {
					t.Fatalf("data mismatch for size %d (got %d bytes, expected %d)", size, len(recovered), len(data))
				}
			}
		})
	}
}

func TestStreamAEAD_TruncationDetection(t *testing.T) {
	key := make([]byte, aead.KeySize)
	_, _ = io.ReadFull(rand.Reader, key)

	cipherInst, _ := aead.New(aead.AES256GCM, key)
	nonceBase := make([]byte, cipherInst.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonceBase)

	data := make([]byte, 5000) // 5 chunks at 1024 chunkSize
	_, _ = io.ReadFull(rand.Reader, data)

	var buf bytes.Buffer
	writer, err := aead.NewStreamWriter(&buf, cipherInst, nonceBase, 1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	streamBytes := buf.Bytes()

	// Truncate last 50 bytes (cuts into final chunk)
	truncated1 := streamBytes[:len(streamBytes)-50]
	reader1, _ := aead.NewStreamReader(bytes.NewReader(truncated1), cipherInst, nonceBase, nil)
	_, err = io.ReadAll(reader1)
	if !errors.Is(err, aead.ErrStreamTruncated) {
		t.Fatalf("expected ErrStreamTruncated, got %v", err)
	}

	// Truncate entire final chunk
	truncated2 := streamBytes[:len(streamBytes)/2]
	reader2, _ := aead.NewStreamReader(bytes.NewReader(truncated2), cipherInst, nonceBase, nil)
	_, err = io.ReadAll(reader2)
	if !errors.Is(err, aead.ErrStreamTruncated) {
		t.Fatalf("expected ErrStreamTruncated, got %v", err)
	}
}

func TestStreamAEAD_TamperDetection(t *testing.T) {
	key := make([]byte, aead.KeySize)
	_, _ = io.ReadFull(rand.Reader, key)

	cipherInst, _ := aead.New(aead.ChaCha20Poly1305, key)
	nonceBase := make([]byte, cipherInst.NonceSize())
	_, _ = io.ReadFull(rand.Reader, nonceBase)

	data := []byte("important payload data to be verified")
	var buf bytes.Buffer
	writer, _ := aead.NewStreamWriter(&buf, cipherInst, nonceBase, 1024, []byte("valid-aad"))
	_, _ = writer.Write(data)
	_ = writer.Close()

	// Wrong AAD on read
	readerWrongAAD, _ := aead.NewStreamReader(bytes.NewReader(buf.Bytes()), cipherInst, nonceBase, []byte("wrong-aad"))
	_, err := io.ReadAll(readerWrongAAD)
	if err == nil {
		t.Fatal("expected read failure on wrong AAD")
	}

	// Corrupt ciphertext byte
	corruptBytes := append([]byte(nil), buf.Bytes()...)
	corruptBytes[len(corruptBytes)-1] ^= 0x55
	readerCorrupt, _ := aead.NewStreamReader(bytes.NewReader(corruptBytes), cipherInst, nonceBase, []byte("valid-aad"))
	_, err = io.ReadAll(readerCorrupt)
	if err == nil {
		t.Fatal("expected read failure on corrupt ciphertext")
	}
}
