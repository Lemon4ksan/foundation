// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"errors"
	"fmt"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
	"github.com/lemon4ksan/foundation/silicon/simd"
)

var (
	errUnexpectedEnd  = errors.New("json: unexpected end of JSON input")
	errInvalidChar    = errors.New("json: invalid character in input")
	errInvalidEscape  = errors.New("json: invalid escape sequence in string")
	errInvalidUnicode = errors.New("json: invalid unicode surrogate pair")
)

// skipWhitespace advances cursor past any ASCII whitespace characters (' ', '\t', '\n', '\r').
//
//go:inline
func skipWhitespace(data []byte, cursor int) int {
	return skipWhitespaceVector(data, cursor)
}

func skipWhitespaceScalar(data []byte, cursor int) int {
	n := len(data)
	for cursor < n {
		b := data[cursor]
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			cursor++
			continue
		}

		return cursor
	}

	return cursor
}

func scanStringSpecialScalar(data []byte, cursor int) int {
	idx := simd.IndexByteTwoSWAR(data[cursor:], '"', '\\')
	if idx == -1 {
		return -1
	}
	return cursor + idx
}

// scanString extracts a JSON string token starting at cursor (which must be '"').
// Returns the raw string content (without quotes), the new cursor position after the closing quote,
// and a boolean indicating whether the string contained escape characters.
func scanString(data []byte, cursor int) (raw []byte, newCursor int, hasEscape bool, err error) {
	if cursor >= len(data) || data[cursor] != '"' {
		return nil, cursor, false, errInvalidChar
	}

	start := cursor + 1
	i := start
	n := len(data)

	for i < n {
		pos := scanStringSpecialVector(data, i)
		if pos == -1 {
			return nil, n, false, errUnexpectedEnd
		}

		if data[pos] == '"' {
			return data[start:pos], pos + 1, hasEscape, nil
		}

		// Escape character '\' found
		hasEscape = true
		pos++ // skip '\'
		if pos >= n {
			return nil, n, true, errUnexpectedEnd
		}

		if data[pos] == 'u' {
			pos += 4 // \uXXXX has 4 hex digits
			if pos >= n {
				return nil, n, true, errUnexpectedEnd
			}
		}

		i = pos + 1
	}

	return nil, n, false, errUnexpectedEnd
}

// unescapeString decodes JSON escape sequences from src into dst.
func unescapeString(dst []byte, src []byte) ([]byte, error) {
	n := len(src)
	for i := 0; i < n; {
		if src[i] != '\\' {
			dst = append(dst, src[i])
			i++
			continue
		}

		i++ // skip '\'
		if i >= n {
			return dst, errInvalidEscape
		}

		switch src[i] {
		case '"', '\\', '/':
			dst = append(dst, src[i])
			i++
		case 'b':
			dst = append(dst, '\b')
			i++
		case 'f':
			dst = append(dst, '\f')
			i++
		case 'n':
			dst = append(dst, '\n')
			i++
		case 'r':
			dst = append(dst, '\r')
			i++
		case 't':
			dst = append(dst, '\t')
			i++
		case 'u':
			if i+4 >= n {
				return dst, errInvalidEscape
			}

			r, err := parseHex4(src[i+1 : i+5])
			if err != nil {
				return dst, err
			}

			i += 5

			// Check for UTF-16 surrogate pair
			if utf16.IsSurrogate(r) {
				if i+5 < n && src[i] == '\\' && src[i+1] == 'u' {
					r2, err := parseHex4(src[i+2 : i+6])
					if err == nil {
						combined := utf16.DecodeRune(r, r2)
						if combined != utf8.RuneError {
							var buf [utf8.UTFMax]byte
							written := utf8.EncodeRune(buf[:], combined)
							dst = append(dst, buf[:written]...)
							i += 6
							continue
						}
					}
				}

				r = utf8.RuneError
			}

			var buf [utf8.UTFMax]byte
			written := utf8.EncodeRune(buf[:], r)
			dst = append(dst, buf[:written]...)
		default:
			return dst, errInvalidEscape
		}
	}

	return dst, nil
}

// parseHex4 decodes a 4-digit hexadecimal integer.
func parseHex4(b []byte) (rune, error) {
	if len(b) < 4 {
		return 0, errInvalidEscape
	}

	var r rune
	for i := 0; i < 4; i++ {
		c := b[i]
		r <<= 4
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			r |= rune(c - 'A' + 10)
		default:
			return 0, errInvalidEscape
		}
	}

	return r, nil
}

