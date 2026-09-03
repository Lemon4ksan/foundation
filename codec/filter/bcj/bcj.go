// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bcj implements Branch-Call-Jump (BCJ) bytecode pre-filters for executable binaries.
// BCJ converts relative branch targets (e.g. x86 E8/E9 CALL/JMP instructions) into absolute addresses,
// significantly improving entropy and compression ratios for subsequent LZMA/Deflate stages.
package bcj

import (
	"encoding/binary"
	"io"
)

// Architecture represents the target CPU instruction set architecture for branch filtering.
type Architecture uint32

const (
	// X86 represents 32-bit and 64-bit x86 branch filtering (BCJ / Bra86).
	X86 Architecture = iota
	// ARM represents 32-bit ARM instruction branch filtering.
	ARM
	// ARMT represents ARM Thumb mode branch filtering.
	ARMT
	// ARM64 represents 64-bit ARM (AArch64) branch filtering.
	ARM64
	// PPC represents PowerPC branch filtering.
	PPC
	// IA64 represents Intel Itanium 64-bit branch filtering.
	IA64
	// SPARC represents SPARC branch filtering.
	SPARC
)

// Filter applies in-place branch conversion on data.
// If encode is true, relative offsets are converted to absolute addresses.
// If encode is false (decode), absolute addresses are converted back to relative offsets.
// ip specifies the starting instruction pointer base (usually 0).
func Filter(arch Architecture, data []byte, ip uint32, encode bool) int {
	switch arch {
	case X86:
		var state uint32
		return filterX86(data, ip, &state, encode)
	case ARM:
		return filterARM(data, ip, encode)
	case ARMT:
		return filterARMT(data, ip, encode)
	case ARM64:
		return filterARM64(data, ip, encode)
	case PPC:
		return filterPPC(data, ip, encode)
	case IA64:
		return filterIA64(data, ip, encode)
	case SPARC:
		return filterSPARC(data, ip, encode)
	default:
		return len(data)
	}
}

// filterX86 implements the 7-Zip x86 branch converter (Bra86).
func filterX86(data []byte, ip uint32, state *uint32, encode bool) int {
	size := len(data)
	if size < 5 {
		return 0
	}

	p := 0
	lim := size - 4
	mask := *state
	ip += 4

	for p < lim {
		if mask == 0 && p+8 <= lim {
			v := binary.LittleEndian.Uint64(data[p:])
			hasE8 := ((v ^ 0xE8E8E8E8E8E8E8E8) - 0x0101010101010101) & ^(v ^ 0xE8E8E8E8E8E8E8E8) & 0x8080808080808080
			hasE9 := ((v ^ 0xE9E9E9E9E9E9E9E9) - 0x0101010101010101) & ^(v ^ 0xE9E9E9E9E9E9E9E9) & 0x8080808080808080
			if (hasE8 | hasE9) == 0 {
				p += 8
				continue
			}
		}

		b := data[p]
		if b != 0xE8 && b != 0xE9 {
			p++
			mask >>= 1
			continue
		}

		offset := p + 1
		src := binary.LittleEndian.Uint32(data[offset : offset+4])

		if mask > 4 || mask == 3 {
			mask >>= 1
			p++
			continue
		}
		mask >>= 1

		msb := byte(src >> 24)
		if (msb+1)&0xFE != 0 {
			p++
			continue
		}

		var dest uint32
		pc := ip + uint32(p)
		if encode {
			dest = src + pc
		} else {
			dest = src - pc
		}

		if (mask << 3) != 0 {
			shift := mask << 3
			testMsb := byte((dest >> shift) & 0xFF)
			if (testMsb+1)&0xFE == 0 {
				dest ^= (uint32(0x100) << shift) - 1
				if encode {
					dest += pc
				} else {
					dest -= pc
				}
			}
		}

		dest &= (1 << 25) - 1
		dest -= (1 << 24)

		binary.LittleEndian.PutUint32(data[offset:offset+4], dest)
		p += 5
		mask = 0
	}

	*state = mask
	return p
}

// filterARM implements 32-bit ARM BL/BLX branch conversion.
func filterARM(data []byte, ip uint32, encode bool) int {
	size := len(data) &^ 3
	for i := 0; i+4 <= size; i += 4 {
		if data[i+3] == 0xEB { // BL instruction
			src := binary.LittleEndian.Uint32(data[i : i+4])
			pc := (ip + uint32(i)) >> 2
			var dest uint32
			if encode {
				dest = (src + pc) & 0x00FFFFFF
			} else {
				dest = (src - pc) & 0x00FFFFFF
			}
			dest |= 0xEB000000
			binary.LittleEndian.PutUint32(data[i:i+4], dest)
		}
	}
	return size
}

// filterARMT implements ARM Thumb branch conversion.
func filterARMT(data []byte, ip uint32, encode bool) int {
	size := len(data) &^ 1
	for i := 0; i+4 <= size; i += 2 {
		b1 := data[i+1]
		if (b1 & 0xF8) == 0xF0 {
			b3 := data[i+3]
			if (b3 & 0xF8) == 0xF8 {
				src := (uint32(b1&0x07) << 19) |
					(uint32(data[i]) << 11) |
					(uint32(b3&0x07) << 8) |
					uint32(data[i+2])
				src <<= 1
				pc := (ip + uint32(i)) >> 1
				var dest uint32
				if encode {
					dest = src + pc
				} else {
					dest = src - pc
				}
				dest >>= 1
				data[i+1] = 0xF0 | byte((dest>>19)&0x07)
				data[i] = byte(dest >> 11)
				data[i+3] = 0xF8 | byte((dest>>8)&0x07)
				data[i+2] = byte(dest)
				i += 2
			}
		}
	}
	return size
}

