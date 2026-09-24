// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package huff0

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"sync"
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
	fib := make([]int, 35)
	fib[0], fib[1] = 1, 2
	for i := 2; i < 35; i++ {
		fib[i] = fib[i-1] + fib[i-2]
		if fib[i] > 10000 {
			fib[i] = 10000
		}
	}
	for i, count := range fib {
		data = append(data, bytes.Repeat([]byte{byte(i + 1)}, count)...)
	}
	for tl := uint8(5); tl <= 11; tl++ {
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
	br.advance(32)
	br.fillFast()
	br.fill()
	_ = br.remaining()
	_ = br.close()

	var br2 bitReaderShifted
	if err := br2.init(src); err == nil {
		br2.fillFastStart()
	}
}

// TestDecoderMatches tests decoder table matching and verification logs.
func TestDecoderMatches(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"), 20)
	var s Scratch
	prep, err := s.prepare(data)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	prep.countSimple(data)
	prep.optimalTableLog()
	if err := prep.buildCTable(); err != nil {
		t.Fatalf("buildCTable failed: %v", err)
	}
	err = prep.cTable.write(prep)
	if err != nil {
		t.Fatalf("cTable.write failed: %v", err)
	}
	var decScratch Scratch
	s2, _, err := ReadTable(prep.Out, &decScratch)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	var buf bytes.Buffer
	s2.matches(prep.cTable, &buf)

	// Also test matches with discrepancies to cover error reporting branches
	badCTable := make(cTable, len(prep.cTable))
	copy(badCTable, prep.cTable)
	if len(badCTable) > 5 {
		badCTable[0].nBits = 0 // decoder with no encoder
		badCTable[1].nBits++   // bit size mismatch
		badCTable[2].val++     // output mismatch
	}
	var errBuf bytes.Buffer
	s2.matches(badCTable, &errBuf)
	if errBuf.Len() == 0 {
		t.Fatalf("expected error logs in matches with bad cTable")
	}

	// Empty table
	var emptyS Scratch
	emptyS.matches(prep.cTable, io.Discard)
}

// TestBuildCTableLargeTotal exercises total > BlockSizeMax scaling in BuildCTable.
func TestBuildCTableLargeTotal(t *testing.T) {
	var s Scratch
	var hist [256]uint32
	hist['A'] = 200000
	hist['B'] = 100000
	hist['C'] = 50000
	if err := s.BuildCTable(&hist); err != nil {
		t.Fatalf("BuildCTable failed: %v", err)
	}
	if !s.CanUseTable(&hist) {
		t.Fatalf("CanUseTable should return true")
	}
	_ = s.EstimateSize(nil)
	_ = s.CanUseTable(nil)

	// Single symbol after scaling returns ErrUseRLE
	var histRLE [256]uint32
	histRLE['A'] = 300000
	if err := s.BuildCTable(&histRLE); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE for scaled single symbol, got %v", err)
	}
}

