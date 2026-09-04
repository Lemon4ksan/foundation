// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compress provides a trimmed, zero-allocation multi-algorithm compression and decompression engine
// supporting RFC 1952 (gzip), RFC 7932 (brotli), RFC 8878 (zstd), and RFC 1951 (deflate).
package compress

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/lemon4ksan/foundation/borrow"
	"github.com/lemon4ksan/foundation/codec/compress/brotli"
	"github.com/lemon4ksan/foundation/codec/compress/flate"
	"github.com/lemon4ksan/foundation/codec/compress/gzip"
	"github.com/lemon4ksan/foundation/codec/compress/lz4"
	"github.com/lemon4ksan/foundation/codec/compress/lzma"
	"github.com/lemon4ksan/foundation/codec/compress/zstd"
	"github.com/lemon4ksan/foundation/silicon/pool"
)

const (
	// DefaultMaxDecompressedSize is the default maximum permitted output size (100 MB).
	DefaultMaxDecompressedSize = 100 * 1024 * 1024

	// MaxAmplificationRatio defines the maximum decompression expansion ratio (250x).
	MaxAmplificationRatio = 250
)

var (
	// ErrUnsupportedEncoding is returned when a Content-Encoding algorithm is unknown.
	ErrUnsupportedEncoding = errors.New("codec: unsupported content encoding")

	// ErrDecompressionFailed is returned when decompression payload is malformed.
	ErrDecompressionFailed = errors.New("codec: decompression failed")

	// ErrDecompressionBomb is returned when decompressed payload exceeds size/amplification limits.
	ErrDecompressionBomb = errors.New(
		"codec: maximum decompressed output limit exceeded (decompression bomb detected)",
	)
)

var (
	zstdDecoderStorage = pool.NewPerPStorage(func() *zstd.Decoder {
		dec, _ := zstd.NewReader(nil, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
		return dec
	})

	gzipReaderStorage = pool.NewPerPStorage(func() *gzip.Reader {
		return new(gzip.Reader)
	})

	flateReaderStorage = pool.NewPerPStorage(func() flate.Resetter {
		return flate.NewReader(nil).(flate.Resetter)
	})

	bytesReaderStorage = pool.NewPerPStorage(func() *bytes.Reader {
		return bytes.NewReader(nil)
	})

	byteBufferStorage = pool.NewPerPStorage(func() *bytes.Buffer {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	})

	gzipWriterStorage = pool.NewPerPStorage(func() *gzip.Writer {
		return gzip.NewWriter(io.Discard)
	})

	flateWriterStorage = pool.NewPerPStorage(func() *flate.Writer {
		w, _ := flate.NewWriter(io.Discard, flate.DefaultCompression)
		return w
	})

	brotliReaderStorage = pool.NewPerPStorage(func() *brotli.Reader {
		return brotli.NewReader(bytes.NewReader(nil))
	})

	brotliWriterStorage = pool.NewPerPStorage(func() *brotli.Writer {
		return brotli.NewWriterLevel(io.Discard, 6)
	})

	lz4ReaderStorage = pool.NewPerPStorage(func() *lz4.Reader {
		return lz4.NewReader(bytes.NewReader(nil))
	})

	lz4WriterStorage = pool.NewPerPStorage(func() *lz4.Writer {
		return lz4.NewWriter(io.Discard)
	})
)

// Decompress decodes compressed src into dst using the specified Content-Encoding algorithm.
// Supports "gzip", "br", "zstd", and "deflate".
func Decompress(encoding string, src, dst []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	// Fast-path for canonical lowercase encodings (zero allocation)
	switch encoding {
	case "gzip", "x-gzip":
		return Gunzip(src, dst)
	case "br":
		return Unbrotli(src, dst)
	case "zstd":
		return Unzstd(src, dst)
	case "deflate":
		return Inflate(src, dst)
	case "lzma2", "7z":
		return Unlzma2(src, dst)
	case "lz4":
		return Unlz4(src, dst)
	case "identity", "":
		if cap(dst) < len(src) {
			dst = make([]byte, len(src))
		} else {
			dst = dst[:len(src)]
		}
		copy(dst, src)
		return dst, nil
	}

	// Fallback for mixed-case or padded strings
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip", "x-gzip":
		return Gunzip(src, dst)
	case "br":
		return Unbrotli(src, dst)
	case "zstd":
		return Unzstd(src, dst)
	case "deflate":
		return Inflate(src, dst)
	case "lzma2", "7z":
		return Unlzma2(src, dst)
	case "lz4":
		return Unlz4(src, dst)
	case "identity", "":
		if cap(dst) < len(src) {
			dst = make([]byte, len(src))
		} else {
			dst = dst[:len(src)]
		}
		copy(dst, src)
		return dst, nil
	default:
		return nil, ErrUnsupportedEncoding
	}
}

