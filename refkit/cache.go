// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"
)

// FieldSchema represents the cached layout and tag metadata of a struct field.
type FieldSchema struct {
	Field  reflect.StructField // StructField descriptor
	Offset uintptr             // Cumulative byte offset from the root struct
	Tag    Tag                 // Pre-parsed tag metadata
}

// StructSchema represents the cached memory layout of a struct type.
type StructSchema struct {
	Type   reflect.Type  // The struct reflect.Type
	Size   uintptr       // Total struct byte size
	Fields []FieldSchema // Visible fields in layout order
}

// SchemaCache provides a thread-safe cache for pre-computed struct layouts and parsed tags.
type SchemaCache struct {
	mu      sync.RWMutex
	schemas map[reflect.Type]*StructSchema
}

// NewSchemaCache constructs an empty, concurrent-safe [SchemaCache].
func NewSchemaCache() *SchemaCache {
	return &SchemaCache{
		schemas: make(map[reflect.Type]*StructSchema),
	}
}

// Get returns the cached [StructSchema] for struct type t.
//
// On a cache hit, Get executes with 0 heap allocations under a shared read lock.
// On a cache miss, it parses all visible fields, computes cumulative byte offsets,
// pre-parses struct tags into [FieldSchema] entries, and caches the result.
//
// If t is a pointer to a struct, Get dereferences the pointer. If t is nil or
// resolves to a non-struct type, Get returns an error.
func (c *SchemaCache) Get(t reflect.Type) (*StructSchema, error) {
	if t == nil {
		return nil, errors.New("refkit: nil type")
	}

	target := DerefType(t)
	if target.Kind() != reflect.Struct {
		return nil, fmt.Errorf("refkit: %v is not a struct", t)
	}

	c.mu.RLock()
	schema, ok := c.schemas[target]
	c.mu.RUnlock()
	if ok {
		return schema, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check under write lock.
	if schema, ok = c.schemas[target]; ok {
		return schema, nil
	}

	var fields []FieldSchema
	for f := range VisibleFields(target) {
		var offset uintptr
		curr := target
		for _, idx := range f.Index {
			sf := curr.Field(idx)
			offset += sf.Offset
			curr = sf.Type
			if curr.Kind() == reflect.Pointer {
				curr = curr.Elem()
			}
		}

		tag := parseFieldTag(f.Tag)
		fields = append(fields, FieldSchema{
			Field:  f,
			Offset: offset,
			Tag:    tag,
		})
	}

	schema = &StructSchema{
		Type:   target,
		Size:   target.Size(),
		Fields: fields,
	}
	c.schemas[target] = schema

	return schema, nil
}

func parseFieldTag(tag reflect.StructTag) Tag {
	raw := string(tag)
	if raw == "" {
		return Tag{}
	}

	var options []string
	var primaryName string

	s := raw
	hasConventional := false
	for s != "" {
		i := 0
		for i < len(s) && s[i] == ' ' {
			i++
		}
		s = s[i:]
		if s == "" {
			break
		}

		i = 0
		for i < len(s) && s[i] > ' ' && s[i] != ':' && s[i] != '"' && s[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(s) || s[i] != ':' || s[i+1] != '"' {
			break
		}
		name := s[:i]
		s = s[i+1:]

		s = s[1:]
		i = 0
		for i < len(s) && s[i] != '"' {
			if s[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(s) {
			break
		}
		val := s[:i]
		s = s[i+1:]

		hasConventional = true
		options = append(options, name+"="+val)

		inner := ParseTag(val)
		if primaryName == "" {
			primaryName = inner.Name
		}
		if strings.Contains(inner.Name, "=") && !slices.Contains(options, inner.Name) {
			options = append(options, inner.Name)
		}
		for _, opt := range inner.Options {
			if !slices.Contains(options, opt) {
				options = append(options, opt)
			}
		}
	}

	if !hasConventional {
		return ParseTag(raw)
	}

	return Tag{
		Name:    primaryName,
		Options: options,
	}
}
