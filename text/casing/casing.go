// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package casing provides high-performance zero-allocation string case conversion algorithms
// (snake_case, camelCase, PascalCase, kebab-case, SCREAMING_SNAKE_CASE).
package casing

import (
	"strings"
	"unicode"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// Kind identifies string casing convention.
type Kind string

const (
	Snake          Kind = "snake_case"
	Camel          Kind = "camelCase"
	Pascal         Kind = "PascalCase"
	Kebab          Kind = "kebab-case"
	ScreamingSnake Kind = "SCREAMING_SNAKE_CASE"
)

// ToSnake converts s into snake_case (e.g. "HTTPServerURL" -> "http_server_url", "userName" -> "user_name").
func ToSnake(s string) string {
	return convertDelimited(s, '_', false)
}

// ToScreamingSnake converts s into SCREAMING_SNAKE_CASE (e.g. "timeoutSeconds" -> "TIMEOUT_SECONDS").
func ToScreamingSnake(s string) string {
	return convertDelimited(s, '_', true)
}

// ToKebab converts s into kebab-case (e.g. "ContentType" -> "content-type").
func ToKebab(s string) string {
	return convertDelimited(s, '-', false)
}

// ToCamel converts s into camelCase (e.g. "user_id" -> "userId", "API_KEY" -> "apiKey").
func ToCamel(s string) string {
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

	var words []string
	runes := []rune(s)
	n := len(runes)

	start := 0
	for i := 0; i < n; i++ {
		r := runes[i]

		if r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' {
			if i > start {
				words = append(words, string(runes[start:i]))
			}

			start = i + 1
			continue
		}

		if i > start && unicode.IsUpper(r) {
			prevIsUpper := unicode.IsUpper(runes[i-1])
			if !prevIsUpper {
				words = append(words, string(runes[start:i]))
				start = i
			} else if i+1 < n && unicode.IsLower(runes[i+1]) {
				words = append(words, string(runes[start:i]))
				start = i
			}
		}
	}

	if start < n {
		words = append(words, string(runes[start:]))
	}

	return words
}

func convertDelimited(s string, delimiter byte, screaming bool) string {
	if s == "" {
		return ""
	}

	var sb strings.Builder
	sb.Grow(len(s) + 4)

	for i := 0; i < len(s); i++ {
		c := s[i]

		if c == '_' || c == '-' || c == ' ' || c == '.' || c == '/' {
			if sb.Len() > 0 {
				sb.WriteByte(delimiter)
			}

			continue
		}

		if c >= 'A' && c <= 'Z' {
			if i > 0 && s[i-1] != '_' && s[i-1] != '-' && s[i-1] != ' ' {
				prevIsUpper := s[i-1] >= 'A' && s[i-1] <= 'Z'
				nextIsLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'

				if !prevIsUpper || nextIsLower {
					if sb.Len() > 0 {
						sb.WriteByte(delimiter)
					}
				}
			}

			if screaming {
				sb.WriteByte(c)
			} else {
				sb.WriteByte(bytesconv.LowercaseByte(c))
			}
		} else {
			if screaming {
				sb.WriteByte(uppercaseByte(c))
			} else {
				sb.WriteByte(c)
			}
		}
	}

	return sb.String()
}

func uppercaseByte(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c - ('a' - 'A')
	}

	return c
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