// DecompressWithDict decodes compressed src into dst using the specified Content-Encoding algorithm and dictionary.
func DecompressWithDict(encoding string, src, dst, dictBytes []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	switch encoding {
	case "dcz":
		return UnzstdDCZ(src, dst, dictBytes)
	case "dcb":
		return UnbrotliDCB(src, dst, dictBytes)
	}

	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "dcz":
		return UnzstdDCZ(src, dst, dictBytes)
	case "dcb":
		return UnbrotliDCB(src, dst, dictBytes)
	default:
		return Decompress(encoding, src, dst)
	}
}

// DecompressScopedWithDict decodes compressed src directly into a zero-allocation scoped buffer with dictionary support.
func DecompressScopedWithDict(s *borrow.Scope, encoding string, src, dictBytes []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	switch encoding {
	case "dcz":
		return UnzstdDCZScoped(s, src, dictBytes)
	case "dcb":
		return UnbrotliDCBScoped(s, src, dictBytes)
	}

	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "dcz":
		return UnzstdDCZScoped(s, src, dictBytes)
	case "dcb":
		return UnbrotliDCBScoped(s, src, dictBytes)
	default:
		return DecompressScoped(s, encoding, src)
	}
}

// DecompressScoped decodes compressed src directly into a zero-allocation scoped buffer bound to s.
func DecompressScoped(s *borrow.Scope, encoding string, src []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	switch encoding {
	case "gzip", "x-gzip":
		return GunzipScoped(s, src)
	case "br":
		return UnbrotliScoped(s, src)
	case "zstd":
		return UnzstdScoped(s, src)
	case "deflate":
		return InflateScoped(s, src)
	case "lzma2", "7z":
		return Unlzma2Scoped(s, src)
	case "lz4":
		return Unlz4Scoped(s, src)
	case "identity", "":
		if s == nil {
			return borrow.NewBytes(src, nil), nil
		}
		b := s.AllocBytes(len(src))
		copy(b.AsSlice(), src)
		return b, nil
	}

	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip", "x-gzip":
		return GunzipScoped(s, src)
	case "br":
		return UnbrotliScoped(s, src)
	case "zstd":
		return UnzstdScoped(s, src)
	case "deflate":
		return InflateScoped(s, src)
	case "lzma2", "7z":
		return Unlzma2Scoped(s, src)
	case "lz4":
		return Unlz4Scoped(s, src)
	case "identity", "":
		if s == nil {
			return borrow.NewBytes(src, nil), nil
		}
		b := s.AllocBytes(len(src))
		copy(b.AsSlice(), src)
		return b, nil
	default:
		return borrow.Bytes{}, ErrUnsupportedEncoding
	}
}

// Compress encodes src into compressed dst using the specified algorithm and level.
func Compress(encoding string, src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}

	lvl := 6
	if len(level) > 0 {
		lvl = level[0]
	}

	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip", "x-gzip":
		return CompressGzip(src, dst, lvl)
	case "deflate":
		return CompressDeflate(src, dst, lvl)
	case "br":
		return CompressBrotli(src, dst, lvl)
	case "zstd":
		return CompressZstd(src, dst, lvl)
	case "lzma2", "7z":
		return CompressLZMA2(src, dst, lvl)
	case "lz4":
		return CompressLZ4(src, dst, lvl)
	case "identity", "":
		if cap(dst) < len(src) {
			dst = make([]byte, len(src))
		} else {
			dst = dst[:len(src)]
		}
		copy(dst, src)
		return dst, nil
	default:
		return nil, ErrUnsupportedEncoding
	}
}

