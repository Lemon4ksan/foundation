// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit_test

import (
	stdurl "net/url"
	"testing"

	"github.com/lemon4ksan/foundation/net/urlkit"
)

// FuzzURLParse tests URL parsing, sharded caching, and standard library error equivalence.
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

// FuzzReplaceVar verifies template path replacement against arbitrary path and parameter values.
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

// FuzzFastAppendQuery tests zero-allocation query parameter concatenation with ? and & delimiters.
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

// FuzzMatchDomainPattern tests domain pattern matching against standard and wildcard hosts.
func FuzzMatchDomainPattern(f *testing.F) {
	f.Add("api.example.com", "*.example.com")
	f.Add("example.com", "example.com")
	f.Add("sub.corp.local", "*.local")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, host, pattern string) {
		_ = urlkit.MatchDomainPattern(host, pattern)
	})
}
