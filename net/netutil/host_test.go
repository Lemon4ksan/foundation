// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package netutil_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"

	"github.com/lemon4ksan/foundation/net/netutil"
)

func TestCleanHost(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fe80::1", netutil.CleanHost("fe80::1%eth0"))
	assert.Equal(t, "[fe80::1]", netutil.CleanHost("[fe80::1%eth0]"))
	assert.Equal(t, "192.168.1.1", netutil.CleanHost("192.168.1.1"))
	assert.Equal(t, "example.com", netutil.CleanHost("example.com"))
	assert.Equal(t, "xn--d1abbgf6aiiy.xn--p1ai", netutil.CleanHost("президент.рф"))
}

func TestCleanHostPort(t *testing.T) {
	t.Parallel()

	h, p := netutil.CleanHostPort("example.com:8080")
	assert.Equal(t, "example.com", h)
	assert.Equal(t, "8080", p)

	h, p = netutil.CleanHostPort("fe80::1%eth0")
	assert.Equal(t, "fe80::1", h)
	assert.Equal(t, "", p)
}
