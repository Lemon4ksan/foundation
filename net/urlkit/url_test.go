// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit_test

import (
	"maps"
	stdurl "net/url"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/net/urlkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// ============================================================================
// 1. URLView & ParseView Tests
// ============================================================================

func TestParseView(t *testing.T) {
	t.Parallel()

	// 1. Comprehensive URL view breakdown
	raw := "https://user:secret@api.gateway.internal:8443/v2/telemetry/events?session_id=9876543210&region=eu-central-1#heading"
	v, err := urlkit.ParseView(raw)
	require.NoError(t, err)

	assert.Equal(t, "https", v.Scheme)
	assert.Equal(t, "user:secret", v.User)
	assert.Equal(t, "api.gateway.internal:8443", v.Host)
	assert.Equal(t, "api.gateway.internal", v.Hostname())
	assert.Equal(t, "8443", v.Port())
	assert.Equal(t, "8443", v.CanonicalPort())
	assert.Equal(t, "/v2/telemetry/events", v.Path)
	assert.Equal(t, "session_id=9876543210&region=eu-central-1", v.RawQuery)
	assert.Equal(t, "heading", v.Fragment)
	assert.True(t, v.IsAbs())

	// QueryValue
	sess, ok := v.QueryValue("session_id")
	assert.True(t, ok)
	assert.Equal(t, "9876543210", sess)

	reg, ok := v.QueryValue("region")
	assert.True(t, ok)
	assert.Equal(t, "eu-central-1", reg)

	_, ok = v.QueryValue("missing")
	assert.False(t, ok)

	// QueryValue edge cases
	vExtra, _ := urlkit.ParseView("http://example.com/?flag&bad=%2&&clean=yes")
	fVal, fOk := vExtra.QueryValue("flag")
	assert.True(t, fOk)
	assert.Equal(t, "", fVal)

	bVal, bOk := vExtra.QueryValue("bad")
	assert.True(t, bOk)
	assert.Equal(t, "%2", bVal)

	cVal, cOk := vExtra.QueryValue("clean")
	assert.True(t, cOk)
	assert.Equal(t, "yes", cVal)

	// CanonicalPort on URLView
	vHTTP, _ := urlkit.ParseView("http://example.com/a")
	assert.Equal(t, "80", vHTTP.CanonicalPort())

	vHTTPS, _ := urlkit.ParseView("https://example.com/a")
	assert.Equal(t, "443", vHTTPS.CanonicalPort())

	vCustom, _ := urlkit.ParseView("ftp://example.com/a")
	assert.Equal(t, "", vCustom.CanonicalPort())

	// ToURL
	u := v.ToURL()
	assert.Equal(t, "https", u.Scheme)
	assert.Equal(t, "api.gateway.internal:8443", u.Host)
	assert.Equal(t, "/v2/telemetry/events", u.Path)

	// String & AppendTo
	assert.Equal(t, raw, v.String())
	var dstBuf [256]byte
	appended := v.AppendTo(dstBuf[:0])
	assert.Equal(t, raw, string(appended))

	// 2. Empty raw URL
	emptyView, err := urlkit.ParseView("")
	require.NoError(t, err)
	assert.Equal(t, "", emptyView.String())
	assert.False(t, emptyView.IsAbs())

	// 3. User without password
	vUser, err := urlkit.ParseView("http://admin@localhost/")
	require.NoError(t, err)
	assert.Equal(t, "admin", vUser.User)
	uUser := vUser.ToURL()
	require.NotNil(t, uUser.User)
	assert.Equal(t, "admin", uUser.User.Username())
}

func TestURLView_Hostname_IPv6Zone(t *testing.T) {
	t.Parallel()

	v, err := urlkit.ParseView("http://[fe80::1%25eth0]:8080/foo")
	require.NoError(t, err)
	assert.Equal(t, "fe80::1%eth0", v.Hostname())
	assert.Equal(t, "8080", v.Port())

	v2, err := urlkit.ParseView("http://[fe80::2]/foo")
	require.NoError(t, err)
	assert.Equal(t, "fe80::2", v2.Hostname())
	assert.Equal(t, "", v2.Port())
}

func TestURLView_ParseView_InvalidHost(t *testing.T) {
	t.Parallel()

	_, err := urlkit.ParseView("http://invalid host.com/foo")
	assert.ErrorIs(t, err, urlkit.ErrInvalidHost)

	_, err2 := urlkit.ParseView("http://foo.com^/foo")
	assert.ErrorIs(t, err2, urlkit.ErrInvalidHost)

	_, err3 := urlkit.ParseView("http://foo{bar}.com/path")
	assert.ErrorIs(t, err3, urlkit.ErrInvalidHost)
}

func TestURLView_Schemes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input       string
		expectedSch string
	}{
		{"HTTP://example.com", "http"},
		{"HTTPS://example.com", "https"},
		{"WS://example.com/socket", "ws"},
		{"WSS://example.com/socket", "wss"},
		{"ftp://files.example.com", "ftp"},
		{"FILE:///local/path", "file"},
		{"CUSTOM-SCHEME://host", "custom-scheme"},
		{"/relative/path", ""},
		{"://bad-scheme", ""},
	}

	for _, tt := range tests {
		v, err := urlkit.ParseView(tt.input)
		if tt.expectedSch != "" {
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSch, v.Scheme)
		}
	}
}

