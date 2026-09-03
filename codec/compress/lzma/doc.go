// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lzma implements LZMA1 and LZMA2 compression and decompression in pure Go.
//
// # Overview
//
// LZMA (Lempel-Ziv-Markov chain Algorithm) combines dictionary-based LZ77 match substitution
// with an adaptive binary arithmetic range coder. Unlike Huffman-based codecs (such as Deflate)
// that emit whole integer bits per symbol, LZMA computes fractional bit lengths by maintaining
// an arithmetic interval [Low, Low + Range) driven by dynamic probability models.
//
// # Range Coder Mechanics
//
// The arithmetic range coder uses 11-bit precision for probabilities (0 to 2048, with 1024
// representing an equal 50% probability). Probabilities are updated after each decision using
// a 5-bit shift (1/32 weight):
//
//	Bit 0: prob += (2048 - prob) >> 5
//	Bit 1: prob -= prob >> 5
//
// When the interval size drops below 2^24, 8 bits are shifted to the output stream. In the encoder,
// potential carries across runs of 0xFF bytes are buffered and resolved via shiftLow.
//
// # Markov State Machine
//
// The encoder and decoder track context using a 12-state automaton:
//   - States 0..3: Consecutive literal bytes.
//   - States 4..6: Literal bytes following a match or repetition.
//   - State 7: Regular dictionary match.
//   - State 8: Repetition match using reps[1..3].
//   - State 9: 1-byte repetition match using reps[0] (ShortRep).
//   - State 10: Repetition match following a literal.
//   - State 11: 1-byte repetition match following a literal.
//
// Whether a state is character-dominant (state < 7) determines the literal decoding pathway.
//
// # Stream Properties (lc, lp, pb)
//
// LZMA streams are configured by three bit-depth properties:
//   - lc (literal context bits, default 3): High bits of previous byte selecting literal sub-models.
//   - lp (literal position bits, default 0): Low bits of current position selecting literal sub-models.
//   - pb (position bits, default 2): Low bits of current position for match/literal decisions.
//
// # Repetition Registers
//
// The last 4 match distances are cached in reps[0..3]. Matches matching these distances
// are encoded via dedicated repetition flags (isRep, isRepG0, isRepG1, isRepG2), avoiding full
// distance coding. If reps[0] matches a single byte, it is emitted as a ShortRep (approx. 4 bits).
//
// # Engine Implementation
//
// This implementation contains several performance-oriented design choices:
//   - Register pinning: Inner decoding and encoding loops hold range coder state in local scalars,
//     allowing the compiler to keep them in CPU registers throughout stream processing.
//   - Branchless literal decoding: The 8-bit literal extraction loop for state < 7 is unrolled
//     to avoid loop branching and pipeline stalls.
//   - Bounds-check elimination: Multi-dimensional probability array lookups use pointer arithmetic
//     to eliminate Go runtime bounds checks on critical paths.
//   - Buffer recycling: Encoder and decoder structures are pooled per logical processor (PerPStorage)
//     to prevent heap allocations during multi-threaded operation.
package lzma
