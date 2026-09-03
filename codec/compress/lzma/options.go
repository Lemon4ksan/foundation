// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma

// Level represents an LZMA compression level preset.
type Level int

const (
	// LevelFastest optimizes for maximum throughput (16 hash chain candidates, 32-byte match limit).
	LevelFastest Level = 1

	// LevelFast provides high throughput with moderate compression (32 candidates, 48-byte match limit).
	LevelFast Level = 3

	// LevelNormal represents the default balanced compression preset (64 candidates, 64-byte match limit).
	LevelNormal Level = 5

	// LevelMaximum optimizes for high compression density (128 candidates, 128-byte match limit).
	LevelMaximum Level = 7

	// LevelUltra achieves maximum compression ratio (256 candidates, 273-byte maximum match limit).
	LevelUltra Level = 9
)

// Options specifies granular tuning parameters for LZMA/LZMA2 compression.
type Options struct {
	Level       Level                        // Preset compression level (1..9, default 5)
	DictSize    uint32                       // Dictionary window size in bytes (e.g. 64KB to 1GB, default 8MB)
	ChainLength int                          // Match finder hash chain search depth (16..256)
	FastBytes   int                          // Match length threshold before early termination (32..273)
	ChunkSize   int                          // LZMA2 chunk partition size in bytes (default 64KB)
	Lc          uint                         // Literal context bits (0..8, default 3)
	Lp          uint                         // Literal position bits (0..4, default 0)
	Pb          uint                         // Position bits (0..4, default 2)
	OnProgress  func(processed, total int64) // Optional progress reporting callback
}

// DefaultOptions returns the standard balanced LZMA2 options preset.
func DefaultOptions() Options {
	return OptionsForLevel(LevelNormal)
}

// OptionsForLevel constructs preset options for the given compression level.
func OptionsForLevel(lvl Level) Options {
	opts := Options{
		Level:     lvl,
		DictSize:  8 * 1024 * 1024,
		ChunkSize: 2 * 1024 * 1024,
		Lc:        3,
		Lp:        0,
		Pb:        2,
	}

	switch {
	case lvl <= LevelFastest:
		opts.Level = LevelFastest
		opts.DictSize = 1 * 1024 * 1024
		opts.ChainLength = 8
		opts.FastBytes = 32
		opts.ChunkSize = 512 * 1024
	case lvl <= LevelFast:
		opts.Level = LevelFast
		opts.DictSize = 2 * 1024 * 1024
		opts.ChainLength = 16
		opts.FastBytes = 32
		opts.ChunkSize = 1 * 1024 * 1024
	case lvl <= LevelNormal:
		opts.Level = LevelNormal
		opts.DictSize = 8 * 1024 * 1024
		opts.ChainLength = 32
		opts.FastBytes = 64
		opts.ChunkSize = 2 * 1024 * 1024
	case lvl <= LevelMaximum:
		opts.Level = LevelMaximum
		opts.DictSize = 16 * 1024 * 1024
		opts.ChainLength = 48
		opts.FastBytes = 96
		opts.ChunkSize = 2 * 1024 * 1024
	default:
		opts.Level = LevelUltra
		opts.DictSize = 32 * 1024 * 1024
		opts.ChainLength = 96
		opts.FastBytes = 128
		opts.ChunkSize = 2 * 1024 * 1024
	}

	return opts
}
