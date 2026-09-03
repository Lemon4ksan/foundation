// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

// The Range Coder maintains an arithmetic interval [Low, Low + Range) mapped to a 32-bit
// integer range. All bit probabilities are stored as uint16 in [0..2048], where 1024 represents
// equal probability. Models adapt using a 5-bit shift (1/32 moving average rate).
//
// When Range drops below 2^24 (kTopValue), the coder shifts 8 bits out to the byte stream.
// In the encoder, potential carries across runs of 0xFF bytes are buffered and resolved via shiftLow.

import (
	"io"
)

const (
	kTopValue             = 1 << 24
	kNumBitModelTotalBits = 11
	kBitModelTotal        = 1 << kNumBitModelTotalBits
	kNumMoveBits          = 5
	probInitVal           = kBitModelTotal >> 1
	kBufferSize           = 64 * 1024
)

// RangeDecoder implements buffered arithmetic range decoding for LZMA decompression.
type RangeDecoder struct {
	r      io.Reader
	range_ uint32
	code   uint32
	buf    [kBufferSize]byte
	pos    int
	limit  int
	eof    bool
}

// Init initializes the range decoder by loading 5 priming bytes from the input stream.
func (rd *RangeDecoder) Init(r io.Reader) error {
	rd.r = r
	rd.range_ = 0xFFFFFFFF
	rd.code = 0
	rd.pos = 0
	rd.limit = 0
	rd.eof = false

	for i := 0; i < 5; i++ {
		b, err := rd.readByte()
		if err != nil {
			return err
		}
		rd.code = (rd.code << 8) | uint32(b)
	}
	return nil
}