func TestURLView_QuerySeq(t *testing.T) {
	t.Parallel()

	v, err := urlkit.ParseView("https://example.com/search?q=hello+world&lang=en&flag&encoded=a%20b%21")
	require.NoError(t, err)

	pairs := maps.Collect(v.QuerySeq())

	assert.Equal(t, "hello world", pairs["q"])
	assert.Equal(t, "en", pairs["lang"])
	assert.Equal(t, "", pairs["flag"])
	assert.Equal(t, "a b!", pairs["encoded"])

	// Early break in iterator
	count := 0
	for range v.QuerySeq() {
		count++
		break
	}
	assert.Equal(t, 1, count)

	// Empty query view
	vEmpty, _ := urlkit.ParseView("https://example.com/no-query")
	emptyCount := 0
	for range vEmpty.QuerySeq() {
		emptyCount++
	}
	assert.Equal(t, 0, emptyCount)
}

// ============================================================================
// 2. Query & Path Escaping / Unescaping Tests
// ============================================================================

func TestUnescapeMode(t *testing.T) {
	t.Parallel()

	// Query unescape replaces '+' with space
	query, err := urlkit.QueryUnescape("foo+bar%20baz")
	require.NoError(t, err)
	assert.Equal(t, "foo bar baz", query)

	// Path unescape preserves '+'
	path, err := urlkit.PathUnescape("foo+bar%20baz")
	require.NoError(t, err)
	assert.Equal(t, "foo+bar baz", path)
}

func TestUnescape_FastPath_And_Hex(t *testing.T) {
	t.Parallel()

	// 1. Fast-path: clean string without '%' or '+'
	clean := "abcdefghijklmnopqrstuvwxyz_0123456789-."
	gotClean, err := urlkit.QueryUnescape(clean)
	require.NoError(t, err)
	assert.Equal(t, clean, gotClean)

	gotCleanPath, err := urlkit.PathUnescape(clean)
	require.NoError(t, err)
	assert.Equal(t, clean, gotCleanPath)

	// 2. Complex unescape with varied hex digits (0-9, a-f, A-F)
	src := "hello%20world%21+hex%00%99%af%AF"
	unescaped, err := urlkit.QueryUnescape(src)
	require.NoError(t, err)
	assert.Equal(t, "hello world! hex\x00\x99\xaf\xaf", unescaped)

	// 3. Truncated '%'
	_, err = urlkit.QueryUnescape("foo%2")
	assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)

	_, err = urlkit.PathUnescape("foo%2")
	assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)

	// 4. Invalid hex characters
	_, err = urlkit.QueryUnescape("foo%2Z")
	assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)

	_, err = urlkit.PathUnescape("foo%2Z")
	assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)

	// 5. Empty string
	empty, err := urlkit.QueryUnescape("")
	require.NoError(t, err)
	assert.Equal(t, "", empty)

	emptyPath, err := urlkit.PathUnescape("")
	require.NoError(t, err)
	assert.Equal(t, "", emptyPath)
}

