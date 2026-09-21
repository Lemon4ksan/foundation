// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// Tag represents a parsed struct tag value (e.g. `json:"user_id,omitempty,inline"`).
type Tag struct {
	Name    string   // Primary field identifier or "-"
	Options []string // Subsequent comma-separated directive flags
}

// ParseTag parses a struct tag string into a structured [Tag], respecting nested brackets and quotes.
//
// For standard ASCII tags, ParseTag executes an optimized lock-free and zero-copy
// scanning path without allocating rune slices. For simple tags without directives,
// ParseTag achieves 0 heap allocations.
func ParseTag(tagStr string) Tag {
	if tagStr == "" {
		return Tag{}
	}

	for i := 0; i < len(tagStr); i++ {
		if tagStr[i] >= 0x80 {
			return parseTagUnicode(tagStr)
		}
	}

	return parseTagASCII(tagStr)
}

func parseTagASCII(tagStr string) Tag {
	// Fast check for tags with no delimiters or options (e.g. "name", "-", "user_id").
	hasDelim := false
	for i := 0; i < len(tagStr); i++ {
		b := tagStr[i]
		if b == ',' || b == '"' || b == '\'' || b == '{' || b == '(' || b == '[' {
			hasDelim = true
			break
		}
	}

	if !hasDelim {
		return Tag{
			Name: strings.TrimSpace(tagStr),
		}
	}

	var stackParts [8]string
	parts := stackParts[:0]
	var heapParts []string

	var start int
	var depth int
	var inQuotes bool
	var quoteChar byte

	for i := 0; i < len(tagStr); i++ {
		b := tagStr[i]
		switch b {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = b
			} else if quoteChar == b {
				inQuotes = false
			}
		case '{', '(', '[':
			if !inQuotes {
				depth++
			}
		case '}', ')', ']':
			if !inQuotes && depth > 0 {
				depth--
			}
		case ',':
			if !inQuotes && depth == 0 {
				part := tagStr[start:i]
				if heapParts == nil && len(parts) < len(stackParts) {
					parts = append(parts, part)
				} else {
					if heapParts == nil {
						heapParts = make([]string, len(parts), len(parts)+8)
						copy(heapParts, parts)
					}
					heapParts = append(heapParts, part)
				}
				start = i + 1
			}
		}
	}

	lastPart := tagStr[start:]
	var allParts []string
	if heapParts != nil {
		heapParts = append(heapParts, lastPart)
		allParts = heapParts
	} else {
		parts = append(parts, lastPart)
		allParts = parts
	}

	name := strings.TrimSpace(allParts[0])
	var options []string
	if len(allParts) > 1 {
		optCount := 0
		for _, opt := range allParts[1:] {
			if strings.TrimSpace(opt) != "" {
				optCount++
			}
		}
		if optCount > 0 {
			options = make([]string, 0, optCount)
			for _, opt := range allParts[1:] {
				if opt = strings.TrimSpace(opt); opt != "" {
					options = append(options, opt)
				}
			}
		}
	}

	return Tag{
		Name:    name,
		Options: options,
	}
}

func parseTagUnicode(tagStr string) Tag {
	var parts []string
	var start int
	var depth int
	var inQuotes bool
	var quoteChar rune

	runes := []rune(tagStr)
	for i, r := range runes {
		switch r {
		case '"', '\'':
			if !inQuotes {
				inQuotes = true
				quoteChar = r
			} else if quoteChar == r {
				inQuotes = false
			}
		case '{', '(', '[':
			if !inQuotes {
				depth++
			}
		case '}', ')', ']':
			if !inQuotes && depth > 0 {
				depth--
			}
		case ',':
			if !inQuotes && depth == 0 {
				parts = append(parts, string(runes[start:i]))
				start = i + 1
			}
		}
	}
	parts = append(parts, string(runes[start:]))

	name := strings.TrimSpace(parts[0])
	var options []string
	if len(parts) > 1 {
		options = make([]string, 0, len(parts)-1)
		for _, opt := range parts[1:] {
			if opt = strings.TrimSpace(opt); opt != "" {
				options = append(options, opt)
			}
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

// Get extracts the value of a key-value directive flag (e.g. `default=10` -> "10", `min=5` -> "5").
func (t Tag) Get(key string) string {
	for _, opt := range t.Options {
		if k, v, ok := strings.Cut(opt, "="); ok && k == key {
			return v
		}
	}
	return ""
}

// GetInt extracts a numeric option parsed as int.
func (t Tag) GetInt(key string) (int, bool) {
	val := t.Get(key)
	if val == "" {
		return 0, false
	}
	n, err := strconv.Atoi(val)
	return n, err == nil
}

// GetFloat extracts a numeric option parsed as float64.
func (t Tag) GetFloat(key string) (float64, bool) {
	val := t.Get(key)
	if val == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(val, 64)
	return f, err == nil
}

// SplitOption splits a key-value option by delim (e.g. `enum=admin|user` with "|" -> ["admin", "user"]).
func (t Tag) SplitOption(key, delim string) []string {
	val := t.Get(key)
	if val == "" {
		return nil
	}
	items := strings.Split(val, delim)
	var res []string
	for _, item := range items {
		if item = strings.TrimSpace(item); item != "" {
			res = append(res, item)
		}
	}
	return res
}

// IsEmpty reports whether the tag contains no name or options.
func (t Tag) IsEmpty() bool {
	return t.Name == "" && len(t.Options) == 0
}

// IsIgnored reports whether the tag explicitly ignores the field via `"-"`.
func (t Tag) IsIgnored() bool {
	return t.Name == "-"
}

// GetTag retrieves and parses the first matching struct tag for keys from field.
func GetTag(field reflect.StructField, keys ...string) Tag {
	for _, key := range keys {
		if raw, ok := field.Tag.Lookup(key); ok {
			return ParseTag(raw)
		}
	}

	return Tag{}
}
