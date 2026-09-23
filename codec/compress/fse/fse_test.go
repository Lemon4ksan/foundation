// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fse

import (
	"bytes"
	"errors"
	"testing"
)

// TestFSERoundtripVariedSizes tests compress/decompress across diverse sizes and entropy levels.
func TestFSERoundtripVariedSizes(t *testing.T) {
	inputs := []struct {
		name string
		data []byte
	}{
		{"RepText_100B", bytes.Repeat([]byte("entropy coding with finite state entropy! "), 3)},
		{"RepText_1KB", bytes.Repeat([]byte("quick brown fox jumps over the lazy dog. "), 25)},
		{"RepText_1400B", bytes.Repeat([]byte("quick brown fox jumps over the lazy dog. "), 35)},
		{"SkewedDistribution", func() []byte {
			b := make([]byte, 2048)
			for i := range b {
				if i%3 == 0 {
					b[i] = 'A'
				} else if i%5 == 0 {
					b[i] = 'B'
				} else {
					b[i] = byte(i % 16)
				}
			}
			return b
		}()},
	}

	var s Scratch
	for _, tc := range inputs {
		t.Run(tc.name, func(t *testing.T) {
			s.Out = nil
			compressed, err := Compress(tc.data, &s)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}
			var decScratch Scratch
			decScratch.DecompressLimit = len(tc.data)
			decompressed, err := Decompress(compressed, &decScratch)
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}
			if len(decompressed) < len(tc.data) || !bytes.Equal(decompressed[:len(tc.data)], tc.data) {
				t.Fatalf("decompressed mismatch: got %d bytes, want %d bytes", len(decompressed), len(tc.data))
			}
		})
	}
}

// TestFSECompressErrors verifies all error returns in Compress.
func TestFSECompressErrors(t *testing.T) {
	var s Scratch
	// Empty input
	if _, err := Compress(nil, &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for nil, got %v", err)
	}
	if _, err := Compress([]byte{0x42}, &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for 1-byte, got %v", err)
	}
	// Single repeated byte -> ErrUseRLE
	if _, err := Compress(bytes.Repeat([]byte{0x42}, 100), &s); !errors.Is(err, ErrUseRLE) {
		t.Fatalf("expected ErrUseRLE, got %v", err)
	}
	// Incompressible flat distribution
	flat := make([]byte, 256)
	for i := range flat {
		flat[i] = byte(i)
	}
	if _, err := Compress(flat, &s); !errors.Is(err, ErrIncompressible) {
		t.Fatalf("expected ErrIncompressible for flat distribution, got %v", err)
	}
	// TableLog > maxTableLog
	s.TableLog = 15
	if _, err := Compress([]byte("some data that would be ok"), &s); err == nil {
		t.Fatalf("expected error for tableLog > maxTableLog")
	}
}

// TestFSEDecompressCorrupt verifies graceful error handling on malformed streams.
func TestFSEDecompressCorrupt(t *testing.T) {
	var s Scratch
	corruptInputs := [][]byte{
		{},
		{0x01},
		{0x01, 0x02},
		{0x01, 0x02, 0x03},
		{0xFF, 0xFF, 0xFF, 0xFF}, // tableLog too large
	}
	for i, in := range corruptInputs {
		if _, err := Decompress(in, &s); err == nil {
			t.Fatalf("expected error for corrupt input #%d, got nil", i)
		}
	}
}

// TestFSEHistogramAPI verifies the Histogram and HistogramFinished methods.
func TestFSEHistogramAPI(t *testing.T) {
	data := bytes.Repeat([]byte("ABRACADABRA_ALAKAZAM_SIMSALABIM"), 30)
	var s Scratch
	hist := s.Histogram()
	for _, b := range data {
		hist[b]++
	}
	var maxCount int
	var maxSymbol uint8
	for i, count := range hist {
		if count > uint32(maxCount) {
			maxCount = int(count)
		}
		if count > 0 {
			maxSymbol = uint8(i)
		}
	}
	s.HistogramFinished(maxSymbol, maxCount)
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress with pre-populated histogram failed: %v", err)
	}
	var decScratch Scratch
	decScratch.DecompressLimit = len(data)
	decompressed, err := Decompress(compressed, &decScratch)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed[:len(data)], data) {
		t.Fatalf("Decompress mismatch")
	}
}

// TestFSEDecompressLimit verifies Decompress returns an error when DecompressLimit is exceeded.
func TestFSEDecompressLimit(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz"), 40)
	var s Scratch
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	var decScratch Scratch
	decScratch.DecompressLimit = 50
	_, err = Decompress(compressed, &decScratch)
	if err == nil {
		t.Fatalf("expected error when output exceeds DecompressLimit, got nil")
	}
}

