// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package assert_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
)

func Example() {
	// In a real test, this would be a *testing.T passed by the test runner.
	var t *testing.T

	// Assertions do not terminate the test on failure.
	assert.Equal(t, 42, 42)
	assert.ElementsMatch(t, []string{"a", "b"}, []string{"b", "a"})
}
