// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zstd

// EncoderLevel defines the compression performance and ratio tradeoff.
type EncoderLevel int

const (
	// SpeedFastest represents fastest compression speed with lower ratio (Level 1).
	SpeedFastest EncoderLevel = 1
	// SpeedDefault represents standard balanced compression (Level 3).
	SpeedDefault EncoderLevel = 3
	// SpeedBetterCompression provides higher compression ratio (Level 7).
	SpeedBetterCompression EncoderLevel = 7
	// SpeedBestCompression provides maximum compression ratio (Level 11).
	SpeedBestCompression EncoderLevel = 11
)

// EncoderOption configures the Zstandard encoder.
type EncoderOption func(*encoderOptions) error

type encoderOptions struct {
	level       EncoderLevel
	windowSize  uint64
	checksum    bool
	concurrency int
}

func defaultEncoderOptions() encoderOptions {
	return encoderOptions{
		level:       SpeedDefault,
		windowSize:  4 * 1024 * 1024, // 4MB default window
		checksum:    true,
		concurrency: 1,
	}
}

// WithEncoderLevel sets the compression level.
func WithEncoderLevel(level EncoderLevel) EncoderOption {
	return func(o *encoderOptions) error {
		if level < SpeedFastest {
			level = SpeedFastest
		}
		if level > SpeedBestCompression {
			level = SpeedBestCompression
		}
		o.level = level
		return nil
	}
}

// WithEncoderChecksum enables or disables xxHash64 checksum at the end of the frame.
func WithEncoderChecksum(enabled bool) EncoderOption {
	return func(o *encoderOptions) error {
		o.checksum = enabled
		return nil
	}
}

// WithEncoderWindowSize configures the sliding window buffer size.
func WithEncoderWindowSize(size uint64) EncoderOption {
	return func(o *encoderOptions) error {
		if size < MinWindowSize {
			size = MinWindowSize
		}
		if size > MaxWindowSize {
			size = MaxWindowSize
		}
		o.windowSize = size
		return nil
	}
}

// WithEncoderConcurrency sets the number of parallel workers.
func WithEncoderConcurrency(n int) EncoderOption {
	return func(o *encoderOptions) error {
		if n <= 0 {
			n = 1
		}
		o.concurrency = n
		return nil
	}
}
