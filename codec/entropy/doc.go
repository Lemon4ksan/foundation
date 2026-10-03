// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package entropy provides Shannon entropy calculation and incompressibility detection heuristics.
//
// Entropy helps avoid wasted CPU cycles by detecting data that is already compressed or
// encrypted before passing it to an expensive compression engine.
// [ShannonEntropy] calculates the entropy of a byte sample, and [IsIncompressibleSample]
// provides a quick heuristic returning true if the entropy indicates the payload cannot be compressed.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: heuristic data analysis
//
// Rejected compromise: Standard library provides no tools to prevent attempting compression on already-dense data.
//
// Accepted cost: Requires one full pass over the sample data before deciding on compression.
//
// Allocations: zero
package entropy