// CompressScoped encodes src into a zero-allocation scoped buffer bound to s.
func CompressScoped(s *borrow.Scope, encoding string, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip", "x-gzip":
		return GzipScoped(s, src, level...)
	case "deflate":
		return DeflateScoped(s, src, level...)
	case "br":
		return BrotliScoped(s, src, level...)
	case "zstd":
		return ZstdScoped(s, src, level...)
	case "lzma2", "7z":
		return LZMA2Scoped(s, src, level...)
	case "lz4":
		return LZ4Scoped(s, src, level...)
	case "identity", "":
		if s == nil {
			return borrow.NewBytes(src, nil), nil
		}
		b := s.AllocBytes(len(src))
		copy(b.AsSlice(), src)
		return b, nil
	default:
		return borrow.Bytes{}, ErrUnsupportedEncoding
	}
}

// Gunzip decompresses a gzip payload (RFC 1952) from src into dst.
func Gunzip(src, dst []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	zr := gzipReaderStorage.Get()
	defer gzipReaderStorage.Put(zr)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)

	br.Reset(src)
	if err := zr.Reset(br); err != nil {
		return nil, err
	}
	defer zr.Close()

	return readAllSlice(zr, dst, gzipEstimatedSize(src))
}

// GunzipScoped decompresses a gzip payload directly into a zero-allocation scoped buffer.
func GunzipScoped(s *borrow.Scope, src []byte) (borrow.Bytes, error) {
	zr := gzipReaderStorage.Get()
	defer gzipReaderStorage.Put(zr)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)

	br.Reset(src)
	if err := zr.Reset(br); err != nil {
		return borrow.Bytes{}, err
	}
	defer zr.Close()

	return readAllSliceScoped(s, zr, gzipEstimatedSize(src))
}

// Unbrotli decompresses a Brotli payload (RFC 7932) from src into dst.
func Unbrotli(src, dst []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	r := brotliReaderStorage.Get()
	defer brotliReaderStorage.Put(r)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)
	br.Reset(src)

	if err := r.Reset(br); err != nil {
		return nil, err
	}

	return readAllSlice(r, dst, len(src)*3)
}

// UnbrotliScoped decompresses a Brotli payload directly into a zero-allocation scoped buffer.
func UnbrotliScoped(s *borrow.Scope, src []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	r := brotliReaderStorage.Get()
	defer brotliReaderStorage.Put(r)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)
	br.Reset(src)

	if err := r.Reset(br); err != nil {
		return borrow.Bytes{}, err
	}

	return readAllSliceScoped(s, r, len(src)*3)
}

// Unzstd decompresses a Zstandard payload (RFC 8878) from src into dst with zero allocations.
func Unzstd(src, dst []byte) ([]byte, error) {
	dec := zstdDecoderStorage.Get()
	defer zstdDecoderStorage.Put(dec)

	return dec.DecodeAll(src, dst)
}

// UnzstdScoped decompresses a Zstandard payload directly into a zero-allocation scoped buffer.
func UnzstdScoped(s *borrow.Scope, src []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	if s == nil {
		raw, err := Unzstd(src, nil)
		if err != nil {
			return borrow.Bytes{}, err
		}
		return borrow.NewBytes(raw, nil), nil
	}

	dec := zstdDecoderStorage.Get()
	defer zstdDecoderStorage.Put(dec)

	initCap := max(len(src)*2, 4096)
	b := s.AllocBytes(initCap)
	dst := b.AsSlice()[:0]

	decompressed, err := dec.DecodeAll(src, dst)
	if err != nil {
		return borrow.Bytes{}, err
	}

	if len(decompressed) <= initCap {
		return b.Slice(0, len(decompressed)), nil
	}

	return borrow.NewBytes(decompressed, nil), nil
}

// Inflate decompresses a raw DEFLATE payload (RFC 1951) from src into dst.
func Inflate(src, dst []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	fr := flateReaderStorage.Get()
	defer flateReaderStorage.Put(fr)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)

	br.Reset(src)
	_ = fr.Reset(br, nil)

	return readAllSlice(fr.(io.Reader), dst, len(src)*2)
}

// InflateScoped decompresses a raw DEFLATE payload directly into a zero-allocation scoped buffer.
func InflateScoped(s *borrow.Scope, src []byte) (borrow.Bytes, error) {
	fr := flateReaderStorage.Get()
	defer flateReaderStorage.Put(fr)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)

	br.Reset(src)
	_ = fr.Reset(br, nil)

	return readAllSliceScoped(s, fr.(io.Reader), len(src)*2)
}

