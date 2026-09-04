// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package aead

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	// DefaultChunkSize is the default plaintext chunk size (64 KiB) for streaming AEAD.
	DefaultChunkSize = 64 * 1024

	// MinChunkSize is the minimum allowable chunk size (1 KiB).
	MinChunkSize = 1024

	// MaxChunkSize is the upper bound on chunk size (16 MiB) to guard against resource exhaustion.
	MaxChunkSize = 16 * 1024 * 1024

	// flagLastChunk indicates this chunk is the final chunk in the stream.
	flagLastChunk uint8 = 0x01
)

var (
	// ErrStreamTruncated indicates that a stream ended abruptly without the authenticated terminal chunk.
	ErrStreamTruncated = errors.New("aead: stream truncated or missing EOF chunk")

	// ErrChunkTooLarge indicates a chunk payload length exceeds MaxChunkSize.
	ErrChunkTooLarge = errors.New("aead: chunk size exceeds maximum allowed")
)

func deriveChunkNonce(baseNonce []byte, chunkIdx uint64) []byte {
	nonce := make([]byte, len(baseNonce))
	copy(nonce, baseNonce)

	// XOR the last 8 bytes with the big-endian chunk index.
	var ctr [8]byte
	binary.BigEndian.PutUint64(ctr[:], chunkIdx)

	offset := len(nonce) - 8
	for i := 0; i < 8; i++ {
		nonce[offset+i] ^= ctr[i]
	}
	return nonce
}

func deriveChunkAAD(streamAAD []byte, chunkIdx uint64, flag uint8) []byte {
	aad := make([]byte, len(streamAAD)+9)
	copy(aad, streamAAD)
	binary.BigEndian.PutUint64(aad[len(streamAAD):len(streamAAD)+8], chunkIdx)
	aad[len(streamAAD)+8] = flag
	return aad
}

// StreamWriter wraps an io.Writer and provides streaming AEAD authenticated chunk encryption.
type StreamWriter struct {
	w         io.Writer
	aead      cipher.AEAD
	nonceBase []byte
	chunkSize int
	streamAAD []byte
	buf       []byte
	chunkIdx  uint64
	closed    bool
}

// NewStreamWriter creates a new authenticated streaming writer.
// If chunkSize <= 0, DefaultChunkSize (64 KiB) is used.
func NewStreamWriter(
	w io.Writer,
	aead cipher.AEAD,
	nonceBase []byte,
	chunkSize int,
	streamAAD []byte,
) (*StreamWriter, error) {
	if aead == nil {
		return nil, errors.New("aead: nil cipher provided")
	}
	if len(nonceBase) != aead.NonceSize() {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidNonceLength, aead.NonceSize(), len(nonceBase))
	}
	switch {
	case chunkSize <= 0:
		chunkSize = DefaultChunkSize
	case chunkSize < MinChunkSize:
		chunkSize = MinChunkSize
	case chunkSize > MaxChunkSize:
		chunkSize = MaxChunkSize
	}

	return &StreamWriter{
		w:         w,
		aead:      aead,
		nonceBase: append([]byte(nil), nonceBase...),
		chunkSize: chunkSize,
		streamAAD: append([]byte(nil), streamAAD...),
		buf:       make([]byte, 0, chunkSize+1024),
	}, nil
}

// Write buffers data and writes out intermediate authenticated chunks.
func (sw *StreamWriter) Write(p []byte) (int, error) {
	if sw.closed {
		return 0, errors.New("aead: write to closed StreamWriter")
	}
	if len(p) == 0 {
		return 0, nil
	}

	written := len(p)
	sw.buf = append(sw.buf, p...)

	// We only flush chunks when len(sw.buf) > sw.chunkSize so that the last remaining bytes
	// can be emitted with flagLastChunk when Close() is called.
	for len(sw.buf) > sw.chunkSize {
		chunk := sw.buf[:sw.chunkSize]
		if err := sw.writeChunk(chunk, 0x00); err != nil {
			return 0, err
		}
		// Shift buffer
		sw.buf = append(sw.buf[:0], sw.buf[sw.chunkSize:]...)
	}

	return written, nil
}

