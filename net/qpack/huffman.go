// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qpack

import (
	"github.com/lemon4ksan/foundation/net/hpack"
	"github.com/lemon4ksan/foundation/silicon/bytesconv"
	"github.com/lemon4ksan/foundation/silicon/pool"
)

var huffmanDecStorage = pool.NewPerPStorage(func() *[]byte {
	b := make([]byte, 0, 512)
	return &b
})

func appendHuffman(dst []byte, s string) []byte {
	return hpack.AppendHuffmanString(dst, s)
}

func huffmanLen(s string) int {
	return int(hpack.HuffmanEncodeLength(s))
}

func decodeHuffman(src []byte, arena *[]byte) (string, error) {
	if arena == nil {
		bufPtr := huffmanDecStorage.Get()
		defer huffmanDecStorage.Put(bufPtr)

		dst, err := hpack.AppendHuffmanDecode((*bufPtr)[:0], src)
		if err != nil {
			return "", err
		}
		*bufPtr = dst

		return string(dst), nil
	}

	start := len(*arena)
	dst, err := hpack.AppendHuffmanDecode(*arena, src)
	if err != nil {
		return "", err
	}
	*arena = dst

	return bytesconv.B2S((*arena)[start:]), nil
}
