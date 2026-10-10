// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secret

import (
	"encoding"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

// Secret wraps sensitive data (passwords, bearer tokens, API keys) to prevent accidental leakage in logs and JSON dumps.
//
// The zero value is empty and ready to use.
type Secret[T any] struct {
	mu  *sync.RWMutex
	val T
}

// New creates a new protected [Secret] container wrapping sensitive value val.
func New[T any](val T) Secret[T] {
	return Secret[T]{
		mu:  new(sync.RWMutex),
		val: val,
	}
}

// Value returns the raw sensitive value for authorized networking operations.
func (s Secret[T]) Value() T {
	if s.mu != nil {
		s.mu.RLock()
	}
	val := s.val
	if s.mu != nil {
		s.mu.RUnlock()
	}
	return val
}

// Expose returns the raw sensitive value. Synonym for [Secret.Value].
func (s Secret[T]) Expose() T {
	return s.Value()
}

// Wipe securely zeroes the underlying sensitive material in memory.
// If the wrapped value is a byte slice, its backing memory is explicitly overwritten with zeroes.
// If the wrapped value implements a Wipe or Destroy method, it is invoked.
// The secret container is then reset to its zero value.
func (s *Secret[T]) Wipe() {
	if s.mu != nil {
		s.mu.Lock()
	}
	if b, ok := any(s.val).([]byte); ok {
		for i := range b {
			b[i] = 0
		}
	} else if w, ok := any(s.val).(interface{ Wipe() }); ok && w != nil {
		w.Wipe()
	} else if d, ok := any(s.val).(interface{ Destroy() }); ok && d != nil {
		d.Destroy()
	}
	var zero T
	s.val = zero
	if s.mu != nil {
		s.mu.Unlock()
	}
}

// String masks the secret when formatted with %s, %v, or fmt.Println.
func (s Secret[T]) String() string {
	return "******"
}

// GoString masks the secret when formatted with %#v in debug prints.
func (s Secret[T]) GoString() string {
	return "secret.Secret(******)"
}

// Format masks the secret during fmt.Sprintf printing.
func (s Secret[T]) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('#') {
		_, _ = f.Write([]byte("secret.Secret(******)"))
		return
	}

	_, _ = f.Write([]byte("******"))
}

// LogValue implements [slog.LogValuer] for structured logging with log/slog.
func (s Secret[T]) LogValue() slog.Value {
	return slog.StringValue("******")
}

// MarshalJSON safely serializes the secret as a masked string to prevent accidental leakage in JSON logs or responses.
func (s Secret[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal("******")
}

// UnmarshalJSON deserializes a raw value into the protected container.
func (s *Secret[T]) UnmarshalJSON(data []byte) error {
	if s.mu == nil {
		s.mu = new(sync.RWMutex)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.val)
}

// MarshalText implements [encoding.TextMarshaler] to safely mask the secret in text serialization (e.g. YAML, TOML, XML).
func (s Secret[T]) MarshalText() ([]byte, error) {
	return []byte("******"), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler] to deserialize text into the secret container.
func (s *Secret[T]) UnmarshalText(text []byte) error {
	if s.mu == nil {
		s.mu = new(sync.RWMutex)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	switch v := any(&s.val).(type) {
	case *string:
		*v = string(text)
		return nil
	case *[]byte:
		*v = append([]byte(nil), text...)
		return nil
	case encoding.TextUnmarshaler:
		return v.UnmarshalText(text)
	default:
		if err := json.Unmarshal(text, &s.val); err == nil {
			return nil
		}
		return json.Unmarshal([]byte(`"`+string(text)+`"`), &s.val)
	}
}

// MarshalBinary implements [encoding.BinaryMarshaler] to safely mask the secret in binary serialization.
func (s Secret[T]) MarshalBinary() ([]byte, error) {
	return []byte("******"), nil
}

// UnmarshalBinary implements [encoding.BinaryUnmarshaler] to deserialize binary data into the secret container.
func (s *Secret[T]) UnmarshalBinary(data []byte) error {
	if s.mu == nil {
		s.mu = new(sync.RWMutex)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	switch v := any(&s.val).(type) {
	case *string:
		*v = string(data)
		return nil
	case *[]byte:
		*v = append([]byte(nil), data...)
		return nil
	case encoding.BinaryUnmarshaler:
		return v.UnmarshalBinary(data)
	default:
		return json.Unmarshal(data, &s.val)
	}
}