// Close flushes any remaining buffered plaintext as the authenticated terminal chunk.
func (sw *StreamWriter) Close() error {
	if sw.closed {
		return nil
	}
	sw.closed = true

	// Emit final chunk (even if empty) with flagLastChunk.
	if err := sw.writeChunk(sw.buf, flagLastChunk); err != nil {
		return err
	}
	sw.buf = nil

	if closer, ok := sw.w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func (sw *StreamWriter) writeChunk(plaintext []byte, flag uint8) error {
	nonce := deriveChunkNonce(sw.nonceBase, sw.chunkIdx)
	aad := deriveChunkAAD(sw.streamAAD, sw.chunkIdx, flag)

	ciphertext := sw.aead.Seal(nil, nonce, plaintext, aad)
	ctLen := uint32(len(ciphertext))

	var header [5]byte
	binary.BigEndian.PutUint32(header[0:4], ctLen)
	header[4] = flag

	if _, err := sw.w.Write(header[:]); err != nil {
		return fmt.Errorf("aead: write chunk %d header: %w", sw.chunkIdx, err)
	}
	if _, err := sw.w.Write(ciphertext); err != nil {
		return fmt.Errorf("aead: write chunk %d ciphertext: %w", sw.chunkIdx, err)
	}

	sw.chunkIdx++
	return nil
}

// StreamReader wraps an io.Reader and decrypts/authenticates streaming AEAD chunks.
type StreamReader struct {
	r          io.Reader
	aead       cipher.AEAD
	nonceBase  []byte
	streamAAD  []byte
	buf        []byte
	chunkIdx   uint64
	reachedEOF bool
}

// NewStreamReader creates a new authenticated streaming reader.
func NewStreamReader(r io.Reader, aead cipher.AEAD, nonceBase, streamAAD []byte) (*StreamReader, error) {
	if aead == nil {
		return nil, errors.New("aead: nil cipher provided")
	}
	if len(nonceBase) != aead.NonceSize() {
		return nil, fmt.Errorf("%w: expected %d bytes, got %d", ErrInvalidNonceLength, aead.NonceSize(), len(nonceBase))
	}

	return &StreamReader{
		r:         r,
		aead:      aead,
		nonceBase: append([]byte(nil), nonceBase...),
		streamAAD: append([]byte(nil), streamAAD...),
	}, nil
}

// Read decrypts chunks on demand and returns the authenticated plaintext.
func (sr *StreamReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	for len(sr.buf) == 0 {
		if sr.reachedEOF {
			return 0, io.EOF
		}

		// Read next chunk frame header: 4 bytes length + 1 byte flag
		var header [5]byte
		_, err := io.ReadFull(sr.r, header[:])
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return 0, ErrStreamTruncated
			}
			return 0, fmt.Errorf("aead: read chunk %d header: %w", sr.chunkIdx, err)
		}

		ctLen := binary.BigEndian.Uint32(header[0:4])
		flag := header[4]

		if ctLen > MaxChunkSize {
			return 0, fmt.Errorf("%w: chunk length %d", ErrChunkTooLarge, ctLen)
		}
		if int(ctLen) < sr.aead.Overhead() {
			return 0, fmt.Errorf("aead: chunk length %d smaller than tag overhead", ctLen)
		}

		ctBuf := make([]byte, ctLen)
		if _, err := io.ReadFull(sr.r, ctBuf); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return 0, ErrStreamTruncated
			}
			return 0, fmt.Errorf("aead: read chunk %d ciphertext: %w", sr.chunkIdx, err)
		}

		nonce := deriveChunkNonce(sr.nonceBase, sr.chunkIdx)
		aad := deriveChunkAAD(sr.streamAAD, sr.chunkIdx, flag)

		pt, err := sr.aead.Open(nil, nonce, ctBuf, aad)
		if err != nil {
			return 0, fmt.Errorf("aead: chunk %d authentication failed: %w", sr.chunkIdx, err)
		}

		if flag&flagLastChunk != 0 {
			sr.reachedEOF = true
		}

		sr.chunkIdx++
		sr.buf = pt
	}

	n := copy(p, sr.buf)
	sr.buf = sr.buf[n:]
	return n, nil
}
