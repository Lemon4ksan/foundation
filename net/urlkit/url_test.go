// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/urlkit"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestParse(t *testing.T) {
	u1, err := urlkit.Parse("https://api.example.com/v1/resource?query=1")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", u1.Host)

	u2, err := urlkit.Parse("https://api.example.com/v1/resource?query=1")
	require.NoError(t, err)
	assert.Equal(t, u1, u2)
}

func TestReplaceVar(t *testing.T) {
	res := urlkit.ReplaceVar("/users/{id}/profile", "id", "42")
	assert.Equal(t, "/users/42/profile", res)

	noMatch := urlkit.ReplaceVar("/users/profile", "id", "42")
	assert.Equal(t, "/users/profile", noMatch)
}

func TestFastAppendQuery(t *testing.T) {
	res1 := urlkit.FastAppendQuery("https://example.com/api", "page", "1")
	assert.Equal(t, "https://example.com/api?page=1", res1)

	res2 := urlkit.FastAppendQuery("https://example.com/api?limit=10", "page", "2")
	assert.Equal(t, "https://example.com/api?limit=10&page=2", res2)
}

func TestCloneURL(t *testing.T) {
	assert.Nil(t, urlkit.CloneURL(nil))

	u1, _ := urlkit.Parse("https://user:pass@example.com:8443/path")
	cloned := urlkit.CloneURL(u1)
	require.NotNil(t, cloned)
	assert.Equal(t, u1.String(), cloned.String())
	assert.NotSame(t, u1, cloned)
	assert.NotSame(t, u1.User, cloned.User)
}

func TestMatchDomainPattern(t *testing.T) {
	assert.True(t, urlkit.MatchDomainPattern("api.example.com", "*.example.com"))
	assert.True(t, urlkit.MatchDomainPattern("example.com", "*.example.com"))
	assert.True(t, urlkit.MatchDomainPattern("example.com", "example.com"))
	assert.False(t, urlkit.MatchDomainPattern("other.com", "*.example.com"))
}

func TestIsCrossOrigin(t *testing.T) {
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
}

func TestCanonicalPort(t *testing.T) {
	uHTTP, _ := urlkit.Parse("http://example.com")
	uHTTPS, _ := urlkit.Parse("https://example.com")
	uCustom, _ := urlkit.Parse("https://example.com:9000")

	assert.Equal(t, "80", urlkit.CanonicalPort(uHTTP))
	assert.Equal(t, "443", urlkit.CanonicalPort(uHTTPS))
	assert.Equal(t, "9000", urlkit.CanonicalPort(uCustom))
	assert.Equal(t, "", urlkit.CanonicalPort(nil))
}

func TestIsSameDomainOrSubdomain(t *testing.T) {
	assert.True(t, urlkit.IsSameDomainOrSubdomain("api.example.com", "example.com"))
	assert.True(t, urlkit.IsSameDomainOrSubdomain("example.com", "api.example.com"))
	assert.True(t, urlkit.IsSameDomainOrSubdomain("example.com", "example.com"))
	assert.False(t, urlkit.IsSameDomainOrSubdomain("other.com", "example.com"))
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

	// 2. Relative path with leading slash -> /v1/users (safe normalization!)
	u, err = urlkit.Resolve(base1, "/users")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/users", u.String())

	// 3. Absolute URL -> bypasses base
	u, err = urlkit.Resolve(base1, "https://other.com/auth")
	require.NoError(t, err)
	assert.Equal(t, "https://other.com/auth", u.String())

	// 4. Root BaseURL fast-path
	baseRoot, _ := urlkit.Parse("https://api.example.com")
	u, err = urlkit.Resolve(baseRoot, "/users/42")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users/42", u.String())
}

func TestUnescape(t *testing.T) {
	unescaped, err := urlkit.Unescape("Hello%20World%21+From+Silicon%2FEngine")
	require.NoError(t, err)
	assert.Equal(t, "Hello World! From Silicon/Engine", unescaped)

	// Clean string fast-path
	clean, err := urlkit.Unescape("clean_alphanumeric_string_without_escapes_12345")
	require.NoError(t, err)
	assert.Equal(t, "clean_alphanumeric_string_without_escapes_12345", clean)

	// Invalid escape
	_, err = urlkit.Unescape("invalid%2")
	assert.Error(t, err)

	// Empty string
	empty, err := urlkit.Unescape("")
	require.NoError(t, err)
	assert.Equal(t, "", empty)
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

	// BuildPath with existing query mark
	path2 := urlkit.BuildPath("/search?type=all", nil, q)
	assert.Equal(t, "/search?type=all&tab=settings", path2)
}

func TestQueryEscape_And_UnescapeBytes(t *testing.T) {
	t.Parallel()

	// 1. AppendQueryEscape & AppendQueryEscapeString
	raw := "Hello World! &foo=bar/baz~_.-"
	escapedBytes := urlkit.AppendQueryEscape(nil, []byte(raw))
	escapedStr := urlkit.AppendQueryEscapeString(nil, raw)
	assert.Equal(t, escapedBytes, escapedStr)

	// 2. UnescapeBytes
	unescapedBytes, err := urlkit.UnescapeBytes(nil, escapedBytes)
	require.NoError(t, err)
	assert.Equal(t, []byte(raw), unescapedBytes)

	// Empty src
	emptyBytes, err := urlkit.UnescapeBytes([]byte("prefix"), nil)
	require.NoError(t, err)
	assert.Equal(t, []byte("prefix"), emptyBytes)

	// Invalid hex in UnescapeBytes
	_, err = urlkit.UnescapeBytes(nil, []byte("bad%ZZ"))
	assert.Error(t, err)
}

func BenchmarkUnescape1KB(b *testing.B) {
	s := "query=hello%20world&tag=fast+engine&path=%2Fapi%2Fv1%2Fresource%3Ffilter%3Dactive"
	for len(s) < 1024 {
		s += "&" + s
	}
	s = s[:1024]

	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = urlkit.Unescape(s)
	}
}
