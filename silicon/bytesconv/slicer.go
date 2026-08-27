// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv

import (
	"bytes"
)

// PatternSlicer is a zero-allocation byte slice pattern matcher and splitter.
// It searches for pattern in data and splits the byte slice at pattern + offset.
type PatternSlicer struct {
	Pattern []byte
	Offset  int
}

// NewPatternSlicer creates a new [PatternSlicer].
func NewPatternSlicer(pattern []byte, offset int) *PatternSlicer {
	return &PatternSlicer{
		Pattern: pattern,
		Offset:  offset,
	}
}

// findIndex locates the byte offset of s.Pattern in data using vector-accelerated SIMD routines.
func (s *PatternSlicer) findIndex(data []byte) int {
	if len(s.Pattern) == 1 {
		return bytes.IndexByte(data, s.Pattern[0])
	}

	return bytes.Index(data, s.Pattern)
}

// Slice finds the first occurrence of Pattern in data, splitting data into two parts at Index + Offset.
// Returns [][]byte{data[:splitPoint], data[splitPoint:]} and true if matched, or [][]byte{data} and false if not matched.
func (s *PatternSlicer) Slice(data []byte) ([][]byte, bool) {
	if len(data) == 0 || len(s.Pattern) == 0 {
		return [][]byte{data}, false
	}

	idx := s.findIndex(data)
	if idx == -1 {
		return [][]byte{data}, false
	}

	splitPoint := idx + s.Offset
	if splitPoint <= 0 || splitPoint >= len(data) {
		return [][]byte{data}, false
	}

	return [][]byte{data[:splitPoint], data[splitPoint:]}, true
}

// SliceAll recursively splits data on every match of Pattern + Offset.
func (s *PatternSlicer) SliceAll(data []byte) [][]byte {
	return s.SliceAllInto(data, nil)
}

// SliceInto finds the first occurrence of Pattern in data, splitting into pre-allocated dst slice.
// If matched, dst is resliced and populated with [data[:splitPoint], data[splitPoint:]], returning (dst, true).
// If not matched, dst is populated with [data], returning (dst, false).
func (s *PatternSlicer) SliceInto(data []byte, dst [][]byte) ([][]byte, bool) {
	dst = dst[:0]
	if len(data) == 0 || len(s.Pattern) == 0 {
		return append(dst, data), false
	}

	idx := s.findIndex(data)
	if idx == -1 {
		return append(dst, data), false
	}

	splitPoint := idx + s.Offset
	if splitPoint <= 0 || splitPoint >= len(data) {
		return append(dst, data), false
	}

	return append(dst, data[:splitPoint], data[splitPoint:]), true
}

// SliceAllInto recursively splits data into pre-allocated dst slice on every match of Pattern + Offset.
func (s *PatternSlicer) SliceAllInto(data []byte, dst [][]byte) [][]byte {
	dst = dst[:0]
	if len(data) == 0 || len(s.Pattern) == 0 {
		return append(dst, data)
	}

	curr := data

	for len(curr) > 0 {
		idx := s.findIndex(curr)
		if idx == -1 {
			dst = append(dst, curr)
			break
		}

		splitPoint := idx + s.Offset
		if splitPoint <= 0 || splitPoint >= len(curr) {
			dst = append(dst, curr)
			break
		}

		dst = append(dst, curr[:splitPoint])
		curr = curr[splitPoint:]
	}

	return dst
}
