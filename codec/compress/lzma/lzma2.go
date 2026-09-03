// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/lemon4ksan/foundation/silicon/pool"
)

// decoderPool recycles DecoderCore instances per scheduler processor P.
// This avoids mutex contention and completely eliminates heap allocation / GC scavenging during multi-core execution.
var decoderPool = pool.NewPerPStorage(func() *DecoderCore {
	return NewDecoderCore(3, 0, 2, 8*1024*1024, 65536)
})

// ErrInvalidLZMA2Property indicates an invalid LZMA2 dictionary property byte (> 40).
var ErrInvalidLZMA2Property = errors.New("lzma2: invalid property byte")

// DictSizeFromProp recalculates dictionary size from the single LZMA2 property byte.
func DictSizeFromProp(p byte) (uint32, error) {
	if p > 40 {
		return 0, ErrInvalidLZMA2Property
	}
	if p == 40 {
		return 0xFFFFFFFF, nil
	}
	return (uint32(2) | uint32(p&1)) << (p/2 + 11), nil
}

// PropFromDictSize calculates the optimal single property byte for a given dictionary size.
func PropFromDictSize(size uint32) byte {
	if size == 0xFFFFFFFF {
		return 40
	}
	if size <= (1 << 12) {
		return 0
	}
	for i := byte(0); i < 40; i++ {
		calc, _ := DictSizeFromProp(i)
		if calc >= size {
			return i
		}
	}
	return 40
}

// Decompressor2 implements decompression for LZMA2 chunked streams.
type Decompressor2 struct {
	DictSize uint32
}

// NewDecompressor2 constructs an LZMA2 decompressor.
func NewDecompressor2(dictSize uint32) *Decompressor2 {
	if dictSize == 0 {
		dictSize = 8 * 1024 * 1024
	}
	return &Decompressor2{DictSize: dictSize}
}

// NewDecompressor2FromProp constructs an LZMA2 decompressor from a 1-byte property header.
func NewDecompressor2FromProp(prop byte) (*Decompressor2, error) {
	dictSize, err := DictSizeFromProp(prop)
	if err != nil {
		return nil, err
	}
	return NewDecompressor2(dictSize), nil
}

func readAllExact(r io.Reader) ([]byte, error) {
	if br, ok := r.(*bytes.Reader); ok {
		buf := make([]byte, br.Len())
		_, err := br.Read(buf)
		return buf, err
	}
	if bb, ok := r.(*bytes.Buffer); ok {
		buf := make([]byte, bb.Len())
		_, err := bb.Read(buf)
		return buf, err
	}
	if s, ok := r.(io.Seeker); ok {
		curr, err := s.Seek(0, io.SeekCurrent)
		if err == nil {
			end, err := s.Seek(0, io.SeekEnd)
			if err == nil {
				_, _ = s.Seek(curr, io.SeekStart)
				size := end - curr
				if size >= 0 && size < 1<<31 {
					buf := make([]byte, size)
					_, err := io.ReadFull(r, buf)
					return buf, err
				}
			}
		}
	}
	return io.ReadAll(r)
}

