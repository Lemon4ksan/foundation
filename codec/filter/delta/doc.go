// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package delta implements the byte distance prediction filter (Delta) used in 7-Zip archives.
//
// The Delta filter converts sequence differences (e.g. in uncompressed audio PCM, table columns,
// RGB bitmap planes) into small delta values, dramatically boosting subsequent entropy compression.
// It supports both in-place byte slice transformation via [Filter] and streaming processing
// via [Reader] and [Writer].
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: numerical delta sequence pre-filters.
//
// Rejected compromise: standard library compression acts only on raw byte streams, yielding poor ratios for structured numeric or multimedia data that compresses better as deltas.
//
// Accepted cost: callers must know the precise column or sample width (distance) of their payload ahead of time.
//
// Allocations: amortized
package delta
