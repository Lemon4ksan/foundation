// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package identifier_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding/internal/identifier"
)

func TestMIB(t *testing.T) {
	mib := identifier.UTF8
	if mib == 0 {
		t.Fatalf("identifier.UTF8 MIB should not be 0")
	}
	if identifier.Replacement <= 10000 {
		t.Fatalf("identifier.Replacement should be an unofficial MIB > 10000")
	}
}