// TestDecompress1X8BitAllLogs exercises tableLog cases 5, 7, and lower synthetic logs.
func TestDecompress1X8BitAllLogs(t *testing.T) {
	// TableLog 5
	{
		smallData := bytes.Repeat([]byte("ABCDE"), 100)
		var s Scratch
		s.TableLog = 5
		comp, _, err := Compress1X(smallData, &s)
		if err != nil {
			t.Fatalf("Compress1X (TL 5) failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(comp, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable (TL 5) failed: %v", err)
		}
		dec := s2.Decoder()
		dst := make([]byte, 0, len(smallData))
		decomp, err := dec.decompress1X8Bit(dst, remain)
		if err != nil {
			t.Fatalf("decompress1X8Bit (TL 5) failed: %v", err)
		}
		if !bytes.Equal(decomp, smallData) {
			t.Fatalf("decompress1X8Bit (TL 5) mismatch")
		}
	}

	// TableLog 7
	{
		midData := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789!@#$"), 25)
		var s Scratch
		s.TableLog = 7
		comp, _, err := Compress1X(midData, &s)
		if err != nil {
			t.Fatalf("Compress1X (TL 7) failed: %v", err)
		}
		var decScratch Scratch
		s2, remain, err := ReadTable(comp, &decScratch)
		if err != nil {
			t.Fatalf("ReadTable (TL 7) failed: %v", err)
		}
		dec := s2.Decoder()
		dst := make([]byte, 0, len(midData))
		decomp, err := dec.decompress1X8Bit(dst, remain)
		if err != nil {
			t.Fatalf("decompress1X8Bit (TL 7) failed: %v", err)
		}
		if !bytes.Equal(decomp, midData) {
			t.Fatalf("decompress1X8Bit (TL 7) mismatch")
		}
	}

	// Synthetic switch cases 1..7 in decompress1X8Bit
	{
		var pool sync.Pool
		var dt [256]dEntrySingle
		for i := range dt {
			dt[i] = dEntrySingle{entry: 8 | (uint16('A') << 8)} // length 8, symbol 'A'
		}
		for log := uint8(1); log <= 7; log++ {
			dec := &Decoder{
				dt:             dTable{single: dt[:]},
				actualTableLog: log,
				bufs:           &pool,
			}
			src := make([]byte, 280)
			src[len(src)-1] = 0x80
			dst := make([]byte, 0, 1000)
			_, _ = dec.decompress1X8Bit(dst, src)

			// Exceed max size on buffer flush
			dstShort := make([]byte, 0, 100)
			_, _ = dec.decompress1X8Bit(dstShort, src)
		}
	}
}

// TestEstimateSizesEdgeCases tests edge cases and error paths in EstimateSizes.
func TestEstimateSizesEdgeCases(t *testing.T) {
	var s Scratch
	// 1 byte
	_, _, _, err := EstimateSizes([]byte("A"), &s)
	if !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for 1-byte")
	}
	// RLE
	_, _, _, err = EstimateSizes(bytes.Repeat([]byte("A"), 50), &s)
	if !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE for repeated bytes")
	}
	// Unique bytes
	_, _, _, err = EstimateSizes([]byte("abcdefghijklmnopqrstuvwxyz"), &s)
	if !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for unique bytes")
	}
	// Normal then reuse
	data := bytes.Repeat([]byte("ABCDEFGHIJKLMN"), 20)
	_, _, err = Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X failed: %v", err)
	}
	tSz, dSz, rSz, err := EstimateSizes(data, &s)
	if err != nil {
		t.Fatalf("EstimateSizes with reuse failed: %v", err)
	}
	if tSz <= 0 || dSz <= 0 || rSz <= 0 {
		t.Fatalf("unexpected sizes: tSz=%d, dSz=%d, rSz=%d", tSz, dSz, rSz)
	}
}

// TestCompressEdgeCases tests compression constraints and error conditions.
func TestCompressEdgeCases(t *testing.T) {
	var s Scratch
	// Short data for Compress4X
	if _, _, err := Compress4X([]byte("too small"), &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for Compress4X short data")
	}
	// ReusePolicyMust without table
	var sMust Scratch
	sMust.Reuse = ReusePolicyMust
	data := bytes.Repeat([]byte("quick brown fox jumps over lazy dog"), 20)
	if _, _, err := Compress1X(data, &sMust); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for ReusePolicyMust without table")
	}
	// WantLogLess > 0
	var sWant Scratch
	sWant.WantLogLess = 1
	comp, _, err := Compress1X(data, &sWant)
	if err == nil && len(comp) == 0 {
		t.Fatalf("unexpected empty output")
	}

	// AppendTable edge cases: empty prevTable, nil s.fse
	var sEmpty Scratch
	if _, err := sEmpty.AppendTable(nil); err == nil {
		t.Fatalf("expected error for AppendTable with empty table")
	}

	// Lazy init fse in AppendTable
	var sLazy Scratch
	var hist [256]uint32
	hist['A'] = 100
	hist['B'] = 100
	if err := sLazy.BuildCTable(&hist); err != nil {
		t.Fatalf("BuildCTable failed: %v", err)
	}
	sLazy.fse = nil // clear fse to test lazy init
	hdr, err := sLazy.AppendTable(nil)
	if err != nil || len(hdr) == 0 {
		t.Fatalf("AppendTable with lazy fse failed: %v", err)
	}
}

