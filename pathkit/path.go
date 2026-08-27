// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"
)

// Path represents an immutable, unified path abstraction bridging
// remote network URLs (https://...), local OS filepaths (C:\... or /etc/...), and URIs (file://..., s3://...).
type Path struct {
	raw    string
	scheme string
	isURL  bool
}

// New creates a new [Path] from a raw string, automatically classifying its scheme and structure.
func New(s string) Path {
	s = strings.TrimSpace(s)
	if s == "" {
		return Path{}
	}

	schemeIdx := strings.Index(s, "://")
	if schemeIdx != -1 {
		scheme := strings.ToLower(s[:schemeIdx])
		if scheme == "file" {
			return Path{
				raw:    s,
				scheme: "file",
				isURL:  false,
			}
		}

		return Path{
			raw:    s,
			scheme: scheme,
			isURL:  true,
		}
	}

	// Local path
	return Path{
		raw:    strings.ReplaceAll(s, "\\", "/"),
		scheme: "",
		isURL:  false,
	}
}

// FromFilePath constructs a [Path] from an OS-native file system path.
func FromFilePath(osPath string) Path {
	return New(osPath)
}

// FromURI constructs a [Path] from a URI string.
func FromURI(uriStr string) (Path, error) {
	p := New(uriStr)
	return p, nil
}

// FromURL constructs a [Path] from standard [*url.URL].
func FromURL(u *url.URL) Path {
	if u == nil {
		return Path{}
	}

	return New(u.String())
}

// String returns the normalized string representation.
func (p Path) String() string {
	return p.raw
}

// IsEmpty reports whether the path is empty.
func (p Path) IsEmpty() bool {
	return p.raw == ""
}

// IsURL reports whether the path is a remote network URL (e.g. http://, https://, s3://, ws://).
func (p Path) IsURL() bool {
	return p.isURL
}

// IsFile reports whether the path represents a local file system path or file:// URI.
func (p Path) IsFile() bool {
	return !p.isURL
}

// Scheme returns the scheme component of the path (e.g. "https", "file", "s3", or empty string).
func (p Path) Scheme() string {
	return p.scheme
}

// IsAbs reports whether the path is absolute.
func (p Path) IsAbs() bool {
	if p.isURL || p.scheme == "file" {
		return true
	}

	if len(p.raw) >= 2 && p.raw[1] == ':' {
		return true
	}

	return strings.HasPrefix(p.raw, "/")
}

// Base returns the last element of path (e.g. "app.json" from "/var/log/app.json" or "42" from "https://api.com/users/42").
func (p Path) Base() string {
	if p.raw == "" {
		return "."
	}

	raw := p.raw
	if p.isURL || p.scheme != "" {
		schemeIdx := strings.Index(raw, "://")
		if schemeIdx != -1 {
			raw = raw[schemeIdx+3:]
		}
	}

	// Trim trailing slashes
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return ""
	}

	slashIdx := strings.LastIndexByte(raw, '/')
	if slashIdx == -1 {
		return raw
	}

	return raw[slashIdx+1:]
}

// Ext returns the file name extension (e.g. ".json" or ".gz").
func (p Path) Ext() string {
	base := p.Base()
	return path.Ext(base)
}

// Stem returns the base file name without its extension (e.g. "app.prod" from "app.prod.json").
func (p Path) Stem() string {
	base := p.Base()
	ext := path.Ext(base)

	return strings.TrimSuffix(base, ext)
}

// Dir returns all but the last element of path, typically the path's directory.
func (p Path) Dir() Path {
	if p.raw == "" {
		return New(".")
	}

	if p.isURL || p.scheme != "" {
		schemeIdx := strings.Index(p.raw, "://")
		prefix := p.raw[:schemeIdx+3]
		rest := p.raw[schemeIdx+3:]

		slashIdx := strings.LastIndexByte(rest, '/')
		if slashIdx == -1 {
			return New(prefix + rest)
		}

		return New(prefix + rest[:slashIdx])
	}

	dir := path.Dir(p.raw)
	return New(dir)
}

// WithExt returns a new [Path] with its extension replaced or appended.
func (p Path) WithExt(newExt string) Path {
	if p.raw == "" {
		return p
	}

	if newExt != "" && !strings.HasPrefix(newExt, ".") {
		newExt = "." + newExt
	}

	currentExt := p.Ext()
	newRaw := strings.TrimSuffix(p.raw, currentExt) + newExt

	return New(newRaw)
}

// Join joins path with subsequent path segments safely.
func (p Path) Join(segments ...string) Path {
	if len(segments) == 0 {
		return p
	}

	if p.raw == "" {
		return New(JoinURL(segments[0], segments[1:]...))
	}

	return New(JoinURL(p.raw, segments...))
}

// Clean returns the shortest path name equivalent to path by purely lexical processing.
func (p Path) Clean() Path {
	return New(Clean(p.raw))
}

// FilePath returns the native OS file path representation suitable for [os.Open] or standard file operations.
// On Windows, returns backslashed paths ("C:\foo\bar"); on Unix, returns forward slashes ("/foo/bar").
func (p Path) FilePath() string {
	if p.scheme == "file" {
		if osPath, err := URIToPath(p.raw); err == nil {
			return osPath
		}
	}

	return filepath.FromSlash(p.raw)
}

// URI returns an RFC 8089 compliant URI string (e.g. "file:///C:/foo/bar" or "https://domain.com/path").
func (p Path) URI() string {
	if p.isURL || p.scheme == "file" {
		return p.raw
	}

	return PathToURI(p.raw)
}
