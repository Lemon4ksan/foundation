// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma_test

import (
	"bytes"
	"io"

	"github.com/lemon4ksan/foundation/codec/compress/lzma"
)

func ExampleNewCompressor2() {
	// Configure level presets: LevelFastest (1), LevelNormal (5), LevelUltra (9)
	opts := lzma.OptionsForLevel(lzma.LevelUltra)
	opts.DictSize = 32 * 1024 * 1024

	comp := lzma.NewCompressor2WithOptions(opts)

	sourceReader := bytes.NewReader([]byte("data"))
	targetWriter := io.Discard

	_, err := comp.Compress(sourceReader, targetWriter)
	if err != nil {
		panic(err)
	}
}

func ExampleNewDecompressor2() {
	decomp := lzma.NewDecompressor2(8 * 1024 * 1024)

	// Dummy compressed data for compilation
	compressedReader := bytes.NewReader([]byte{})
	destWriter := io.Discard

	rc, err := decomp.Decompress(compressedReader)
	if err != nil {
		return
	}
	defer rc.Close()

	_, _ = io.Copy(destWriter, rc)
}
