// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package huff0

import (
	"bytes"
	"errors"
	"testing"
)

// TestHuff0Roundtrip1X_Varied tests Compress1X and Decompress1X across multiple sizes.
func TestHuff0Roundtrip1X_Varied(t *testing.T) {
	testData := [][]byte{
		bytes.Repeat([]byte("Huffman coding in foundation Go library! 1234567890"), 10),
		bytes.Repeat([]byte("Huffman coding in foundation Go library! 1234567890"), 50),
		bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()"), 80),
	}

	for i, data := range testData {
		var s Scratch
		compressed, reUsed, err := Compress1X(data, &s)
		if err != nil {
			t.Fatalf("case %d Compress1X failed: %v", i, err)
		}
		if reUsed {
			t.Fatalf("case %d expected reUsed = false on first run", i)
		}

		var decScratch Scratch
		s2, remain, err := ReadTable(compressed, &decScratch)
		if err != nil {
			t.Fatalf("case %d ReadTable failed: %v", i, err)
		}
		s2.MaxDecodedSize = len(data)
		decompressed, err := s2.Decompress1X(remain)
		if err != nil {
			t.Fatalf("case %d Decompress1X failed: %v", i, err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("case %d Decompress1X mismatch", i)
		}
	}
}

// TestHuff0Roundtrip4X tests 4-stream interleaved Compress4X and Decompress4X.
func TestHuff0Roundtrip4X(t *testing.T) {
	t.Run("Small4X_8Bit", func(t *testing.T) {
		data := bytes.Repeat([]byte("ABCDE"), 100) // 500 bytes < 800
		var s Scratch
		compressed, _, err := Compress4X(data, &s)
		if err != nil {
			t.Fatalf("Compress4X failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(compressed, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable failed: %v", err)
		}
		s2.MaxDecodedSize = len(data)
		decompressed, err := s2.Decompress4X(remain, len(data))
		if err != nil {
			t.Fatalf("Decompress4X failed: %v", err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("Decompress4X mismatch")
		}
	})

	sizes := []int{1024, 2048, 8192, 32768}
	for _, sz := range sizes {
		data := bytes.Repeat([]byte("0123456789abcdefghijklmnopqrstuvwxyz!@#$%^"), sz/42+1)[:sz]
		var s Scratch
		compressed, _, err := Compress4X(data, &s)
		if err != nil {
			t.Fatalf("Compress4X failed for size %d: %v", sz, err)
		}

		var decScratch Scratch
		s2, remain, err := ReadTable(compressed, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable failed for size %d: %v", sz, err)
		}
		s2.MaxDecodedSize = len(data)
		decompressed, err := s2.Decompress4X(remain, len(data))
		if err != nil {
			t.Fatalf("Decompress4X failed for size %d: %v", sz, err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("Decompress4X mismatch for size %d", sz)
		}
	}
}

// TestHuff0BuildCTableAndTableAPI tests BuildCTable, CanUseTable, EstimateSize, AppendTable.
func TestHuff0BuildCTableAndTableAPI(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDABCDABCDWXYZWXYZ"), 50)
	var hist [256]uint32
	for _, b := range data {
		hist[b]++
	}

	var s Scratch
	if err := s.BuildCTable(&hist); err != nil {
		t.Fatalf("BuildCTable failed: %v", err)
	}

	if !s.CanUseTable(&hist) {
		t.Fatalf("CanUseTable should return true for matching histogram")
	}

	est := s.EstimateSize(&hist)
	if est <= 0 {
		t.Fatalf("EstimateSize returned non-positive: %d", est)
	}

	// Incompatible histogram
	var otherHist [256]uint32
	otherHist['!'] = 10
	if s.CanUseTable(&otherHist) {
		t.Fatalf("CanUseTable should return false for symbol not in table")
	}
	if s.EstimateSize(&otherHist) != -1 {
		t.Fatalf("EstimateSize should return -1 for incompatible histogram")
	}

	hdr, err := s.AppendTable(nil)
	if err != nil {
		t.Fatalf("AppendTable failed: %v", err)
	}
	if len(hdr) == 0 {
		t.Fatalf("AppendTable generated empty header")
	}

	// Verify header can be read back by ReadTable
	var decScratch Scratch
	_, remain, err := ReadTable(hdr, &decScratch)
	if err != nil {
		t.Fatalf("ReadTable on AppendTable output failed: %v", err)
	}
	if len(remain) != 0 {
		t.Fatalf("expected 0 remaining bytes, got %d", len(remain))
	}

	// Test TransferCTable
	var s2 Scratch
	s2.TransferCTable(&s)
	if !s2.CanUseTable(&hist) {
		t.Fatalf("Transferred table should be usable")
	}
}

// TestHuff0TableReuse tests table reuse policies.
func TestHuff0TableReuse(t *testing.T) {
	data := bytes.Repeat([]byte("THE_QUICK_BROWN_FOX_JUMPS_OVER_THE_LAZY_DOG"), 40)

	var s Scratch
	s.Reuse = ReusePolicyAllow

	// First compress generates table
	comp1, reUsed1, err := Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress 1 failed: %v", err)
	}
	if reUsed1 {
		t.Fatalf("expected reUsed = false on first call")
	}

	// Second compress with same data and Prefer policy should reuse table
	s.Reuse = ReusePolicyPrefer
	s.Out = nil
	comp2, reUsed2, err := Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress 2 failed: %v", err)
	}
	if !reUsed2 {
		t.Fatalf("expected reUsed = true on second call with Prefer policy")
	}
	if len(comp2) >= len(comp1) {
		t.Logf("reused table compressed size %d vs original with header %d", len(comp2), len(comp1))
	}
}

// TestEstimateSizes exercises EstimateSizes function.
func TestEstimateSizes(t *testing.T) {
	data := bytes.Repeat([]byte("quick brown fox jumps over the lazy dog! "), 20)
	var s Scratch
	tableSz, dataSz, reuseSz, err := EstimateSizes(data, &s)
	if err != nil {
		t.Fatalf("EstimateSizes failed: %v", err)
	}
	if tableSz <= 0 || dataSz <= 0 {
		t.Fatalf("unexpected sizes: tableSz=%d, dataSz=%d, reuseSz=%d", tableSz, dataSz, reuseSz)
	}
}

// TestStatelessDecoder tests s.Decoder() stateless decoding methods.
func TestStatelessDecoder(t *testing.T) {
	data := bytes.Repeat([]byte("Foundation High-Performance Concurrency & SIMD Framework"), 30)

	var s Scratch
	compressed, _, err := Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X failed: %v", err)
	}

	var decScratch Scratch
	s2, remain, err := ReadTable(compressed, &decScratch)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}

	dec := s2.Decoder()
	dst := make([]byte, 0, len(data))
	decomp, err := dec.Decompress1X(dst, remain)
	if err != nil {
		t.Fatalf("Decoder.Decompress1X failed: %v", err)
	}
	if !bytes.Equal(decomp, data) {
		t.Fatalf("Decoder.Decompress1X mismatch")
	}

	// Also test direct decompress1X8Bit on 8-bit table if applicable
	if dec.actualTableLog <= 8 {
		dst8 := make([]byte, 0, len(data))
		decomp8, err := dec.decompress1X8Bit(dst8, remain)
		if err == nil && !bytes.Equal(decomp8, data) {
			t.Fatalf("decompress1X8Bit mismatch")
		}
	}
}

