// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cookie_test

import (
	"net/http"
	"testing"

	"github.com/lemon4ksan/foundation/net/cookie"
)

func FuzzParseSetCookieHeader(f *testing.F) {
	f.Add("session=abc1234; Path=/; Domain=example.com; Secure; HttpOnly; SameSite=Strict", "example.com", "/")
	f.Add("token=xyz; Max-Age=3600; Partitioned; SameSite=Lax", "sub.example.com", "/api")
	f.Add("__Secure-ID=123; Secure; Path=/", "example.com", "/")
	f.Add("__Host-ID=123; Secure; Path=/", "", "/")
	f.Add("empty=", "example.com", "/")
	f.Add("", "example.com", "/")
	f.Add("bad\x00char=123", "example.com", "/")

	f.Fuzz(func(t *testing.T, headerVal, defaultDomain, defaultPath string) {
		c := cookie.ParseSetCookieHeader(headerVal, defaultDomain, defaultPath)
		_ = cookie.ValidatePrefix(c)

		if c.Name != "" {
			stdCookie := &http.Cookie{
				Name:  c.Name,
				Value: c.Value,
				Path:  c.Path,
			}
			_ = cookie.BuildCookieHeader([]*http.Cookie{stdCookie})
		}
	})
}

func FuzzPathMatch(f *testing.F) {
	f.Add("/api/v1/users", "/api")
	f.Add("/api/v1/users", "/api/")
	f.Add("/api", "/api")
	f.Add("/", "/")
	f.Add("/auth", "/api")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, reqPath, cookiePath string) {
		_ = cookie.PathMatch(reqPath, cookiePath)
	})
}