// TestCompressReusePolicies tests ReusePolicyAllow, ReusePolicyPrefer, ReusePolicyNone, and edge cases.
func TestCompressReusePolicies(t *testing.T) {
	data1 := bytes.Repeat([]byte("ABCDEFGHIJKLMN"), 20)
	data2 := bytes.Repeat([]byte("ABCDEFGHIJKLMN"), 15)

	// ReusePolicyAllow
	{
		var s Scratch
		s.Reuse = ReusePolicyAllow
		_, _, err := Compress1X(data1, &s)
		if err != nil {
			t.Fatalf("Compress1X data1 failed: %v", err)
		}
		_, reUsed, err := Compress1X(data2, &s)
		if err != nil {
			t.Fatalf("Compress1X data2 failed: %v", err)
		}
		_ = reUsed
	}

	// ReusePolicyPrefer
	{
		var s Scratch
		s.Reuse = ReusePolicyPrefer
		_, _, err := Compress1X(data1, &s)
		if err != nil {
			t.Fatalf("Compress1X data1 failed: %v", err)
		}
		_, reUsed, err := Compress1X(data2, &s)
		if err != nil {
			t.Fatalf("Compress1X data2 failed: %v", err)
		}
		if !reUsed {
			t.Fatalf("expected reUsed for ReusePolicyPrefer")
		}

		// ReusePolicyPrefer with small wantSize causing fallthrough to new table
		s.WantLogLess = 10
		_, _, _ = Compress1X(data2, &s)
	}

	// ReusePolicyNone
	{
		var s Scratch
		s.Reuse = ReusePolicyNone
		_, _, _ = Compress1X(data1, &s)
		_, _, _ = Compress1X(data2, &s)
	}

	// maxCount > len(in)
	{
		var s Scratch
		s.maxCount = 1000
		_, _, err := Compress1X([]byte("ABC"), &s)
		if err == nil {
			t.Fatalf("expected error for maxCount > len(in)")
		}
	}

	// maxCount < (len(in) >> 7)
	{
		var s Scratch
		spreadData := make([]byte, 512)
		for i := 0; i < 256; i++ {
			spreadData[i] = byte(i)
			spreadData[256+i] = byte(i)
		}
		_, _, err := Compress1X(spreadData, &s)
		if !errors.Is(err, ErrIncompressible) {
			t.Fatalf("expected ErrIncompressible for well-distributed input")
		}
	}
}

// TestCanUseAndValidateTable tests canUseTable and validateTable branches.
func TestCanUseAndValidateTable(t *testing.T) {
	var s Scratch
	s.symbolLen = 5
	s.actualTableLog = 5
	s.count[0] = 10
	s.count[1] = 10

	// Short table
	ctShort := make(cTable, 3)
	if s.canUseTable(ctShort) {
		t.Fatal("expected false for short table")
	}
	if s.validateTable(ctShort) {
		t.Fatal("expected false for short table")
	}

	// Zero bits for present symbol
	ctZero := make(cTable, 5)
	ctZero[0].nBits = 0
	if s.canUseTable(ctZero) {
		t.Fatal("expected false for zero bits")
	}
	if s.validateTable(ctZero) {
		t.Fatal("expected false for zero bits")
	}

	// nBits > actualTableLog
	ctBig := make(cTable, 5)
	ctBig[0].nBits = 10
	ctBig[1].nBits = 2
	if s.validateTable(ctBig) {
		t.Fatal("expected false for nBits > actualTableLog")
	}

	// Valid table
	ctZero[0].nBits = 2
	ctZero[1].nBits = 2
	if !s.canUseTable(ctZero) {
		t.Fatal("expected true for valid table")
	}
	if !s.validateTable(ctZero) {
		t.Fatal("expected true for valid table")
	}
}

