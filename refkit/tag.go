// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
	"slices"
	"strings"
)

// Tag represents a parsed struct tag value (e.g. `json:"user_id,omitempty,inline"`).
type Tag struct {
	Name    string   // Primary field identifier or "-"
	Options []string // Subsequent comma-separated directive flags
}

// ParseTag parses a standard comma-separated tag string into a structured [Tag].
func ParseTag(tagStr string) Tag {
	if tagStr == "" {
		return Tag{}
	}

	parts := strings.Split(tagStr, ",")
	name := strings.TrimSpace(parts[0])

	var options []string
	if len(parts) > 1 {
		options = make([]string, 0, len(parts)-1)
		for _, opt := range parts[1:] {
			options = append(options, strings.TrimSpace(opt))
		}
	}

	return Tag{
		Name:    name,
		Options: options,
	}
}

// Has reports whether the specified directive flag is present in the tag options list.
func (t Tag) Has(option string) bool {
	return slices.Contains(t.Options, option)
}

// IsIgnored reports whether the tag specifies that the field should be skipped (name is "-").
func (t Tag) IsIgnored() bool {
	return t.Name == "-"
}

// GetTag inspects field for the first non-empty tag matching candidate keys in priority order.
func GetTag(field reflect.StructField, keys ...string) Tag {
	for _, k := range keys {
		if raw := field.Tag.Get(k); raw != "" {
			return ParseTag(raw)
		}
	}

	return Tag{}
}