// filterARM64 implements AArch64 (ARM64) B/BL branch conversion.
func filterARM64(data []byte, ip uint32, encode bool) int {
	size := len(data) &^ 3
	for i := 0; i+4 <= size; i += 4 {
		v := binary.LittleEndian.Uint32(data[i : i+4])
		// Matches ARM64 B and BL: opcodes 0x14000000 and 0x94000000
		if (v & 0x7C000000) == 0x14000000 {
			pc := (ip + uint32(i)) >> 2
			var dest uint32
			if encode {
				dest = v + pc
			} else {
				dest = v - pc
			}
			dest = (dest & 0x03FFFFFF) | (v & 0xFC000000)
			binary.LittleEndian.PutUint32(data[i:i+4], dest)
		}
	}
	return size
}

// filterPPC implements PowerPC branch conversion (big-endian).
func filterPPC(data []byte, ip uint32, encode bool) int {
	size := len(data) &^ 3
	for i := 0; i+4 <= size; i += 4 {
		v := binary.BigEndian.Uint32(data[i : i+4])
		// PowerPC B/BA/BL/BLA: opcode 18 (0x48000000)
		if (v & 0xFC000002) == 0x48000001 {
			pc := (ip + uint32(i)) >> 2
			var dest uint32
			if encode {
				dest = v + pc
			} else {
				dest = v - pc
			}
			dest = (dest & 0x03FFFFFC) | (v & 0xFC000003)
			binary.BigEndian.PutUint32(data[i:i+4], dest)
		}
	}
	return size
}

// filterIA64 implements Intel Itanium 64-bit instruction bundle branch conversion.
func filterIA64(data []byte, ip uint32, encode bool) int {
	size := (len(data) / 16) * 16
	for i := 0; i < size; i += 16 {
		mask := data[i] & 0x1F
		if mask != 0x16 && mask != 0x17 {
			continue
		}
		for slot := 2; slot < 3; slot++ {
			bitPos := uint(5 + slot*41)
			bytePos := bitPos >> 3
			bitOffset := bitPos & 7
			raw := binary.LittleEndian.Uint64(data[i+int(bytePos) : i+int(bytePos)+8])
			inst := raw >> bitOffset
			if (inst & 0x03F00000000) == 0x00500000000 {
				src := uint32((inst >> 13) & 0xFFFFF)
				pc := (ip + uint32(i)) >> 4
				var dest uint32
				if encode {
					dest = src + pc
				} else {
					dest = src - pc
				}
				dest &= 0xFFFFF
				inst = (inst &^ (0xFFFFF << 13)) | (uint64(dest) << 13)
				raw = (raw &^ (uint64(0x1FFFFFFFFFF) << bitOffset)) | (inst << bitOffset)
				binary.LittleEndian.PutUint64(data[i+int(bytePos):i+int(bytePos)+8], raw)
			}
		}
	}
	return size
}

// filterSPARC implements SPARC CALL branch conversion (big-endian).
func filterSPARC(data []byte, ip uint32, encode bool) int {
	size := len(data) &^ 3
	for i := 0; i+4 <= size; i += 4 {
		v := binary.BigEndian.Uint32(data[i : i+4])
		// SPARC CALL: opcode 0x40000000
		if (v & 0xC0000000) == 0x40000000 {
			pc := (ip + uint32(i)) >> 2
			var dest uint32
			if encode {
				dest = v + pc
			} else {
				dest = v - pc
			}
			dest = (dest & 0x3FFFFFFF) | (v & 0xC0000000)
			binary.BigEndian.PutUint32(data[i:i+4], dest)
		}
	}
	return size
}

// Reader wraps an io.Reader and decodes BCJ-filtered bytes on the fly.
type Reader struct {
	r    io.Reader
	arch Architecture
	ip   uint32
	buf  []byte
	off  int
	end  int
	err  error
}

// NewReader constructs a new BCJ streaming decompressor filter.
func NewReader(r io.Reader, arch Architecture) *Reader {
	return &Reader{
		r:    r,
		arch: arch,
		buf:  make([]byte, 32*1024),
	}
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	if r.off < r.end {
		n := copy(p, r.buf[r.off:r.end])
		r.off += n
		return n, nil
	}

	if r.err != nil {
		return 0, r.err
	}

	n, err := r.r.Read(r.buf)
	if n > 0 {
		r.off = 0
		r.end = n
		Filter(r.arch, r.buf[:n], r.ip, false)
		r.ip += uint32(n)
		toCopy := copy(p, r.buf[r.off:r.end])
		r.off += toCopy
		r.err = err
		return toCopy, nil
	}

	r.err = err
	return 0, err
}

// Writer wraps an io.Writer and encodes BCJ branch instructions before forwarding.
type Writer struct {
	w    io.Writer
	arch Architecture
	ip   uint32
}

// NewWriter constructs a new BCJ streaming encoder filter.
func NewWriter(w io.Writer, arch Architecture) *Writer {
	return &Writer{
		w:    w,
		arch: arch,
	}
}

// Write implements io.Writer.
func (w *Writer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	chunk := make([]byte, len(p))
	copy(chunk, p)
	Filter(w.arch, chunk, w.ip, true)
	w.ip += uint32(len(p))
	return w.w.Write(chunk)
}
