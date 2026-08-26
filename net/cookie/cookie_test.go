// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cookie_test

import (
	"net/http"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"

	"github.com/lemon4ksan/foundation/net/cookie"
)

func TestCookie_ParseSetCookieHeader(t *testing.T) {
	header := "session_id=xyz123; Domain=example.com; Path=/api; Secure; HttpOnly; SameSite=Lax; Max-Age=3600"
	c := cookie.ParseSetCookieHeader(header, "default.com", "/")

	assert.Equal(t, "session_id", c.Name)
	assert.Equal(t, "xyz123", c.Value)
	assert.Equal(t, "example.com", c.Domain)
	assert.Equal(t, "/api", c.Path)
	assert.True(t, c.Secure)
	assert.True(t, c.HTTPOnly)
	assert.Equal(t, "Lax", c.SameSite)
	assert.Equal(t, 3600, c.MaxAge)
}

func TestCookie_PathMatch(t *testing.T) {
	assert.True(t, cookie.PathMatch("/api/v1", "/api"))
	assert.False(t, cookie.PathMatch("/apiv1", "/api"))
	assert.True(t, cookie.PathMatch("/api", "/api"))
}

func TestCookie_BuildCookieHeader(t *testing.T) {
	cookies := []*http.Cookie{
		{Name: "c1", Value: "v1", Path: "/"},
		{Name: "c2", Value: "v2", Path: "/api"},
	}

	hdr := cookie.BuildCookieHeader(cookies)
	assert.Equal(t, "c2=v2; c1=v1", hdr)
}

func TestCookie_ExportNetscape(t *testing.T) {
	cookies := []*http.Cookie{
		{Name: "session", Value: "123", Domain: ".example.com", Path: "/api", Secure: true},
	}

	netscape := cookie.ExportNetscape(cookies, "example.com")
	assert.Contains(t, netscape, "# Netscape HTTP Cookie File")
	assert.Contains(t, netscape, ".example.com\tTRUE\t/api\tTRUE\t0\tsession\t123")
	assert.Equal(t, "", cookie.ExportNetscape(nil, ""))
}

func TestCookie_ParseSingleCookie(t *testing.T) {
	ck := cookie.ParseSingleCookie([]byte("foo"), []byte("foo=bar; Path=/"))
	require.NotNil(t, ck)
	assert.Equal(t, "foo", ck.Name)
	assert.Equal(t, "bar", ck.Value)

	emptyCk := cookie.ParseSingleCookie(nil, []byte(""))
	assert.Nil(t, emptyCk)
}

func TestCookie_EmptyParse(t *testing.T) {
	c := cookie.ParseSetCookieHeader("", "domain.com", "/")
	assert.Equal(t, "", c.Name)
}

func TestCookie_RFC6265bis_Limits(t *testing.T) {
	badHeader := "name=val\x00ue; Domain=example.com"
	c := cookie.ParseSetCookieHeader(badHeader, "example.com", "/")
	assert.Equal(t, "", c.Name)

	hdrMaxAge := "name=value; Max-Age=50000000; SameSite=strict"
	c = cookie.ParseSetCookieHeader(hdrMaxAge, "example.com", "/")
	assert.Equal(t, "name", c.Name)
	assert.Equal(t, cookie.MaxCookieAgeSeconds, c.MaxAge)
	assert.Equal(t, "Strict", c.SameSite)

	cNone := cookie.ParseSetCookieHeader("n=v; SameSite=none", "example.com", "/")
	assert.Equal(t, "None", cNone.SameSite)

	cInvalid := cookie.ParseSetCookieHeader("n=v; SameSite=invalid_value", "example.com", "/")
	assert.Equal(t, "Default", cInvalid.SameSite)
}

func TestCookie_RFC6265bis_Prefixes(t *testing.T) {
	assert.True(t, cookie.ValidatePrefix(cookie.Cookie{
		Name:   "__Secure-SID",
		Value:  "12345",
		Secure: true,
	}))
	assert.False(t, cookie.ValidatePrefix(cookie.Cookie{
		Name:   "__Secure-SID",
		Value:  "12345",
		Secure: false,
	}))

	assert.True(t, cookie.ValidatePrefix(cookie.Cookie{
		Name:   "__Host-SID",
		Value:  "12345",
		Secure: true,
		Path:   "/",
		Domain: "",
	}))
	assert.False(t, cookie.ValidatePrefix(cookie.Cookie{
		Name:   "__Host-SID",
		Value:  "12345",
		Secure: false,
		Path:   "/",
		Domain: "",
	}))

	assert.False(t, cookie.ValidatePrefix(cookie.Cookie{
		Name:  "",
		Value: "__Secure-bad",
	}))
	assert.False(t, cookie.ValidatePrefix(cookie.Cookie{
		Name:  "",
		Value: "__Host-bad",
	}))
}