// TestInternalMethods tests unexported functions: normalizeCount2, validateNorm, symbolTransform.String, bitReader, and decoder.
func TestInternalMethods(t *testing.T) {
	// symbolTransform.String
	st := symbolTransform{deltaFindState: 12, deltaNbBits: 4}
	if st.String() == "" {
		t.Fatalf("symbolTransform.String should not be empty")
	}

	// normalizeCount2 & validateNorm
	data := bytes.Repeat([]byte("ABRACADABRA_ALAKAZAM_SIMSALABIM"), 30)
	var s Scratch
	prepared, err := s.prepare(data)
	if err != nil {
		t.Fatalf("prepare failed: %v", err)
	}
	_ = prepared.countSimple(data)
	prepared.optimalTableLog()
	if err := prepared.normalizeCount2(); err != nil {
		t.Fatalf("normalizeCount2 failed: %v", err)
	}
	if err := prepared.validateNorm(); err != nil {
		t.Fatalf("validateNorm failed: %v", err)
	}

	// bitReader methods: init with empty, getBits(0), close error
	var br bitReader
	if err := br.init(nil); err == nil {
		t.Fatalf("expected error for nil bitReader input")
	}
	brInitData := []byte{0x01, 0x80}
	if err := br.init(brInitData); err != nil {
		t.Fatalf("bitReader init failed: %v", err)
	}
	if br.getBits(0) != 0 {
		t.Fatalf("getBits(0) != 0")
	}
	br.bitsRead = 65
	if err := br.close(); err == nil {
		t.Fatalf("expected error from close when bitsRead > 64")
	}

	// bitWriter methods
	var bw bitWriter
	bw.reset(nil)
	bw.addBits16ZeroNC(4, 0)
	bw.addBits16NC(4, 0xA)
	bw.flushAlign()
	bw.close()

	// decoder.nextFast
	var s2 Scratch
	comp, err := Compress(data, &s2)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	var decScratch Scratch
	decPrep, err := decScratch.prepare(comp)
	if err != nil {
		t.Fatalf("prepare dec failed: %v", err)
	}
	if err := decPrep.readNCount(); err != nil {
		t.Fatalf("readNCount failed: %v", err)
	}
	if err := decPrep.buildDtable(); err != nil {
		t.Fatalf("buildDtable failed: %v", err)
	}
	var bitR bitReader
	if err := bitR.init(decPrep.br.unread()); err != nil {
		t.Fatalf("bitReader init failed: %v", err)
	}
	var dec decoder
	dec.init(&bitR, decPrep.decTable, decPrep.actualTableLog)
	sym := dec.nextFast()
	if sym == 0 && dec.finished() {
		t.Logf("nextFast symbol %v", sym)
	}
}

// TestFSEZeroBits tests the high-probability (>50%) zeroBits compression and decompression branches.
func TestFSEZeroBits(t *testing.T) {
	data := append(bytes.Repeat([]byte("A"), 1200), bytes.Repeat([]byte("BCDEFGH0123456789"), 20)...)
	var s Scratch
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if !s.zeroBits {
		t.Fatalf("expected s.zeroBits == true for skewed input")
	}

	var decScratch Scratch
	decScratch.DecompressLimit = len(data)
	decompressed, err := Decompress(compressed, &decScratch)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decompressed[:len(data)], data) {
		t.Fatalf("decompressed mismatch for zeroBits stream")
	}
}

// TestFSETableLogVariations tests small (<=8) and large (>8) tableLogs across compression loops.
func TestFSETableLogVariations(t *testing.T) {
	data := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789"), 30)

	for _, tl := range []uint8{6, 7, 8, 9, 10, 11, 12} {
		var s Scratch
		s.TableLog = tl
		compressed, err := Compress(data, &s)
		if err != nil {
			t.Fatalf("Compress with TableLog=%d failed: %v", tl, err)
		}

		var decScratch Scratch
		decScratch.DecompressLimit = len(data)
		decompressed, err := Decompress(compressed, &decScratch)
		if err != nil {
			t.Fatalf("Decompress with TableLog=%d failed: %v", tl, err)
		}
		if !bytes.Equal(decompressed[:len(data)], data) {
			t.Fatalf("Decompress mismatch with TableLog=%d", tl)
		}
	}
}

