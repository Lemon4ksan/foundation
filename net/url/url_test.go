// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package url_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"

	"github.com/lemon4ksan/foundation/net/url"
)

func TestParse(t *testing.T) {
	u1, err := url.Parse("https://api.example.com/v1/resource?query=1")
	require.NoError(t, err)
	assert.Equal(t, "api.example.com", u1.Host)

	u2, err := url.Parse("https://api.example.com/v1/resource?query=1")
	require.NoError(t, err)
	assert.Equal(t, u1, u2)
}

func TestReplaceVar(t *testing.T) {
	res := url.ReplaceVar("/users/{id}/profile", "id", "42")
	assert.Equal(t, "/users/42/profile", res)

	noMatch := url.ReplaceVar("/users/profile", "id", "42")
	assert.Equal(t, "/users/profile", noMatch)
}

func TestFastAppendQuery(t *testing.T) {
	res1 := url.FastAppendQuery("https://example.com/api", "page", "1")
	assert.Equal(t, "https://example.com/api?page=1", res1)

	res2 := url.FastAppendQuery("https://example.com/api?limit=10", "page", "2")
	assert.Equal(t, "https://example.com/api?limit=10&page=2", res2)
}

func TestCloneURL(t *testing.T) {
	assert.Nil(t, url.CloneURL(nil))

	u1, _ := url.Parse("https://user:pass@example.com:8443/path")
	cloned := url.CloneURL(u1)
	require.NotNil(t, cloned)
	assert.Equal(t, u1.String(), cloned.String())
	assert.NotSame(t, u1, cloned)
	assert.NotSame(t, u1.User, cloned.User)
}

func TestMatchDomainPattern(t *testing.T) {
	assert.True(t, url.MatchDomainPattern("api.example.com", "*.example.com"))
	assert.True(t, url.MatchDomainPattern("example.com", "*.example.com"))
	assert.True(t, url.MatchDomainPattern("example.com", "example.com"))
	assert.False(t, url.MatchDomainPattern("other.com", "*.example.com"))
}

func TestIsCrossOrigin(t *testing.T) {
	u1, _ := url.Parse("https://example.com/a")
	u2, _ := url.Parse("https://example.com/b")
	uDiffScheme, _ := url.Parse("http://example.com/a")
	uDiffDomain, _ := url.Parse("https://other.com/a")
	uDiffPort, _ := url.Parse("https://example.com:8443/a")

	assert.False(t, url.IsCrossOrigin(u1, u2))
	assert.True(t, url.IsCrossOrigin(u1, uDiffScheme))
	assert.True(t, url.IsCrossOrigin(u1, uDiffDomain))
	assert.True(t, url.IsCrossOrigin(u1, uDiffPort))
	assert.False(t, url.IsCrossOrigin(nil, u1))
}

func TestCanonicalPort(t *testing.T) {
	uHTTP, _ := url.Parse("http://example.com")
	uHTTPS, _ := url.Parse("https://example.com")
	uCustom, _ := url.Parse("https://example.com:9000")

	assert.Equal(t, "80", url.CanonicalPort(uHTTP))
	assert.Equal(t, "443", url.CanonicalPort(uHTTPS))
	assert.Equal(t, "9000", url.CanonicalPort(uCustom))
	assert.Equal(t, "", url.CanonicalPort(nil))
}

func TestIsSameDomainOrSubdomain(t *testing.T) {
	assert.True(t, url.IsSameDomainOrSubdomain("api.example.com", "example.com"))
	assert.True(t, url.IsSameDomainOrSubdomain("example.com", "api.example.com"))
	assert.True(t, url.IsSameDomainOrSubdomain("example.com", "example.com"))
	assert.False(t, url.IsSameDomainOrSubdomain("other.com", "example.com"))
}

func TestResolve(t *testing.T) {
	t.Parallel()

	base1, err := url.NormalizeBaseURL("https://api.example.com/v1")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/", base1.String())

	// 1. Relative path without slash -> /v1/users
	u, err := url.Resolve(base1, "users")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/users", u.String())

	// 2. Relative path with leading slash -> /v1/users (safe normalization!)
	u, err = url.Resolve(base1, "/users")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/v1/users", u.String())

	// 3. Absolute URL -> bypasses base
	u, err = url.Resolve(base1, "https://other.com/auth")
	require.NoError(t, err)
	assert.Equal(t, "https://other.com/auth", u.String())

	// 4. Root BaseURL fast-path
	baseRoot, _ := url.Parse("https://api.example.com")
	u, err = url.Resolve(baseRoot, "/users/42")
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com/users/42", u.String())
}

func TestUnescape(t *testing.T) {
	unescaped, err := url.Unescape("Hello%20World%21+From+Silicon%2FEngine")
	require.NoError(t, err)
	assert.Equal(t, "Hello World! From Silicon/Engine", unescaped)

	// Clean string fast-path
	clean, err := url.Unescape("clean_alphanumeric_string_without_escapes_12345")
	require.NoError(t, err)
	assert.Equal(t, "clean_alphanumeric_string_without_escapes_12345", clean)

	// Invalid escape
	_, err = url.Unescape("invalid%2")
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
		_, _ = url.Unescape(s)
	}
}
