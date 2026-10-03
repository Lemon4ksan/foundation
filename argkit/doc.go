// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package argkit provides a zero-dependency POSIX argument and flag parsing engine.
//
// It supports short flag clumping (e.g., -la expands to -l and -a), attached values
// (-o=output.bin), interspersed arguments and flags, and shell token normalization.
// It also provides fuzzy typo suggestions for unknown options.
//
// # Compared to the standard library
//
// Stdlib counterpart: flag
//
// Rejected compromise: The flag package stops parsing flags at the first positional argument.
//
// Accepted cost: Parsing requires more allocations to track interspersed state and normalize tokens.
//
// Allocations: unconstrained
package argkit