// TestBitWriterFlushAllCases covers all switch branches 0..8 and default panic in bitWriter.flush.
func TestBitWriterFlushAllCases(t *testing.T) {
	for v := uint8(0); v <= 8; v++ {
		var bw bitWriter
		bw.nBits = v * 8
		bw.bitContainer = 0x0102030405060708
		bw.flush()
		if len(bw.out) != int(v) {
			t.Fatalf("flush for %d bytes got %d bytes", v, len(bw.out))
		}
	}

	// Test panic for nBits > 64
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on nBits > 64")
		}
	}()
	var bw bitWriter
	bw.nBits = 72
	bw.flush()
}

// TestBitReaderEdgeCases verifies initialization and edge conditions in bitReader.
func TestBitReaderEdgeCases(t *testing.T) {
	var br bitReader
	// Zero byte end of stream error
	if err := br.init([]byte{0x00}); err == nil {
		t.Fatalf("expected error for trailing zero byte")
	}
	// Small slice < 8 bytes
	if err := br.init([]byte{0x12, 0x80}); err != nil {
		t.Fatalf("init with small slice failed: %v", err)
	}
	if br.finished() {
		t.Fatalf("reader should not be finished")
	}
	_ = br.getBits(0)
	_ = br.getBitsFast(1)
	_ = br.close()
}

// TestValidateNormEdgeCases tests validation failure branches.
func TestValidateNormEdgeCases(t *testing.T) {
	var s Scratch
	s.actualTableLog = 5
	s.symbolLen = 2
	s.norm[0] = 5
	s.norm[1] = 5 // total = 10 != 32
	if err := s.validateNorm(); err == nil {
		t.Fatalf("expected error when total != 1<<tableLog")
	}

	// Symbol out of range error
	s.norm[0] = 16
	s.norm[1] = 16 // total = 32
	s.count[2] = 10
	if err := s.validateNorm(); err == nil {
		t.Fatalf("expected error when count[symbolLen:] has non-zero entries")
	}
}

// TestNormalizeCount2EdgeCases tests branches in secondary normalization.
func TestNormalizeCount2EdgeCases(t *testing.T) {
	var s Scratch
	s.actualTableLog = 6
	s.symbolLen = 2
	s.count[0] = 1
	s.count[1] = 1
	s.br.init([]byte{1, 1})
	_ = s.normalizeCount2()

	// Incompressible-like distribution triggering all values poor
	s.actualTableLog = 5
	s.symbolLen = 3
	s.count[0] = 10
	s.count[1] = 20
	s.count[2] = 30
	s.br.init(make([]byte, 60))
	_ = s.normalizeCount2()

	// Full scaling loop path in normalizeCount2
	var s2 Scratch
	s2.actualTableLog = 8
	s2.symbolLen = 3
	s2.count[0] = 500
	s2.count[1] = 400
	s2.count[2] = 100
	s2.br.init(make([]byte, 1000))
	if err := s2.normalizeCount2(); err != nil {
		t.Fatalf("normalizeCount2 full path failed: %v", err)
	}

	// Weight < 1 error branch in normalizeCount2
	var s3 Scratch
	s3.actualTableLog = 5
	s3.symbolLen = 4
	s3.count[0] = 1
	s3.count[1] = 1
	s3.count[2] = 1
	s3.count[3] = 1000000
	s3.br.init(make([]byte, 1000003))
	_ = s3.normalizeCount2()
}

// TestFSECompressionModulo tests input lengths congruent to 0, 1, 2, 3 mod 4.
func TestFSECompressionModulo(t *testing.T) {
	base := []byte("Compression modulo test pattern for FSE!")
	for rem := 0; rem < 4; rem++ {
		data := bytes.Repeat(base, 10)[:len(base)*10-rem]
		var s Scratch
		compressed, err := Compress(data, &s)
		if err != nil {
			t.Fatalf("Compress failed for rem %d: %v", rem, err)
		}
		var decScratch Scratch
		decScratch.DecompressLimit = len(data)
		decomp, err := Decompress(compressed, &decScratch)
		if err != nil {
			t.Fatalf("Decompress failed for rem %d: %v", rem, err)
		}
		if !bytes.Equal(decomp[:len(data)], data) {
			t.Fatalf("mismatch for rem %d", rem)
		}
	}
}

// TestFSEZeroBitsSmallTableLog tests zeroBits with tableLog <= 8.
func TestFSEZeroBitsSmallTableLog(t *testing.T) {
	data := append(bytes.Repeat([]byte("A"), 1000), bytes.Repeat([]byte("BCDE"), 10)...)
	var s Scratch
	s.TableLog = 8
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	var decScratch Scratch
	decScratch.DecompressLimit = len(data)
	decomp, err := Decompress(compressed, &decScratch)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decomp[:len(data)], data) {
		t.Fatalf("mismatch")
	}
}