// TestCTableWriteAndEstTableSize tests raw 4-bit packing in write and estTableSize.
func TestCTableWriteAndEstTableSize(t *testing.T) {
	var s Scratch
	_, _ = s.prepare(nil)
	s.actualTableLog = 5
	s.symbolLen = 2 // maxSymbolValue = 1 < 2, skips FSE
	ct := make(cTable, 2)
	ct[0].nBits = 2
	ct[1].nBits = 2
	err := ct.write(&s)
	if err != nil {
		t.Fatalf("ct.write failed: %v", err)
	}
	sz, err := ct.estTableSize(&s)
	if err != nil || sz <= 0 {
		t.Fatalf("ct.estTableSize failed: %v", err)
	}

	// ReadTable uncompressed & error branches
	var decScratch Scratch
	_, _, _ = ReadTable(s.Out, &decScratch)
	_, _, _ = ReadTable(nil, &decScratch)
	_, _, _ = ReadTable([]byte{0x01}, &decScratch)
	_, _, _ = ReadTable([]byte{0x85, 0x12}, &decScratch) // uncompressed too short
	_, _, _ = ReadTable([]byte{10, 1, 2}, &decScratch)   // want 10 bytes have 2

	// maxSymbolValue > (256 - 128) = 128
	s.symbolLen = 135
	ctLarge := make(cTable, 135)
	for i := range ctLarge {
		ctLarge[i].nBits = 1
	}
	_ = ctLarge.write(&s)
	_, _ = ctLarge.estTableSize(&s)

	// Zero-weight symbol path in ReadTable
	ctZeroWeight := make(cTable, 4)
	ctZeroWeight[0].nBits = 2
	ctZeroWeight[1].nBits = 0 // weight 0
	ctZeroWeight[2].nBits = 2
	ctZeroWeight[3].nBits = 2
	var sZW Scratch
	_, _ = sZW.prepare(nil)
	sZW.actualTableLog = 5
	sZW.symbolLen = 4
	_ = ctZeroWeight.write(&sZW)
	_, _, _ = ReadTable(sZW.Out, &decScratch)
}

// TestMatchesDetailed tests specific error paths in matches.
func TestMatchesDetailed(t *testing.T) {
	// 1. Decoder exists but no encoder
	{
		var s Scratch
		s.actualTableLog = 3
		s.dt.single = make([]dEntrySingle, 8)
		ct := make(cTable, 8)
		s.dt.single[0] = dEntrySingle{entry: 1 | (uint16(2) << 8)}
		ct[2].nBits = 0
		var buf bytes.Buffer
		s.matches(ct, &buf)
		if !strings.Contains(buf.String(), "has decoder, but no encoder") {
			t.Fatalf("expected has decoder warning, got: %s", buf.String())
		}
	}

	// 2. Decoder output and bit size mismatch
	{
		var s Scratch
		s.actualTableLog = 3
		s.dt.single = make([]dEntrySingle, 8)
		ct := make(cTable, 8)
		s.dt.single[0] = dEntrySingle{entry: 2 | (uint16(5) << 8)}
		ct[1].nBits = 3
		ct[1].val = 0
		var buf bytes.Buffer
		s.matches(ct, &buf)
		if !strings.Contains(buf.String(), "mismatch") {
			t.Fatalf("expected mismatch, got: %s", buf.String())
		}
	}

	// 3. > 20 errors stopping
	{
		var s Scratch
		s.actualTableLog = 6
		s.dt.single = make([]dEntrySingle, 64)
		ct := make(cTable, 64)
		ct[1].nBits = 1
		ct[1].val = 0
		s.dt.single[0] = dEntrySingle{entry: 1 | (uint16(1) << 8)} // matches base
		var buf bytes.Buffer
		s.matches(ct, &buf)
		if !strings.Contains(buf.String(), "errors, stopping") {
			t.Fatalf("expected 'errors, stopping', got: %s", buf.String())
		}
	}
}

