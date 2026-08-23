// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"bytes"
	"io"
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
func (dec *Decoder) DisallowUnknownFields() {
	dec.cfg.DisallowUnknownFields = true
}

// UseNumber causes the Decoder to unmarshal a number into an interface{} as a
// Number instead of as a float64.
func (dec *Decoder) UseNumber() {
	dec.cfg.UseNumber = true
}

// Decode reads the next JSON-encoded value from its input and stores it in the value pointed to by v.
func (dec *Decoder) Decode(v any) error {
	if dec.err != nil {
		return dec.err
	}

	if len(dec.buf) == 0 || dec.cursor >= len(dec.buf) {
		data, err := io.ReadAll(dec.r)
		if err != nil && err != io.EOF {
			dec.err = err
			return err
		}

		dec.buf = data
		dec.cursor = 0
	}

	if dec.cursor >= len(dec.buf) {
		return io.EOF
	}

	err := UnmarshalWithConfig(dec.buf[dec.cursor:], v, dec.cfg)
	if err != nil {
		return err
	}

	newCursor, _ := skipValue(dec.buf, dec.cursor)
	dec.cursor = newCursor

	return nil
}

// InputOffset returns the input stream byte offset of the current decoder position.
func (dec *Decoder) InputOffset() int64 {
	return int64(dec.cursor)
}

// More reports whether there is another element in the current array or object being parsed.
func (dec *Decoder) More() bool {
	dec.cursor = skipWhitespace(dec.buf, dec.cursor)
	if dec.cursor >= len(dec.buf) {
		return false
	}

	c := dec.buf[dec.cursor]
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
func (enc *Encoder) SetEscapeHTML(on bool) {
	enc.escapeHTML = on
}

// SetIndent instructs the encoder to format each subsequent encoded
// value as if indented by the package-level function Indent.
func (enc *Encoder) SetIndent(prefix, indent string) {
	enc.prefix = prefix
	enc.indent = indent
}

// Encode writes the JSON encoding of v to the stream, followed by a newline character.
func (enc *Encoder) Encode(v any) error {
	var data []byte
	var err error

	if enc.indent != "" || enc.prefix != "" {
		data, err = MarshalIndent(v, enc.prefix, enc.indent)
	} else {
		data, err = Marshal(v)
	}

	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = enc.w.Write(data)

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
	for i := 0; i < len(src); i++ {
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
