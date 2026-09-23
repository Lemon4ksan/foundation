// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matchfinder

import (
	"bytes"
	"testing"
)

func BenchmarkM0(b *testing.B) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 50)
	var m M0
	var matches []Match
	matches = m.FindMatches(matches, data)
	m.Reset()

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches = m.FindMatches(matches[:0], data)
	}
}

func BenchmarkM4(b *testing.B) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 50)
	var m M4
	var matches []Match
	for range 60 {
		matches = m.FindMatches(matches[:0], data)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches = m.FindMatches(matches[:0], data)
	}
}

func BenchmarkZFast(b *testing.B) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 50)
	var m ZFast
	var matches []Match
	matches = m.FindMatches(matches, data)
	m.Reset()

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches = m.FindMatches(matches[:0], data)
	}
}

func BenchmarkZDFast(b *testing.B) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 50)
	var m ZDFast
	var matches []Match
	matches = m.FindMatches(matches, data)
	m.Reset()

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches = m.FindMatches(matches[:0], data)
	}
}

func BenchmarkBargain1(b *testing.B) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. 1234567890"), 50)
	var m Bargain1
	var matches []Match
	for range 60 {
		matches = m.FindMatches(matches[:0], data)
	}

	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matches = m.FindMatches(matches[:0], data)
	}
}