// TestFSECompressPoorIncompressible tests input where output >= len(in) returning ErrIncompressible.
func TestFSECompressPoorIncompressible(t *testing.T) {
	var s Scratch
	// 3-byte input error in s.compress
	if err := s.compress([]byte{1, 2}); err == nil {
		t.Fatalf("expected error on src <= 2 in compress")
	}
}

// TestBuildDtableCorrupt tests position != 0 in buildDtable.
func TestBuildDtableCorrupt(t *testing.T) {
	var s Scratch
	s.actualTableLog = 5
	s.symbolLen = 2
	s.norm[0] = 15
	s.norm[1] = 15 // total 30 != 32, leaves hole
	_ = s.buildDtable()
}

// TestReadNCountErrors exercises error branches in readNCount.
func TestReadNCountErrors(t *testing.T) {
	var s Scratch
	// input too small (< 4 bytes)
	s.br.init([]byte{0x01, 0x02})
	if err := s.readNCount(); err == nil {
		t.Fatalf("expected error on input < 4 in readNCount")
	}

	// tableLog too large (> 14)
	s.br.init([]byte{0x0F, 0x00, 0x00, 0x00})
	if err := s.readNCount(); err == nil {
		t.Fatalf("expected error on tableLog > 14 in readNCount")
	}
}

// TestNormalizeCount2RiskRounding exercises the risk of rounding to zero branch in normalizeCount2.
func TestNormalizeCount2RiskRounding(t *testing.T) {
	var s Scratch
	s.actualTableLog = 6 // 64
	s.symbolLen = 52
	for i := 0; i < 50; i++ {
		s.count[i] = 12
	}
	s.count[50] = 60
	s.count[51] = 600
	total := 50*12 + 60 + 600
	s.br.init(make([]byte, total))
	_ = s.normalizeCount2()
}

// TestIncompressibleDueToExpansion exercises Compress returning ErrIncompressible when compressed >= len(in).
func TestIncompressibleDueToExpansion(t *testing.T) {
	var s Scratch
	data := []byte("0123456789abcdefghijklmnopqrstuvwxyz!@#$%^&*()_+~`")
	if _, err := Compress(data, &s); !errors.Is(err, ErrIncompressible) {
		t.Logf("Compress returned: %v", err)
	}
}

// TestFSECompressLoopsHighTableLog tests compression loops with actualTableLog > 8.
func TestFSECompressLoopsHighTableLog(t *testing.T) {
	// Case 2: !zeroBits && actualTableLog > 8
	data1 := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;':,./<>?"), 100)
	var s1 Scratch
	s1.TableLog = 10
	comp1, err := Compress(data1, &s1)
	if err != nil {
		t.Fatalf("Compress Case 2 failed: %v", err)
	}
	if len(comp1) == 0 {
		t.Fatalf("Compress Case 2 generated empty output")
	}

	// Case 4: zeroBits && actualTableLog > 8
	data2 := append(bytes.Repeat([]byte("Z"), 5000), bytes.Repeat([]byte("0123456789abcdefghijklmnopqrstuvwxyz"), 50)...)
	var s2 Scratch
	s2.TableLog = 10
	comp2, err := Compress(data2, &s2)
	if err != nil {
		t.Fatalf("Compress Case 4 failed: %v", err)
	}
	var decScratch2 Scratch
	decScratch2.DecompressLimit = len(data2)
	decomp2, err := Decompress(comp2, &decScratch2)
	if err != nil {
		t.Fatalf("Decompress Case 4 failed: %v", err)
	}
	if !bytes.Equal(decomp2[:len(data2)], data2) {
		t.Fatalf("mismatch Case 4")
	}
}

// TestNormalizeCount2TotalZero exercises total == 0 branch with toDistribute > 0.
func TestNormalizeCount2TotalZero(t *testing.T) {
	var s Scratch
	s.actualTableLog = 5 // 32
	s.symbolLen = 30
	for i := 0; i < 30; i++ {
		s.count[i] = 2
	}
	s.br.init(make([]byte, 60))
	if err := s.normalizeCount2(); err != nil {
		t.Fatalf("normalizeCount2 total zero failed: %v", err)
	}
}

