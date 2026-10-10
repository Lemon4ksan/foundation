// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/urlkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestCleanHost(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fe80::1", urlkit.CleanHost("fe80::1%eth0"))
	assert.Equal(t, "[fe80::1]", urlkit.CleanHost("[fe80::1%eth0]"))
	assert.Equal(t, "192.168.1.1", urlkit.CleanHost("192.168.1.1"))
	assert.Equal(t, "example.com", urlkit.CleanHost("example.com"))
	assert.Equal(t, "xn--d1abbgf6aiiy.xn--p1ai", urlkit.CleanHost("президент.рф"))
}

func TestCleanHostPort(t *testing.T) {
	t.Parallel()

	h, p := urlkit.CleanHostPort("example.com:8080")
	assert.Equal(t, "example.com", h)
	assert.Equal(t, "8080", p)

	h, p = urlkit.CleanHostPort("fe80::1%eth0")
	assert.Equal(t, "fe80::1", h)
	assert.Equal(t, "", p)
}