func TestQueryEscape_And_UnescapeBytes(t *testing.T) {
	t.Parallel()

	// 1. AppendQueryEscape & AppendQueryEscapeString
	raw := "Hello World! &foo=bar/baz~_.-"
	escapedBytes := urlkit.AppendQueryEscape(nil, []byte(raw))
	escapedStr := urlkit.AppendQueryEscapeString(nil, raw)
	assert.Equal(t, escapedBytes, escapedStr)

	// 2. QueryUnescapeBytes
	unescapedBytes, err := urlkit.QueryUnescapeBytes(nil, escapedBytes)
	require.NoError(t, err)
	assert.Equal(t, []byte(raw), unescapedBytes)

	// Destination with existing capacity
	dstCap := make([]byte, 0, 128)
	unescapedCap, err := urlkit.QueryUnescapeBytes(dstCap, escapedBytes)
	require.NoError(t, err)
	assert.Equal(t, []byte(raw), unescapedCap)

	// Empty src
	emptyBytes, err := urlkit.QueryUnescapeBytes([]byte("prefix"), nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("prefix"), emptyBytes)

	// Invalid hex in QueryUnescapeBytes
	_, err = urlkit.QueryUnescapeBytes(nil, []byte("bad%ZZ"))
	assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)
}

func TestPathEscape_And_UnescapeBytes(t *testing.T) {
	t.Parallel()

	raw := "user/profile with spaces+symbols"
	escapedBytes := urlkit.AppendPathEscape(nil, []byte(raw))
	escapedStr := urlkit.AppendPathEscapeString(nil, raw)
	assert.Equal(t, escapedBytes, escapedStr)

	// PathUnescapeBytes preserves '+'
	unescapedBytes, err := urlkit.PathUnescapeBytes(nil, escapedBytes)
	require.NoError(t, err)
	assert.Equal(t, []byte(raw), unescapedBytes)

	// Empty src
	empty, err := urlkit.PathUnescapeBytes([]byte("pre"), nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("pre"), empty)

	// Invalid hex
	_, err = urlkit.PathUnescapeBytes(nil, []byte("bad%ZZ"))
	assert.ErrorIs(t, err, urlkit.ErrInvalidEscape)
}

// ============================================================================
// 3. URL Resolution, Routing & Path Operations
// ============================================================================

func TestParse(t *testing.T) {
	t.Parallel()

	u1, err := urlkit.Parse("https://api.example.com/v1/resource?query=1")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", u1.Host)

	u2, err := urlkit.Parse("https://api.example.com/v1/resource?query=1")
	require.NoError(t, err)
	assert.Equal(t, u1.String(), u2.String())

	// Empty string
	emptyURL, err := urlkit.Parse("")
	require.NoError(t, err)
	assert.Equal(t, "", emptyURL.String())

	// Invalid URL
	_, err = urlkit.Parse("http://[invalid-ipv6")
	assert.Error(t, err)
}

func TestReplaceVar_And_Append(t *testing.T) {
	t.Parallel()

	// 1. ReplaceVar
	res := urlkit.ReplaceVar("/users/{id}/profile", "id", "42")
	assert.Equal(t, "/users/42/profile", res)

	noMatch := urlkit.ReplaceVar("/users/profile", "id", "42")
	assert.Equal(t, "/users/profile", noMatch)

	emptyKey := urlkit.ReplaceVar("/users/{id}", "", "42")
	assert.Equal(t, "/users/{id}", emptyKey)

	shortPath := urlkit.ReplaceVar("a", "longkey", "val")
	assert.Equal(t, "a", shortPath)

	// 2. AppendReplaceVar
	appended := urlkit.AppendReplaceVar(nil, "/items/{item_id}", "item_id", "book-99")
	assert.Equal(t, "/items/book-99", string(appended))

	appendedNoMatch := urlkit.AppendReplaceVar([]byte("base:"), "/items/all", "key", "val")
	assert.Equal(t, "base:/items/all", string(appendedNoMatch))

	assert.Equal(t, []byte("short"), urlkit.AppendReplaceVar(nil, "short", "", "val"))
	assert.Equal(t, []byte("a"), urlkit.AppendReplaceVar(nil, "a", "too_long_key", "val"))
}