// Decompress decodes an LZMA2 chunked stream from src using zero-copy parallel direct slicing.
//
// LZMA2 Chunk Header Specification:
//   - 0x00: End marker of the stream.
//   - 0x01: Uncompressed chunk (dictionary reset).
//   - 0x02: Uncompressed chunk (dictionary retained).
//   - 0x80 | (mode << 5): Compressed LZMA chunk:
//   - Mode 0: Retain dictionary and state.
//   - Mode 1: Reset state.
//   - Mode 2: Reset state and reload (lc, lp, pb) properties.
//   - Mode 3: Reset dictionary, state, and properties (fully independent chunk).
func (d *Decompressor2) Decompress(src io.Reader) (io.ReadCloser, error) {
	rawComp, err := readAllExact(src)
	if err != nil {
		return nil, err
	}

	type chunkTask struct {
		isRaw      bool
		rawSlice   []byte
		chunkData  []byte
		unpackSize uint64
		offset     int
		mode       int
		lc, lp, pb uint
	}

	var tasks []chunkTask
	r := bytes.NewReader(rawComp)
	var totalUnpackSize int

	var lc, lp, pb uint = 3, 0, 2
	for {
		var ctrl [1]byte
		_, err := r.Read(ctrl[:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		control := ctrl[0]
		if control == 0x00 {
			// Stream End Marker
			break
		}

		if control < 0x80 {
			// Uncompressed Chunk: control 1 resets dictionary, control 2 retains dictionary history.
			var sizeBuf [2]byte
			if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
				return nil, err
			}
			uncompressedSize := int(binary.BigEndian.Uint16(sizeBuf[:])) + 1
			data := make([]byte, uncompressedSize)
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, err
			}
			tasks = append(tasks, chunkTask{
				isRaw:      true,
				rawSlice:   data,
				unpackSize: uint64(uncompressedSize),
				offset:     totalUnpackSize,
				mode:       int(control),
			})
			totalUnpackSize += uncompressedSize
			continue
		}

		mode := (control >> 5) & 3
		var header [4]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			return nil, err
		}

		unpackSize := ((uint64(control&0x1F) << 16) | (uint64(header[0]) << 8) | uint64(header[1])) + 1
		packSize := ((int(header[2]) << 8) | int(header[3])) + 1

		if mode >= 2 {
			var prop [1]byte
			if _, err := io.ReadFull(r, prop[:]); err != nil {
				return nil, err
			}
			p := prop[0]
			pb = uint(p / (9 * 5))
			p %= 9 * 5
			lp = uint(p / 9)
			lc = uint(p % 9)
		}

		chunkBytes := make([]byte, packSize)
		if _, err := io.ReadFull(r, chunkBytes); err != nil {
			return nil, err
		}

		tasks = append(tasks, chunkTask{
			isRaw:      false,
			chunkData:  chunkBytes,
			unpackSize: unpackSize,
			offset:     totalUnpackSize,
			mode:       int(mode),
			lc:         lc,
			lp:         lp,
			pb:         pb,
		})
		totalUnpackSize += int(unpackSize)
	}

	finalOut := make([]byte, totalUnpackSize)

	type blockRange struct {
		startTask int
		endTask   int
	}
	var blocks []blockRange
	curBlockStart := 0
	for i := range tasks {
		t := &tasks[i]
		if i > 0 && (t.mode == 3 || (t.isRaw && t.mode == 1)) {
			blocks = append(blocks, blockRange{startTask: curBlockStart, endTask: i})
			curBlockStart = i
		}
	}
	if len(tasks) > 0 {
		blocks = append(blocks, blockRange{startTask: curBlockStart, endTask: len(tasks)})
	}

	numWorkers := min(runtime.GOMAXPROCS(0), len(blocks))
	if numWorkers <= 0 {
		numWorkers = 1
	}

	var wg sync.WaitGroup
	var blockCounter atomic.Int64
	var decErr atomic.Pointer[error]

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			core := decoderPool.Get()
			if d.DictSize > core.dictSize {
				core = NewDecoderCore(3, 0, 2, d.DictSize, 0)
			}
			defer decoderPool.Put(core)
			var rd RangeDecoder

			for {
				bIdx := int(blockCounter.Add(1) - 1)
				if bIdx >= len(blocks) || decErr.Load() != nil {
					return
				}
				blk := blocks[bIdx]
				core.winPos = 0
				core.InitProbs()

				for i := blk.startTask; i < blk.endTask; i++ {
					t := &tasks[i]
					if t.isRaw {
						if t.mode == 1 {
							core.winPos = 0
						}
						copy(finalOut[t.offset:], t.rawSlice)
						for _, b := range t.rawSlice {
							core.win[core.winPos] = b
							core.winPos = (core.winPos + 1) & core.dictMask
						}
						if t.mode == 1 {
							core.InitProbs()
						}
						continue
					}

					if t.mode == 3 {
						core.winPos = 0
					}
					if t.mode >= 2 {
						core.lc = t.lc
						core.lp = t.lp
						core.pb = t.pb
						core.posMask = (1 << t.pb) - 1
					}
					if t.mode >= 1 {
						core.InitProbs()
					}

					if err := rd.Init(bytes.NewReader(t.chunkData)); err != nil {
						decErr.Store(&err)
						return
					}

					destSlice := finalOut[t.offset : t.offset+int(t.unpackSize)]
					if _, err := core.DecodeToSlice(
						&rd,
						destSlice,
						t.unpackSize,
					); err != nil && !errors.Is(err, io.EOF) &&
						!errors.Is(err, io.ErrUnexpectedEOF) {
						decErr.Store(&err)
						return
					}
				}
			}
		}()
	}

	wg.Wait()

	if ptr := decErr.Load(); ptr != nil {
		return nil, *ptr
	}

	return io.NopCloser(bytes.NewReader(finalOut)), nil
}

