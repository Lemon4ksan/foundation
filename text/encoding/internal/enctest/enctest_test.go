// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enctest_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding"
	"github.com/lemon4ksan/foundation/text/encoding/internal/enctest"
)

func TestNopEncoding(t *testing.T) {
	enctest.TestEncoding(t, encoding.Nop, "hello world", "hello world", "", "")
}