// TestInternalHelpers exercises bitReaderBytes, encTwoSymbols, and scratch estimates.
func TestInternalHelpers(t *testing.T) {
	// bitReaderBytes
	var brb bitReaderBytes
	if err := brb.init(nil); err == nil {
		t.Fatalf("expected error on nil input to bitReaderBytes")
	}
	src := []byte{0x80, 0x12, 0x34, 0x56, 0x78}
	if err := brb.init(src); err != nil {
		t.Fatalf("bitReaderBytes init failed: %v", err)
	}
	_ = brb.peekByteFast()
	brb.advance(1)
	brb.fillFast()
	brb.fill()
	_ = brb.finished()
	_ = brb.remaining()
	_ = brb.close()

	// bitWriter encTwoSymbols
	var bw bitWriter
	// scratch estimates
	var s Scratch
	data := bytes.Repeat([]byte("ABCDEF"), 20)
	prep, err := s.prepare(data)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	_, _ = prep.countSimple(data)
	prep.optimalTableLog()
	_ = prep.buildCTable()
	bw.encTwoSymbols(prep.cTable, data[0], data[1])
	bw.flush32()
	bw.flushAlign()
	bw.close()

	_, _ = prep.cTable.estTableSize(prep)
	_ = prep.cTable.estimateSize(prep.count[:])
	_ = prep.minSize(len(data))
	_ = prep.canUseTable(prep.cTable)
	_ = prep.validateTable(prep.cTable)
}

