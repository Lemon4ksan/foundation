// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit

import (
	"iter"
	"net/url"
	"path/filepath"
	"runtime"
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
// When s contains no backslashes and already has a normalized scheme, New performs zero heap allocations.
func New(s string) Path {
	s = strings.TrimSpace(s)
	if s == "" {
		return Path{}
	}

	if rawScheme, _, ok := strings.Cut(s, "://"); ok {
		scheme := normalizeScheme(rawScheme)
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

	// Local path: only replace backslashes if present
	if strings.IndexByte(s, '\\') != -1 {
		return Path{
			raw:    strings.ReplaceAll(s, "\\", "/"),
			scheme: "",
			isURL:  false,
		}
	}

	return Path{
		raw:    s,
		scheme: "",
		isURL:  false,
	}
}

// normalizeScheme returns a lowercase scheme string without allocating if already lowercase.
func normalizeScheme(s string) string {
	switch s {
	case "http", "https", "file", "s3", "ws", "wss", "ftp", "ssh", "git":
		return s
	case "HTTP":
		return "http"
	case "HTTPS":
		return "https"
	case "FILE":
		return "file"
	case "S3":
		return "s3"
	case "WS":
		return "ws"
	case "WSS":
		return "wss"
	}

	hasUpper := false
	for i := 0; i < len(s); i++ {
		if s[i] >= 'A' && s[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return s
	}

	return strings.ToLower(s)
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

// Append appends the normalized string representation of p to dst without heap allocations when dst has capacity.
func (p Path) Append(dst []byte) []byte {
	return append(dst, p.raw...)
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

// Authority returns the authority (host/domain and optional port/userinfo) component of the path
// between "://" and the subsequent slash, or empty string if not present.
func (p Path) Authority() string {
	idx := strings.Index(p.raw, "://")
	if idx == -1 {
		return ""
	}
	rest := p.raw[idx+3:]
	slash := strings.IndexAny(rest, "/\\")
	if slash == -1 {
		return rest
	}
	return rest[:slash]
}

// IsAbs reports whether the path is absolute.
func (p Path) IsAbs() bool {
	if p.isURL || p.scheme == "file" {
		return true
	}

	// Windows UNC paths (e.g. \\server\share or //server/share)
	if strings.HasPrefix(p.raw, `\\`) || strings.HasPrefix(p.raw, "//") {
		return true
	}

	// Windows drive-letter paths require a root slash (e.g. C:\ or C:/).
	// Bare "C:" or "C:foo" is drive-relative, not absolute.
	if len(p.raw) >= 3 && p.raw[1] == ':' &&
		((p.raw[0] >= 'a' && p.raw[0] <= 'z') || (p.raw[0] >= 'A' && p.raw[0] <= 'Z')) &&
		(p.raw[2] == '/' || p.raw[2] == '\\') {
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

	if raw == "/" || raw == "\\" {
		return "/"
	}

	// Trim trailing slashes
	trimmed := strings.TrimRight(raw, "/\\")
	if trimmed == "" {
		return "/"
	}

	slashIdx := strings.LastIndexAny(trimmed, "/\\")
	if slashIdx == -1 {
		return trimmed
	}

	return trimmed[slashIdx+1:]
}

// Ext returns the file name extension (e.g. ".json" or ".gz").
func (p Path) Ext() string {
	base := p.Base()
	i := strings.LastIndexByte(base, '.')
	if i <= 0 {
		return ""
	}
	return base[i:]
}

// Stem returns the base file name without its extension (e.g. "app.prod" from "app.prod.json").
func (p Path) Stem() string {
	base := p.Base()
	ext := p.Ext()
	return strings.TrimSuffix(base, ext)
}

// Dir returns all but the last element of path, typically the path's directory.
func (p Path) Dir() Path {
	if p.raw == "" {
		return Path{raw: ".", scheme: "", isURL: false}
	}

	if p.isURL || p.scheme != "" {
		schemeIdx := strings.Index(p.raw, "://")
		prefix := p.raw[:schemeIdx+3]
		rest := p.raw[schemeIdx+3:]

		trimmedRest := strings.TrimRight(rest, "/\\")
		slashIdx := strings.LastIndexAny(trimmedRest, "/\\")
		if slashIdx == -1 {
			return Path{
				raw:    p.raw,
				scheme: p.scheme,
				isURL:  p.isURL,
			}
		}

		return Path{
			raw:    prefix + trimmedRest[:slashIdx],
			scheme: p.scheme,
			isURL:  p.isURL,
		}
	}

	// Local path
	trimmed := strings.TrimRight(p.raw, "/\\")
	if trimmed == "" {
		return Path{raw: "/", scheme: "", isURL: false}
	}

	slashIdx := strings.LastIndexAny(trimmed, "/\\")
	if slashIdx == -1 {
		// Windows drive root (e.g. "C:" or "C:/" or "C:\")
		if len(trimmed) == 2 && trimmed[1] == ':' {
			if len(p.raw) > 2 {
				return Path{raw: p.raw[:3], scheme: "", isURL: false}
			}
			return Path{raw: trimmed, scheme: "", isURL: false}
		}
		return Path{raw: ".", scheme: "", isURL: false}
	}

	if slashIdx == 0 {
		return Path{raw: "/", scheme: "", isURL: false}
	}

	// Windows drive letter e.g. "C:/"
	if slashIdx == 2 && len(trimmed) >= 2 && trimmed[1] == ':' {
		return Path{raw: trimmed[:3], scheme: "", isURL: false}
	}

	return Path{
		raw:    trimmed[:slashIdx],
		scheme: "",
		isURL:  false,
	}
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

	return Path{
		raw:    newRaw,
		scheme: p.scheme,
		isURL:  p.isURL,
	}
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
	return Path{
		raw:    Clean(p.raw),
		scheme: p.scheme,
		isURL:  p.isURL,
	}
}

// AppendFilePath appends the native OS file path representation suitable for [os.Open] into dst.
// On Windows, appends backslashed paths ("C:\foo\bar"); on Unix, appends forward slashes ("/foo/bar").
func (p Path) AppendFilePath(dst []byte) []byte {
	if p.scheme == "file" {
		var err error
		dst, err = AppendURIToPath(dst, p.raw)
		if err == nil {
			return dst
		}
	}

	start := len(dst)
	dst = append(dst, p.raw...)
	if runtime.GOOS == "windows" {
		for i := start; i < len(dst); i++ {
			if dst[i] == '/' {
				dst[i] = '\\'
			}
		}
	}
	return dst
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

// AppendURI appends an RFC 8089 compliant URI string to dst.
func (p Path) AppendURI(dst []byte) []byte {
	if p.isURL || p.scheme == "file" {
		return append(dst, p.raw...)
	}

	return AppendPathToURI(dst, p.raw)
}

// URI returns an RFC 8089 compliant URI string (e.g. "file:///C:/foo/bar" or "https://domain.com/path").
func (p Path) URI() string {
	if p.isURL || p.scheme == "file" {
		return p.raw
	}

	return PathToURI(p.raw)
}

// Segments returns an iterator over the non-empty path segments of pathStr.
// Iteration stops immediately when yield returns false.
// Segments operates purely by slicing substrings without heap allocations across all operating systems.
func Segments(pathStr string) iter.Seq[string] {
	return func(yield func(string) bool) {
		p := pathStr
		if idx := strings.Index(p, "://"); idx != -1 {
			rest := p[idx+3:]
			if slash := strings.IndexAny(rest, "/\\"); slash != -1 {
				p = rest[slash:]
			} else {
				p = ""
			}
		}

		for len(p) > 0 {
			// Skip leading slashes
			for len(p) > 0 && (p[0] == '/' || p[0] == '\\') {
				p = p[1:]
			}
			if len(p) == 0 {
				break
			}

			nextSlash := strings.IndexAny(p, "/\\")
			var seg string
			if nextSlash == -1 {
				seg = p
				p = ""
			} else {
				seg = p[:nextSlash]
				p = p[nextSlash+1:]
			}

			if !yield(seg) {
				return
			}
		}
	}
}

// Segments returns an iterator over the path segments of p.
func (p Path) Segments() iter.Seq[string] {
	return Segments(p.raw)
}

// Segments2 returns a push iterator over index and segment pairs of p.
func (p Path) Segments2() iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		idx := 0
		for seg := range p.Segments() {
			if !yield(idx, seg) {
				return
			}
			idx++
		}
	}
}

// Ancestors returns a push iterator yielding directory ancestors from immediate parent up to the root.
// For example, for "/a/b/c", yields "/a/b", "/a", and "/".
func (p Path) Ancestors() iter.Seq[Path] {
	return func(yield func(Path) bool) {
		curr := p.Dir()
		prev := p
		for !curr.IsEmpty() && curr != prev && curr.raw != "." {
			if !yield(curr) {
				return
			}
			prev = curr
			curr = curr.Dir()
		}
	}
}

func isDriveSegment(s string) bool {
	return len(s) == 2 && s[1] == ':' &&
		((s[0] >= 'a' && s[0] <= 'z') || (s[0] >= 'A' && s[0] <= 'Z'))
}

// HasPrefix reports whether p has the directory prefix prefix.
// Both paths are compared segment by segment.
func (p Path) HasPrefix(prefix Path) bool {
	if p.isURL != prefix.isURL || p.scheme != prefix.scheme {
		return false
	}
	if p.Authority() != prefix.Authority() {
		return false
	}

	pSegs := p.Segments()
	prefixSegs := prefix.Segments()

	pNext, pStop := iter.Pull(pSegs)
	defer pStop()

	prefixNext, prefixStop := iter.Pull(prefixSegs)
	defer prefixStop()

	for {
		preSeg, preOk := prefixNext()
		if !preOk {
			// All prefix segments matched
			return true
		}

		pSeg, pOk := pNext()
		segMatch := pSeg == preSeg
		if !segMatch && isDriveSegment(pSeg) && isDriveSegment(preSeg) && strings.EqualFold(pSeg, preSeg) {
			segMatch = true
		}
		if !pOk || !segMatch {
			return false
		}
	}
}

// volume returns the root volume prefix for Windows drive letters (e.g. "C:")
// or UNC paths (e.g. "//server/share").
func volume(raw string) string {
	if len(raw) >= 2 && raw[1] == ':' &&
		((raw[0] >= 'a' && raw[0] <= 'z') || (raw[0] >= 'A' && raw[0] <= 'Z')) {
		return raw[:2]
	}
	if strings.HasPrefix(raw, "//") || strings.HasPrefix(raw, `\\`) {
		rest := raw[2:]
		idx := strings.IndexAny(rest, "/\\")
		if idx == -1 {
			return raw
		}
		shareRest := rest[idx+1:]
		shareIdx := strings.IndexAny(shareRest, "/\\")
		if shareIdx == -1 {
			return raw
		}
		return raw[:2+idx+1+shareIdx]
	}
	return ""
}

// Rel computes the relative path from base to p.
// If the paths do not share a common base or have incompatible schemes, returns (Path{}, false).
// When paths are identical or incompatible, Rel performs strictly zero heap allocations.
// Parallel segment traversal uses iter.Pull without slice allocations.
func (p Path) Rel(base Path) (Path, bool) {
	if p.isURL != base.isURL || p.scheme != base.scheme {
		return Path{}, false
	}
	if p.Authority() != base.Authority() {
		return Path{}, false
	}
	if p.IsAbs() != base.IsAbs() {
		return Path{}, false
	}
	if !strings.EqualFold(volume(p.raw), volume(base.raw)) {
		return Path{}, false
	}

	pClean := strings.TrimRight(p.raw, "/\\")
	baseClean := strings.TrimRight(base.raw, "/\\")
	if pClean == baseClean {
		return Path{raw: ".", scheme: "", isURL: false}, true
	}

	baseNext, baseStop := iter.Pull(base.Segments())
	defer baseStop()

	pNext, pStop := iter.Pull(p.Segments())
	defer pStop()

	var (
		diverged  bool
		upCount   int
		pFirst    string
		pHasFirst bool
	)

	for {
		bSeg, bOk := baseNext()
		pSeg, pOk := pNext()

		if !bOk && !pOk {
			// Both ended simultaneously without divergence
			break
		}

		segMatch := bSeg == pSeg
		if !segMatch && isDriveSegment(bSeg) && isDriveSegment(pSeg) && strings.EqualFold(bSeg, pSeg) {
			segMatch = true
		}

		if bOk && pOk && segMatch && !diverged {
			// In lockstep matching
			continue
		}

		// Divergence reached!
		diverged = true

		if bOk {
			upCount = 1
		}
		if pOk {
			pFirst = pSeg
			pHasFirst = true
		}
		break
	}

	if !diverged {
		return Path{raw: ".", scheme: "", isURL: false}, true
	}

	// Count remaining base segments that need "../"
	for {
		_, bOk := baseNext()
		if !bOk {
			break
		}
		upCount++
	}

	var sb strings.Builder
	for i := 0; i < upCount; i++ {
		if sb.Len() > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString("..")
	}

	if pHasFirst {
		if sb.Len() > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(pFirst)
	}

	for {
		pSeg, pOk := pNext()
		if !pOk {
			break
		}
		if sb.Len() > 0 {
			sb.WriteByte('/')
		}
		sb.WriteString(pSeg)
	}

	if sb.Len() == 0 {
		return Path{raw: ".", scheme: "", isURL: false}, true
	}

	return Path{raw: sb.String(), scheme: "", isURL: false}, true
}

// Match reports whether the path's Base matches the shell pattern.
// Supported patterns: '*' (matches any sequence of non-separator chars),
// '?' (matches any single non-separator char), and character classes '[...]'.
func (p Path) Match(pattern string) bool {
	base := p.Base()
	matched, _ := filepath.Match(pattern, base)
	return matched
}