// DecompressStream reads LZMA2 chunks directly from r until the 0x00 end marker is reached, returning the decompressed payload and total compressed bytes consumed.
func (d *Decompressor2) DecompressStream(r io.Reader) ([]byte, int, error) {
	type chunkTask struct {
		isRaw      bool
		rawSlice   []byte
		chunkData  []byte
		unpackSize uint64
		offset     int
		mode       int
		lc, lp, pb uint
	}

	var tasks []chunkTask
	var totalUnpackSize int
	totalRead := 0
	allIndependent := true

	var lc, lp, pb uint = 3, 0, 2
	for {
		var ctrl [1]byte
		n, err := io.ReadFull(r, ctrl[:])
		totalRead += n
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return nil, totalRead, err
		}

		control := ctrl[0]
		if control == 0x00 {
			// Stream End Marker (1 byte 0x00)
			break
		}

		if control < 0x80 {
			// Uncompressed chunk
			if control != 1 {
				allIndependent = false
			}
			var sizeBuf [2]byte
			if _, err := io.ReadFull(r, sizeBuf[:]); err != nil {
				return nil, totalRead, err
			}
			totalRead += 2
			uncompressedSize := int(binary.BigEndian.Uint16(sizeBuf[:])) + 1
			data := make([]byte, uncompressedSize)
			if _, err := io.ReadFull(r, data); err != nil {
				return nil, totalRead, err
			}
			totalRead += uncompressedSize
			tasks = append(tasks, chunkTask{
				isRaw:      true,
				rawSlice:   data,
				unpackSize: uint64(uncompressedSize),
				offset:     totalUnpackSize,
				mode:       int(control),
			})
			totalUnpackSize += uncompressedSize
			continue
		}

		mode := (control >> 5) & 3
		if mode != 3 {
			allIndependent = false
		}
		var header [4]byte
		if _, err := io.ReadFull(r, header[:]); err != nil {
			return nil, totalRead, err
		}
		totalRead += 4

		unpackSize := ((uint64(control&0x1F) << 16) | (uint64(header[0]) << 8) | uint64(header[1])) + 1
		packSize := ((int(header[2]) << 8) | int(header[3])) + 1

		if mode >= 2 {
			var prop [1]byte
			if _, err := io.ReadFull(r, prop[:]); err != nil {
				return nil, totalRead, err
			}
			totalRead += 1
			p := prop[0]
			pb = uint(p / (9 * 5))
			p %= 9 * 5
			lp = uint(p / 9)
			lc = uint(p % 9)
		}

		chunkBytes := make([]byte, packSize)
		if _, err := io.ReadFull(r, chunkBytes); err != nil {
			return nil, totalRead, err
		}
		totalRead += packSize

		tasks = append(tasks, chunkTask{
			isRaw:      false,
			chunkData:  chunkBytes,
			unpackSize: unpackSize,
			offset:     totalUnpackSize,
			mode:       int(mode),
			lc:         lc,
			lp:         lp,
			pb:         pb,
		})
		totalUnpackSize += int(unpackSize)
	}

	finalOut := make([]byte, totalUnpackSize)

	if !allIndependent {
		core := NewDecoderCore(3, 0, 2, d.DictSize, 0)
		for i := range tasks {
			t := &tasks[i]
			if t.isRaw {
				copy(finalOut[t.offset:], t.rawSlice)
				for _, b := range t.rawSlice {
					core.win[core.winPos] = b
					core.winPos = (core.winPos + 1) & core.dictMask
				}
				if t.mode == 1 {
					core.InitProbs()
				}
				continue
			}

			if t.mode >= 2 {
				core.lc = t.lc
				core.lp = t.lp
				core.pb = t.pb
				core.posMask = (1 << t.pb) - 1
			}
			if t.mode >= 1 {
				core.InitProbs()
			}
			if t.mode == 3 {
				core.winPos = 0
			}

			var rd RangeDecoder
			if err := rd.Init(bytes.NewReader(t.chunkData)); err != nil {
				return nil, totalRead, err
			}
			destSlice := finalOut[t.offset : t.offset+int(t.unpackSize)]
			if _, err := core.DecodeToSlice(&rd, destSlice, t.unpackSize); err != nil &&
				!errors.Is(err, io.EOF) &&
				!errors.Is(err, io.ErrUnexpectedEOF) {
				return nil, totalRead, err
			}
		}
	} else {
		for i := range tasks {
			if tasks[i].isRaw {
				copy(finalOut[tasks[i].offset:], tasks[i].rawSlice)
			}
		}

		numWorkers := runtime.GOMAXPROCS(0)
		if numWorkers > len(tasks) {
			numWorkers = len(tasks)
		}
		if numWorkers < 1 {
			numWorkers = 1
		}

		var wg sync.WaitGroup
		var taskCounter atomic.Int64
		var decErr atomic.Pointer[error]

		for range numWorkers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				core := decoderPool.Get()
				defer decoderPool.Put(core)
				var rd RangeDecoder

				for {
					idx := int(taskCounter.Add(1) - 1)
					if idx >= len(tasks) || decErr.Load() != nil {
						return
					}
					t := &tasks[idx]
					if t.isRaw {
						continue
					}

					core.lc = t.lc
					core.lp = t.lp
					core.pb = t.pb
					core.posMask = (1 << t.pb) - 1
					core.InitProbs()

					if err := rd.Init(bytes.NewReader(t.chunkData)); err != nil {
						decErr.Store(&err)
						return
					}

					destSlice := finalOut[t.offset : t.offset+int(t.unpackSize)]
					if _, err := core.DecodeToSlice(
						&rd,
						destSlice,
						t.unpackSize,
					); err != nil && !errors.Is(err, io.EOF) &&
						!errors.Is(err, io.ErrUnexpectedEOF) {
						decErr.Store(&err)
						return
					}
				}
			}()
		}
		wg.Wait()
		if ptr := decErr.Load(); ptr != nil {
			return nil, totalRead, *ptr
		}
	}

	return finalOut, totalRead, nil
}

