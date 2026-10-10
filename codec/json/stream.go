// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

// Decoder reads and decodes JSON values from an input stream.
type Decoder struct {
	r      io.Reader
	buf    []byte
	cursor int
	cfg    DecoderConfig
	err    error
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r:   r,
		buf: make([]byte, 0, 512),
	}
}

// DisallowUnknownFields causes the Decoder to return an error when the destination
// is a struct and the input contains object keys which do not match any
// non-ignored, exported fields in the destination.
func (d *Decoder) DisallowUnknownFields() {
	d.cfg.DisallowUnknownFields = true
}

// UseNumber causes the Decoder to unmarshal a number into an interface{} as a
// Number instead of as a float64.
func (d *Decoder) UseNumber() {
	d.cfg.UseNumber = true
}

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v.
func (d *Decoder) Decode(v any) error {
	if d.err != nil {
		return d.err
	}

	if len(d.buf) == 0 || d.cursor >= len(d.buf) {
		data, err := io.ReadAll(d.r)
		if err != nil && !errors.Is(err, io.EOF) {
			d.err = err
			return err
		}

		d.buf = data
		d.cursor = 0
	}

	d.cursor = skipWhitespace(d.buf, d.cursor)
	if d.cursor >= len(d.buf) {
		return io.EOF
	}

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("json: Unmarshal(non-pointer %s)", reflect.TypeOf(v).String())
	}

	elemType := rv.Type().Elem()
	dec, err := getDecoder(elemType)
	if err != nil {
		return err
	}

	ptr := unsafe.Pointer(rv.Pointer())
	newCursor, err := dec(d.buf, d.cursor, ptr, &d.cfg)
	if err != nil {
		return err
	}

	d.cursor = newCursor
	return nil
}

// InputOffset returns the input stream byte offset of the current decoder position.
func (d *Decoder) InputOffset() int64 {
	return int64(d.cursor)
}

// More reports whether there is another element in the current array or object being parsed.
func (d *Decoder) More() bool {
	d.cursor = skipWhitespace(d.buf, d.cursor)
	if d.cursor >= len(d.buf) {
		return false
	}

	c := d.buf[d.cursor]
	return c != ']' && c != '}'
}

// Encoder writes JSON values to an output stream.
type Encoder struct {
	w          io.Writer
	escapeHTML bool
	indent     string
	prefix     string
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:          w,
		escapeHTML: true,
	}
}

// SetEscapeHTML specifies whether problematic HTML characters
// should be escaped inside JSON quoted strings.
func (e *Encoder) SetEscapeHTML(on bool) {
	e.escapeHTML = on
}

// SetIndent instructs the encoder to format each subsequent encoded
// value as if indented by the package-level function Indent.
func (e *Encoder) SetIndent(prefix, indent string) {
	e.prefix = prefix
	e.indent = indent
}

// Encode writes the JSON encoding of v to the stream, followed by a newline character.
//
//nolint:prealloc
func (e *Encoder) Encode(v any) error {
	var data []byte
	var err error

	if e.indent != "" || e.prefix != "" {
		data, err = MarshalIndent(v, e.prefix, e.indent)
	} else {
		data, err = Marshal(v)
	}

	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = e.w.Write(data)

	return err
}

// Compact appends to dst the JSON-encoded src with insignificant space characters elided.
func Compact(dst *bytes.Buffer, src []byte) error {
	cursor := 0
	n := len(src)

	for cursor < n {
		cursor = skipWhitespace(src, cursor)
		if cursor >= n {
			break
		}

		b := src[cursor]
		if b == '"' {
			raw, newCursor, _, err := scanString(src, cursor)
			if err != nil {
				return err
			}

			dst.WriteByte('"')
			dst.Write(raw)
			dst.WriteByte('"')
			cursor = newCursor
			continue
		}

		dst.WriteByte(b)
		cursor++
	}

	return nil
}

// HTMLEscape appends to dst the JSON-encoded src with <, >, &, U+2028, and U+2029
// characters escaped inside string literals.
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	for i := range src {
		b := src[i]
		switch b {
		case '<':
			dst.WriteString(`\u003c`)
		case '>':
			dst.WriteString(`\u003e`)
		case '&':
			dst.WriteString(`\u0026`)
		default:
			dst.WriteByte(b)
		}
	}
}