// scanNumber extracts the raw numeric token starting at cursor.
func scanNumber(data []byte, cursor int) (numBytes []byte, newCursor int, isFloat bool, err error) {
	n := len(data)
	if cursor >= n {
		return nil, cursor, false, errUnexpectedEnd
	}

	start := cursor
	if data[cursor] == '-' {
		cursor++
		if cursor >= n {
			return nil, n, false, errUnexpectedEnd
		}
	}

	for cursor < n {
		b := data[cursor]
		if b >= '0' && b <= '9' {
			cursor++
			continue
		}

		if b == '.' || b == 'e' || b == 'E' || b == '+' || b == '-' {
			isFloat = true
			cursor++
			continue
		}

		break
	}

	if cursor == start || (cursor == start+1 && data[start] == '-') {
		return nil, cursor, false, errInvalidChar
	}

	return data[start:cursor], cursor, isFloat, nil
}

// parseInt parses signed 64-bit integer from raw bytes without allocations.
func parseInt(b []byte) (int64, error) {
	if len(b) == 0 {
		return 0, errInvalidChar
	}

	neg := false
	if b[0] == '-' {
		neg = true
		b = b[1:]
	}

	if len(b) == 0 {
		return 0, errInvalidChar
	}

	var n uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("json: invalid integer %q", string(b))
		}

		n = n*10 + uint64(c-'0')
	}

	if neg {
		return -int64(n), nil
	}

	return int64(n), nil
}

// parseUint parses unsigned 64-bit integer from raw bytes without allocations.
func parseUint(b []byte) (uint64, error) {
	if len(b) == 0 {
		return 0, errInvalidChar
	}

	var n uint64
	for _, c := range b {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("json: invalid unsigned integer %q", string(b))
		}

		n = n*10 + uint64(c-'0')
	}

	return n, nil
}

// parseFloat parses float64 from raw bytes without heap allocations.
func parseFloat(b []byte) (float64, error) {
	return strconv.ParseFloat(bytesconv.B2S(b), 64)
}

// validateValue validates and advances past any valid JSON value (object, array, string, number, bool, null).
func validateValue(data []byte, cursor int) (int, error) {
	cursor = skipWhitespace(data, cursor)
	if cursor >= len(data) {
		return cursor, errUnexpectedEnd
	}

	switch data[cursor] {
	case '"':
		_, newCursor, _, err := scanString(data, cursor)
		return newCursor, err
	case '{':
		cursor++
		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}
			if data[cursor] == '}' {
				return cursor + 1, nil
			}
			if !first {
				if data[cursor] != ',' {
					return cursor, errInvalidChar
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
				if cursor < len(data) && data[cursor] == '}' {
					return cursor, errInvalidChar // trailing comma
				}
			}
			first = false

			if cursor >= len(data) || data[cursor] != '"' {
				return cursor, errInvalidChar
			}
			_, newCursor, _, err := scanString(data, cursor)
			if err != nil {
				return cursor, err
			}
			cursor = skipWhitespace(data, newCursor)
			if cursor >= len(data) || data[cursor] != ':' {
				return cursor, errInvalidChar
			}
			cursor++
			newCursor, err = validateValue(data, cursor)
			if err != nil {
				return cursor, err
			}
			cursor = newCursor
		}
	case '[':
		cursor++
		first := true
		for {
			cursor = skipWhitespace(data, cursor)
			if cursor >= len(data) {
				return cursor, errUnexpectedEnd
			}
			if data[cursor] == ']' {
				return cursor + 1, nil
			}
			if !first {
				if data[cursor] != ',' {
					return cursor, errInvalidChar
				}
				cursor++
				cursor = skipWhitespace(data, cursor)
				if cursor < len(data) && data[cursor] == ']' {
					return cursor, errInvalidChar // trailing comma
				}
			}
			first = false

			newCursor, err := validateValue(data, cursor)
			if err != nil {
				return cursor, err
			}
			cursor = newCursor
		}
	case 't':
		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "true" {
			return cursor + 4, nil
		}
		return cursor, errInvalidChar
	case 'f':
		if cursor+5 <= len(data) && string(data[cursor:cursor+5]) == "false" {
			return cursor + 5, nil
		}
		return cursor, errInvalidChar
	case 'n':
		if cursor+4 <= len(data) && string(data[cursor:cursor+4]) == "null" {
			return cursor + 4, nil
		}
		return cursor, errInvalidChar
	default:
		_, newCursor, _, err := scanNumber(data, cursor)
		return newCursor, err
	}
}

// skipValue skips over any valid JSON value (object, array, string, number, bool, null).
func skipValue(data []byte, cursor int) (int, error) {
	return validateValue(data, cursor)
}
