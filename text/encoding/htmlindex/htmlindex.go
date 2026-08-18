// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gen.go

// Package htmlindex maps character set encoding names to Encodings as
// recommended by the W3C for use in HTML 5. See http://www.w3.org/TR/encoding.
package htmlindex

// TODO: perhaps have a "bare" version of the index (used by this package) that
// is not pre-loaded with all encodings. Global variables in encodings prevent
// the linker from being able to purge unneeded tables. This means that
// referencing all encodings, as this package does for the default index, links
// in all encodings unconditionally.
//
// This issue can be solved by either solving the linking issue (see
// https://github.com/golang/go/issues/6330) or refactoring the encoding tables
// (e.g. moving the tables to internal packages that do not use global
// variables).

// TODO: allow canonicalizing names

import (
	"errors"
	"strings"

	"github.com/lemon4ksan/foundation/text/encoding"
	"github.com/lemon4ksan/foundation/text/encoding/internal/identifier"
)

var (
	errInvalidName = errors.New("htmlindex: invalid encoding name")
	errUnknown     = errors.New("htmlindex: unknown Encoding")
	errUnsupported = errors.New("htmlindex: this encoding is not supported")
)

var languageDefaults = map[string]string{
	"ar": "windows-1256",
	"ba": "windows-1251",
	"be": "windows-1251",
	"bg": "windows-1251",
	"cs": "windows-1250",
	"el": "iso-8859-7",
	"et": "windows-1257",
	"fa": "windows-1256",
	"he": "windows-1255",
	"hr": "windows-1250",
	"hu": "iso-8859-2",
	"ja": "shift_jis",
	"kk": "windows-1251",
	"ko": "euc-kr",
	"ku": "windows-1254",
	"ky": "windows-1251",
	"lt": "windows-1257",
	"lv": "windows-1257",
	"mk": "windows-1251",
	"pl": "iso-8859-2",
	"ru": "windows-1251",
	"sah": "windows-1251",
	"sk": "windows-1250",
	"sl": "iso-8859-2",
	"sr": "windows-1251",
	"tg": "windows-1251",
	"th": "windows-874",
	"tr": "windows-1254",
	"tt": "windows-1251",
	"uk": "windows-1251",
	"vi": "windows-1258",
	"zh-hans": "gb18030",
	"zh-hant": "big5",
	"zh": "gb18030",
	"bs": "windows-1250",
}

// LanguageDefault returns the canonical name of the default encoding for a given language tag string.
func LanguageDefault(tag string) string {
	tag = strings.ToLower(strings.TrimSpace(tag))
	if enc, ok := languageDefaults[tag]; ok {
		return enc
	}
	if idx := strings.IndexAny(tag, "-_"); idx > 0 {
		if enc, ok := languageDefaults[tag[:idx]]; ok {
			return enc
		}
	}
	return "windows-1252"
}

// Get returns an Encoding for one of the names listed in
// http://www.w3.org/TR/encoding using the Default Index. Matching is case-
// insensitive.
func Get(name string) (encoding.Encoding, error) {
	x, ok := nameMap[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, errInvalidName
	}
	return encodings[x], nil
}

// Name reports the canonical name of the given Encoding. It will return
// an error if e is not associated with a supported encoding scheme.
func Name(e encoding.Encoding) (string, error) {
	id, ok := e.(identifier.Interface)
	if !ok {
		return "", errUnknown
	}
	mib, _ := id.ID()
	if mib == 0 {
		return "", errUnknown
	}
	v, ok := mibMap[mib]
	if !ok {
		return "", errUnsupported
	}
	return canonical[v], nil
}
