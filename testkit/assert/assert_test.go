// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package assert

import (
	"errors"
	"testing"
	"time"
)

type mockTB struct {
	testing.TB
	failed   bool
	messages []string
}

func (m *mockTB) Helper() {}

func (m *mockTB) Errorf(format string, args ...any) {
	m.failed = true
}

func (m *mockTB) FailNow() {
	m.failed = true
}

func TestAssert_Basic(t *testing.T) {
	mock := &mockTB{}

	if !Equal(mock, 10, 10) || mock.failed {
		t.Fatalf("Equal failed on identical values")
	}

	mock.failed = false
	if Equal(mock, 10, 20) || !mock.failed {
		t.Fatalf("Equal should fail on different values")
	}

	mock.failed = false
	if !NotEqual(mock, 10, 20) || mock.failed {
		t.Fatalf("NotEqual failed on different values")
	}

	mock.failed = false
	if !True(mock, true) || mock.failed {
		t.Fatalf("True failed")
	}

	mock.failed = false
	if !False(mock, false) || mock.failed {
		t.Fatalf("False failed")
	}

	mock.failed = false
	if !Nil(mock, nil) || mock.failed {
		t.Fatalf("Nil failed on nil")
	}

	mock.failed = false
	var ptr *int
	if !Nil(mock, ptr) || mock.failed {
		t.Fatalf("Nil failed on nil pointer")
	}

	mock.failed = false
	val := 5
	if !NotNil(mock, &val) || mock.failed {
		t.Fatalf("NotNil failed on non-nil pointer")
	}

	mock.failed = false
	if !NoError(mock, nil) || mock.failed {
		t.Fatalf("NoError failed on nil error")
	}

	mock.failed = false
	err := errors.New("custom error")
	if !Error(mock, err) || mock.failed {
		t.Fatalf("Error failed on non-nil error")
	}

	mock.failed = false
	if !ErrorContains(mock, err, "custom") || mock.failed {
		t.Fatalf("ErrorContains failed")
	}

	mock.failed = false
	if !Contains(mock, "hello world", "world") || mock.failed {
		t.Fatalf("Contains failed on string")
	}

	mock.failed = false
	if !Contains(mock, []int{1, 2, 3}, 2) || mock.failed {
		t.Fatalf("Contains failed on slice")
	}

	mock.failed = false
	if !NotContains(mock, []int{1, 2, 3}, 5) || mock.failed {
		t.Fatalf("NotContains failed on slice")
	}

	mock.failed = false
	if !Len(mock, []int{1, 2, 3}, 3) || mock.failed {
		t.Fatalf("Len failed")
	}

	mock.failed = false
	if !Empty(mock, []int{}) || mock.failed {
		t.Fatalf("Empty failed")
	}

	mock.failed = false
	if !NotEmpty(mock, []int{1}) || mock.failed {
		t.Fatalf("NotEmpty failed")
	}

	mock.failed = false
	if !Greater(mock, 10, 5) || mock.failed {
		t.Fatalf("Greater failed")
	}

	mock.failed = false
	if !Less(mock, 5, 10) || mock.failed {
		t.Fatalf("Less failed")
	}

	mock.failed = false
	if !Panics(mock, func() { panic("oops") }) || mock.failed {
		t.Fatalf("Panics failed")
	}

	mock.failed = false
	if !NotPanics(mock, func() {}) || mock.failed {
		t.Fatalf("NotPanics failed")
	}

	mock.failed = false
	count := 0
	if !Eventually(mock, func() bool {
		count++
		return count >= 3
	}, 100*time.Millisecond, 5*time.Millisecond) || mock.failed {
		t.Fatalf("Eventually failed")
	}
}
