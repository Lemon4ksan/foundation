// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"errors"
)

// DefaultMaxDepth is the default maximum nesting depth allowed for recursive message traversal.
const DefaultMaxDepth = 64

// DefaultMaxFields is the default maximum total fields visited during recursive traversal
// to prevent exponential recursion blowup (Protobuf Billion Laughs / entity expansion).
const DefaultMaxFields = 100_000

var (
	// ErrMaxDepthExceeded is returned when message nesting exceeds the configured maximum depth.
	ErrMaxDepthExceeded = errors.New("proto: recursion depth limit exceeded")

	// ErrRecursionBudgetExceeded is returned when the total number of traversed fields
	// exceeds the budget, preventing exponential Billion Laughs attacks.
	ErrRecursionBudgetExceeded = errors.New("proto: recursion field budget exceeded")

	// ErrStop stops message traversal early without an error.
	ErrStop = errors.New("proto: stop iteration")

	// ErrSkipSubmessage skips descending into the current submessage.
	ErrSkipSubmessage = errors.New("proto: skip submessage")
)

// MessageVisitor is called for each field during recursive message scanning.
// depth indicates the current nesting depth (0 for root message fields).
// Return recurse = true to descend into f.Message() if f.Type == WireBytes.
// Return err != nil to abort traversal (or return ErrStop to finish cleanly, ErrSkipSubmessage to skip this submessage).
type MessageVisitor func(depth int, f Field) (recurse bool, err error)

// ScanMessage recursively scans data up to maxDepth using the provided visitor.
// If maxDepth <= 0, DefaultMaxDepth (64) is used.
func ScanMessage(data []byte, maxDepth int, visit MessageVisitor) error {
	if maxDepth <= 0 {
		maxDepth = DefaultMaxDepth
	}
	budget := DefaultMaxFields
	err := scanMessageDepth(data, 0, maxDepth, &budget, visit)
	if errors.Is(err, ErrStop) {
		return nil
	}
	return err
}

func scanMessageDepth(data []byte, depth, maxDepth int, budget *int, visit MessageVisitor) error {
	for f, err := range Scan(data) {
		if err != nil {
			return err
		}
		*budget--
		if *budget <= 0 {
			return ErrRecursionBudgetExceeded
		}
		recurse, visitErr := visit(depth, f)
		if visitErr != nil {
			if errors.Is(visitErr, ErrStop) {
				return ErrStop
			}
			if errors.Is(visitErr, ErrSkipSubmessage) {
				continue
			}
			return visitErr
		}
		if recurse && f.Type == WireBytes {
			if depth >= maxDepth {
				return ErrMaxDepthExceeded
			}
			if err := scanMessageDepth(f.Data, depth+1, maxDepth, budget, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

// WalkMessage recursively scans data, visiting all fields.
// shouldRecurse determines if a WireBytes field should be descended into.
func WalkMessage(data []byte, shouldRecurse func(depth int, f Field) bool, visit func(depth int, f Field) error) error {
	return ScanMessage(data, DefaultMaxDepth, func(depth int, f Field) (bool, error) {
		if err := visit(depth, f); err != nil {
			return false, err
		}
		if f.Type == WireBytes && shouldRecurse != nil {
			return shouldRecurse(depth, f), nil
		}
		return false, nil
	})
}

// ScanMessage recursively scans this field as an embedded submessage.
func (f Field) ScanMessage(maxDepth int, visit MessageVisitor) error {
	if f.Type != WireBytes {
		return ErrInvalidWireType
	}
	return ScanMessage(f.Data, maxDepth, visit)
}