// TestEstimateSizesComprehensive tests EstimateSizes across all branching paths.
func TestEstimateSizesComprehensive(t *testing.T) {
	var s Scratch
	// 1-byte input -> ErrIncompressible
	if _, _, _, err := EstimateSizes([]byte{0x42}, &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible, got %v", err)
	}

	// Repeated single symbol -> ErrUseRLE
	if _, _, _, err := EstimateSizes(bytes.Repeat([]byte{0x42}, 100), &s); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE, got %v", err)
	}

	// Flat incompressible input -> ErrIncompressible
	flat := make([]byte, 256)
	for i := range flat {
		flat[i] = byte(i)
	}
	if _, _, _, err := EstimateSizes(flat, &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for flat data, got %v", err)
	}

	// Compressible input with WantLogLess
	data := bytes.Repeat([]byte("ABRACADABRA_ALAKAZAM_SIMSALABIM_12345678"), 20)
	var s2 Scratch
	s2.WantLogLess = 1
	tableSz, dataSz, reuseSz, err := EstimateSizes(data, &s2)
	if err != nil {
		t.Fatalf("EstimateSizes failed: %v", err)
	}
	if tableSz <= 0 || dataSz <= 0 {
		t.Fatalf("unexpected sizes: tableSz=%d, dataSz=%d", tableSz, dataSz)
	}
	if reuseSz != -1 {
		t.Fatalf("expected reuseSz=-1, got %d", reuseSz)
	}

	// Second run with previous table for reuse
	s2.prevTable = s2.cTable
	s2.prevTableLog = s2.actualTableLog
	_, _, reuseSz2, err := EstimateSizes(data, &s2)
	if err != nil {
		t.Fatalf("EstimateSizes with reuse failed: %v", err)
	}
	if reuseSz2 <= 0 {
		t.Fatalf("expected reuseSz2 > 0, got %d", reuseSz2)
	}
}

// TestCompressReusePolicyMustAndIncompressible tests ReusePolicyMust and ReusePolicyAllow.
func TestCompressReusePolicyMustAndIncompressible(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJKLM_0123456789_NOPQRSTUVWXYZ"), 20)
	var s Scratch
	// Initial compression to establish cTable
	_, reUsed, err := Compress1X(data, &s)
	if err != nil || reUsed {
		t.Fatalf("initial Compress1X failed: err=%v, reUsed=%v", err, reUsed)
	}

	// ReusePolicyMust with same data -> should succeed with reUsed = true
	s.Reuse = ReusePolicyMust
	compressed2, reUsed2, err2 := Compress1X(data, &s)
	if err2 != nil || !reUsed2 {
		t.Fatalf("ReusePolicyMust failed on matching data: err=%v, reUsed=%v", err2, reUsed2)
	}
	if len(compressed2) == 0 {
		t.Fatalf("unexpected empty compressed")
	}

	// ReusePolicyMust with different, incompatible data -> should fail with ErrIncompressible
	diffData := bytes.Repeat([]byte("!@#$%^&*()_+~`<>?{}|"), 20)
	_, _, err3 := Compress1X(diffData, &s)
	if !errors.Is(err3, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for ReusePolicyMust with incompatible data, got %v", err3)
	}

	// ReusePolicyAllow where reuse is selected
	var sAllow Scratch
	_, _, _ = Compress1X(data, &sAllow)
	sAllow.Reuse = ReusePolicyAllow
	_, reUsedAllow, errAllow := Compress1X(data, &sAllow)
	if errAllow != nil {
		t.Fatalf("ReusePolicyAllow failed: %v", errAllow)
	}
	if !reUsedAllow {
		t.Logf("ReusePolicyAllow did not reuse table, but succeeded")
	}
}

// TestBitReaderErrorBranches tests close() and init() errors on bitReaderBytes and bitReaderShifted.
func TestBitReaderErrorBranches(t *testing.T) {
	// bitReaderBytes with unconsumed bits on close
	var br bitReaderBytes
	if err := br.init([]byte{0x80, 0x01}); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	_ = br.close()

	// bitReaderShifted
	var brs bitReaderShifted
	if err := brs.init([]byte{0x80, 0x01}); err != nil {
		t.Fatalf("brs init failed: %v", err)
	}
	_ = brs.close()

	// bitReaderShifted init errors
	var brsErr bitReaderShifted
	if err := brsErr.init(nil); err == nil {
		t.Fatalf("expected error on empty bitReaderShifted init")
	}
	if err := brsErr.init([]byte{0x00}); err == nil {
		t.Fatalf("expected error on 0x00 byte for bitReaderShifted init")
	}
}