func TestFastAppendQuery(t *testing.T) {
	t.Parallel()

	res1 := urlkit.FastAppendQuery("https://example.com/api", "page", "1")
	assert.Equal(t, "https://example.com/api?page=1", res1)

	res2 := urlkit.FastAppendQuery("https://example.com/api?limit=10", "page", "2")
	assert.Equal(t, "https://example.com/api?limit=10&page=2", res2)

	resEmpty := urlkit.FastAppendQuery("https://example.com/api", "", "1")
	assert.Equal(t, "https://example.com/api", resEmpty)
}

func TestAppendRawQuery_And_BuildPath(t *testing.T) {
	t.Parallel()

	// 1. AppendRawQuery
	assert.Equal(t, "https://example.com", urlkit.AppendRawQuery("https://example.com", ""))
	assert.Equal(t, "https://example.com?a=1", urlkit.AppendRawQuery("https://example.com", "a=1"))
	assert.Equal(t, "https://example.com?a=1&b=2", urlkit.AppendRawQuery("https://example.com?a=1", "b=2"))

	// 2. BuildPath
	params := map[string]string{"user_id": "123", "action": "edit"}
	q := make(map[string][]string)
	q["tab"] = []string{"settings"}

	path1 := urlkit.BuildPath("/users/{user_id}/{action}", params, q)
	assert.Equal(t, "/users/123/edit?tab=settings", path1)

	path2 := urlkit.BuildPath("/search?type=all", nil, q)
	assert.Equal(t, "/search?type=all&tab=settings", path2)

	// Empty params and queries
	pathEmpty := urlkit.BuildPath("/plain/path", nil, nil)
	assert.Equal(t, "/plain/path", pathEmpty)
}

func TestCloneURL(t *testing.T) {
	t.Parallel()

	assert.Nil(t, urlkit.CloneURL(nil))

	u1, _ := urlkit.Parse("https://user:pass@example.com:8443/path")
	cloned := urlkit.CloneURL(u1)
	require.NotNil(t, cloned)
	assert.Equal(t, u1.String(), cloned.String())
	assert.NotSame(t, u1, cloned)
	assert.NotSame(t, u1.User, cloned.User)
}

func TestResolve(t *testing.T) {
	t.Parallel()

	base1, err := urlkit.NormalizeBaseURL("https://api.example.com/v1")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/", base1.String())

	// 1. Relative path without slash -> /v1/users
	u, err := urlkit.Resolve(base1, "users")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/users", u.String())

	// 2. Relative path with leading slash -> /users (RFC 3986 compliant absolute path)
	u, err = urlkit.Resolve(base1, "/users")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users", u.String())

	// 3. Absolute URL -> bypasses base
	u, err = urlkit.Resolve(base1, "https://other.com/auth")
	require.NoError(t, err)
	assert.Equal(t, "https://other.com/auth", u.String())

	// 4. Root BaseURL fast-path
	baseRoot, _ := urlkit.Parse("https://api.example.com")
	u, err = urlkit.Resolve(baseRoot, "/users/42")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users/42", u.String())

	// 5. Empty path and slash against base
	uEmpty, err := urlkit.Resolve(base1, "")
	require.NoError(t, err)
	assert.Equal(t, base1.String(), uEmpty.String())

	// 6. NormalizeBaseURL with empty string
	normEmpty, err := urlkit.NormalizeBaseURL("")
	require.NoError(t, err)
	assert.Equal(t, "", normEmpty.String())

	// 7. ResolveString
	resStr, err := urlkit.ResolveString(base1, "")
	require.NoError(t, err)
	assert.Equal(t, base1.String(), resStr)

	resAbs, err := urlkit.ResolveString(nil, "https://api.com/users")
	require.NoError(t, err)
	assert.Equal(t, "https://api.com/users", resAbs)

	_, err = urlkit.ResolveString(base1, ":bad-url")
	assert.Error(t, err)
}

