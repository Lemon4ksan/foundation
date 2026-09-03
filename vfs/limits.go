// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
)

// ResourceLimits defines runtime constraints for archive extraction.
type ResourceLimits struct {
	MaxTotalSize int64
	MaxFileSize  int64
	MaxRatio     float64
}

// ErrResourceLimit indicates that extraction exceeded configured size or ratio thresholds.
var ErrResourceLimit = errors.New("vfs: resource limit exceeded")

const (
	defaultGracePeriod = 10 * 1024 * 1024 // 10 MB grace period before ratio enforcement
	defaultRatioBuffer = 4096             // Buffer to smooth ratio on small payloads
)

// SecureWriter wraps an [io.Writer] to enforce file size, total uncompressed size, and decompression ratio limits.
type SecureWriter struct {
	writer         io.Writer
	written        int64
	totalWritten   *int64
	maxFileSize    int64
	maxTotalSize   int64
	maxRatio       float64
	compressedSize int64
}

// NewSecureWriter constructs a rate and size constrained writer.
func NewSecureWriter(w io.Writer, limits ResourceLimits, compressedSize int64, globalTotal *int64) *SecureWriter {
	return &SecureWriter{
		writer:         w,
		totalWritten:   globalTotal,
		maxFileSize:    limits.MaxFileSize,
		maxTotalSize:   limits.MaxTotalSize,
		maxRatio:       limits.MaxRatio,
		compressedSize: compressedSize,
	}
}

// Write writes bytes while evaluating resource limits.
func (sw *SecureWriter) Write(p []byte) (int, error) {
	n := len(p)
	writeLen := int64(n)

	if sw.totalWritten != nil && sw.maxTotalSize > 0 {
		newTotal := atomic.AddInt64(sw.totalWritten, writeLen)
		if newTotal > sw.maxTotalSize {
			atomic.AddInt64(sw.totalWritten, -writeLen)
			return 0, fmt.Errorf("%w: global extraction limit %d bytes exceeded", ErrResourceLimit, sw.maxTotalSize)
		}
	}

	if sw.maxFileSize > 0 {
		if sw.written+writeLen > sw.maxFileSize {
			return 0, fmt.Errorf("%w: entry limit %d bytes exceeded", ErrResourceLimit, sw.maxFileSize)
		}
	}

	if sw.maxRatio > 0 && (sw.written+writeLen) > defaultGracePeriod {
		denominator := float64(sw.compressedSize + defaultRatioBuffer)
		ratio := float64(sw.written+writeLen) / denominator
		if ratio > sw.maxRatio {
			return 0, fmt.Errorf(
				"%w: decompression ratio %.1fx exceeds threshold %.1fx (possible archive bomb)",
				ErrResourceLimit,
				ratio,
				sw.maxRatio,
			)
		}
	}

	n, err := sw.writer.Write(p)
	sw.written += int64(n)
	return n, err
}
