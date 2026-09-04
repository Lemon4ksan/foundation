// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package url provides zero-allocation URL parsing, caching, path variable expansion,
// and fast query parameter appending.
package urlkit

import (
	"hash/crc32"
	"net/url"
	"strings"
	"sync"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// fastHash computes a hardware CRC32 hash of string s to select a cache shard index.
//
//go:inline
func fastHash(s string) uint32 {
	return crc32.ChecksumIEEE(bytesconv.S2B(s))
}

// cacheShard protects an isolated hash map partition with an RWMutex to minimize lock contention.
type cacheShard struct {
	mu sync.RWMutex
	m  map[string]*url.URL
}

type shardedURLCache struct {
	shards [16]cacheShard
}

var (
	globalURLCache shardedURLCache
	bufPool        = sync.Pool{
		New: func() any {
			b := make([]byte, 0, 512)
			return &b
		},
	}
)

func init() {
	for i := 0; i < 16; i++ {
		globalURLCache.shards[i].m = make(map[string]*url.URL, 16)
	}
}

// URLView represents a parsed URL using direct string slices of rawURL with zero heap allocations.
type URLView struct {
	Scheme   string
	User     string
	Host     string
	Path     string
	RawQuery string
	Fragment string
}

// Hostname returns the host part of u without port, if any.
func (u URLView) Hostname() string {
	h := u.Host
	if strings.HasPrefix(h, "[") {
		// IPv6 literal
		if end := strings.IndexByte(h, ']'); end >= 0 {
			return h[1:end]
		}
	}
	if colon := strings.LastIndexByte(h, ':'); colon >= 0 {
		return h[:colon]
	}
	return h
}

// Port returns the port part of u.Host, or an empty string if no port is present.
func (u URLView) Port() string {
	h := u.Host
	if strings.HasPrefix(h, "[") {
		if end := strings.IndexByte(h, ']'); end >= 0 {
			h = h[end+1:]
		}
	}
	if colon := strings.LastIndexByte(h, ':'); colon >= 0 {
		return h[colon+1:]
	}
	return ""
}

// CanonicalPort returns the effective port number considering standard HTTP/HTTPS defaults.
func (u URLView) CanonicalPort() string {
	p := u.Port()
	if p != "" {
		return p
	}
	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}
	if strings.EqualFold(u.Scheme, "http") {
		return "80"
	}
	return ""
}

// IsAbs reports whether the URLView specifies an absolute scheme.
func (u URLView) IsAbs() bool {
	return u.Scheme != ""
}

// QueryValue returns the first value associated with the given key in RawQuery, or ("", false).
// If the key exists without percent encoding, it returns a zero-alloc slice directly.
func (u URLView) QueryValue(key string) (string, bool) {
	q := u.RawQuery
	for len(q) > 0 {
		var pair string
		if idx := strings.IndexByte(q, '&'); idx >= 0 {
			pair = q[:idx]
			q = q[idx+1:]
		} else {
			pair = q
			q = ""
		}
		if pair == "" {
			continue
		}
		k, val, found := strings.Cut(pair, "=")
		if !found {
			if k == key {
				return "", true
			}
			continue
		}
		if k == key {
			if strings.IndexByte(val, '%') >= 0 || strings.IndexByte(val, '+') >= 0 {
				unesc, err := Unescape(val)
				if err == nil {
					return unesc, true
				}
			}
			return val, true
		}
	}
	return "", false
}

// String reconstructs the full URL string.
func (u URLView) String() string {
	var sb strings.Builder
	if u.Scheme != "" {
		sb.WriteString(u.Scheme)
		sb.WriteString(":")
	}
	if u.Host != "" || u.User != "" || (u.Scheme != "" && (u.Scheme == "http" || u.Scheme == "https")) {
		sb.WriteString("//")
		if u.User != "" {
			sb.WriteString(u.User)
			sb.WriteString("@")
		}
		sb.WriteString(u.Host)
	}
	if u.Path != "" {
		sb.WriteString(u.Path)
	}
	if u.RawQuery != "" {
		sb.WriteString("?")
		sb.WriteString(u.RawQuery)
	}
	if u.Fragment != "" {
		sb.WriteString("#")
		sb.WriteString(u.Fragment)
	}
	return sb.String()
}

