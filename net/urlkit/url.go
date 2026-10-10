// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit

import (
	"errors"
	"iter"
	"net/url"
	"strings"
	"sync"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

type UnescapeMode int

const (
	UnescapeModeQuery UnescapeMode = iota
	UnescapeModePath
)

// ErrInvalidHost indicates that a host name contains forbidden or invalid ASCII characters.
var ErrInvalidHost = errors.New("url: invalid character in host name")

// fastHash computes a hardware fast hash of string s to select a cache shard index.
//
//go:inline
func fastHash(s string) uint32 {
	return uint32(bytesconv.FastHash64(bytesconv.S2B(s)))
}

// cacheShard protects an isolated hash map partition with an RWMutex to minimize lock contention.
//
// It is padded to 64 bytes to eliminate false sharing across CPU cachelines.
type cacheShard struct {
	mu sync.RWMutex
	m  map[string]*url.URL
	_  [32]byte
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
	for i := range 16 {
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

// Hostname returns the host part without the port, if any.
func (u URLView) Hostname() string {
	h := u.Host
	if strings.HasPrefix(h, "[") {
		// IPv6 literal
		if end := strings.IndexByte(h, ']'); end >= 0 {
			ip := h[1:end]
			if zoneIdx := strings.Index(ip, "%25"); zoneIdx >= 0 {
				ip = ip[:zoneIdx] + "%" + ip[zoneIdx+3:]
			}
			return ip
		}
		return h
	}
	colon := strings.LastIndexByte(h, ':')
	if colon >= 0 && strings.IndexByte(h, ':') == colon {
		return h[:colon]
	}
	return h
}

// Port returns the port part of the host, or an empty string if none is present.
func (u URLView) Port() string {
	h := u.Host
	if strings.HasPrefix(h, "[") {
		if end := strings.IndexByte(h, ']'); end >= 0 {
			h = h[end+1:]
		} else {
			return ""
		}
	}
	colon := strings.LastIndexByte(h, ':')
	if colon >= 0 && strings.IndexByte(h, ':') == colon {
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

// IsAbs reports whether the view specifies an absolute scheme.
func (u URLView) IsAbs() bool {
	return u.Scheme != ""
}

// QueryValue returns the first value associated with the given key in the raw query, or an empty string and false.
//
// If the key exists without percent encoding, it returns a slice directly.
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
		
		if strings.IndexByte(k, '%') >= 0 || strings.IndexByte(k, '+') >= 0 {
			if unescK, err := QueryUnescape(k); err == nil {
				k = unescK
			}
		}

		if !found {
			if k == key {
				return "", true
			}
			continue
		}
		if k == key {
			if strings.IndexByte(val, '%') >= 0 || strings.IndexByte(val, '+') >= 0 {
				unesc, err := QueryUnescape(val)
				if err == nil {
					return unesc, true
				}
			}
			return val, true
		}
	}
	return "", false
}

// QuerySeq returns an iterator yielding key-value pairs from the raw query.
//
// Keys and values with percent-encoding or '+' are unescaped using fast vector routines.
func (u URLView) QuerySeq() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
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
				val = ""
			}
			if strings.IndexByte(k, '%') >= 0 || strings.IndexByte(k, '+') >= 0 {
				if unescK, err := QueryUnescape(k); err == nil {
					k = unescK
				}
			}
			if strings.IndexByte(val, '%') >= 0 || strings.IndexByte(val, '+') >= 0 {
				if unescVal, err := QueryUnescape(val); err == nil {
					val = unescVal
				}
			}
			if !yield(k, val) {
				return
			}
		}
	}
}

// AppendTo appends the reconstructed URL string to dst without heap allocations.
func (u URLView) AppendTo(dst []byte) []byte {
	if u.Scheme != "" {
		dst = append(dst, u.Scheme...)
		dst = append(dst, ':')
	}
	if u.Host != "" || u.User != "" || (u.Scheme != "" && (u.Scheme == "http" || u.Scheme == "https")) {
		dst = append(dst, '/', '/')
		if u.User != "" {
			dst = append(dst, u.User...)
			dst = append(dst, '@')
		}
		dst = append(dst, u.Host...)
	}
	if u.Path != "" {
		dst = append(dst, u.Path...)
	}
	if u.RawQuery != "" {
		dst = append(dst, '?')
		dst = append(dst, u.RawQuery...)
	}
	if u.Fragment != "" {
		dst = append(dst, '#')
		dst = append(dst, u.Fragment...)
	}
	return dst
}

// String reconstructs the full URL string.
func (u URLView) String() string {
	estLen := len(u.Scheme) + len(u.User) + len(u.Host) + len(u.Path) + len(u.RawQuery) + len(u.Fragment) + 6
	buf := make([]byte, 0, estLen)
	return string(u.AppendTo(buf))
}

