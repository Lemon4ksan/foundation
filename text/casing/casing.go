// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package casing

import (
	"strings"
	"unicode"
)

// Kind identifies a string casing convention.
// The zero value is an empty string and represents an unspecified casing.
type Kind string

const (
	// Snake represents lowercase underscore-separated casing (e.g. "snake_case").
	Snake Kind = "snake_case"
	// Camel represents lowerCamelCase casing (e.g. "camelCase").
	Camel Kind = "camelCase"
	// Pascal represents UpperCamelCase casing (e.g. "PascalCase").
	Pascal Kind = "PascalCase"
	// Kebab represents lowercase hyphen-separated casing (e.g. "kebab-case").
	Kebab Kind = "kebab-case"
)

// ToSnake converts s into snake_case (e.g. "HTTPServerURL" -> "http_server_url", "userName" -> "user_name").
func ToSnake(s string) string {
	return convertDelimited(s, '_', false)
}

// ToScreamingSnake converts s into SCREAMING_SNAKE_CASE (e.g. "HTTPServerURL" -> "HTTP_SERVER_URL").
func ToScreamingSnake(s string) string {
	return convertDelimited(s, '_', true)
}

// ToKebab converts s into kebab-case (e.g. "ContentType" -> "content-type").
func ToKebab(s string) string {
	return convertDelimited(s, '-', false)
}

// ToCamel converts s into camelCase (e.g. "user_id" -> "userId", "API_KEY" -> "apiKey").
func ToCamel(s string) string {
	if s == "" {
		return ""
	}
	if isASCII(s) {
		var sb strings.Builder
		sb.Grow(len(s))
		first := true
		forEachWordASCII(s, func(w string) {
			if first {
				writeLowerWordASCII(&sb, w)
				first = false
			} else {
				writeTitleWordASCII(&sb, w)
			}
		})
		return sb.String()
	}

	words := SplitWords(s)
	if len(words) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(s))

	for i, w := range words {
		if i == 0 {
			sb.WriteString(strings.ToLower(w))
		} else {
			sb.WriteString(titleWord(w))
		}
	}

	return sb.String()
}

// ToPascal converts s into PascalCase (e.g. "user_id" -> "UserId", "http_server" -> "HttpServer").
func ToPascal(s string) string {
	if s == "" {
		return ""
	}
	if isASCII(s) {
		var sb strings.Builder
		sb.Grow(len(s))
		forEachWordASCII(s, func(w string) {
			writeTitleWordASCII(&sb, w)
		})
		return sb.String()
	}

	words := SplitWords(s)
	if len(words) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(s))

	for _, w := range words {
		sb.WriteString(titleWord(w))
	}

	return sb.String()
}

// SplitWords splits an identifier into word tokens, respecting camelCase, snake_case, kebab-case, and acronyms.
func SplitWords(s string) []string {
	if s == "" {
		return nil
	}

	if isASCII(s) {
		words := make([]string, 0, 8)
		forEachWordASCII(s, func(w string) {
			words = append(words, w)
		})
		return words
	}

	words := make([]string, 0, 8)
	forEachWordUnicode(s, func(w string) {
		words = append(words, w)
	})
	return words
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func forEachWordASCII(s string, yield func(word string)) {
	n := len(s)
	start := 0
	for i := 0; i < n; i++ {
		c := s[i]
		if c == '_' || c == '-' || c == ' ' || c == '.' || c == '/' {
			if i > start {
				yield(s[start:i])
			}
			start = i + 1
			continue
		}

		if i > start && c >= 'A' && c <= 'Z' {
			prev := s[i-1]
			prevIsUpper := prev >= 'A' && prev <= 'Z'
			if !prevIsUpper {
				yield(s[start:i])
				start = i
			} else if i+1 < n && s[i+1] >= 'a' && s[i+1] <= 'z' {
				yield(s[start:i])
				start = i
			}
		}
	}
	if start < n {
		yield(s[start:])
	}
}

func forEachWordUnicode(s string, yield func(word string)) {
	runes := []rune(s)
	n := len(runes)
	start := 0
	for i := 0; i < n; i++ {
		r := runes[i]
		if r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' {
			if i > start {
				yield(string(runes[start:i]))
			}
			start = i + 1
			continue
		}

		if i > start && unicode.IsUpper(r) {
			prevIsUpper := unicode.IsUpper(runes[i-1])
			if !prevIsUpper {
				yield(string(runes[start:i]))
				start = i
			} else if i+1 < n && unicode.IsLower(runes[i+1]) {
				yield(string(runes[start:i]))
				start = i
			}
		}
	}
	if start < n {
		yield(string(runes[start:]))
	}
}

func writeTitleWordASCII(sb *strings.Builder, w string) {
	if len(w) == 0 {
		return
	}
	c := w[0]
	if c >= 'a' && c <= 'z' {
		c -= 32
	}
	sb.WriteByte(c)
	for i := 1; i < len(w); i++ {
		c = w[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sb.WriteByte(c)
	}
}

func writeLowerWordASCII(sb *strings.Builder, w string) {
	for i := 0; i < len(w); i++ {
		c := w[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		sb.WriteByte(c)
	}
}

func writeUpperWordASCII(sb *strings.Builder, w string) {
	for i := 0; i < len(w); i++ {
		c := w[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		sb.WriteByte(c)
	}
}

func convertDelimited(s string, delimiter byte, screaming bool) string {
	if s == "" {
		return ""
	}

	if isASCII(s) {
		var sb strings.Builder
		sb.Grow(len(s) + 4)
		first := true
		forEachWordASCII(s, func(w string) {
			if !first {
				sb.WriteByte(delimiter)
			}
			first = false
			if screaming {
				writeUpperWordASCII(&sb, w)
			} else {
				writeLowerWordASCII(&sb, w)
			}
		})
		return sb.String()
	}

	words := SplitWords(s)
	if len(words) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(s) + 4)

	for i, w := range words {
		if i > 0 {
			sb.WriteByte(delimiter)
		}
		if screaming {
			sb.WriteString(strings.ToUpper(w))
		} else {
			sb.WriteString(strings.ToLower(w))
		}
	}

	return sb.String()
}

func titleWord(w string) string {
	if w == "" {
		return ""
	}

	runes := []rune(w)
	runes[0] = unicode.ToUpper(runes[0])

	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}

	return string(runes)
}
