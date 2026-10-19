// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base64_test

import (
	"math"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/base64"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestRepro_URLEncodedLen_NegativeAndOverflow(t *testing.T) {
	t.Run("negative length returns 0", func(t *testing.T) {
		assert.Equal(t, 0, base64.URLEncodedLen(-1))
		assert.Equal(t, 0, base64.URLEncodedLen(-100))
	})

	t.Run("overflow panics", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r)
		}()
		_ = base64.URLEncodedLen(math.MaxInt / 4)
	})
}