// ToURL converts the view to a standard [*url.URL].
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
				switch schemeCandidate {
				case "http", "https", "ws", "wss", "ftp", "file", "tcp", "udp":
					view.Scheme = schemeCandidate
				case "HTTP":
					view.Scheme = "http"
				case "HTTPS":
					view.Scheme = "https"
				case "WS":
					view.Scheme = "ws"
				case "WSS":
					view.Scheme = "wss"
				default:
					view.Scheme = strings.ToLower(schemeCandidate)
				}
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

	// 5. Host Validation
	for i := 0; i < len(view.Host); i++ {
		b := view.Host[i]
		if b <= ' ' || b == '<' || b == '>' || b == '"' || b == 92 || b == '^' || b == 96 || b == '{' || b == '|' ||
			b == '}' {
			return URLView{}, ErrInvalidHost
		}
	}

	return view, nil
}

// IsAbsURL reports whether the path begins with an absolute HTTP or HTTPS scheme identifier (RFC 3986 §3.1 & §4.3).
func IsAbsURL(path string) bool {
	return len(path) >= 7 && (strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://"))
}

// Parse parses the string or returns a cached [*url.URL].
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
		return CloneURL(u), nil
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

// CloneURL returns a deep copy of the [*url.URL].
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

// NormalizeBaseURL parses and normalizes a raw Base URI string into a [*url.URL].
//
// It ensures a trailing slash per RFC 3986 §5.2.3.
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

// Resolve resolves a relative path or absolute URL against the base [*url.URL] (RFC 3986 §5).
//
// It features a zero-allocation fast-path for root-domain paths and normalizes slash boundaries.
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

// ResolveString computes the resolved target URL string from the base [*url.URL] and path (RFC 3986 §5).
func ResolveString(baseURL *url.URL, path string) (string, error) {
	if IsAbsURL(path) || baseURL == nil || baseURL.Host == "" {
		return path, nil
	}
	if path == "" {
		return baseURL.String(), nil
	}

	rel, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	return baseURL.ResolveReference(rel).String(), nil
}

// ReplaceVar performs path variable replacement ({key} -> value) in the path.
func ReplaceVar(path, key, value string) string {
	if key == "" || len(path) < len(key)+2 {
		return path
	}

	idx := -1
	for i := 0; i <= len(path)-len(key)-2; i++ {
		if path[i] == '{' && path[i+1+len(key)] == '}' && path[i+1:i+1+len(key)] == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return path
	}

	before := path[:idx]
	after := path[idx+len(key)+2:]

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

// AppendReplaceVar appends the path with {key} replaced by value into dst without heap allocations.
func AppendReplaceVar(dst []byte, path, key, value string) []byte {
	if key == "" || len(path) < len(key)+2 {
		return append(dst, path...)
	}

	idx := -1
	for i := 0; i <= len(path)-len(key)-2; i++ {
		if path[i] == '{' && path[i+1+len(key)] == '}' && path[i+1:i+1+len(key)] == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return append(dst, path...)
	}

	dst = append(dst, path[:idx]...)
	dst = append(dst, value...)
	dst = append(dst, path[idx+len(key)+2:]...)
	return dst
}

// FastAppendQuery appends key=value to the target URL using SIMD byte detection and pooled buffers.
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

// AppendRawQuery appends the raw query string to the target URL using SIMD byte detection and pooled buffers.
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

// MatchDomainPattern checks if the host matches the pattern (supporting exact and *.wildcard matches).
func MatchDomainPattern(host, pattern string) bool {
	host = strings.TrimSuffix(host, ".")
	pattern = strings.TrimSuffix(pattern, ".")

	if !strings.HasPrefix(pattern, "*.") {
		return bytesconv.EqualFoldASCII(host, pattern)
	}

	domain := pattern[2:] // "example.com"
	if bytesconv.EqualFoldASCII(host, domain) {
		return true
	}

	if len(host) > len(domain) && host[len(host)-len(domain)-1] == '.' {
		return bytesconv.EqualFoldASCII(host[len(host)-len(domain):], domain)
	}

	return false
}

// IsCrossOrigin determines whether the given [*url.URL] instances belong to different RFC 6454 web origins.
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

// CanonicalPort resolves the effective port number considering scheme defaults.
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

// IsSameDomainOrSubdomain reports whether the clean hosts match domain or subdomain suffix.
func IsSameDomainOrSubdomain(clean1, clean2 string) bool {
	clean1 = strings.TrimSuffix(clean1, ".")
	clean2 = strings.TrimSuffix(clean2, ".")

	if bytesconv.EqualFoldASCII(clean1, clean2) {
		return true
	}

	if len(clean1) > len(clean2) && clean1[len(clean1)-len(clean2)-1] == '.' {
		return bytesconv.EqualFoldASCII(clean1[len(clean1)-len(clean2):], clean2)
	}

	if len(clean2) > len(clean1) && clean2[len(clean2)-len(clean1)-1] == '.' {
		return bytesconv.EqualFoldASCII(clean2[len(clean2)-len(clean1):], clean1)
	}

	return false
}

// BuildPath constructs a final URL path by interpolating pathParams and appending query parameters from [url.Values].
func BuildPath(basePath string, pathParams map[string]string, queryParams url.Values) string {
	if len(pathParams) == 0 && len(queryParams) == 0 {
		return basePath
	}

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

var noEscapeTable = [256]bool{
	'-': true, '.': true, '_': true, '~': true,
	'0': true, '1': true, '2': true, '3': true, '4': true, '5': true, '6': true, '7': true, '8': true, '9': true,
	'A': true, 'B': true, 'C': true, 'D': true, 'E': true, 'F': true, 'G': true, 'H': true, 'I': true, 'J': true,
	'K': true, 'L': true, 'M': true, 'N': true, 'O': true, 'P': true, 'Q': true, 'R': true, 'S': true, 'T': true,
	'U': true, 'V': true, 'W': true, 'X': true, 'Y': true, 'Z': true,
	'a': true, 'b': true, 'c': true, 'd': true, 'e': true, 'f': true, 'g': true, 'h': true, 'i': true, 'j': true,
	'k': true, 'l': true, 'm': true, 'n': true, 'o': true, 'p': true, 'q': true, 'r': true, 's': true, 't': true,
	'u': true, 'v': true, 'w': true, 'x': true, 'y': true, 'z': true,
}

//go:inline
//go:nosplit
func shouldEscape(c byte) bool {
	return !noEscapeTable[c]
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
//
//go:inline
func AppendQueryEscapeString(dst []byte, src string) []byte {
	return AppendQueryEscape(dst, bytesconv.S2B(src))
}

// AppendPathEscape appends percent-encoded bytes of src to dst according to RFC 3986 §3.3.
//
//go:inline
func AppendPathEscape(dst, src []byte) []byte {
	return AppendQueryEscape(dst, src)
}

// AppendPathEscapeString appends percent-encoded bytes of string src to dst.
//
//go:inline
func AppendPathEscapeString(dst []byte, src string) []byte {
	return AppendQueryEscape(dst, bytesconv.S2B(src))
}

// QueryUnescape unescapes URL percent-encoded characters (%XX) and replaces '+' with ' '.
//
// It uses hardware vector acceleration.
func QueryUnescape(s string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}
	// Zero-allocation fast-path: if no '%' and no '+', return s directly!
	if strings.IndexByte(s, '%') < 0 && strings.IndexByte(s, '+') < 0 {
		return s, nil
	}
	src := bytesconv.S2B(s)
	buf := make([]byte, len(src))
	n, err := unescapeVector(buf, src, UnescapeModeQuery)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// QueryUnescapeBytes unescapes URL percent-encoded characters from src into dst.
func QueryUnescapeBytes(dst, src []byte) ([]byte, error) {
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
	n, err := unescapeVector(dst[start:], src, UnescapeModeQuery)
	if err != nil {
		return nil, err
	}
	return dst[:start+n], nil
}

func unescapeScalar(dst, src []byte, mode UnescapeMode) (int, error) {
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
			if mode == UnescapeModeQuery {
				dst[out] = ' '
			} else {
				dst[out] = '+'
			}
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

// PathUnescape unescapes URL percent-encoded characters (%XX) without touching '+'.
//
// It complies with RFC 3986 Section 2.2 for URI components.
func PathUnescape(s string) (string, error) {
	if len(s) == 0 {
		return "", nil
	}
	// Zero-allocation fast-path: if no '%', return s directly!
	if strings.IndexByte(s, '%') < 0 {
		return s, nil
	}
	src := bytesconv.S2B(s)
	buf := make([]byte, len(src))
	n, err := unescapeScalar(buf, src, UnescapeModePath)
	if err != nil {
		return "", err
	}
	return string(buf[:n]), nil
}

// PathUnescapeBytes unescapes URL percent-encoded characters (%XX) from src into dst without touching '+'.
func PathUnescapeBytes(dst, src []byte) ([]byte, error) {
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
	n, err := unescapeScalar(dst[start:], src, UnescapeModePath)
	if err != nil {
		return nil, err
	}
	return dst[:start+n], nil
}
