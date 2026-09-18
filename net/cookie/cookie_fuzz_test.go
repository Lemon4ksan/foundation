// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cookie_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/cookie"
)

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