// ============================================================================
// 4. Domain & Origin Matching Tests
// ============================================================================

func TestMatchDomainPattern(t *testing.T) {
	t.Parallel()

	assert.True(t, urlkit.MatchDomainPattern("api.example.com", "*.example.com"))
	assert.True(t, urlkit.MatchDomainPattern("example.com", "*.example.com"))
	assert.True(t, urlkit.MatchDomainPattern("example.com", "example.com"))
	assert.False(t, urlkit.MatchDomainPattern("other.com", "*.example.com"))
	assert.False(t, urlkit.MatchDomainPattern("badexample.com", "*.example.com"))

	// Case insensitivity
	assert.True(t, urlkit.MatchDomainPattern("API.EXAMPLE.COM", "*.example.com"))
	assert.True(t, urlkit.MatchDomainPattern("api.example.com", "*.EXAMPLE.COM"))
}

func TestIsCrossOrigin(t *testing.T) {
	t.Parallel()

	u1, _ := urlkit.Parse("https://example.com/a")
	u2, _ := urlkit.Parse("https://example.com/b")
	uDiffScheme, _ := urlkit.Parse("http://example.com/a")
	uDiffDomain, _ := urlkit.Parse("https://other.com/a")
	uDiffPort, _ := urlkit.Parse("https://example.com:8443/a")

	assert.False(t, urlkit.IsCrossOrigin(u1, u2))
	assert.True(t, urlkit.IsCrossOrigin(u1, uDiffScheme))
	assert.True(t, urlkit.IsCrossOrigin(u1, uDiffDomain))
	assert.True(t, urlkit.IsCrossOrigin(u1, uDiffPort))
	assert.False(t, urlkit.IsCrossOrigin(nil, u1))
	assert.False(t, urlkit.IsCrossOrigin(u1, nil))
}

func TestCanonicalPort(t *testing.T) {
	t.Parallel()

	uHTTP, _ := urlkit.Parse("http://example.com")
	uHTTPS, _ := urlkit.Parse("https://example.com")
	uCustom, _ := urlkit.Parse("https://example.com:9000")

	assert.Equal(t, "80", urlkit.CanonicalPort(uHTTP))
	assert.Equal(t, "443", urlkit.CanonicalPort(uHTTPS))
	assert.Equal(t, "9000", urlkit.CanonicalPort(uCustom))
	assert.Equal(t, "", urlkit.CanonicalPort(nil))
}

func TestIsSameDomainOrSubdomain(t *testing.T) {
	t.Parallel()

	assert.True(t, urlkit.IsSameDomainOrSubdomain("api.example.com", "example.com"))
	assert.True(t, urlkit.IsSameDomainOrSubdomain("example.com", "api.example.com"))
	assert.True(t, urlkit.IsSameDomainOrSubdomain("example.com", "example.com"))
	assert.False(t, urlkit.IsSameDomainOrSubdomain("other.com", "example.com"))
	assert.False(t, urlkit.IsSameDomainOrSubdomain("notexample.com", "example.com"))

	// Case insensitivity
	assert.True(t, urlkit.IsSameDomainOrSubdomain("API.EXAMPLE.COM", "example.com"))
	assert.True(t, urlkit.IsSameDomainOrSubdomain("example.com", "API.EXAMPLE.COM"))
}

// ============================================================================
// 5. Concurrency & Cache Isolation Tests
// ============================================================================

func TestURLCache_Concurrent_And_CloneIsolation(t *testing.T) {
	t.Parallel()

	const workers = 32
	const iters = 200

	var wg sync.WaitGroup
	wg.Add(workers)

	for w := range workers {
		go func(workerID int) {
			defer wg.Done()
			for range iters {
				raw := "https://service-user:pass@api.internal:8080/v1/query?q=item"
				u, err := urlkit.Parse(raw)
				if err != nil {
					t.Errorf("worker %d failed to parse: %v", workerID, err)
					return
				}
				// Verify mutation isolation
				if u.User != nil {
					u.Path = "/mutated"
				}
			}
		}(w)
	}

	wg.Wait()

	// Verify original cache entry remains pristine
	uClean, err := urlkit.Parse("https://service-user:pass@api.internal:8080/v1/query?q=item")
	require.NoError(t, err)
	assert.Equal(t, "/v1/query", uClean.Path)
}

