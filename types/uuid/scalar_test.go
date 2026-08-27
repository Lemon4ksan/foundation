// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uuid

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestUUID_ScalarFallback(t *testing.T) {
	t.Parallel()

	u := MustNewV4()
	var buf [36]byte
	formatScalar(&u, &buf)
	assert.Equal(t, u.String(), string(buf[:]))

	parsed, ok := parseScalar(u.String())
	assert.True(t, ok)
	assert.Equal(t, u, parsed)

	// Invalid cases in parseScalar
	_, ok = parseScalar("6ba7b810_9dad_11d1_80b4_00c04fd430c8") // bad delimiters
	assert.False(t, ok)

	_, ok = parseScalar("6ba7b810-9dad-11d1-80b4-00c04fd430cg") // bad hex char
	assert.False(t, ok)
}
