// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package require

import (
	"testing"
)

type mockTB struct {
	testing.TB
	failed bool
}

func (m *mockTB) Helper() {}

func (m *mockTB) Errorf(format string, args ...any) {
	m.failed = true
}

func (m *mockTB) FailNow() {
	m.failed = true
}

func TestRequire_Basic(t *testing.T) {
	mock := &mockTB{}

	Equal(mock, "a", "a")
	if mock.failed {
		t.Fatalf("Equal failed")
	}

	NoError(mock, nil)
	if mock.failed {
		t.Fatalf("NoError failed")
	}

	True(mock, true)
	if mock.failed {
		t.Fatalf("True failed")
	}
}
