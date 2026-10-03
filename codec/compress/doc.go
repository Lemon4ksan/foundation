// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package compress provides a unified, zero-allocation multi-algorithm compression engine.
//
// The package acts as a high-performance substrate for compressing and decompressing
// data streams using various codecs (gzip, zstd, brotli, deflate, lz4, lzma). It exposes
// transparent decompression mechanisms that automatically detect the payload format
// and decode it into pre-allocated destination slices.
//
// To protect against resource exhaustion attacks (decompression bombs), the engine
// enforces maximum uncompressed payload thresholds and amplification limits. It leverages
// thread-local CPU storage for engines like zstd, gzip, and deflate, entirely avoiding
// lock contention and garbage collection overhead during concurrent stream processing.
// Use [Decompress] and [DecompressScoped] for payload extraction.
//
// Sub-packages contain specific algorithmic codecs, while this package provides
// the shared abstractions and format multiplexing.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: unified transparent decompression and streaming safety limits.
//
// Rejected compromise: stdlib decoders allocate buffers and structs per stream and do not limit expansion ratios out of the box, pushing safety and GC costs to the user.
//
// Accepted cost: increased binary size by statically linking multiple compression engines.
//
// Allocations: amortized
package compress