// CompressGzip compresses src using RFC 1952 (gzip) format into dst.
func CompressGzip(src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}

	buf := byteBufferStorage.Get()
	defer byteBufferStorage.Put(buf)
	buf.Reset()

	zw := gzipWriterStorage.Get()
	defer gzipWriterStorage.Put(zw)
	zw.Reset(buf)

	if _, err := zw.Write(src); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	compressed := buf.Bytes()
	if cap(dst) < len(compressed) {
		dst = make([]byte, len(compressed))
	} else {
		dst = dst[:len(compressed)]
	}
	copy(dst, compressed)
	return dst, nil
}

// GzipScoped compresses src into a zero-allocation scoped buffer bound to s.
func GzipScoped(s *borrow.Scope, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	if s == nil {
		raw, err := CompressGzip(src, nil, level...)
		if err != nil {
			return borrow.Bytes{}, err
		}
		return borrow.NewBytes(raw, nil), nil
	}

	buf := byteBufferStorage.Get()
	defer byteBufferStorage.Put(buf)
	buf.Reset()

	zw := gzipWriterStorage.Get()
	defer gzipWriterStorage.Put(zw)
	zw.Reset(buf)

	if _, err := zw.Write(src); err != nil {
		return borrow.Bytes{}, err
	}
	if err := zw.Close(); err != nil {
		return borrow.Bytes{}, err
	}

	compressed := buf.Bytes()
	b := s.AllocBytes(len(compressed))
	copy(b.AsSlice(), compressed)
	return b, nil
}

// CompressDeflate compresses src using raw RFC 1951 deflate format into dst.
func CompressDeflate(src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}

	buf := byteBufferStorage.Get()
	defer byteBufferStorage.Put(buf)
	buf.Reset()

	fw := flateWriterStorage.Get()
	defer flateWriterStorage.Put(fw)
	fw.Reset(buf)

	if _, err := fw.Write(src); err != nil {
		return nil, err
	}
	if err := fw.Close(); err != nil {
		return nil, err
	}

	compressed := buf.Bytes()
	if cap(dst) < len(compressed) {
		dst = make([]byte, len(compressed))
	} else {
		dst = dst[:len(compressed)]
	}
	copy(dst, compressed)
	return dst, nil
}

// DeflateScoped compresses src using raw deflate into a zero-allocation scoped buffer bound to s.
func DeflateScoped(s *borrow.Scope, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	if s == nil {
		raw, err := CompressDeflate(src, nil, level...)
		if err != nil {
			return borrow.Bytes{}, err
		}
		return borrow.NewBytes(raw, nil), nil
	}

	buf := byteBufferStorage.Get()
	defer byteBufferStorage.Put(buf)
	buf.Reset()

	fw := flateWriterStorage.Get()
	defer flateWriterStorage.Put(fw)
	fw.Reset(buf)

	if _, err := fw.Write(src); err != nil {
		return borrow.Bytes{}, err
	}
	if err := fw.Close(); err != nil {
		return borrow.Bytes{}, err
	}

	compressed := buf.Bytes()
	b := s.AllocBytes(len(compressed))
	copy(b.AsSlice(), compressed)
	return b, nil
}

// CompressBrotli compresses src using RFC 7932 format into dst.
func CompressBrotli(src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}

	lvl := 6
	if len(level) > 0 {
		lvl = level[0]
	}
	_ = lvl

	w := brotliWriterStorage.Get()
	defer brotliWriterStorage.Put(w)

	var buf bytes.Buffer
	w.Reset(&buf)

	if _, err := w.Write(src); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	compressed := buf.Bytes()
	if cap(dst) < len(compressed) {
		dst = make([]byte, len(compressed))
	} else {
		dst = dst[:len(compressed)]
	}
	copy(dst, compressed)
	return dst, nil
}

// BrotliScoped compresses src using Brotli into a zero-allocation scoped buffer bound to s.
func BrotliScoped(s *borrow.Scope, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	compressed, err := CompressBrotli(src, nil, level...)
	if err != nil {
		return borrow.Bytes{}, err
	}

	if s == nil {
		return borrow.NewBytes(compressed, nil), nil
	}

	b := s.AllocBytes(len(compressed))
	copy(b.AsSlice(), compressed)
	return b, nil
}