// ============================================================================
// 6. Zero Allocation Verification Tests
// ============================================================================

func TestZeroAllocations(t *testing.T) {
	// 1. QueryUnescape clean fast-path
	cleanStr := "clean_alphanumeric_string_without_escapes_12345"
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = urlkit.QueryUnescape(cleanStr)
	})
	assert.Equal(t, float64(0), allocs)

	// 2. PathUnescape clean fast-path
	allocs = testing.AllocsPerRun(100, func() {
		_, _ = urlkit.PathUnescape(cleanStr)
	})
	assert.Equal(t, float64(0), allocs)

	// 3. MatchDomainPattern
	allocs = testing.AllocsPerRun(100, func() {
		_ = urlkit.MatchDomainPattern("api.example.com", "*.example.com")
	})
	assert.Equal(t, float64(0), allocs)

	// 4. IsSameDomainOrSubdomain
	allocs = testing.AllocsPerRun(100, func() {
		_ = urlkit.IsSameDomainOrSubdomain("api.example.com", "example.com")
	})
	assert.Equal(t, float64(0), allocs)

	// 5. ReplaceVar no match
	allocs = testing.AllocsPerRun(100, func() {
		_ = urlkit.ReplaceVar("/users/profile", "id", "42")
	})
	assert.Equal(t, float64(0), allocs)

	// 6. ParseView
	rawURL := "https://api.example.com/v1/users?page=1#top"
	allocs = testing.AllocsPerRun(100, func() {
		_, _ = urlkit.ParseView(rawURL)
	})
	assert.Equal(t, float64(0), allocs)

	// 7. URLView.QuerySeq iteration over clean params
	v, _ := urlkit.ParseView("https://api.com/?a=1&b=2&c=3")
	allocs = testing.AllocsPerRun(100, func() {
		for range v.QuerySeq() {
		}
	})
	assert.Equal(t, float64(0), allocs)
}

// ============================================================================
// 7. Fuzz Tests
// ============================================================================

func FuzzURLParse(f *testing.F) {
	f.Add("https://example.com/api/v1/users?page=1&limit=50#section")
	f.Add("http://localhost:8080/test")
	f.Add("")
	f.Add("://bad-scheme")
	f.Add("https://user:pass@127.0.0.1:8443/secure?token=abc#frag")
	f.Add("https://[::1]:9000/ipv6")

	f.Fuzz(func(t *testing.T, raw string) {
		got, gotErr := urlkit.Parse(raw)
		expected, expErr := stdurl.Parse(raw)

		if (gotErr == nil) != (expErr == nil) {
			t.Fatalf("Parse error discrepancy on %q: got error %v, expected %v", raw, gotErr, expErr)
		}

		if gotErr == nil && expected != nil {
			if got.Scheme != expected.Scheme || got.Host != expected.Host || got.Path != expected.Path {
				t.Fatalf("Parse mismatch on %q: got %+v, expected %+v", raw, got, expected)
			}
		}
	})
}

func FuzzReplaceVar(f *testing.F) {
	f.Add("/users/{id}/profile", "id", "12345")
	f.Add("/items/{category}/{item_id}", "category", "electronics")
	f.Add("no_variable", "key", "val")
	f.Add("", "k", "v")

	f.Fuzz(func(t *testing.T, path, key, val string) {
		res := urlkit.ReplaceVar(path, key, val)
		if key != "" {
			target := "{" + key + "}"
			if len(path) > 0 && len(val) > 0 {
				_ = res
			}
			_ = target
		}
	})
}

func FuzzFastAppendQuery(f *testing.F) {
	f.Add("https://api.com/users", "page", "2")
	f.Add("https://api.com/users?sort=asc", "limit", "100")
	f.Add("", "k", "v")

	f.Fuzz(func(t *testing.T, targetURL, key, val string) {
		res := urlkit.FastAppendQuery(targetURL, key, val)
		if key == "" {
			if res != targetURL {
				t.Fatalf("expected unchanged URL for empty key")
			}
		} else if len(targetURL) > 0 {
			if len(res) <= len(targetURL) {
				t.Fatalf("expected expanded URL")
			}
		}
	})
}