// TestHuff0Errors verifies boundary and error conditions.
func TestHuff0Errors(t *testing.T) {
	var s Scratch
	// Empty input returns ErrUseRLE, 1-byte returns ErrIncompressible
	if _, _, err := Compress1X(nil, &s); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE for nil input, got %v", err)
	}
	if _, _, err := Compress1X([]byte("A"), &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for 1-byte input, got %v", err)
	}
	// Single repeated byte -> ErrUseRLE
	if _, _, err := Compress1X(bytes.Repeat([]byte("A"), 100), &s); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE, got %v", err)
	}
	// ReadTable too small
	if _, _, err := ReadTable([]byte{0x01}, &s); err == nil {
		t.Fatalf("expected error for 1-byte table input")
	}
	// Decompress without loaded table
	var emptyScratch Scratch
	if _, err := emptyScratch.Decompress1X([]byte{1, 2, 3}); err == nil {
		t.Fatalf("expected error for Decompress1X without table")
	}

	// BuildCTable errors
	if err := s.BuildCTable(nil); err == nil {
		t.Fatalf("expected error for nil count in BuildCTable")
	}
	var emptyHist [256]uint32
	if err := s.BuildCTable(&emptyHist); err == nil {
		t.Fatalf("expected error for empty histogram in BuildCTable")
	}
	var singleSymbolHist [256]uint32
	singleSymbolHist['A'] = 100
	if err := s.BuildCTable(&singleSymbolHist); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE for single symbol in BuildCTable, got %v", err)
	}

	// Decompress4X errors
	var decScratch Scratch
	decScratch.MaxDecodedSize = 100
	if _, err := decScratch.Decompress4X([]byte{1, 2, 3}, 200); !errors.Is(err, ErrMaxDecodedSizeExceeded) {
		t.Fatalf("expected ErrMaxDecodedSizeExceeded, got %v", err)
	}
}

// TestCompress4XpDirect tests compress4Xp parallel compressor.
func TestCompress4XpDirect(t *testing.T) {
	data := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog! 1234567890"), 50)
	var s Scratch
	prep, err := s.prepare(data)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	compressed, reUsed, err := compress(data, prep, prep.compress4Xp)
	if err != nil {
		t.Fatalf("compress4Xp failed: %v", err)
	}
	if reUsed {
		t.Fatalf("unexpected reUsed")
	}
	var decScratch Scratch
	s2, remain, err := ReadTable(compressed, &decScratch)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	s2.MaxDecodedSize = len(data)
	decomp, err := s2.Decompress4X(remain, len(data))
	if err != nil {
		t.Fatalf("Decompress4X failed: %v", err)
	}
	if !bytes.Equal(decomp, data) {
		t.Fatalf("decompressed mismatch")
	}

	// Short data error branch
	var sShort Scratch
	if _, err = sShort.compress4Xp([]byte("too small")); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for short data in compress4Xp")
	}
}

// TestSetMaxHeightAndTableLimits exercises setMaxHeight with skewed frequencies.
func TestSetMaxHeightAndTableLimits(t *testing.T) {
	var data []byte
	fib := []int{1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144, 233, 377, 610, 987, 1597, 2584, 4181, 6765, 10946}
	for i, count := range fib {
		data = append(data, bytes.Repeat([]byte{byte(i + 1)}, count)...)
	}
	for tl := uint8(5); tl <= 9; tl++ {
		var s Scratch
		s.TableLog = tl
		prep, err := s.prepare(data)
		if err != nil {
			t.Fatalf("prepare failed: %v", err)
		}
		prep.countSimple(data)
		prep.optimalTableLog()
		err = prep.buildCTable()
		if err != nil {
			t.Fatalf("buildCTable failed for TableLog %d: %v", tl, err)
		}
		var w bytes.Buffer
		prep.matches(prep.cTable, &w)
	}
}