// CompressZstd compresses src using RFC 8878 Zstandard format into dst.
func CompressZstd(src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}

	srcLen := len(src)
	maxCap := 4 + 5 + ((srcLen/131072)+1)*3 + srcLen
	if cap(dst) < maxCap {
		dst = make([]byte, 0, maxCap)
	} else {
		dst = dst[:0]
	}

	// Emit RFC 8878 Zstandard magic identifier (0xFD2FB528 in little-endian order)
	dst = append(dst, 0x28, 0xB5, 0x2F, 0xFD)

	// Format single-segment frame header with appropriately sized content length field
	switch {
	case srcLen < 256:
		dst = append(dst, 0x20, byte(srcLen))
	case srcLen < 65536+256:
		fcs := uint16(srcLen - 256)
		dst = append(dst, 0x20|(1<<6), byte(fcs), byte(fcs>>8))
	default:
		dst = append(dst, 0x20|(2<<6), byte(srcLen), byte(srcLen>>8), byte(srcLen>>16), byte(srcLen>>24))
	}

	// Chunk payload into standard uncompressed raw blocks (capped at 128KB max block size)
	offset := 0
	for offset < srcLen {
		chunkSize := srcLen - offset
		isLast := true
		if chunkSize > 131072 {
			chunkSize = 131072
			isLast = false
		}

		var lastFlag uint32
		if isLast {
			lastFlag = 1
		}
		// Block_Type = 0 (Raw)
		blockHeader := lastFlag | (0 << 1) | (uint32(chunkSize) << 3)
		dst = append(dst, byte(blockHeader), byte(blockHeader>>8), byte(blockHeader>>16))
		dst = append(dst, src[offset:offset+chunkSize]...)
		offset += chunkSize
	}

	return dst, nil
}

// ZstdScoped compresses src using Zstandard into a zero-allocation scoped buffer bound to s.
func ZstdScoped(s *borrow.Scope, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	compressed, err := CompressZstd(src, nil, level...)
	if err != nil {
		return borrow.Bytes{}, err
	}

	if s == nil {
		return borrow.NewBytes(compressed, nil), nil
	}

	b := s.AllocBytes(len(compressed))
	copy(b.AsSlice(), compressed)
	return b, nil
}

func gzipEstimatedSize(src []byte) int {
	if len(src) >= 4 {
		isize := int(binary.LittleEndian.Uint32(src[len(src)-4:]))
		if isize > 0 && isize < 50*1024*1024 {
			return isize
		}
	}
	return len(src) * 2
}

func readAllSlice(r io.Reader, dst []byte, estimatedCap int) ([]byte, error) {
	if cap(dst) < estimatedCap {
		dst = make([]byte, 0, estimatedCap)
	} else {
		dst = dst[:0]
	}

	for {
		if len(dst) == cap(dst) {
			if len(dst) >= DefaultMaxDecompressedSize {
				return nil, ErrDecompressionBomb
			}
			newCap := cap(dst) * 2
			if newCap < 4096 {
				newCap = 4096
			}
			newDst := make([]byte, len(dst), newCap)
			copy(newDst, dst)
			dst = newDst
		}

		n, err := r.Read(dst[len(dst):cap(dst)])
		dst = dst[:len(dst)+n]

		if err != nil {
			if err == io.EOF {
				return dst, nil
			}
			return nil, fmt.Errorf("%w: %w", ErrDecompressionFailed, err)
		}
	}
}

func readAllSliceScoped(s *borrow.Scope, r io.Reader, estimatedCap int) (borrow.Bytes, error) {
	if s == nil {
		res, err := readAllSlice(r, nil, estimatedCap)
		if err != nil {
			return borrow.Bytes{}, err
		}
		return borrow.NewBytes(res, nil), nil
	}

	initCap := max(estimatedCap, 4096)
	b := s.AllocBytes(initCap)
	raw := b.AsSlice()[:0]

	for {
		if len(raw) == cap(raw) {
			if len(raw) >= DefaultMaxDecompressedSize {
				return borrow.Bytes{}, ErrDecompressionBomb
			}
			newCap := cap(raw) * 2
			newB := s.AllocBytes(newCap)
			newRaw := newB.AsSlice()[:len(raw)]
			copy(newRaw, raw)
			b = newB
			raw = newRaw
		}

		n, err := r.Read(raw[len(raw):cap(raw)])
		raw = raw[:len(raw)+n]

		if err != nil {
			if err == io.EOF {
				return b.Slice(0, len(raw)), nil
			}
			return borrow.Bytes{}, fmt.Errorf("%w: %w", ErrDecompressionFailed, err)
		}
	}
}