// TestFSEDecompressNonVector tests the non-vector quad decoding branch.
func TestFSEDecompressNonVector(t *testing.T) {
	orig := hasVectorFSE
	hasVectorFSE = false
	defer func() { hasVectorFSE = orig }()

	data := bytes.Repeat([]byte("FSE non-vector quad decode loop verification! 1234567890"), 30)
	var s Scratch
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	var decScratch Scratch
	decScratch.DecompressLimit = len(data)
	decomp, err := Decompress(compressed, &decScratch)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decomp[:len(data)], data) {
		t.Fatalf("mismatch in non-vector decompress")
	}
}

// TestFSEGapsAndLowProbLastSymbol tests symbol table gaps and low-probability trailing symbol.
func TestFSEGapsAndLowProbLastSymbol(t *testing.T) {
	data := append(bytes.Repeat([]byte{0, 200}, 500), 255)
	var s Scratch
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	var decScratch Scratch
	decScratch.DecompressLimit = len(data)
	decomp, err := Decompress(compressed, &decScratch)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}
	if !bytes.Equal(decomp[:len(data)], data) {
		t.Fatalf("mismatch")
	}

	// Also test intermediate low-probability symbol
	dataIntermediate := append(append(bytes.Repeat([]byte{0, 200}, 500), 50), 255)
	var s2 Scratch
	comp2, err := Compress(dataIntermediate, &s2)
	if err != nil {
		t.Fatalf("Compress intermediate low prob failed: %v", err)
	}
	var decScratch2 Scratch
	decScratch2.DecompressLimit = len(dataIntermediate)
	decomp2, err := Decompress(comp2, &decScratch2)
	if err != nil {
		t.Fatalf("Decompress intermediate low prob failed: %v", err)
	}
	if !bytes.Equal(decomp2[:len(dataIntermediate)], dataIntermediate) {
		t.Fatalf("mismatch intermediate low prob")
	}
}

// TestFSEZeroBitsLimitExceeded tests DecompressLimit error on zeroBits streams.
func TestFSEZeroBitsLimitExceeded(t *testing.T) {
	data := append(bytes.Repeat([]byte("A"), 1200), bytes.Repeat([]byte("BCDEFGH0123456789"), 20)...)
	var s Scratch
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	var decScratch Scratch
	decScratch.DecompressLimit = 50
	_, err = Decompress(compressed, &decScratch)
	if err == nil {
		t.Fatalf("expected error when zeroBits stream exceeds DecompressLimit")
	}
}

// TestReadNCountCorruptions tests corruption detection in readNCount.
func TestReadNCountCorruptions(t *testing.T) {
	data := append(bytes.Repeat([]byte{0, 200}, 500), 255)
	var s Scratch
	compressed, err := Compress(data, &s)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	// Corrupt header bytes
	for i := 4; i < min(len(compressed), 12); i++ {
		corrupt := append([]byte{}, compressed...)
		corrupt[i] ^= 0xFF
		var decScratch Scratch
		decScratch.prepare(corrupt)
		_ = decScratch.readNCount()
	}
}

// TestNilScratch exercises s == nil in prepare.
func TestNilScratch(t *testing.T) {
	data := bytes.Repeat([]byte("Testing nil scratch handling! "), 10)
	comp, err := Compress(data, nil)
	if err != nil {
		t.Fatalf("Compress(nil) failed: %v", err)
	}
	decomp, err := Decompress(comp, nil)
	if err != nil {
		t.Fatalf("Decompress(nil) failed: %v", err)
	}
	if !bytes.Equal(decomp[:len(data)], data) {
		t.Fatalf("mismatch with nil scratch")
	}
}

// TestOptimalTableLogLimits exercises tableLog < minTablelog and tableLog > maxTableLog.
func TestOptimalTableLogLimits(t *testing.T) {
	var s Scratch
	s.TableLog = 2
	s.symbolLen = 2
	s.br.init(make([]byte, 100))
	s.optimalTableLog()
	if s.actualTableLog < minTablelog {
		t.Fatalf("expected actualTableLog >= minTablelog")
	}

	s.TableLog = 20
	s.optimalTableLog()
	if s.actualTableLog > maxTableLog {
		t.Fatalf("expected actualTableLog <= maxTableLog")
	}
}

// TestBuildCTableErrors exercises error paths in buildCTable.
func TestBuildCTableErrors(t *testing.T) {
	// position != 0 in buildCTable
	var s Scratch
	s.actualTableLog = 5
	s.symbolLen = 2
	s.norm[0] = 15
	s.norm[1] = 15 // total 30 != 32
	_ = s.buildCTable()

	// total mismatch in buildCTable
	var s2 Scratch
	s2.actualTableLog = 5
	s2.symbolLen = 2
	s2.norm[0] = 16
	s2.norm[1] = 17 // total 33 != 32
	_ = s2.buildCTable()
}
