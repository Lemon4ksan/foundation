// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package values

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

var (
	bOne  = []byte("1")
	bZero = []byte("0")
	bNull = []byte("null")
)

// ErrInvalidFormat is returned when a raw string representation fails parsing into a structured type.
var ErrInvalidFormat = errors.New("values: invalid value format")

// ValueError describes an error encountered during structure reflection or value unmarshaling.
type ValueError struct {
	Err   error
	Type  string
	Field string
	Index int
}

// Error is an alias for [ValueError] to eliminate package stuttering.
type Error = ValueError

// Error returns a formatted description of the value unmarshaling error.
func (e *ValueError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var sb strings.Builder
	sb.Grow(64)
	sb.WriteString("values: ")

	if e.Field != "" {
		sb.WriteString("field ")
		sb.WriteString(e.Field)

		if e.Index >= 0 {
			var numBuf [12]byte
			sb.WriteByte('[')
			sb.Write(strconv.AppendInt(numBuf[:0], int64(e.Index), 10))
			sb.WriteByte(']')
		}

		sb.WriteString(": ")
		sb.WriteString(e.Err.Error())

		return sb.String()
	}

	if e.Type != "" {
		sb.WriteString(e.Type)
		sb.WriteString(": ")
		sb.WriteString(e.Err.Error())

		return sb.String()
	}

	sb.WriteString(e.Err.Error())

	return sb.String()
}

// Unwrap returns the underlying cause of the value unmarshaling error.
func (e *ValueError) Unwrap() error { return e.Err }

// Uint64String parses uint64 values from numeric or quoted string JSON payloads.
type Uint64String uint64

// UnmarshalJSON parses JSON byte data into [Uint64String].
func (u *Uint64String) UnmarshalJSON(b []byte) error {
	s := bytesconv.B2S(bytesconv.TrimQuotes(b))
	if len(s) == 0 || s == "null" {
		*u = 0
		return nil
	}

	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return &ValueError{Type: "Uint64String", Err: err}
	}

	*u = Uint64String(val)

	return nil
}

// MarshalJSON serializes [Uint64String] as a quoted JSON string.
func (u Uint64String) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 24)
	buf = append(buf, '"')
	buf = strconv.AppendUint(buf, uint64(u), 10)
	buf = append(buf, '"')
	return buf, nil
}

// Int64String parses int64 values from numeric or quoted string JSON payloads.
type Int64String int64

// UnmarshalJSON parses JSON byte data into [Int64String].
func (i *Int64String) UnmarshalJSON(b []byte) error {
	s := bytesconv.B2S(bytesconv.TrimQuotes(b))
	if len(s) == 0 || s == "null" {
		*i = 0
		return nil
	}

	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &ValueError{Type: "Int64String", Err: err}
	}

	*i = Int64String(val)

	return nil
}

// MarshalJSON serializes [Int64String] as a quoted JSON string.
func (i Int64String) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 24)
	buf = append(buf, '"')
	buf = strconv.AppendInt(buf, int64(i), 10)
	buf = append(buf, '"')
	return buf, nil
}

// Float64String parses float64 values from numeric or quoted string JSON payloads.
type Float64String float64

// UnmarshalJSON parses JSON byte data into [Float64String].
func (f *Float64String) UnmarshalJSON(b []byte) error {
	s := bytesconv.B2S(bytesconv.TrimQuotes(b))
	if len(s) == 0 || s == "null" {
		*f = 0
		return nil
	}

	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return &ValueError{Type: "Float64String", Err: err}
	}

	*f = Float64String(val)

	return nil
}

// MarshalJSON serializes [Float64String] as a JSON string.
func (f Float64String) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 32)
	buf = append(buf, '"')
	buf = strconv.AppendFloat(buf, float64(f), 'f', -1, 64)
	buf = append(buf, '"')
	return buf, nil
}

// BoolInt parses boolean flags represented as numbers or strings in JSON.
type BoolInt bool

// UnmarshalJSON implements [json.Unmarshaler].
func (bi *BoolInt) UnmarshalJSON(b []byte) error {
	s := bytesconv.B2S(bytesconv.TrimQuotes(b))

	switch {
	case s == "1" || bytesconv.EqualFoldASCII(s, "true"):
		*bi = true
	case s == "0" || bytesconv.EqualFoldASCII(s, "false") || len(s) == 0 || s == "null":
		*bi = false
	default:
		val, err := strconv.Atoi(s)
		*bi = (err == nil && val != 0)
	}

	return nil
}

// MarshalJSON serializes [BoolInt] back as numeric "1" or "0" JSON values.
func (bi BoolInt) MarshalJSON() ([]byte, error) {
	if bi {
		return bOne, nil
	}

	return bZero, nil
}

// UnixTimestamp parses UNIX epoch timestamps from strings or numbers in JSON.
type UnixTimestamp time.Time

// UnmarshalJSON implements [json.Unmarshaler].
func (t *UnixTimestamp) UnmarshalJSON(b []byte) error {
	s := bytesconv.B2S(bytesconv.TrimQuotes(b))
	if len(s) == 0 || s == "null" || s == "0" {
		*t = UnixTimestamp(time.Time{})
		return nil
	}

	unix, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return &ValueError{Type: "UnixTimestamp", Err: err}
	}

	*t = UnixTimestamp(time.Unix(unix, 0).UTC())

	return nil
}

// MarshalJSON serializes [UnixTimestamp] as a numeric Unix epoch timestamp.
func (t UnixTimestamp) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return bZero, nil
	}

	return strconv.AppendInt(nil, time.Time(t).Unix(), 10), nil
}

// Time returns the underlying [time.Time].
func (t UnixTimestamp) Time() time.Time { return time.Time(t) }

// RFC3339Timestamp parses ISO-8601 / RFC-3339 formatted date-time strings in JSON.
type RFC3339Timestamp time.Time

// UnmarshalJSON implements [json.Unmarshaler].
func (t *RFC3339Timestamp) UnmarshalJSON(b []byte) error {
	s := bytesconv.B2S(bytesconv.TrimQuotes(b))
	if len(s) == 0 || s == "null" {
		*t = RFC3339Timestamp(time.Time{})
		return nil
	}

	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return &ValueError{Type: "RFC3339Timestamp", Err: err}
	}

	*t = RFC3339Timestamp(parsed.UTC())

	return nil
}

// MarshalJSON implements [json.Marshaler].
func (t RFC3339Timestamp) MarshalJSON() ([]byte, error) {
	if time.Time(t).IsZero() {
		return bNull, nil
	}

	buf := make([]byte, 0, 22)
	buf = append(buf, '"')
	buf = time.Time(t).AppendFormat(buf, time.RFC3339)
	buf = append(buf, '"')
	return buf, nil
}

// Time returns the underlying [time.Time] value.
func (t RFC3339Timestamp) Time() time.Time { return time.Time(t) }

// String returns the RFC-3339 formatted date-time string.
func (t RFC3339Timestamp) String() string {
	if time.Time(t).IsZero() {
		return ""
	}

	return time.Time(t).Format(time.RFC3339)
}

// NumberConstraint represents all signed, unsigned integer, and floating-point types.
type NumberConstraint interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64
}

// NumberString formats any integer or floating-point number into its decimal string representation.
//
// For integers and floats, formatting uses strconv with zero heap allocations beyond the returned string.
func NumberString[T NumberConstraint](v T) string {
	return bytesconv.FormatNumber(v)
}
