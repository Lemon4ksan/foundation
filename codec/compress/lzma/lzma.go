// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lzma implements zero-dependency LZMA1 and LZMA2 compression and decompression engines.
package lzma

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

// ErrInvalidProperties indicates malformed LZMA header properties.
var ErrInvalidProperties = errors.New("lzma: invalid header properties")

// Header holds LZMA1 stream parameters.
type Header struct {
	Properties byte
	DictSize   uint32
	UnpackSize uint64
}

// ParseHeader decodes a standard 13-byte or 5-byte LZMA1 header.
func ParseHeader(data []byte) (Header, error) {
	if len(data) < 5 {
		return Header{}, ErrInvalidProperties
	}
	h := Header{
		Properties: data[0],
		DictSize:   binary.LittleEndian.Uint32(data[1:5]),
	}
	if len(data) >= 13 {
		h.UnpackSize = binary.LittleEndian.Uint64(data[5:13])
	} else {
		h.UnpackSize = ^uint64(0)
	}
	return h, nil
}

// Decompressor implements decompression for raw LZMA1 streams.
type Decompressor struct {
	DictSize   uint32
	UnpackSize uint64
	LC         uint
	LP         uint
	PB         uint
}

// NewDecompressor creates an LZMA1 decompressor.
func NewDecompressor(dictSize uint32, unpackSize uint64) *Decompressor {
	if dictSize <= 0 {
		dictSize = 8 * 1024 * 1024
	}
	return &Decompressor{
		DictSize:   dictSize,
		UnpackSize: unpackSize,
		LC:         3,
		LP:         0,
		PB:         2,
	}
}

// Decompress wraps src with an LZMA1 streaming decompressor.
func (d *Decompressor) Decompress(src io.Reader) (io.ReadCloser, error) {
	var rd RangeDecoder
	if err := rd.Init(src); err != nil {
		return nil, err
	}

	core := NewDecoderCore(d.LC, d.LP, d.PB, d.DictSize, d.UnpackSize)
	var out bytes.Buffer
	if _, err := core.DecodeStream(
		&rd,
		&out,
		d.UnpackSize,
	); err != nil && !errors.Is(err, io.EOF) &&
		!errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	return io.NopCloser(&out), nil
}

// Compressor implements compression for LZMA1 streams.
type Compressor struct {
	Level    int
	DictSize uint32
	LC       uint
	LP       uint
	PB       uint
}

// NewCompressor creates an LZMA1 compressor configured for the given level.
func NewCompressor(level int) *Compressor {
	dictSize := uint32(8 * 1024 * 1024)
	if level >= 9 {
		dictSize = 32 * 1024 * 1024
	} else if level <= 1 {
		dictSize = 1 * 1024 * 1024
	}
	return &Compressor{
		Level:    level,
		DictSize: dictSize,
		LC:       3,
		LP:       0,
		PB:       2,
	}
}

// Compress reads from src and compresses into dest.
func (c *Compressor) Compress(src io.Reader, dest io.Writer) (int64, error) {
	raw, err := io.ReadAll(src)
	if err != nil {
		return 0, err
	}

	var re RangeEncoder
	re.Init(dest)

	core := NewEncoderCore(c.LC, c.LP, c.PB, c.DictSize)
	if err := core.EncodeStream(raw, &re); err != nil {
		return 0, err
	}

	if err := re.Flush(); err != nil {
		return 0, err
	}
	return int64(len(raw)), nil
}
