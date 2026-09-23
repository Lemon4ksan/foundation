// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fse

import (
	"bytes"
	"testing"
)

func BenchmarkFSECompress(b *testing.B) {
	data := bytes.Repeat([]byte("ABRACADABRA_ALAKAZAM_SIMSALABIM_FSE_COMPRESS"), 100)
	var s Scratch
	_, err := Compress(data, &s)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Compress(data, &s)
	}
}

func BenchmarkFSEDecompress(b *testing.B) {
	data := bytes.Repeat([]byte("ABRACADABRA_ALAKAZAM_SIMSALABIM_FSE_COMPRESS"), 100)
	var s Scratch
	comp, err := Compress(data, &s)
	if err != nil {
		b.Fatal(err)
	}
	var decScratch Scratch
	decScratch.DecompressLimit = len(data)
	_, err = Decompress(comp, &decScratch)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Decompress(comp, &decScratch)
	}
}
