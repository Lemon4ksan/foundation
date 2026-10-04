// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package fskit provides high-performance filesystem primitives.
//
// It includes concurrent multi-threaded directory traversal ([FastWalk], [FastWalkSeq]),
// push iterator directory traversal ([WalkSeq]), cross-platform memory-mapped I/O ([OpenMmap]),
// atomic file replacement ([AtomicWriteFile]), bounded file ingestion ([ReadFileLimited]),
// and zero-allocation file media heuristics ([IsIncompressibleExtension]).
//
// # Compared to the standard library
//
// Stdlib counterpart: io/fs, path/filepath, and os
//
// Rejected compromise: filepath.Walk and filepath.WalkDir traverse directories sequentially.
//
// Accepted cost: FastWalk yields paths out of order and spawns goroutines.
//
// Allocations: unconstrained
package fskit
