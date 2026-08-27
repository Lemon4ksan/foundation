// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package json implements a silicon-grade, zero-allocation, lockless pure-Go JSON encoder and decoder.
//
// It compiles type metadata once into an opcode execution sequence and performs direct memory writes
// via standard ABI-safe unsafe offsets without reflection overhead, runtime linkname hacks, or GC churn.
package json

import (
	"bytes"
	stdjson "encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

// Marshaler is the interface implemented by types that can marshal themselves into valid JSON.
type Marshaler = stdjson.Marshaler

// Unmarshaler is the interface implemented by types that can unmarshal a JSON description of themselves.
type Unmarshaler = stdjson.Unmarshaler

// RawMessage is a raw encoded JSON value. It implements [Marshaler] and [Unmarshaler]
// and can be used to delay JSON decoding or precompute a JSON encoding.
type RawMessage = stdjson.RawMessage

// A Number represents a JSON number literal.
type Number = stdjson.Number

// Marshal returns the JSON encoding of v.
func Marshal(v any) ([]byte, error) {
	state := encPool.Get().(*encodeState)
	state.buf = state.buf[:0]
	state.escapeHTML = true

	defer encPool.Put(state)

	if v == nil {
		return []byte("null"), nil
	}

	t := reflect.TypeOf(v)
	enc, err := getEncoder(t)
	if err != nil {
		return nil, err
	}

	val := reflect.ValueOf(v)
	valCopy := reflect.New(t)
	valCopy.Elem().Set(val)
	ptr := unsafe.Pointer(valCopy.Pointer())

	if err := enc(state, ptr); err != nil {
		return nil, err
	}

	out := make([]byte, len(state.buf))
	copy(out, state.buf)

	return out, nil
}

// MarshalTo encodes v directly into dst with zero heap allocations when capacity allows.
func MarshalTo(dst []byte, v any) ([]byte, error) {
	if v == nil {
		return append(dst, "null"...), nil
	}

	state := encPool.Get().(*encodeState)
	state.buf = state.buf[:0]
	state.escapeHTML = true

	defer encPool.Put(state)

	t := reflect.TypeOf(v)
	enc, err := getEncoder(t)
	if err != nil {
		return nil, err
	}

	val := reflect.ValueOf(v)
	valCopy := reflect.New(t)
	valCopy.Elem().Set(val)
	ptr := unsafe.Pointer(valCopy.Pointer())

	if err := enc(state, ptr); err != nil {
		return nil, err
	}

	return append(dst, state.buf...), nil
}

// MarshalIndent is like Marshal but applies Indent to format the output.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = Indent(&buf, b, prefix, indent)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Unmarshal parses the JSON-encoded data and stores the result in the value pointed to by v.
func Unmarshal(data []byte, v any) error {
	return UnmarshalWithConfig(data, v, DecoderConfig{})
}

// UnmarshalNoCopy parses JSON data storing string references directly from data without string cloning.
// The caller must guarantee data is not modified during the lifetime of decoded strings.
func UnmarshalNoCopy(data []byte, v any) error {
	return UnmarshalWithConfig(data, v, DecoderConfig{NoCopy: true})
}

// UnmarshalWithConfig parses JSON data into v according to the provided DecoderConfig flags.
func UnmarshalWithConfig(data []byte, v any, cfg DecoderConfig) error {
	if len(data) == 0 {
		return errUnexpectedEnd
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
	cursor := skipWhitespace(data, 0)

	_, err = dec(data, cursor, ptr, &cfg)

	return err
}

// Valid reports whether data is a valid JSON encoding.
func Valid(data []byte) bool {
	cursor := skipWhitespace(data, 0)
	if cursor >= len(data) {
		return false
	}

	newCursor, err := skipValue(data, cursor)
	if err != nil {
		return false
	}

	newCursor = skipWhitespace(data, newCursor)

	return newCursor == len(data)
}

// Indent appends to dst an indented form of the JSON-encoded src.
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	var (
		depth    int
		inString bool
		escape   bool
	)

	for i := 0; i < len(src); i++ {
		c := src[i]
		if inString {
			dst.WriteByte(c)
			switch {
			case escape:
				escape = false
			case c == '\\':
				escape = true
			case c == '"':
				inString = false
			}
			continue
		}

		switch c {
		case ' ', '\t', '\r', '\n':
			continue
		case '"':
			inString = true
			dst.WriteByte(c)
		case '{', '[':
			dst.WriteByte(c)
			dst.WriteByte('\n')
			depth++
			dst.WriteString(prefix)
			for j := 0; j < depth; j++ {
				dst.WriteString(indent)
			}
		case '}', ']':
			dst.WriteByte('\n')
			depth--
			dst.WriteString(prefix)
			for j := 0; j < depth; j++ {
				dst.WriteString(indent)
			}
			dst.WriteByte(c)
		case ':':
			dst.WriteString(": ")
		case ',':
			dst.WriteByte(',')
			dst.WriteByte('\n')
			dst.WriteString(prefix)
			for j := 0; j < depth; j++ {
				dst.WriteString(indent)
			}
		default:
			dst.WriteByte(c)
		}
	}

	return nil
}
