// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package huff0

import (
	"bytes"
	"testing"
)

func BenchmarkHuff0Compress1X(b *testing.B) {
	data := bytes.Repeat([]byte("Huffman coding benchmarks with zero allocations! 1234567890"), 30)
	var s Scratch
	_, _, err := Compress1X(data, &s)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Compress1X(data, &s)
	}
}

func BenchmarkHuff0Decompress1X(b *testing.B) {
	data := bytes.Repeat([]byte("Huffman coding benchmarks with zero allocations! 1234567890"), 30)
	var s Scratch
	comp, _, err := Compress1X(data, &s)
	if err != nil {
		b.Fatal(err)
	}

	var decScratch Scratch
	s2, remain, err := ReadTable(comp, &decScratch)
	if err != nil {
		b.Fatal(err)
	}
	s2.MaxDecodedSize = len(data)
	// Warm up
	_, err = s2.Decompress1X(remain)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s2.Decompress1X(remain)
	}
}

func BenchmarkHuff0Compress4X(b *testing.B) {
	data := bytes.Repeat([]byte("Huffman coding 4X stream benchmarks with zero allocations! 1234567890"), 30)
	var s Scratch
	_, _, err := Compress4X(data, &s)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Compress4X(data, &s)
	}
}

func BenchmarkHuff0Decompress4X(b *testing.B) {
	data := bytes.Repeat([]byte("Huffman coding 4X stream benchmarks with zero allocations! 1234567890"), 30)
	var s Scratch
	comp, _, err := Compress4X(data, &s)
	if err != nil {
		b.Fatal(err)
	}

	var decScratch Scratch
	s2, remain, err := ReadTable(comp, &decScratch)
	if err != nil {
		b.Fatal(err)
	}
	s2.MaxDecodedSize = len(data)
	// Warm up
	_, err = s2.Decompress4X(remain, len(data))
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = s2.Decompress4X(remain, len(data))
	}
}