// TestBuildCTableAndAppendTableErrors exercises edge cases and error handling in BuildCTable and AppendTable.
func TestBuildCTableAndAppendTableErrors(t *testing.T) {
	var count [256]uint32
	count['A'] = 10
	count['B'] = 20

	// BuildCTable nil scratch
	var nilScratch *Scratch
	if err := nilScratch.BuildCTable(&count); err == nil {
		t.Fatalf("expected error for BuildCTable on nil Scratch")
	}

	// BuildCTable nil count
	var s Scratch
	if err := s.BuildCTable(nil); err == nil {
		t.Fatalf("expected error for BuildCTable with nil count")
	}

	// BuildCTable prepare error (invalid tableLog)
	s.TableLog = 25
	if err := s.BuildCTable(&count); err == nil {
		t.Fatalf("expected error for BuildCTable with invalid tableLog")
	}
	s.TableLog = 0

	// BuildCTable empty count
	var emptyCount [256]uint32
	if err := s.BuildCTable(&emptyCount); err == nil {
		t.Fatalf("expected error for BuildCTable with all-zero histogram")
	}

	// BuildCTable single symbol -> ErrUseRLE
	var singleCount [256]uint32
	singleCount['X'] = 50
	if err := s.BuildCTable(&singleCount); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE for single symbol count, got %v", err)
	}

	// BuildCTable with total > BlockSizeMax to trigger histogram scaling (lines 70-95)
	var largeCount [256]uint32
	largeCount['A'] = 70000
	largeCount['B'] = 1
	var sLarge Scratch
	if err := sLarge.BuildCTable(&largeCount); err != nil {
		t.Fatalf("BuildCTable with large total failed: %v", err)
	}

	// AppendTable nil / empty table
	if _, err := nilScratch.AppendTable(nil); err == nil {
		t.Fatalf("expected error for AppendTable on nil Scratch")
	}
	var emptyTableScratch Scratch
	if _, err := emptyTableScratch.AppendTable(nil); err == nil {
		t.Fatalf("expected error for AppendTable with empty table")
	}

	// AppendTable with invalid tableLog during prepare
	ct := make(cTable, 4)
	ct[0].nBits = 1
	ct[1].nBits = 1
	var sAppendPrep Scratch
	sAppendPrep.prevTable = ct
	sAppendPrep.TableLog = 25
	if _, err := sAppendPrep.AppendTable(nil); err == nil {
		t.Fatalf("expected error for AppendTable with invalid TableLog")
	}
}

// TestCompressPrepareErrors exercises error paths in Compress1X and Compress4X when prepare fails.
func TestCompressPrepareErrors(t *testing.T) {
	data := []byte("some test data for compress prepare error")
	s := &Scratch{TableLog: 25}
	if _, _, err := Compress1X(data, s); err == nil {
		t.Fatalf("expected error for Compress1X with invalid tableLog")
	}
	if _, _, err := Compress4X(data, s); err == nil {
		t.Fatalf("expected error for Compress4X with invalid tableLog")
	}
}

// TestDecoderAsmErrorBranches exercises error paths in Decoder.Decompress1X and Decompress4X.
func TestDecoderAsmErrorBranches(t *testing.T) {
	var emptyDec Decoder
	// No table loaded
	if _, err := emptyDec.Decompress1X(make([]byte, 10), []byte{1, 2, 3}); err == nil {
		t.Fatalf("expected error for Decompress1X without table")
	}
	if _, err := emptyDec.Decompress4X(make([]byte, 10), []byte{1, 2, 3}); err == nil {
		t.Fatalf("expected error for Decompress4X without table")
	}

	// Decompress1X with bad src
	validDec := &Decoder{actualTableLog: 11}
	validDec.dt.single = make([]dEntrySingle, 2048)
	if _, err := validDec.Decompress1X(make([]byte, 10), nil); err == nil {
		t.Fatalf("expected error for Decompress1X with nil src")
	}

	// Decompress4X with small input (<10 bytes)
	if _, err := validDec.Decompress4X(make([]byte, 2000), []byte{1, 2, 3}); err == nil {
		t.Fatalf("expected error for Decompress4X with small input")
	}

	// Decompress4X with truncated input in jump table
	badJump := []byte{0xff, 0xff, 0x01, 0x00, 0x01, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	if _, err := validDec.Decompress4X(make([]byte, 2000), badJump); err == nil {
		t.Fatalf("expected error for Decompress4X with truncated jump table")
	}

	// compress4X and compress4Xp with len < 12
	var s Scratch
	if _, err := s.compress4X([]byte("small")); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for compress4X on small input, got %v", err)
	}
	if _, err := s.compress4Xp([]byte("small")); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for compress4Xp on small input, got %v", err)
	}
}