// ToURL converts URLView to a standard [*url.URL].
func (u URLView) ToURL() *url.URL {
	res := &url.URL{
		Scheme:   u.Scheme,
		Host:     u.Host,
		Path:     u.Path,
		RawQuery: u.RawQuery,
		Fragment: u.Fragment,
	}
	if u.User != "" {
		if username, password, ok := strings.Cut(u.User, ":"); ok {
			res.User = url.UserPassword(username, password)
		} else {
			res.User = url.User(u.User)
		}
	}
	return res
}

//go:inline
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

//go:inline
func isSchemeChar(c byte) bool {
	return isAlpha(c) || (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.'
}

// ParseView parses rawURL into a zero-allocation [URLView] backed directly by the rawURL string memory.
func ParseView(rawURL string) (URLView, error) {
	if rawURL == "" {
		return URLView{}, nil
	}

	var view URLView

	// 1. Fragment (#)
	if hashIdx := strings.IndexByte(rawURL, '#'); hashIdx >= 0 {
		view.Fragment = rawURL[hashIdx+1:]
		rawURL = rawURL[:hashIdx]
	}

	// 2. Query (?)
	if qIdx := strings.IndexByte(rawURL, '?'); qIdx >= 0 {
		view.RawQuery = rawURL[qIdx+1:]
		rawURL = rawURL[:qIdx]
	}

	// 3. Scheme (RFC 3986 §3.1: ALPHA *( ALPHA / DIGIT / "+" / "-" / "." ))
	rest := rawURL
	if colonIdx := strings.IndexByte(rawURL, ':'); colonIdx > 0 {
		slashIdx := strings.IndexByte(rawURL, '/')
		if slashIdx < 0 || colonIdx < slashIdx {
			schemeCandidate := rawURL[:colonIdx]
			validScheme := isAlpha(schemeCandidate[0])
			if validScheme {
				for i := 1; i < len(schemeCandidate); i++ {
					if !isSchemeChar(schemeCandidate[i]) {
						validScheme = false
						break
					}
				}
			}
			if validScheme {
				view.Scheme = strings.ToLower(schemeCandidate)
				rest = rawURL[colonIdx+1:]
			}
		}
	}

	// 4. Authority (//)
	if strings.HasPrefix(rest, "//") {
		rest = rest[2:]
		slashIdx := strings.IndexByte(rest, '/')
		var authority string
		if slashIdx >= 0 {
			authority = rest[:slashIdx]
			view.Path = rest[slashIdx:]
		} else {
			authority = rest
			view.Path = ""
		}
		if atIdx := strings.LastIndexByte(authority, '@'); atIdx >= 0 {
			view.User = authority[:atIdx]
			view.Host = authority[atIdx+1:]
		} else {
			view.Host = authority
		}
	} else {
		view.Path = rest
	}

	return view, nil
}

// IsAbsURL reports whether path begins with an absolute HTTP or HTTPS scheme identifier (RFC 3986 §3.1 & §4.3).
func IsAbsURL(path string) bool {
	return len(path) >= 7 && (strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://"))
}

// Parse parses rawURL string or returns a cached [*url.URL] pointer.
func Parse(rawURL string) (*url.URL, error) {
	if rawURL == "" {
		return &url.URL{}, nil
	}

	idx := fastHash(rawURL) & 15
	_ = &globalURLCache.shards[15]
	sh := &globalURLCache.shards[idx]

	sh.mu.RLock()
	u, ok := sh.m[rawURL]
	sh.mu.RUnlock()

	if ok {
		clone := *u
		return &clone, nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	cached := *parsed

	sh.mu.Lock()
	if len(sh.m) > 512 {
		clear(sh.m)
	}

	sh.m[rawURL] = &cached
	sh.mu.Unlock()

	return parsed, nil
}

// CloneURL returns a deep copy of u.
func CloneURL(u *url.URL) *url.URL {
	if u == nil {
		return nil
	}

	cloned := *u
	if u.User != nil {
		userCopy := *u.User
		cloned.User = &userCopy
	}

	return &cloned
}

// NormalizeBaseURL parses and normalizes a raw Base URI string, ensuring a trailing slash per RFC 3986 §5.2.3.
func NormalizeBaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return &url.URL{}, nil
	}

	formatted := raw
	if !strings.HasSuffix(formatted, "/") {
		formatted += "/"
	}

	return Parse(formatted)
}

// Resolve resolves a relative path or absolute URL against baseURL (RFC 3986 §5).
// It features an O(1) zero-allocation fast-path for root-domain paths and normalizes slash boundaries.
func Resolve(baseURL *url.URL, path string) (*url.URL, error) {
	if (path == "" || path == "/") && baseURL != nil && baseURL.Host != "" {
		return CloneURL(baseURL), nil
	}

	if IsAbsURL(path) || baseURL == nil || baseURL.Host == "" {
		return Parse(path)
	}

	// Fast-path: root domain (no subpath) + root-relative path (e.g. "https://api.com" + "/users")
	if len(path) > 0 && path[0] == '/' && (baseURL.Path == "" || baseURL.Path == "/") {
		return &url.URL{
			Scheme:   baseURL.Scheme,
			Host:     baseURL.Host,
			Path:     path,
			User:     baseURL.User,
			RawQuery: baseURL.RawQuery,
		}, nil
	}

	targetStr, err := ResolveString(baseURL, path)
	if err != nil {
		return nil, err
	}

	return Parse(targetStr)
}

// ResolveString computes the resolved target URL string from baseURL and path (RFC 3986 §5).
func ResolveString(baseURL *url.URL, path string) (string, error) {
	if IsAbsURL(path) || baseURL == nil || baseURL.Host == "" {
		return path, nil
	}

	if path == "" || path == "/" {
		return baseURL.String(), nil
	}

	baseStr := strings.TrimSuffix(baseURL.String(), "/")
	if path[0] == '/' {
		return baseStr + path, nil
	}

	if !strings.HasPrefix(path, ".") {
		return baseStr + "/" + path, nil
	}

	// If relative path with dot segments (e.g. "../api"), use RFC 3986 reference resolution
	rel, err := Parse(path)
	if err != nil {
		return "", err
	}

	return baseURL.ResolveReference(rel).String(), nil
}

// ReplaceVar performs path variable replacement ({key} -> value) in path.
func ReplaceVar(path, key, value string) string {
	target := "{" + key + "}"

	before, after, ok := strings.Cut(path, target)
	if !ok {
		return path
	}

	bufPtr := bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]

	buf = append(buf, before...)
	buf = append(buf, value...)
	buf = append(buf, after...)

	res := string(buf)

	*bufPtr = buf
	bufPool.Put(bufPtr)

	return res
}

