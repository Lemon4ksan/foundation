// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"errors"
	"io"
	"iter"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// Delim is a JSON delimiter character: one of '{', '}', '[', or ']'.
type Delim rune

// String returns the string representation of delimiter d.
func (d Delim) String() string { return string(d) }

// Token represents a JSON token: Delim, bool, float64, Number, string, or nil.
type Token = any

// ObjectEntries returns a push iterator yielding key-value pairs from a valid JSON object byte slice.
// Values are returned as [RawMessage] referencing the underlying slice with zero allocations.
// Iteration halts if data is malformed, not an object, or if yield returns false.
func ObjectEntries(data []byte) iter.Seq2[string, RawMessage] {
	return func(yield func(string, RawMessage) bool) {
		cursor := skipWhitespace(data, 0)
		if cursor >= len(data) || data[cursor] != '{' {
			return
		}
		cursor++

		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) || data[cursor] == '}' {
				return
			}
			if !first {
				if data[cursor] != ',' {
					return
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
			}
			first = false

			if cursor >= len(data) || data[cursor] != '"' {
				return
			}

			keyRaw, newCursor, hasEscape, err := scanString(data, cursor)
			if err != nil {
				return
			}
			cursor = newCursor

			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) || data[cursor] != ':' {
				return
			}
			cursor++

			valStart := skipWhitespace(data, cursor)
			valEnd, err := skipValue(data, valStart)
			if err != nil {
				return
			}
			cursor = valEnd

			var key string
			if hasEscape {
				unescaped, uerr := unescapeString(nil, keyRaw)
				if uerr != nil {
					return
				}
				key = string(unescaped)
			} else {
				key = bytesconv.B2S(keyRaw)
			}

			if !yield(key, RawMessage(data[valStart:valEnd])) {
				return
			}
		}
	}
}

// ArrayElements returns a push iterator yielding elements and their indices from a JSON array byte slice.
// Values are returned as [RawMessage] referencing the underlying slice with zero allocations.
// Iteration halts if data is malformed, not an array, or if yield returns false.
func ArrayElements(data []byte) iter.Seq2[int, RawMessage] {
	return func(yield func(int, RawMessage) bool) {
		cursor := skipWhitespace(data, 0)
		if cursor >= len(data) || data[cursor] != '[' {
			return
		}
		cursor++

		first := true
		idx := 0
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) || data[cursor] == ']' {
				return
			}
			if !first {
				if data[cursor] != ',' {
					return
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
			}
			first = false

			valStart := cursor
			valEnd, err := skipValue(data, valStart)
			if err != nil {
				return
			}
			cursor = valEnd

			if !yield(idx, RawMessage(data[valStart:valEnd])) {
				return
			}
			idx++
		}
	}
}

// DecodeSeq returns a push iterator yielding successfully decoded values of type T from r.
// Iteration stops upon reaching io.EOF or encountering a decoding error.
func DecodeSeq[T any](r io.Reader) iter.Seq2[T, error] {
	dec := NewDecoder(r)
	return func(yield func(T, error) bool) {
		for {
			var v T
			err := dec.Decode(&v)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				yield(v, err)
				return
			}
			if !yield(v, nil) {
				return
			}
		}
	}
}
