// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package compress

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"

	"github.com/lemon4ksan/foundation/borrow"
)

// Magic header constants for Dictionary-Compressed HTTP streams (RFC 9842).
var (
	MagicDCB = [4]byte{0xff, 0x44, 0x43, 0x42}                         // RFC 9842 §4
	MagicDCZ = [8]byte{0x5e, 0x2a, 0x4d, 0x18, 0x20, 0x00, 0x00, 0x00} // RFC 9842 §5
)

var (
	// ErrDictionaryMismatch signals that the SHA-256 hash in the stream header does not match the local dictionary.
	ErrDictionaryMismatch = errors.New("codec: dictionary SHA-256 hash mismatch")

	// ErrInvalidDCZHeader signals a corrupted or non-conformant DCZ stream header (RFC 9842 §5).
	ErrInvalidDCZHeader = errors.New("codec: invalid Dictionary-Compressed Zstandard (dcz) header")

	// ErrInvalidDCBHeader signals a corrupted or non-conformant DCB stream header (RFC 9842 §4).
	ErrInvalidDCBHeader = errors.New("codec: invalid Dictionary-Compressed Brotli (dcb) header")
)

// UnbrotliDCB decompresses a Dictionary-Compressed Brotli payload (RFC 9842 §4)
// using the provided raw dictionary.
func UnbrotliDCB(src, dst, dictBytes []byte) ([]byte, error) {
	if len(src) < 36 {
		return nil, ErrInvalidDCBHeader
	}

	// Validate RFC 9842 Section 4 magic signature (\xFFDCB)
	if !bytes.Equal(src[:4], MagicDCB[:]) {
		return nil, ErrInvalidDCBHeader
	}

	// Validate expected dictionary hash using constant-time comparison
	expectedHash := sha256.Sum256(dictBytes)
	if subtle.ConstantTimeCompare(src[4:36], expectedHash[:]) != 1 {
		return nil, fmt.Errorf("%w: expected %x, got %x", ErrDictionaryMismatch, expectedHash, src[4:36])
	}

	// Decompress trailing Brotli stream against shared dictionary context
	return Unbrotli(src[36:], dst)
}

// UnbrotliDCBScoped decompresses a DCB payload directly into a zero-allocation scoped buffer.
func UnbrotliDCBScoped(s *borrow.Scope, src, dictBytes []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	if len(src) < 36 {
		return borrow.Bytes{}, ErrInvalidDCBHeader
	}

	if !bytes.Equal(src[:4], MagicDCB[:]) {
		return borrow.Bytes{}, ErrInvalidDCBHeader
	}

	expectedHash := sha256.Sum256(dictBytes)
	if subtle.ConstantTimeCompare(src[4:36], expectedHash[:]) != 1 {
		return borrow.Bytes{}, fmt.Errorf("%w: expected %x, got %x", ErrDictionaryMismatch, expectedHash, src[4:36])
	}

	return UnbrotliScoped(s, src[36:])
}

// NewDCBReader returns an io.ReadCloser that decompresses a streaming DCB response (RFC 9842 §4).
func NewDCBReader(r io.Reader, dictBytes []byte) (io.ReadCloser, error) {
	if r == nil {
		return nil, errors.New("codec: nil reader")
	}

	var header [36]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDCBHeader, err)
	}

	if !bytes.Equal(header[:4], MagicDCB[:]) {
		return nil, ErrInvalidDCBHeader
	}

	expectedHash := sha256.Sum256(dictBytes)
	if subtle.ConstantTimeCompare(header[4:36], expectedHash[:]) != 1 {
		return nil, fmt.Errorf("%w: expected %x, got %x", ErrDictionaryMismatch, expectedHash, header[4:36])
	}

	br, err := AcquireBrotliReader(r)
	if err != nil {
		return nil, err
	}

	return &decompressReadCloser{
		reader: br,
		closer: closerOf(r),
		release: func() {
			ReleaseBrotliReader(br)
		},
	}, nil
}

// WrapDCBHeader prepends the standard 36-byte RFC 9842 §4 header (4-byte magic + 32-byte SHA-256).
func WrapDCBHeader(brotliStream []byte, dictHash [32]byte) []byte {
	out := make([]byte, 36+len(brotliStream))
	copy(out[:4], MagicDCB[:])
	copy(out[4:36], dictHash[:])
	copy(out[36:], brotliStream)
	return out
}