// FastAppendQuery appends key=value to targetURL using SIMD byte detection and pooled buffers.
func FastAppendQuery(targetURL, key, value string) string {
	if key == "" {
		return targetURL
	}

	bufPtr := bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]

	buf = append(buf, targetURL...)
	if strings.IndexByte(targetURL, '?') >= 0 {
		buf = append(buf, '&')
	} else {
		buf = append(buf, '?')
	}

	buf = append(buf, key...)
	buf = append(buf, '=')
	buf = append(buf, value...)

	res := string(buf)

	*bufPtr = buf
	bufPool.Put(bufPtr)

	return res
}

// AppendRawQuery appends raw query string to targetURL using SIMD byte detection and pooled buffers.
func AppendRawQuery(targetURL, rawQuery string) string {
	if rawQuery == "" {
		return targetURL
	}

	bufPtr := bufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0]

	buf = append(buf, targetURL...)
	if strings.IndexByte(targetURL, '?') >= 0 {
		buf = append(buf, '&')
	} else {
		buf = append(buf, '?')
	}

	buf = append(buf, rawQuery...)
	res := string(buf)

	*bufPtr = buf
	bufPool.Put(bufPtr)

	return res
}

// MatchDomainPattern checks if host matches pattern (supporting exact and *.wildcard matches).
func MatchDomainPattern(host, pattern string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	pattern = strings.ToLower(strings.TrimSuffix(pattern, "."))

	if !strings.HasPrefix(pattern, "*.") {
		return host == pattern
	}

	suffix := pattern[1:] // ".example.com"

	return strings.HasSuffix(host, suffix) || host == pattern[2:]
}

// IsCrossOrigin determines whether u1 and u2 belong to different RFC 6454 web origins.
func IsCrossOrigin(u1, u2 *url.URL) bool {
	if u1 == nil || u2 == nil {
		return false
	}

	if !strings.EqualFold(u1.Scheme, u2.Scheme) {
		return true
	}

	h1 := strings.TrimSuffix(u1.Hostname(), ".")
	h2 := strings.TrimSuffix(u2.Hostname(), ".")

	if !bytesconv.EqualFoldASCII(h1, h2) {
		return true
	}

	return CanonicalPort(u1) != CanonicalPort(u2)
}

