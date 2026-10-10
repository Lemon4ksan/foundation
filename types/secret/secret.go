// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secret

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

// Secret wraps sensitive data (passwords, bearer tokens, API keys) to prevent accidental leakage in logs and JSON dumps.
//
// The zero value is empty and ready to use.
type Secret[T any] struct {
	val T
}

// New creates a new protected [Secret] container wrapping sensitive value val.
//
// It executes in O(1) time and is safe for concurrent use.
func New[T any](val T) Secret[T] {
	return Secret[T]{val: val}
}

// Value returns the raw sensitive value for authorized networking operations.
//
// It executes in O(1) time and is safe for concurrent use.
func (s Secret[T]) Value() T {
	return s.val
}

// Expose returns the raw sensitive value.
//
// Synonym for [Secret.Value].
// It executes in O(1) time and is safe for concurrent use.
func (s Secret[T]) Expose() T {
	return s.val
}

// String masks the secret when formatted with %s, %v, or fmt.Println.
//
// It executes in O(1) time and is safe for concurrent use.
func (s Secret[T]) String() string {
	return "******"
}

// GoString masks the secret when formatted with %#v in debug prints.
//
// It executes in O(1) time and is safe for concurrent use.
func (s Secret[T]) GoString() string {
	return "secret.Secret(******)"
}

// Format masks the secret during fmt.Sprintf printing.
//
// It executes in O(1) time and is safe for concurrent use.
func (s Secret[T]) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('#') {
		_, _ = f.Write([]byte("secret.Secret(******)"))
		return
	}

	_, _ = f.Write([]byte("******"))
}

// LogValue implements [slog.LogValuer] for structured logging with log/slog.
//
// It executes in O(1) time and is safe for concurrent use.
func (s Secret[T]) LogValue() slog.Value {
	return slog.StringValue("******")
}

// MarshalJSON safely serializes the secret as a masked string to prevent accidental leakage in JSON logs or responses.
//
// It executes in O(1) time and allocates memory. It is safe for concurrent use.
func (s Secret[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal("******")
}

// UnmarshalJSON deserializes a raw value into the protected container.
//
// It executes in O(N) time where N is the length of data and allocates memory.
// It is safe for concurrent use.
func (s *Secret[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &s.val)
}