func FuzzMatchDomainPattern(f *testing.F) {
	f.Add("api.example.com", "*.example.com")
	f.Add("example.com", "example.com")
	f.Add("sub.corp.local", "*.local")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, host, pattern string) {
		_ = urlkit.MatchDomainPattern(host, pattern)
	})
}

// ============================================================================
// 8. Benchmarks
// ============================================================================

func BenchmarkUnescape1KB(b *testing.B) {
	s := "query=hello%20world&tag=fast+engine&path=%2Fapi%2Fv1%2Fresource%3Ffilter%3Dactive"
	for len(s) < 1024 {
		s += "&" + s
	}
	s = s[:1024]

	b.SetBytes(1024)
	b.ReportAllocs()

	for b.Loop() {
		_, _ = urlkit.QueryUnescape(s)
	}
}

func BenchmarkQueryUnescape_FastPath(b *testing.B) {
	clean := "alphanumeric_clean_component_identifier_with_no_escapes"
	b.ReportAllocs()

	for b.Loop() {
		_, _ = urlkit.QueryUnescape(clean)
	}
}

func BenchmarkPathUnescape_FastPath(b *testing.B) {
	clean := "alphanumeric_clean_component_identifier_with_no_escapes"
	b.ReportAllocs()

	for b.Loop() {
		_, _ = urlkit.PathUnescape(clean)
	}
}

func BenchmarkParseView(b *testing.B) {
	raw := "https://user:secret@api.gateway.internal:8443/v2/telemetry/events?session_id=9876543210&region=eu-central-1#heading"
	b.ReportAllocs()

	for b.Loop() {
		v, err := urlkit.ParseView(raw)
		if err != nil || v.Host == "" {
			b.Fatal(err)
		}
	}
}

func BenchmarkURLView_AppendTo(b *testing.B) {
	raw := "https://user:secret@api.gateway.internal:8443/v2/telemetry/events?session_id=9876543210&region=eu-central-1#heading"
	v, _ := urlkit.ParseView(raw)
	buf := make([]byte, 0, 256)
	b.ReportAllocs()

	for b.Loop() {
		_ = v.AppendTo(buf[:0])
	}
}

func BenchmarkURLView_QuerySeq(b *testing.B) {
	raw := "https://api.example.com/v1/search?q=golang&sort=asc&limit=100&offset=200"
	v, _ := urlkit.ParseView(raw)
	b.ReportAllocs()

	for b.Loop() {
		for range v.QuerySeq() {
		}
	}
}

func BenchmarkReplaceVar(b *testing.B) {
	path := "/api/v1/organizations/{org_id}/teams/{team_id}/members"
	b.ReportAllocs()

	for b.Loop() {
		_ = urlkit.ReplaceVar(path, "team_id", "eng-core")
	}
}

func BenchmarkMatchDomainPattern(b *testing.B) {
	host := "api.internal.production.example.com"
	pattern := "*.example.com"
	b.ReportAllocs()

	for b.Loop() {
		_ = urlkit.MatchDomainPattern(host, pattern)
	}
}

func BenchmarkIsSameDomainOrSubdomain(b *testing.B) {
	h1 := "api.service.internal"
	h2 := "service.internal"
	b.ReportAllocs()

	for b.Loop() {
		_ = urlkit.IsSameDomainOrSubdomain(h1, h2)
	}
}

func TestReproduce_URLViewQueryValueKeyEncodingBug(t *testing.T) {
	u, err := urlkit.ParseView("http://example.com?my%20key=value")
	if err != nil {
		t.Fatal(err)
	}
	
	// QueryValue should find "my key", but it only checks raw equality,
	// missing the percent-encoded key.
	val, ok := u.QueryValue("my key")
	
	if !ok {
		t.Fatalf("Bug triggered! QueryValue failed to find a percent-encoded key. val=%q", val)
	}
}