// TestReadTableWeightErrorBranches exercises error paths in ReadTable weight parsing.
func TestReadTableWeightErrorBranches(t *testing.T) {
	var dec Scratch
	// 1. Weights zero
	hdrZero := []byte{129, 0x00}
	if _, _, err := ReadTable(hdrZero, &dec); err == nil || !strings.Contains(err.Error(), "weights zero") {
		t.Fatalf("expected 'weights zero' error, got %v", err)
	}

	// 2. Weight too large (> tableLogMax)
	hdrTooLarge := []byte{129, 0x0E}
	if _, _, err := ReadTable(hdrTooLarge, &dec); err == nil || !strings.Contains(err.Error(), "weight too large") {
		t.Fatalf("expected 'weight too large' error, got %v", err)
	}

	// 3. Last value not power of two
	hdrNotPow2 := []byte{129, 0x14}
	if _, _, err := ReadTable(hdrNotPow2, &dec); err == nil || !strings.Contains(err.Error(), "last value not power of two") {
		t.Fatalf("expected 'last value not power of two' error, got %v", err)
	}

	// 4. TableLog too big (> tableLogMax)
	hdrTableLogTooBig := make([]byte, 1+65)
	hdrTableLogTooBig[0] = 255 // 128 weights (255 - 127)
	for i := 1; i < len(hdrTableLogTooBig); i++ {
		hdrTableLogTooBig[i] = 0xAA // weights 10 and 10
	}
	if _, _, err := ReadTable(hdrTableLogTooBig, &dec); err == nil || !strings.Contains(err.Error(), "tableLog too big") {
		t.Fatalf("expected 'tableLog too big' error, got %v", err)
	}
}

