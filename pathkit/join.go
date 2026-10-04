// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit

import (
	"strings"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// AppendJoinURL appends base and subsequent path segments into dst as a canonical URL path
// without mangling scheme prefixes (e.g. https://).
//
// Concurrency & Zero-Allocation Semantics:
// When dst has sufficient capacity, AppendJoinURL performs strictly zero heap allocations.
func AppendJoinURL(dst []byte, base string, parts ...string) []byte {
	if len(parts) == 0 {
		return append(dst, base...)
	}

	if base == "" {
		for len(parts) > 0 && parts[0] == "" {
			parts = parts[1:]
		}
		if len(parts) == 0 {
			return dst
		}
		base = parts[0]
		parts = parts[1:]
	}

	schemeIdx := strings.Index(base, "://")
	var prefix string
	var basePath string

	if schemeIdx != -1 {
		prefix = base[:schemeIdx+3]
		basePath = base[schemeIdx+3:]
	} else {
		basePath = base
	}

	dst = append(dst, prefix...)
	dst = append(dst, basePath...)

	for _, part := range parts {
		if part == "" {
			continue
		}

		hasTrailing := len(dst) > len(prefix) && dst[len(dst)-1] == '/'
		hasLeading := part[0] == '/'

		switch {
		case len(dst) == len(prefix):
			dst = append(dst, part...)
		case hasTrailing && hasLeading:
			// Strip leading slashes from part to avoid duplicate slashes
			trimmed := strings.TrimLeft(part, "/")
			dst = append(dst, trimmed...)
		case !hasTrailing && !hasLeading:
			dst = append(dst, '/')
			dst = append(dst, part...)
		default:
			dst = append(dst, part...)
		}
	}

	return dst
}

// JoinURL joins base and subsequent path segments into a canonical URL path without mangling scheme prefixes (e.g. https://).
// Unlike path.Join, it guarantees that "https://" is not collapsed into "https:/",
// and unlike filepath.Join on Windows, it strictly preserves forward slashes ("/").
//
// Allocations: exactly 1 heap allocation for the returned string buffer.
func JoinURL(base string, parts ...string) string {
	if len(parts) == 0 {
		return base
	}

	// Pre-calculate exact capacity to avoid multiple slice growths
	totalLen := len(base)
	for _, p := range parts {
		totalLen += len(p) + 1
	}

	buf := make([]byte, 0, totalLen)
	buf = AppendJoinURL(buf, base, parts...)
	return bytesconv.B2S(buf)
}

// AppendClean appends the shortest path name equivalent to p by purely lexical processing into dst.
// If the path contains a scheme (e.g. "https://domain.com/a/../b"), the scheme and authority are preserved
// while the path portion is cleaned.
//
// Concurrency & Zero-Allocation Semantics:
// When dst has sufficient capacity, AppendClean performs strictly zero heap allocations.
func AppendClean(dst []byte, p string) []byte {
	if p == "" {
		return append(dst, '.')
	}

	schemeIdx := strings.Index(p, "://")
	if schemeIdx != -1 {
		prefix := p[:schemeIdx+3]
		rest := p[schemeIdx+3:]

		// Find first slash after authority
		slashIdx := strings.IndexByte(rest, '/')
		if slashIdx == -1 {
			return append(dst, p...)
		}

		authority := rest[:slashIdx]
		pathPart := rest[slashIdx:]

		dst = append(dst, prefix...)
		dst = append(dst, authority...)
		return cleanLexicalPath(dst, pathPart)
	}

	return cleanLexicalPath(dst, p)
}

// Clean returns the shortest path name equivalent to path by purely lexical processing.
// If the path contains a scheme (e.g. "https://domain.com/a/../b"), the scheme and authority are preserved
// while the path portion is cleaned.
func Clean(p string) string {
	if p == "" {
		return "."
	}

	buf := make([]byte, 0, len(p))
	buf = AppendClean(buf, p)
	return bytesconv.B2S(buf)
}

// AppendSanitize strips Directory Traversal sequences ("../", "..\", etc.) from p to ensure safe usage in file lookups,
// appending the result into dst.
//
// Concurrency & Zero-Allocation Semantics:
// When dst has sufficient capacity, AppendSanitize performs strictly zero heap allocations.
func AppendSanitize(dst []byte, p string) []byte {
	if p == "" {
		return dst
	}

	// Convert any backslashes to slashes and ensure leading slash for safe traversal stripping
	startLen := len(dst)

	// Clean as a rooted path: leading slash causes lexical cleaner to drop leading ".."
	dst = append(dst, '/')
	for i := 0; i < len(p); i++ {
		b := p[i]
		if b == '\\' {
			dst = append(dst, '/')
		} else {
			dst = append(dst, b)
		}
	}

	// Perform in-place lexical clean on the segment added
	rawAdded := bytesconv.B2S(dst[startLen:])
	dst = dst[:startLen]
	dst = cleanLexicalPath(dst, rawAdded)

	// Trim leading slash and single "."
	res := dst[startLen:]
	for len(res) > 0 && res[0] == '/' {
		res = res[1:]
	}
	if len(res) == 1 && res[0] == '.' {
		res = res[:0]
	}

	// Shift back if needed
	if len(res) < len(dst[startLen:]) {
		copy(dst[startLen:], res)
		dst = dst[:startLen+len(res)]
	}

	return dst
}

// Sanitize strips Directory Traversal sequences ("../", "..\", etc.) from path to ensure safe usage in file lookups.
func Sanitize(p string) string {
	if p == "" {
		return ""
	}

	buf := make([]byte, 0, len(p))
	buf = AppendSanitize(buf, p)
	return bytesconv.B2S(buf)
}

// cleanLexicalPath applies standard lexical path cleaning rules directly into dst.
func cleanLexicalPath(dst []byte, s string) []byte {
	if s == "" {
		return append(dst, '.')
	}

	rooted := s[0] == '/'
	startLen := len(dst)

	if rooted {
		dst = append(dst, '/')
	}

	// Track segment boundaries for popping on ".."
	// Using a stack of segment start offsets
	var segStarts [32]int
	segStack := segStarts[:0]

	numRegular := 0

	i := 0
	for i < len(s) {
		// Skip multiple slashes
		for i < len(s) && s[i] == '/' {
			i++
		}
		if i >= len(s) {
			break
		}

		// Find end of segment
		segStart := i
		for i < len(s) && s[i] != '/' {
			i++
		}
		seg := s[segStart:i]

		if seg == "." {
			continue
		}

		if seg == ".." {
			if numRegular > 0 {
				// Pop previous regular segment
				lastStart := segStack[len(segStack)-1]
				segStack = segStack[:len(segStack)-1]
				numRegular--
				dst = dst[:lastStart]
				// If we popped down to the root slash, keep the root slash
				if rooted && len(dst) == startLen {
					dst = append(dst, '/')
				}
				continue
			}
			if !rooted {
				// Cannot pop; append ".."
				if len(dst) > startLen && dst[len(dst)-1] != '/' {
					dst = append(dst, '/')
				}
				segStack = append(segStack, len(dst))
				dst = append(dst, '.', '.')
			}
			continue
		}

		// Regular segment
		if len(dst) > startLen && dst[len(dst)-1] != '/' {
			dst = append(dst, '/')
		}
		segStack = append(segStack, len(dst))
		numRegular++
		dst = append(dst, seg...)
	}

	if len(dst) == startLen {
		if rooted {
			return append(dst, '/')
		}
		return append(dst, '.')
	}

	return dst
}
