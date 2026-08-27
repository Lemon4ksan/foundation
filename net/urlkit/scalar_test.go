// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit

import (
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestUnescapeScalar_FallbackAndHex(t *testing.T) {
	t.Parallel()

	// fromHexChar
	assert.Equal(t, 0, fromHexChar('0'))
	assert.Equal(t, 9, fromHexChar('9'))
	assert.Equal(t, 10, fromHexChar('a'))
	assert.Equal(t, 15, fromHexChar('f'))
	assert.Equal(t, 10, fromHexChar('A'))
	assert.Equal(t, 15, fromHexChar('F'))
	assert.Equal(t, -1, fromHexChar('g'))
	assert.Equal(t, -1, fromHexChar('Z'))

	// unescapeScalar valid
	src := []byte("hello%20world+plus%21")
	dst := make([]byte, len(src))
	n, err := unescapeScalar(dst, src)
	require.NoError(t, err)
	assert.Equal(t, "hello world plus!", string(dst[:n]))

	// unescapeScalar truncated %
	_, err = unescapeScalar(dst, []byte("hello%2"))
	assert.ErrorIs(t, err, ErrInvalidEscape)

	// unescapeScalar invalid hex
	_, err = unescapeScalar(dst, []byte("hello%2Z"))
	assert.ErrorIs(t, err, ErrInvalidEscape)
}
