// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit

import (
	"path"
	"strings"
)

// JoinURL joins base and subsequent path segments into a canonical URL path without mangling scheme prefixes (e.g. https://).
// Unlike path.Join, it guarantees that "https://" is not collapsed into "https:/",
// and unlike filepath.Join on Windows, it strictly preserves forward slashes ("/").
func JoinURL(base string, parts ...string) string {
	if len(parts) == 0 {
		return base
	}

	if base == "" {
		return JoinURL(parts[0], parts[1:]...)
	}

	var sb strings.Builder
	// Approximate allocation size
	totalLen := len(base) + 1
	for _, p := range parts {
		totalLen += len(p) + 1
	}
	sb.Grow(totalLen)

	schemeIdx := strings.Index(base, "://")
	var prefix string
	var basePath string

	if schemeIdx != -1 {
		prefix = base[:schemeIdx+3]
		basePath = base[schemeIdx+3:]
	} else {
		basePath = base
	}

	sb.WriteString(prefix)

	// Clean base path and parts
	curr := basePath
	for _, part := range parts {
		if part == "" {
			continue
		}

		currHasSlash := strings.HasSuffix(curr, "/")
		partHasSlash := strings.HasPrefix(part, "/")

		switch {
		case curr == "":
			curr = part
		case currHasSlash && partHasSlash:
			curr += strings.TrimPrefix(part, "/")
		case !currHasSlash && !partHasSlash:
			curr += "/" + part
		default:
			curr += part
		}
	}

	sb.WriteString(curr)

	return sb.String()
}

// Clean returns the shortest path name equivalent to path by purely lexical processing.
// If the path contains a scheme (e.g. "https://domain.com/a/../b"), the scheme and authority are preserved
// while the path portion is cleaned.
func Clean(p string) string {
	if p == "" {
		return "."
	}

	schemeIdx := strings.Index(p, "://")
	if schemeIdx != -1 {
		prefix := p[:schemeIdx+3]
		rest := p[schemeIdx+3:]

		// Find first slash after authority
		slashIdx := strings.IndexByte(rest, '/')
		if slashIdx == -1 {
			return p
		}

		authority := rest[:slashIdx]
		pathPart := rest[slashIdx:]
		cleanedPath := path.Clean(pathPart)

		return prefix + authority + cleanedPath
	}

	return path.Clean(p)
}

// Sanitize strips Directory Traversal sequences ("../", "..\", etc.) from path to ensure safe usage in file lookups.
func Sanitize(p string) string {
	if p == "" {
		return ""
	}

	if strings.IndexByte(p, '\\') != -1 {
		p = strings.ReplaceAll(p, "\\", "/")
	}

	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}

	cleaned := path.Clean(p)
	return strings.TrimPrefix(cleaned, "/")
}