// AcquireZstdReader borrows a pooled [*zstd.Decoder] bound to r.
func AcquireZstdReader(r io.Reader) (*zstd.Decoder, error) {
	dec := zstdDecoderStorage.Get()
	if err := dec.Reset(r); err != nil {
		zstdDecoderStorage.Put(dec)
		return nil, err
	}
	return dec, nil
}

// ReleaseZstdReader returns dec back to the pool.
func ReleaseZstdReader(dec *zstd.Decoder) {
	if dec != nil {
		_ = dec.Reset(nil)
		zstdDecoderStorage.Put(dec)
	}
}

// AcquireGzipReader borrows a pooled [*gzip.Reader] bound to r.
func AcquireGzipReader(r io.Reader) (*gzip.Reader, error) {
	zr := gzipReaderStorage.Get()
	if err := zr.Reset(r); err != nil {
		gzipReaderStorage.Put(zr)
		return nil, err
	}
	return zr, nil
}

// ReleaseGzipReader returns zr back to the pool.
func ReleaseGzipReader(zr *gzip.Reader) {
	if zr != nil {
		_ = zr.Close()
		gzipReaderStorage.Put(zr)
	}
}

// AcquireBrotliReader borrows a pooled [*brotli.Reader] bound to r.
func AcquireBrotliReader(r io.Reader) (*brotli.Reader, error) {
	br := brotliReaderStorage.Get()
	if err := br.Reset(r); err != nil {
		brotliReaderStorage.Put(br)
		return nil, err
	}
	return br, nil
}

// ReleaseBrotliReader returns br back to the pool.
func ReleaseBrotliReader(br *brotli.Reader) {
	if br != nil {
		brotliReaderStorage.Put(br)
	}
}

// IsCompressed checks magic header bytes to detect gzip, zstd, or lz4 compressed payloads.
func IsCompressed(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	if data[0] == 0x1f && data[1] == 0x8b {
		return true
	}
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
		return true
	}
	if len(data) >= 4 && data[0] == 0x04 && data[1] == 0x22 && data[2] == 0x4d && data[3] == 0x18 {
		return true
	}
	return false
}

// MatchesEncoding reports whether enc is present within the Content-Encoding header value.
func MatchesEncoding(headerValue []byte, enc string) bool {
	if len(headerValue) == 0 || len(enc) == 0 {
		return false
	}
	return strings.Contains(strings.ToLower(string(headerValue)), strings.ToLower(enc))
}

// NewDictionaryReader returns an io.ReadCloser that decompresses data from r
// using the specified Content-Encoding algorithm ("dcz", "dcb", "gzip", "br", "zstd", "deflate")
// with the provided dictionary payload (RFC 9842).
func NewDictionaryReader(encoding string, r io.Reader, dictBytes []byte) (io.ReadCloser, error) {
	if r == nil {
		return nil, errors.New("codec: nil reader")
	}

	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "dcz":
		return NewDCZReader(r, dictBytes)
	case "dcb":
		return NewDCBReader(r, dictBytes)
	default:
		return NewReader(encoding, r)
	}
}

// NewReader returns a pooled [io.ReadCloser] that decompresses data from r.
func NewReader(encoding string, r io.Reader) (io.ReadCloser, error) {
	if r == nil {
		return nil, errors.New("codec: nil reader")
	}

	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "gzip", "x-gzip":
		zr, err := AcquireGzipReader(r)
		if err != nil {
			return nil, err
		}
		return &decompressReadCloser{
			reader: zr,
			closer: closerOf(r),
			release: func() {
				ReleaseGzipReader(zr)
			},
		}, nil

	case "br":
		br, err := AcquireBrotliReader(r)
		if err != nil {
			return nil, err
		}
		return &decompressReadCloser{
			reader: br,
			closer: closerOf(r),
			release: func() {
				ReleaseBrotliReader(br)
			},
		}, nil

	case "zstd":
		dec, err := AcquireZstdReader(r)
		if err != nil {
			return nil, err
		}
		return &decompressReadCloser{
			reader: dec,
			closer: closerOf(r),
			release: func() {
				ReleaseZstdReader(dec)
			},
		}, nil

	case "deflate":
		fr := flateReaderStorage.Get()
		br := bytesReaderStorage.Get()
		br.Reset(nil)
		if err := fr.Reset(r, nil); err != nil {
			flateReaderStorage.Put(fr)
			bytesReaderStorage.Put(br)
			return nil, err
		}
		return &decompressReadCloser{
			reader: fr.(io.Reader),
			closer: closerOf(r),
			release: func() {
				flateReaderStorage.Put(fr)
				bytesReaderStorage.Put(br)
			},
		}, nil

	case "lz4":
		zr := lz4ReaderStorage.Get()
		zr.Reset(r)
		return &decompressReadCloser{
			reader: zr,
			closer: closerOf(r),
			release: func() {
				lz4ReaderStorage.Put(zr)
			},
		}, nil

	case "identity", "":
		return io.NopCloser(r), nil

	default:
		return nil, ErrUnsupportedEncoding
	}
}