// TestDecompress8BitDirect tests 1X and 4X 8-bit decoder methods directly.
func TestDecompress8BitDirect(t *testing.T) {
	// Construct data8 with 90 symbols and skewed counts so Huffman tree depth >= 8 and actualTableLog == 8
	var data8 []byte
	for i := 0; i < 90; i++ {
		data8 = append(data8, bytes.Repeat([]byte{byte(i + 1)}, (i+1)*5)...)
	}

	// Test 1X with TableLog = 8
	{
		var s Scratch
		s.TableLog = 8
		comp, _, err := Compress1X(data8, &s)
		if err != nil {
			t.Fatalf("Compress1X failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(comp, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable failed: %v", err)
		}
		if s2.actualTableLog != 8 {
			t.Fatalf("expected actualTableLog == 8, got %d", s2.actualTableLog)
		}
		dec := s2.Decoder()
		dst := make([]byte, 0, len(data8))
		decomp, err := dec.decompress1X8Bit(dst, remain)
		if err != nil {
			t.Fatalf("decompress1X8Bit (tableLog=8) failed: %v", err)
		}
		if !bytes.Equal(decomp, data8) {
			t.Fatalf("decompress1X8Bit mismatch")
		}
		dstExact := make([]byte, 0, len(data8))
		decompExact, err := dec.decompress1X8BitExactly(dstExact, remain)
		if err != nil {
			t.Fatalf("decompress1X8BitExactly failed: %v", err)
		}
		if !bytes.Equal(decompExact, data8) {
			t.Fatalf("decompress1X8BitExactly mismatch")
		}
	}

	// Test 1X with TableLog < 8 (TableLog = 6)
	{
		smallData := bytes.Repeat([]byte("ABCDEFGH"), 50)
		var s Scratch
		s.TableLog = 6
		comp, _, err := Compress1X(smallData, &s)
		if err != nil {
			t.Fatalf("Compress1X failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(comp, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable failed: %v", err)
		}
		dec := s2.Decoder()
		dst := make([]byte, 0, len(smallData))
		decomp, err := dec.decompress1X8Bit(dst, remain)
		if err != nil {
			t.Fatalf("decompress1X8Bit (tableLog<8) failed: %v", err)
		}
		if !bytes.Equal(decomp, smallData) {
			t.Fatalf("decompress1X8Bit (<8) mismatch")
		}
	}

	// Test 4X with TableLog = 8
	{
		var s Scratch
		s.TableLog = 8
		comp, _, err := Compress4X(data8, &s)
		if err != nil {
			t.Fatalf("Compress4X failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(comp, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable failed: %v", err)
		}
		if s2.actualTableLog != 8 {
			t.Fatalf("expected actualTableLog == 8, got %d", s2.actualTableLog)
		}
		dec := s2.Decoder()
		dst := make([]byte, 0, len(data8))
		decomp, err := dec.decompress4X8bit(dst, remain)
		if err != nil {
			t.Fatalf("decompress4X8bit (tableLog=8) failed: %v", err)
		}
		if !bytes.Equal(decomp, data8) {
			t.Fatalf("decompress4X8bit mismatch")
		}
		dstExact := make([]byte, 0, len(data8))
		decompExact, err := dec.decompress4X8bitExactly(dstExact, remain)
		if err != nil {
			t.Fatalf("decompress4X8bitExactly failed: %v", err)
		}
		if !bytes.Equal(decompExact, data8) {
			t.Fatalf("decompress4X8bitExactly mismatch")
		}
	}

	// Test 4X with TableLog < 8
	{
		smallData := bytes.Repeat([]byte("ABCDEFGH"), 50)
		var s Scratch
		s.TableLog = 6
		comp, _, err := Compress4X(smallData, &s)
		if err != nil {
			t.Fatalf("Compress4X failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(comp, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable failed: %v", err)
		}
		dec := s2.Decoder()
		dst := make([]byte, 0, len(smallData))
		decomp, err := dec.decompress4X8bit(dst, remain)
		if err != nil {
			t.Fatalf("decompress4X8bit (tableLog<8) failed: %v", err)
		}
		if !bytes.Equal(decomp, smallData) {
			t.Fatalf("decompress4X8bit (<8) mismatch")
		}
	}
}

// TestHighTableLogAndAsm exercises TableLog >= 9 and ASM decompression paths.
func TestHighTableLogAndAsm(t *testing.T) {
	var data []byte
	for i := 0; i < 200; i++ {
		repeat := (i % 20) + 1
		data = append(data, bytes.Repeat([]byte{byte(i)}, repeat*20)...)
	}
	var s Scratch
	s.TableLog = 11
	comp, _, err := Compress4X(data, &s)
	if err != nil {
		t.Fatalf("Compress4X failed: %v", err)
	}
	var decScratch Scratch
	s2, remain, err := ReadTable(comp, &decScratch)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	s2.MaxDecodedSize = len(data)
	decomp, err := s2.Decompress4X(remain, len(data))
	if err != nil {
		t.Fatalf("Decompress4X (TableLog 11) failed: %v", err)
	}
	if !bytes.Equal(decomp, data) {
		t.Fatalf("Decompress4X mismatch")
	}

	var s1X Scratch
	s1X.TableLog = 11
	comp1, _, err := Compress1X(data, &s1X)
	if err != nil {
		t.Fatalf("Compress1X failed: %v", err)
	}
	var decScratch1 Scratch
	s3, remain1, err := ReadTable(comp1, &decScratch1)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	s3.MaxDecodedSize = len(data)
	decomp1, err := s3.Decompress1X(remain1)
	if err != nil {
		t.Fatalf("Decompress1X (TableLog 11) failed: %v", err)
	}
	if !bytes.Equal(decomp1, data) {
		t.Fatalf("Decompress1X mismatch")
	}
}

// TestBitReaderShiftedHelpers exercises bitReaderShifted helper methods.
func TestBitReaderShiftedHelpers(t *testing.T) {
	var br bitReaderShifted
	if err := br.init(nil); err == nil {
		t.Fatalf("expected error on nil input")
	}
	src := []byte{0x80, 0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0, 0x11}
	if err := br.init(src); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	_ = br.peekBitsFast(5)
	br.advance(5)
	br.fillFast()
	br.fillFastStart()
	br.fill()
	_ = br.remaining()
	_ = br.close()
}

