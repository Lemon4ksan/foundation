// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"iter"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
)

type tagCacheEntry struct {
	key string
	tag Tag
}

const tagCacheSize = 256

var (
	tagCache [tagCacheSize]atomic.Pointer[tagCacheEntry]

	optOmitempty       = []string{"omitempty"}
	optInline          = []string{"inline"}
	optString          = []string{"string"}
	optOmitemptyInline = []string{"omitempty", "inline"}
	optInlineOmitempty = []string{"inline", "omitempty"}
)

func hashString(s string) uint64 {
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h = (h ^ uint64(s[i])) * 1099511628211
	}
	return h
}

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

	h := hashString(tagStr)
	slot := h % tagCacheSize
	entry := tagCache[slot].Load()
	if entry != nil && entry.key == tagStr {
		return entry.tag
	}

	tag := parseTagSlow(tagStr)
	tagCache[slot].Store(&tagCacheEntry{key: tagStr, tag: tag})
	return tag
}

func parseTagSlow(tagStr string) Tag {
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
		switch optCount {
		case 1:
			for _, opt := range allParts[1:] {
				if opt = strings.TrimSpace(opt); opt != "" {
					switch opt {
					case "omitempty":
						options = optOmitempty
					case "inline":
						options = optInline
					case "string":
						options = optString
					default:
						options = []string{opt}
					}
					break
				}
			}
		case 2:
			var opts [2]string
			idx := 0
			for _, opt := range allParts[1:] {
				if opt = strings.TrimSpace(opt); opt != "" {
					if idx < 2 {
						opts[idx] = opt
						idx++
					}
				}
			}
			switch {
			case opts[0] == "omitempty" && opts[1] == "inline":
				options = optOmitemptyInline
			case opts[0] == "inline" && opts[1] == "omitempty":
				options = optInlineOmitempty
			default:
				options = []string{opts[0], opts[1]}
			}
		default:
			if optCount > 0 {
				options = make([]string, 0, optCount)
				for _, opt := range allParts[1:] {
					if opt = strings.TrimSpace(opt); opt != "" {
						options = append(options, opt)
					}
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
	var (
		parts     []string
		start     int
		depth     int
		inQuotes  bool
		quoteChar rune
	)

	for i, r := range tagStr {
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
				parts = append(parts, tagStr[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, tagStr[start:])

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
//
//go:inline
func (t Tag) Has(option string) bool {
	return slices.Contains(t.Options, option)
}

// Value extracts the value of a key-value directive flag (e.g. `default=10` -> "10", `min=5` -> "5").
//
//go:inline
func (t Tag) Value(key string) string {
	for _, opt := range t.Options {
		if len(opt) > len(key) && opt[len(key)] == '=' && opt[:len(key)] == key {
			return opt[len(key)+1:]
		}
	}
	return ""
}

// Int extracts a numeric option parsed as int.
func (t Tag) Int(key string) (int, bool) {
	val := t.Value(key)
	if val == "" {
		return 0, false
	}
	n, err := strconv.Atoi(val)
	return n, err == nil
}

// Float extracts a numeric option parsed as float64.
func (t Tag) Float(key string) (float64, bool) {
	val := t.Value(key)
	if val == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(val, 64)
	return f, err == nil
}

// SplitOption splits a key-value option by delim, returning an iterator over the non-empty parts.
func (t Tag) SplitOption(key, delim string) iter.Seq[string] {
	val := t.Value(key)
	return func(yield func(string) bool) {
		if val == "" {
			return
		}

		remaining := val
		for {
			idx := strings.Index(remaining, delim)
			if idx < 0 {
				if item := strings.TrimSpace(remaining); item != "" {
					yield(item)
				}
				return
			}

			if item := strings.TrimSpace(remaining[:idx]); item != "" {
				if !yield(item) {
					return
				}
			}
			remaining = remaining[idx+len(delim):]
		}
	}
}

// IsEmpty reports whether the tag contains no name or options.
//
//go:inline
func (t Tag) IsEmpty() bool {
	return t.Name == "" && len(t.Options) == 0
}

// IsIgnored reports whether the tag explicitly ignores the field via `"-"`.
//
//go:inline
func (t Tag) IsIgnored() bool {
	return t.Name == "-"
}

// TagOf retrieves and parses the first matching struct tag for keys from field.
func TagOf(field reflect.StructField, keys ...string) Tag {
	for _, key := range keys {
		if raw, ok := field.Tag.Lookup(key); ok {
			return ParseTag(raw)
		}
	}

	return Tag{}
}
