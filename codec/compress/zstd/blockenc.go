// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zstd

import (
	"bytes"
)

const (
	maxBlockPayloadSize = 128 * 1024 // 128 KB
)

// BlockEncoder handles encoding and packing of individual Zstd blocks.
type BlockEncoder struct {
	dst []byte
}

// NewBlockEncoder constructs a block encoder.
func NewBlockEncoder() *BlockEncoder {
	return &BlockEncoder{
		dst: make([]byte, 0, maxBlockPayloadSize+1024),
	}
}

// EncodeBlock encodes a chunk of up to 128KB into dest according to RFC 8878.
func (be *BlockEncoder) EncodeBlock(src []byte, last bool, dest *bytes.Buffer) error {
	srcLen := len(src)
	if srcLen == 0 {
		if last {
			// Emit empty last raw block
			dest.Write([]byte{0x01, 0x00, 0x00})
		}
		return nil
	}

	// Check if all bytes are identical for fast RLE block
	allSame := true
	b0 := src[0]
	for _, b := range src[1:] {
		if b != b0 {
			allSame = false
			break
		}
	}

	if allSame && srcLen > 8 {
		// Emit RLE block: blockType = 1
		var lastBit int
		if last {
			lastBit = 1
		}
		header := [3]byte{
			byte(lastBit | (1 << 1) | ((srcLen & 0x1F) << 3)),
			byte((srcLen >> 5) & 0xFF),
			byte((srcLen >> 13) & 0xFF),
		}
		dest.Write(header[:])
		dest.WriteByte(b0)
		return nil
	}

	// Emit Raw block: blockType = 0
	var lastBit int
	if last {
		lastBit = 1
	}
	header := [3]byte{
		byte(lastBit | (0 << 1) | ((srcLen & 0x1F) << 3)),
		byte((srcLen >> 5) & 0xFF),
		byte((srcLen >> 13) & 0xFF),
	}
	dest.Write(header[:])
	dest.Write(src)
	return nil
}
