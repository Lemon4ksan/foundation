// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zstd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/bits"

	"github.com/lemon4ksan/seal/codec/zstd/xxhash"
)

// Writer implements a streaming Zstandard (RFC 8878) encoder.
type Writer struct {
	w        io.Writer
	opts     encoderOptions
	blockEnc *BlockEncoder
	buf      []byte
	hasher   *xxhash.Digest
	closed   bool
	headerW  bool
}

// NewWriter creates a new Zstandard writer compressing to w.
func NewWriter(w io.Writer, opts ...EncoderOption) (*Writer, error) {
	if w == nil {
		return nil, errors.New("zstd: nil writer")
	}
	cfg := defaultEncoderOptions()
	for _, opt := range opts {
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	wr := &Writer{
		w:        w,
		opts:     cfg,
		blockEnc: NewBlockEncoder(),
		buf:      make([]byte, 0, maxBlockPayloadSize),
		hasher:   xxhash.New(),
	}
	return wr, nil
}

func (w *Writer) writeHeader() error {
	if w.headerW {
		return nil
	}
	// Magic Number: 0xFD2FB528 (4 bytes little-endian)
	var hdr [6]byte
	copy(hdr[0:4], []byte(frameMagic))

	fhd := byte(0x00)
	if w.opts.checksum {
		fhd |= 0x04
	}
	hdr[4] = fhd

	// Window Descriptor: for 4MB: (22-10)<<3 = 0x60
	winSize := w.opts.windowSize
	if winSize < 1<<10 {
		winSize = 1 << 10
	}
	winLog := bits.Len64(winSize - 1)
	if winLog < 10 {
		winLog = 10
	}
	if winLog > 31 {
		winLog = 31
	}
	hdr[5] = byte((winLog - 10) << 3)

	if _, err := w.w.Write(hdr[:]); err != nil {
		return err
	}
	w.headerW = true
	return nil
}

// Write writes uncompressed bytes into the Zstd stream.
func (w *Writer) Write(p []byte) (int, error) {
	if w.closed {
		return 0, ErrDecoderClosed
	}
	if len(p) == 0 {
		return 0, nil
	}

	if err := w.writeHeader(); err != nil {
		return 0, err
	}

	if w.opts.checksum {
		_, _ = w.hasher.Write(p)
	}

	totalWritten := len(p)
	for len(p) > 0 {
		avail := maxBlockPayloadSize - len(w.buf)
		if avail <= 0 {
			if err := w.flushBlock(false); err != nil {
				return 0, err
			}
			avail = maxBlockPayloadSize
		}

		toCopy := min(len(p), avail)
		w.buf = append(w.buf, p[:toCopy]...)
		p = p[toCopy:]
	}

	return totalWritten, nil
}

func (w *Writer) flushBlock(last bool) error {
	var blockBuf bytes.Buffer
	if err := w.blockEnc.EncodeBlock(w.buf, last, &blockBuf); err != nil {
		return err
	}
	w.buf = w.buf[:0]

	if blockBuf.Len() > 0 {
		if _, err := w.w.Write(blockBuf.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// Flush flushes buffered data to the output stream.
func (w *Writer) Flush() error {
	if w.closed {
		return ErrDecoderClosed
	}
	if err := w.writeHeader(); err != nil {
		return err
	}
	if len(w.buf) > 0 {
		return w.flushBlock(false)
	}
	return nil
}

// Close flushes final blocks and frame checksum.
func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	if err := w.writeHeader(); err != nil {
		return err
	}

	if err := w.flushBlock(true); err != nil {
		return err
	}

	if w.opts.checksum {
		crc := uint32(w.hasher.Sum64())
		var crcBuf [4]byte
		binary.LittleEndian.PutUint32(crcBuf[:], crc)
		if _, err := w.w.Write(crcBuf[:]); err != nil {
			return err
		}
	}

	return nil
}

// Reset resets the writer to write to a new destination.
func (w *Writer) Reset(dest io.Writer) {
	w.w = dest
	w.buf = w.buf[:0]
	w.hasher.Reset()
	w.closed = false
	w.headerW = false
}

// EncodeAll compresses src into dst, returning the compressed buffer.
func (w *Writer) EncodeAll(src, dst []byte) []byte {
	var buf bytes.Buffer
	if dst != nil {
		buf = *bytes.NewBuffer(dst[:0])
	}
	w.Reset(&buf)
	_, _ = w.Write(src)
	_ = w.Close()
	return buf.Bytes()
}