// CanonicalPort resolves effective port number considering scheme defaults.
func CanonicalPort(u *url.URL) string {
	if u == nil {
		return ""
	}

	port := u.Port()
	if port != "" {
		return port
	}

	if strings.EqualFold(u.Scheme, "https") {
		return "443"
	}

	if strings.EqualFold(u.Scheme, "http") {
		return "80"
	}

	return ""
}

// IsSameDomainOrSubdomain reports whether clean host h1 and h2 match domain or subdomain suffix.
func IsSameDomainOrSubdomain(clean1, clean2 string) bool {
	clean1 = strings.ToLower(clean1)
	clean2 = strings.ToLower(clean2)

	if clean1 == clean2 {
		return true
	}

	return strings.HasSuffix(clean1, "."+clean2) || strings.HasSuffix(clean2, "."+clean1)
}

// BuildPath constructs a final URL path by interpolating pathParams and appending queryParams efficiently.
func BuildPath(basePath string, pathParams map[string]string, queryParams url.Values) string {
	res := basePath

	for k, v := range pathParams {
		res = ReplaceVar(res, k, url.PathEscape(v))
	}

	if len(queryParams) > 0 {
		encoded := queryParams.Encode()
		if encoded != "" {
			if strings.Contains(res, "?") {
				res += "&" + encoded
			} else {
				res += "?" + encoded
			}
		}
	}

	return res
}

const hexDigits = "0123456789ABCDEF"

//go:inline
//go:nosplit
func shouldEscape(c byte) bool {
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' {
		return false
	}

	switch c {
	case '-', '_', '.', '~':
		return false
	}

	return true
}

// AppendQueryEscape appends percent-encoded bytes of src to dst without heap allocations.
func AppendQueryEscape(dst, src []byte) []byte {
	for _, b := range src {
		if !shouldEscape(b) {
			dst = append(dst, b)
		} else {
			dst = append(dst, '%', hexDigits[b>>4], hexDigits[b&15])
		}
	}

	return dst
}

// AppendQueryEscapeString appends percent-encoded bytes of string src to dst without heap allocations.
func AppendQueryEscapeString(dst []byte, src string) []byte {
	for i := 0; i < len(src); i++ {
		b := src[i]
		if !shouldEscape(b) {
			dst = append(dst, b)
		} else {
			dst = append(dst, '%', hexDigits[b>>4], hexDigits[b&15])
		}
	}

	return dst
}

// Unescape unescapes URL percent-encoded characters (%XX) and replaces '+' with ' '
// using hardware vector acceleration.
func Unescape(s string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}
	src := bytesconv.S2B(s)
	buf := make([]byte, len(src))
	n, err := unescapeVector(buf, src)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// UnescapeBytes unescapes URL percent-encoded characters from src into dst.
func UnescapeBytes(dst, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return dst, nil
	}
	start := len(dst)
	if cap(dst)-start < len(src) {
		newDst := make([]byte, start+len(src))
		copy(newDst, dst)
		dst = newDst
	} else {
		dst = dst[:start+len(src)]
	}
	n, err := unescapeVector(dst[start:], src)
	if err != nil {
		return nil, err
	}
	return dst[:start+n], nil
}

func unescapeScalar(dst, src []byte) (int, error) {
	out := 0
	for i := 0; i < len(src); {
		c := src[i]
		switch c {
		case '%':
			if i+2 >= len(src) {
				return 0, ErrInvalidEscape
			}
			hi := fromHexChar(src[i+1])
			lo := fromHexChar(src[i+2])
			if hi < 0 || lo < 0 {
				return 0, ErrInvalidEscape
			}
			dst[out] = byte((hi << 4) | lo)
			out++
			i += 3
		case '+':
			dst[out] = ' '
			out++
			i++
		default:
			dst[out] = c
			out++
			i++
		}
	}
	return out, nil
}

func fromHexChar(c byte) int {
	if c >= '0' && c <= '9' {
		return int(c - '0')
	}
	if c >= 'a' && c <= 'f' {
		return int(c - 'a' + 10)
	}
	if c >= 'A' && c <= 'F' {
		return int(c - 'A' + 10)
	}
	return -1
}
