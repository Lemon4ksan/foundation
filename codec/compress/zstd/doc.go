// Copyright (c) 2019-2023 Klaus Post. All rights reserved.
// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package zstd provides decompression and compression of Zstandard streams.
//
// Zstandard (RFC 8878) is a real-time compression algorithm, providing high compression
// ratios. It offers a very wide range of compression/speed trade-offs, while being backed
// by a very fast decoder.
//
// # Performance
//
// This package is heavily optimized for speed and low allocation overhead, making extensive
// use of assembly routines on amd64 and arm64 for performance-critical components.
//
// Decoders and encoders are designed to be reused to prevent garbage collection pressure.
// Under the hood, decoders utilize thread-local CPU storage (pool.PerPStorage) to avoid
// lock contention and heap allocations during multi-threaded operation.
//
// # Dictionary Compression
//
// The package supports shared dictionary compression for situations where many small payloads
// with similar structure are compressed independently. Dictionaries can be pre-computed to
// drastically improve compression ratios.
//
// VENDORED: Originates from github.com/klauspost/compress/zstd.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: Zstandard compression format
//
// Rejected compromise: No stdlib Zstd; third-party packages allocate per decoder and rely on heavy concurrency primitives.
//
// Accepted cost: Large codebase size, dependency on architecture-specific assembly.
//
// Allocations: amortized
package zstd
