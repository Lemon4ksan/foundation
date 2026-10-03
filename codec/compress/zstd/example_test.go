// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zstd_test

import (
	"bytes"
	"io"

	"github.com/lemon4ksan/foundation/codec/compress/zstd"
)

func ExampleNewReader() {
	compressedData := []byte{
		40, 181, 47, 253, 0, 88, 32, 0, 0, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100,
	}

	decoder, err := zstd.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		panic(err)
	}
	defer decoder.Close()

	decompressed, err := io.ReadAll(decoder)
	if err != nil {
		panic(err)
	}

	_ = decompressed
}

func ExampleNewWriter() {
	var buf bytes.Buffer

	encoder, err := zstd.NewWriter(&buf, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		panic(err)
	}

	_, err = encoder.Write([]byte("hello world"))
	if err != nil {
		panic(err)
	}

	err = encoder.Close()
	if err != nil {
		panic(err)
	}
}