func (rd *RangeDecoder) fillAndReadByte() (byte, error) {
	n, err := rd.r.Read(rd.buf[:])
	if n == 0 {
		rd.eof = true
		if err == nil {
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	rd.pos = 1
	rd.limit = n
	return rd.buf[0], nil
}

func (rd *RangeDecoder) readByte() (byte, error) {
	if rd.pos < rd.limit {
		b := rd.buf[rd.pos]
		rd.pos++
		return b, nil
	}
	return rd.fillAndReadByte()
}

func (rd *RangeDecoder) normalizeSlow() error {
	for rd.range_ < kTopValue {
		if rd.pos < rd.limit {
			b := rd.buf[rd.pos]
			rd.pos++
			rd.range_ <<= 8
			rd.code = (rd.code << 8) | uint32(b)
		} else {
			b, err := rd.fillAndReadByte()
			if err != nil {
				return err
			}
			rd.range_ <<= 8
			rd.code = (rd.code << 8) | uint32(b)
		}
	}
	return nil
}

// DecodeBit decodes a single binary decision against the adaptive probability model prob.
func (rd *RangeDecoder) DecodeBit(prob *uint16) (int, error) {
	p := uint32(*prob)
	bound := (rd.range_ >> kNumBitModelTotalBits) * p
	if rd.code < bound {
		rd.range_ = bound
		*prob += uint16((kBitModelTotal - p) >> kNumMoveBits)
		if rd.range_ < kTopValue {
			rd.range_ <<= 8
			if rd.pos < rd.limit {
				rd.code = (rd.code << 8) | uint32(rd.buf[rd.pos])
				rd.pos++
			} else {
				if err := rd.normalizeSlow(); err != nil {
					return 0, err
				}
			}
		}
		return 0, nil
	}
	rd.range_ -= bound
	rd.code -= bound
	*prob -= uint16(p >> kNumMoveBits)
	if rd.range_ < kTopValue {
		rd.range_ <<= 8
		if rd.pos < rd.limit {
			rd.code = (rd.code << 8) | uint32(rd.buf[rd.pos])
			rd.pos++
		} else {
			if err := rd.normalizeSlow(); err != nil {
				return 0, err
			}
		}
	}
	return 1, nil
}

// DecodeDirectBits decodes unmodeled direct bits (fixed 50/50 probability).
func (rd *RangeDecoder) DecodeDirectBits(numBits int) (uint32, error) {
	var val uint32
	for i := 0; i < numBits; i++ {
		rd.range_ >>= 1
		val <<= 1
		if rd.code >= rd.range_ {
			rd.code -= rd.range_
			val |= 1
		}
		if rd.range_ < kTopValue {
			if rd.pos < rd.limit {
				b := rd.buf[rd.pos]
				rd.pos++
				rd.range_ <<= 8
				rd.code = (rd.code << 8) | uint32(b)
			} else {
				if err := rd.normalizeSlow(); err != nil {
					return 0, err
				}
			}
		}
	}
	return val, nil
}

// DecodeTree decodes a binary probability decision tree of given depth.
func (rd *RangeDecoder) DecodeTree(probs []uint16, numBits int) (uint32, error) {
	var m uint32 = 1
	for i := 0; i < numBits; i++ {
		bit, err := rd.DecodeBit(&probs[m])
		if err != nil {
			return 0, err
		}
		m = (m << 1) | uint32(bit)
	}
	return m - (1 << numBits), nil
}

// DecodeReverseTree decodes a reversed binary decision tree (lowest bit first).
func (rd *RangeDecoder) DecodeReverseTree(probs []uint16, numBits int) (uint32, error) {
	var m uint32 = 1
	var symbol uint32
	for i := 0; i < numBits; i++ {
		bit, err := rd.DecodeBit(&probs[m])
		if err != nil {
			return 0, err
		}
		m = (m << 1) | uint32(bit)
		symbol |= uint32(bit) << i
	}
	return symbol, nil
}

// RangeEncoder implements arithmetic range encoding for LZMA stream compression.
type RangeEncoder struct {
	w         io.Writer
	low       uint64
	range_    uint32
	cacheSize uint64
	cache     byte
	buf       [kBufferSize]byte
	pos       int
}

// Init initializes the range encoder writing to w.
func (re *RangeEncoder) Init(w io.Writer) {
	re.w = w
	re.low = 0
	re.range_ = 0xFFFFFFFF
	re.cacheSize = 1
	re.cache = 0
	re.pos = 0
}

func (re *RangeEncoder) flushAndWriteByte(b byte) error {
	if re.pos > 0 {
		if _, err := re.w.Write(re.buf[:re.pos]); err != nil {
			return err
		}
		re.pos = 0
	}
	re.buf[0] = b
	re.pos = 1
	return nil
}

func (re *RangeEncoder) writeByte(b byte) error {
	if re.pos < len(re.buf) {
		re.buf[re.pos] = b
		re.pos++
		return nil
	}
	return re.flushAndWriteByte(b)
}

// shiftLow outputs high bytes and resolves potential carry overflows across 0xFF runs.
func (re *RangeEncoder) shiftLow() error {
	low := uint32(re.low)
	high := uint32(re.low >> 32)
	if low < 0xFF000000 || high != 0 {
		temp := re.cache
		for {
			if err := re.writeByte(temp + byte(high)); err != nil {
				return err
			}
			temp = 0xFF
			re.cacheSize--
			if re.cacheSize == 0 {
				break
			}
		}
		re.cache = byte(low >> 24)
	}
	re.cacheSize++
	re.low = uint64(low << 8)
	return nil
}

// EncodeBit encodes a single bit with adaptive probability model prob.
func (re *RangeEncoder) EncodeBit(prob *uint16, bit int) error {
	p := uint32(*prob)
	bound := (re.range_ >> kNumBitModelTotalBits) * p
	if bit == 0 {
		re.range_ = bound
		*prob += uint16((kBitModelTotal - p) >> kNumMoveBits)
	} else {
		re.low += uint64(bound)
		re.range_ -= bound
		*prob -= uint16(p >> kNumMoveBits)
	}
	if re.range_ < kTopValue {
		re.range_ <<= 8
		return re.shiftLow()
	}
	return nil
}

// EncodeDirectBits encodes raw unmodeled bits with fixed 50/50 probability.
func (re *RangeEncoder) EncodeDirectBits(val uint32, numBits int) error {
	for i := numBits - 1; i >= 0; i-- {
		re.range_ >>= 1
		if ((val >> i) & 1) == 1 {
			re.low += uint64(re.range_)
		}
		if re.range_ < kTopValue {
			re.range_ <<= 8
			if err := re.shiftLow(); err != nil {
				return err
			}
		}
	}
	return nil
}

// EncodeTree encodes a symbol through a binary probability decision tree.
func (re *RangeEncoder) EncodeTree(probs []uint16, numBits int, symbol uint32) error {
	var m uint32 = 1
	for i := numBits - 1; i >= 0; i-- {
		bit := int((symbol >> i) & 1)
		if err := re.EncodeBit(&probs[m], bit); err != nil {
			return err
		}
		m = (m << 1) | uint32(bit)
	}
	return nil
}

// EncodeReverseTree encodes a symbol through a reverse binary probability decision tree (lowest bit first).
func (re *RangeEncoder) EncodeReverseTree(probs []uint16, numBits int, symbol uint32) error {
	var m uint32 = 1
	for i := 0; i < numBits; i++ {
		bit := int((symbol >> i) & 1)
		if err := re.EncodeBit(&probs[m], bit); err != nil {
			return err
		}
		m = (m << 1) | uint32(bit)
	}
	return nil
}

// Flush flushes all buffered arithmetic bits into the underlying writer.
func (re *RangeEncoder) Flush() error {
	for i := 0; i < 5; i++ {
		if err := re.shiftLow(); err != nil {
			return err
		}
	}
	if re.pos > 0 {
		_, err := re.w.Write(re.buf[:re.pos])
		re.pos = 0
		return err
	}
	return nil
}