// NewWriter returns a pooled [io.WriteCloser] that compresses written data to w.
func NewWriter(encoding string, w io.Writer, level ...int) (io.WriteCloser, error) {
	if w == nil {
		return nil, errors.New("codec: nil writer")
	}

	enc := strings.ToLower(strings.TrimSpace(encoding))
	switch enc {
	case "gzip", "x-gzip":
		zw := gzipWriterStorage.Get()
		zw.Reset(w)
		return &compressWriteCloser{
			writer: zw,
			release: func() {
				gzipWriterStorage.Put(zw)
			},
		}, nil

	case "deflate":
		fw := flateWriterStorage.Get()
		fw.Reset(w)
		return &compressWriteCloser{
			writer: fw,
			release: func() {
				flateWriterStorage.Put(fw)
			},
		}, nil

	case "br":
		bw := brotliWriterStorage.Get()
		bw.Reset(w)
		return &compressWriteCloser{
			writer: bw,
			release: func() {
				brotliWriterStorage.Put(bw)
			},
		}, nil

	case "lz4":
		zw := lz4WriterStorage.Get()
		zw.Reset(w)
		return &compressWriteCloser{
			writer: zw,
			release: func() {
				lz4WriterStorage.Put(zw)
			},
		}, nil

	case "identity", "":
		return nopWriteCloser{Writer: w}, nil

	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedEncoding, encoding)
	}
}

type decompressReadCloser struct {
	reader  io.Reader
	closer  io.Closer
	release func()
}

func (d *decompressReadCloser) Read(p []byte) (int, error) {
	return d.reader.Read(p)
}

func (d *decompressReadCloser) Close() error {
	if d.release != nil {
		d.release()
		d.release = nil
	}
	if d.closer != nil {
		return d.closer.Close()
	}
	return nil
}

type compressWriteCloser struct {
	writer  io.WriteCloser
	release func()
}

func (c *compressWriteCloser) Write(p []byte) (int, error) {
	return c.writer.Write(p)
}

func (c *compressWriteCloser) Close() error {
	var err error
	if c.writer != nil {
		err = c.writer.Close()
	}
	if c.release != nil {
		c.release()
		c.release = nil
	}
	return err
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func closerOf(r io.Reader) io.Closer {
	if c, ok := r.(io.Closer); ok {
		return c
	}
	return nil
}

// Unlzma2 decompresses an LZMA2 payload from src into dst.
func Unlzma2(src, dst []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}
	dec := lzma.NewDecompressor2(8 * 1024 * 1024)
	rc, err := dec.Decompress(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return readAllSlice(rc, dst, len(src)*3)
}

// Unlzma2Scoped decompresses an LZMA2 payload directly into a zero-allocation scoped buffer.
func Unlzma2Scoped(s *borrow.Scope, src []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}
	dec := lzma.NewDecompressor2(8 * 1024 * 1024)
	rc, err := dec.Decompress(bytes.NewReader(src))
	if err != nil {
		return borrow.Bytes{}, err
	}
	defer rc.Close()
	return readAllSliceScoped(s, rc, len(src)*3)
}

// CompressLZMA2 compresses src into dst using LZMA2 chunk framing.
func CompressLZMA2(src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}
	comp := lzma.NewCompressor2(8 * 1024 * 1024)
	var buf bytes.Buffer
	if _, err := comp.Compress(bytes.NewReader(src), &buf); err != nil {
		return nil, err
	}
	res := buf.Bytes()
	if cap(dst) >= len(res) {
		dst = dst[:len(res)]
		copy(dst, res)
		return dst, nil
	}
	return res, nil
}

