// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
)

var errNilType = errors.New("refkit: nil type")

// OffsetInvalid is assigned to [FieldSchema.Offset] when a field is accessed via
// an embedded pointer and cannot be addressed via a single contiguous offset from the root struct.
const OffsetInvalid uintptr = ^uintptr(0)

// FieldSchema represents the cached layout and tag metadata of a struct field.
type FieldSchema struct {
	Field    reflect.StructField // StructField descriptor
	Offset   uintptr             // Cumulative byte offset from the root struct (OffsetInvalid if Indirect)
	Indirect bool                // True if accessing this field requires dereferencing an embedded pointer
	Tag      Tag                 // Pre-parsed tag metadata
}

// StructSchema represents the cached memory layout of a struct type.
type StructSchema struct {
	Type   reflect.Type  // The struct reflect.Type
	Size   uintptr       // Total struct byte size
	Fields []FieldSchema // Visible fields in layout order
}

// SchemaCache provides a thread-safe cache for pre-computed struct layouts and parsed tags.
type SchemaCache struct {
	mu      sync.Mutex
	schemas atomic.Pointer[map[reflect.Type]*StructSchema]
}

// NewSchemaCache constructs an empty, concurrent-safe [SchemaCache].
func NewSchemaCache() *SchemaCache {
	c := &SchemaCache{}
	m := make(map[reflect.Type]*StructSchema)
	c.schemas.Store(&m)
	return c
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
		return nil, errNilType
	}

	target := DerefType(t)
	if target.Kind() != reflect.Struct {
		return nil, fmt.Errorf("refkit: %v is not a struct", t)
	}

	currMap := c.schemas.Load()
	if currMap != nil {
		if schema, ok := (*currMap)[target]; ok {
			return schema, nil
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check under write lock.
	currMap = c.schemas.Load()
	if currMap != nil {
		if schema, ok := (*currMap)[target]; ok {
			return schema, nil
		}
	}

	var fields []FieldSchema
	for f := range VisibleFields(target) {
		var offset uintptr
		var indirect bool
		curr := target
		for i, idx := range f.Index {
			sf := curr.Field(idx)
			if indirect {
				curr = sf.Type
				if curr.Kind() == reflect.Pointer {
					curr = curr.Elem()
				}
				continue
			}
			offset += sf.Offset
			curr = sf.Type
			if curr.Kind() == reflect.Pointer {
				curr = curr.Elem()
				if i < len(f.Index)-1 {
					indirect = true
					offset = OffsetInvalid
				}
			}
		}

		tag := parseFieldTag(f.Tag)
		fields = append(fields, FieldSchema{
			Field:    f,
			Offset:   offset,
			Indirect: indirect,
			Tag:      tag,
		})
	}

	schema := &StructSchema{
		Type:   target,
		Size:   target.Size(),
		Fields: fields,
	}

	var oldLen int
	if currMap != nil {
		oldLen = len(*currMap)
	}
	newMap := make(map[reflect.Type]*StructSchema, oldLen+1)
	if currMap != nil {
		maps.Copy(newMap, *currMap)
	}
	newMap[target] = schema
	c.schemas.Store(&newMap)

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
			if s[i] == '\\' && i+1 < len(s) {
				i += 2
				continue
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