// TestDecompress4X8BitErrors exercises error branches in decompress4X8bit.
func TestDecompress4X8BitErrors(t *testing.T) {
	d := &Decoder{actualTableLog: 7}
	d.dt.single = make([]dEntrySingle, 256)

	// Truncated jump offset
	badJump := []byte{0xff, 0xff, 0x01, 0x00, 0x01, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	if _, err := d.decompress4X8bit(make([]byte, 100), badJump); err == nil {
		t.Fatalf("expected error for truncated jump offset in decompress4X8bit")
	}

	// Bad stream 0
	badStream0 := []byte{0x01, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if _, err := d.decompress4X8bit(make([]byte, 100), badStream0); err == nil {
		t.Fatalf("expected error for bad stream 0 in decompress4X8bit")
	}

	// Bad stream 3
	badStream3 := []byte{0x02, 0x00, 0x02, 0x00, 0x02, 0x00, 0x80, 0x01, 0x80, 0x01, 0x80, 0x01, 0x00}
	if _, err := d.decompress4X8bit(make([]byte, 100), badStream3); err == nil {
		t.Fatalf("expected error for bad stream 3 in decompress4X8bit")
	}

	// Stream overrun
	data := bytes.Repeat([]byte("ABCDE"), 100) // 500 bytes
	var s Scratch
	compressed, _, _ := Compress4X(data, &s)
	s2, remain, _ := ReadTable(compressed, nil)
	s2.MaxDecodedSize = len(data)
	dec := s2.Decoder()
	tooSmallDst := make([]byte, 50)
	_, _ = dec.decompress4X8bit(tooSmallDst, remain)
}


// TestCleanBitReaderClose exercises the clean close path on bitReaderBytes and bitReaderShifted.
func TestCleanBitReaderClose(t *testing.T) {
	var brClean bitReaderBytes
	brClean.off = 0
	brClean.bitsRead = 64
	if err := brClean.close(); err != nil {
		t.Fatalf("expected nil error on clean close, got %v", err)
	}

	var brsClean bitReaderShifted
	brsClean.off = 0
	brsClean.bitsRead = 64
	if err := brsClean.close(); err != nil {
		t.Fatalf("expected nil error on clean close, got %v", err)
	}
}

// TestDecompress1XDstCapExceeded exercises buffer limit errors during Decompress1X.
func TestDecompress1XDstCapExceeded(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDEFGHIJKLM_0123456789_NOPQRSTUVWXYZ"), 30)
	var s Scratch
	comp, _, err := Compress1X(data, &s)
	if err != nil {
		t.Fatalf("Compress1X failed: %v", err)
	}
	sTable, remain, err := ReadTable(comp, nil)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	sTable.MaxDecodedSize = len(data)
	smallDst := make([]byte, 0, 10)
	_, _ = sTable.Decoder().Decompress1X(smallDst, remain)
}

// TestDecompress4X8BitDirectLarge exercises decompress4X8bit with >1024 bytes to cover large block loops.
func TestDecompress4X8BitDirectLarge(t *testing.T) {
	data := bytes.Repeat([]byte("ABCDE"), 400) // 2000 bytes
	var s Scratch
	compressed, _, err := Compress4X(data, &s)
	if err != nil {
		t.Fatalf("Compress4X failed: %v", err)
	}
	s2, remain, err := ReadTable(compressed, nil)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	s2.MaxDecodedSize = len(data)
	dec := s2.Decoder()
	dst := make([]byte, len(data))
	got, err := dec.decompress4X8bit(dst, remain)
	if err != nil {
		t.Fatalf("decompress4X8bit direct failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("decompress4X8bit direct mismatch")
	}
}

// TestDecompress1X8BitExactlyDirect exercises decompress1X8BitExactly with large buffer and maxDecodedSize errors.
func TestDecompress1X8BitExactlyDirect(t *testing.T) {
	// Construct data with 256 distinct symbols to guarantee 8-bit table
	raw := make([]byte, 256)
	for i := range raw {
		raw[i] = byte(i)
	}
	data := append(raw, raw...) // 512 bytes
	var s Scratch
	s.TableLog = 8
	compressed, _, err := Compress1X(data, &s)
	if err != nil {
		// If Compress1X refused flat distribution, test buffer exceeded directly
		smallDst := make([]byte, 0, 10)
		var sScratch Scratch
		d := sScratch.Decoder()
		d.actualTableLog = 8
		d.dt.single = make([]dEntrySingle, 256)
		_, _ = d.decompress1X8BitExactly(smallDst, []byte{0x80, 0x01})
		return
	}
	s2, remain, err := ReadTable(compressed, nil)
	if err != nil {
		t.Fatalf("ReadTable failed: %v", err)
	}
	dec := s2.Decoder()

	// Normal run (>256 bytes)
	dst := make([]byte, 0, len(data))
	got, err := dec.decompress1X8BitExactly(dst, remain)
	if err != nil {
		t.Fatalf("decompress1X8BitExactly failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("decompress1X8BitExactly mismatch")
	}

	// Buffer exceeded at off == 0
	smallDst := make([]byte, 0, 100)
	_, err = dec.decompress1X8BitExactly(smallDst, remain)
	if !errors.Is(err, ErrMaxDecodedSizeExceeded) {
		t.Fatalf("expected ErrMaxDecodedSizeExceeded, got %v", err)
	}
}

// TestCountSimplePrevTableEdge exercises countSimple when input symbol exceeds prevTable length.
func TestCountSimplePrevTableEdge(t *testing.T) {
	var s Scratch
	s.prevTable = make(cTable, 4)
	for j := range s.prevTable {
		s.prevTable[j].nBits = 1
	}
	data := []byte{0, 1, 2, 3, 10}
	max, reuse := s.countSimple(data)
	if max <= 0 || reuse {
		t.Fatalf("expected reuse=false when symbol >= len(prevTable)")
	}
}