// Compressor2 implements compression for LZMA2 streams.
type Compressor2 struct {
	Options Options
}

// NewCompressor2 constructs an LZMA2 compressor with default preset options.
func NewCompressor2(dictSize uint32) *Compressor2 {
	opts := DefaultOptions()
	if dictSize > 0 {
		opts.DictSize = dictSize
	}
	return &Compressor2{Options: opts}
}

// NewCompressor2WithOptions constructs an LZMA2 compressor with custom tuning options.
func NewCompressor2WithOptions(opts Options) *Compressor2 {
	if opts.DictSize == 0 {
		opts.DictSize = 8 * 1024 * 1024
	}
	if opts.ChunkSize == 0 {
		opts.ChunkSize = 64 * 1024
	}
	if opts.ChainLength == 0 || opts.FastBytes == 0 {
		preset := OptionsForLevel(opts.Level)
		if opts.ChainLength == 0 {
			opts.ChainLength = preset.ChainLength
		}
		if opts.FastBytes == 0 {
			opts.FastBytes = preset.FastBytes
		}
	}
	return &Compressor2{Options: opts}
}

// Compress reads from src and compresses into dest using LZMA2 chunk framing with continuous dictionary blocks.
// Parallel workers compress independent multi-megabyte blocks, preserving dictionary history within each block.
func (c *Compressor2) Compress(src io.Reader, dest io.Writer) (int64, error) {
	numWorkers := runtime.GOMAXPROCS(0)
	if numWorkers <= 0 {
		numWorkers = 1
	}

	blockSize := int(c.Options.DictSize)
	if numWorkers > 1 {
		if blockSize > 2*1024*1024 {
			blockSize = 2 * 1024 * 1024
		}
	}
	if blockSize < 512*1024 {
		blockSize = 512 * 1024
	}
	chunkSize := 64 * 1024

	batchBufSize := numWorkers * blockSize
	if batchBufSize < 8*1024*1024 {
		batchBufSize = 8 * 1024 * 1024
	}
	batchBuf := make([]byte, batchBufSize)

	type encodedChunk struct {
		isRaw     bool
		mode      byte
		rawSlice  []byte
		compBytes []byte
		chunkLen  int
	}

	type blockResult struct {
		chunks []encodedChunk
	}

	var totalWritten int64
	var totalProcessedChunks int64
	var totalUncompressedBytes int64
	firstBatch := true

	for {
		n, err := io.ReadFull(src, batchBuf)
		if n == 0 {
			if firstBatch {
				if _, err := dest.Write([]byte{0x00}); err != nil {
					return 0, err
				}
				return 0, nil
			}
			break
		}
		firstBatch = false
		totalUncompressedBytes += int64(n)
		rawBatch := batchBuf[:n]

		numBlocks := (n + blockSize - 1) / blockSize
		results := make([]blockResult, numBlocks)

		var wg sync.WaitGroup
		var blockCounter atomic.Int64
		var compErr atomic.Pointer[error]

		workersForBatch := min(numWorkers, numBlocks)
		for w := 0; w < workersForBatch; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				core := NewEncoderCoreWithOptions(c.Options)
				var re RangeEncoder
				var compBuf bytes.Buffer

				for {
					bIdx := int(blockCounter.Add(1) - 1)
					if bIdx >= numBlocks || compErr.Load() != nil {
						return
					}

					blockStart := bIdx * blockSize
					blockEnd := min(blockStart+blockSize, n)
					blockData := rawBatch[blockStart:blockEnd]
					blockLen := blockEnd - blockStart

					numChunksInBlock := (blockLen + chunkSize - 1) / chunkSize
					chunks := make([]encodedChunk, 0, numChunksInBlock)

					for cIdx := 0; cIdx < numChunksInBlock; cIdx++ {
						cStart := cIdx * chunkSize
						cEnd := min(cStart+chunkSize, blockLen)
						cLen := cEnd - cStart
						rawSlice := blockData[cStart:cEnd]

						compBuf.Reset()
						re.Init(&compBuf)

						mode := byte(1)
						if cIdx == 0 {
							mode = 3
							core.Reset()
						} else {
							core.ResetStateKeepDict()
						}

						if err := core.EncodeChunk(blockData, cStart, cEnd, &re); err != nil {
							compErr.Store(&err)
							return
						}
						if err := re.Flush(); err != nil {
							compErr.Store(&err)
							return
						}

						compBytes := compBuf.Bytes()
						if len(compBytes) >= cLen || len(compBytes) > 65536 {
							rawCopy := make([]byte, cLen)
							copy(rawCopy, rawSlice)
							chunks = append(chunks, encodedChunk{
								isRaw:    true,
								mode:     mode,
								rawSlice: rawCopy,
								chunkLen: cLen,
							})
						} else {
							chunkCopy := make([]byte, len(compBytes))
							copy(chunkCopy, compBytes)
							chunks = append(chunks, encodedChunk{
								isRaw:     false,
								mode:      mode,
								compBytes: chunkCopy,
								chunkLen:  cLen,
							})
						}
					}
					results[bIdx] = blockResult{chunks: chunks}
				}
			}()
		}

		wg.Wait()

		if ptr := compErr.Load(); ptr != nil {
			return totalUncompressedBytes, *ptr
		}

		// Write completed blocks and chunks directly to dest
		for bIdx := 0; bIdx < numBlocks; bIdx++ {
			for _, res := range results[bIdx].chunks {
				if res.isRaw {
					var uncompHeader [3]byte
					if res.mode == 3 {
						uncompHeader[0] = 0x01 // reset dict
					} else {
						uncompHeader[0] = 0x02 // keep dict
					}
					uncompHeader[1] = byte((res.chunkLen - 1) >> 8)
					uncompHeader[2] = byte(res.chunkLen - 1)
					if _, err := dest.Write(uncompHeader[:]); err != nil {
						return totalUncompressedBytes, err
					}
					totalWritten += 3
					if _, err := dest.Write(res.rawSlice); err != nil {
						return totalUncompressedBytes, err
					}
					totalWritten += int64(len(res.rawSlice))
				} else {
					unpackSizeMinus1 := uint32(res.chunkLen - 1)
					packSizeMinus1 := uint16(len(res.compBytes) - 1)

					if res.mode == 3 {
						var control byte = 0x80 | (3 << 5) // mode 3
						control |= byte((unpackSizeMinus1 >> 16) & 0x1F)

						var chunkHeader [6]byte
						chunkHeader[0] = control
						chunkHeader[1] = byte(unpackSizeMinus1 >> 8)
						chunkHeader[2] = byte(unpackSizeMinus1)
						chunkHeader[3] = byte(packSizeMinus1 >> 8)
						chunkHeader[4] = byte(packSizeMinus1)
						chunkHeader[5] = byte((c.Options.Pb*5+c.Options.Lp)*9 + c.Options.Lc)

						if _, err := dest.Write(chunkHeader[:]); err != nil {
							return totalUncompressedBytes, err
						}
						totalWritten += 6
					} else {
						var control byte = 0x80 | (1 << 5) // mode 1
						control |= byte((unpackSizeMinus1 >> 16) & 0x1F)

						var chunkHeader [5]byte
						chunkHeader[0] = control
						chunkHeader[1] = byte(unpackSizeMinus1 >> 8)
						chunkHeader[2] = byte(unpackSizeMinus1)
						chunkHeader[3] = byte(packSizeMinus1 >> 8)
						chunkHeader[4] = byte(packSizeMinus1)

						if _, err := dest.Write(chunkHeader[:]); err != nil {
							return totalUncompressedBytes, err
						}
						totalWritten += 5
					}

					if _, err := dest.Write(res.compBytes); err != nil {
						return totalUncompressedBytes, err
					}
					totalWritten += int64(len(res.compBytes))
				}

				totalProcessedChunks++
				if c.Options.OnProgress != nil {
					c.Options.OnProgress(totalProcessedChunks, totalProcessedChunks)
				}
			}
		}

		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			return totalUncompressedBytes, err
		}
	}

	// Write LZMA2 stream terminator (0x00)
	if _, err := dest.Write([]byte{0x00}); err != nil {
		return totalUncompressedBytes, err
	}
	totalWritten++
	_ = totalWritten

	return totalUncompressedBytes, nil
}