// LZMA2Scoped compresses src directly into a zero-allocation scoped buffer.
func LZMA2Scoped(s *borrow.Scope, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}
	comp := lzma.NewCompressor2(8 * 1024 * 1024)
	var buf bytes.Buffer
	if _, err := comp.Compress(bytes.NewReader(src), &buf); err != nil {
		return borrow.Bytes{}, err
	}
	res := buf.Bytes()
	if s == nil {
		return borrow.NewBytes(res, nil), nil
	}
	b := s.AllocBytes(len(res))
	copy(b.AsSlice(), res)
	return b, nil
}

// Unlz4 decompresses an LZ4 frame payload from src into dst.
func Unlz4(src, dst []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, nil
	}

	zr := lz4ReaderStorage.Get()
	defer lz4ReaderStorage.Put(zr)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)

	br.Reset(src)
	zr.Reset(br)

	return readAllSlice(zr, dst, len(src)*3)
}

// Unlz4Scoped decompresses an LZ4 payload directly into a zero-allocation scoped buffer.
func Unlz4Scoped(s *borrow.Scope, src []byte) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	zr := lz4ReaderStorage.Get()
	defer lz4ReaderStorage.Put(zr)

	br := bytesReaderStorage.Get()
	defer bytesReaderStorage.Put(br)

	br.Reset(src)
	zr.Reset(br)

	return readAllSliceScoped(s, zr, len(src)*3)
}

// CompressLZ4 compresses src using LZ4 framed format into dst.
func CompressLZ4(src, dst []byte, level ...int) ([]byte, error) {
	if len(src) == 0 {
		return dst[:0], nil
	}

	buf := byteBufferStorage.Get()
	defer byteBufferStorage.Put(buf)
	buf.Reset()

	zw := lz4WriterStorage.Get()
	defer lz4WriterStorage.Put(zw)
	zw.Reset(buf)

	if len(level) > 0 && level[0] > 0 {
		lvl := level[0]
		var cl lz4.CompressionLevel
		switch {
		case lvl <= 1:
			cl = lz4.Level1
		case lvl == 2:
			cl = lz4.Level2
		case lvl == 3:
			cl = lz4.Level3
		case lvl == 4:
			cl = lz4.Level4
		case lvl == 5:
			cl = lz4.Level5
		case lvl == 6:
			cl = lz4.Level6
		case lvl == 7:
			cl = lz4.Level7
		case lvl == 8:
			cl = lz4.Level8
		default:
			cl = lz4.Level9
		}
		_ = zw.Apply(lz4.CompressionLevelOption(cl))
	}

	if _, err := zw.Write(src); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}

	compressed := buf.Bytes()
	if cap(dst) < len(compressed) {
		dst = make([]byte, len(compressed))
	} else {
		dst = dst[:len(compressed)]
	}
	copy(dst, compressed)
	return dst, nil
}

// LZ4Scoped compresses src using LZ4 framed format into a zero-allocation scoped buffer bound to s.
func LZ4Scoped(s *borrow.Scope, src []byte, level ...int) (borrow.Bytes, error) {
	if len(src) == 0 {
		return borrow.Bytes{}, nil
	}

	compressed, err := CompressLZ4(src, nil, level...)
	if err != nil {
		return borrow.Bytes{}, err
	}

	if s == nil {
		return borrow.NewBytes(compressed, nil), nil
	}

	b := s.AllocBytes(len(compressed))
	copy(b.AsSlice(), compressed)
	return b, nil
}

// CompressLZ4Block compresses src into dst using raw un-framed LZ4 block format.
// Returns the number of bytes written to dst.
func CompressLZ4Block(src, dst []byte) (int, error) {
	return lz4.CompressBlock(src, dst)
}

// Unlz4Block decompresses raw LZ4 block src into dst.
// Returns the number of bytes written to dst. dst must have sufficient capacity.
func Unlz4Block(src, dst []byte) (int, error) {
	return lz4.UncompressBlock(src, dst)
}

// CompressLZ4BlockBound returns the maximum compressed size for n bytes in raw block format.
func CompressLZ4BlockBound(n int) int {
	return lz4.CompressBlockBound(n)
}
