// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gomock_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/gomock"
)

type threadSafeGomockReporter struct {
	mu     sync.Mutex
	errors []string
	fatals []string
}

func (r *threadSafeGomockReporter) Errorf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, fmt.Sprintf(format, args...))
}

func (r *threadSafeGomockReporter) Fatalf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.fatals = append(r.fatals, fmt.Sprintf(format, args...))
}

func (r *threadSafeGomockReporter) Helper() {}

func TestAdversarial_Gomock_DropInCompatibility(t *testing.T) {
	t.Parallel()

	rep := &threadSafeGomockReporter{}
	ctrl := gomock.NewController(rep)

	// Verify Controller type alias and methods
	assert.NotNil(t, ctrl)

	// Record an expected call
	receiver := "dummyService"
	call := ctrl.RecordCall(receiver, "Execute", gomock.Any(), gomock.Eq(100))
	call.Return("result", nil).Times(10)

	// Concurrent invocation through ctrl.Call
	const count = 10
	var wg sync.WaitGroup
	wg.Add(count)

	for i := range count {
		go func(idx int) {
			defer wg.Done()
			rets := ctrl.Call(receiver, "Execute", "param", 100)
			assert.Equal(t, "result", rets[0])
			assert.Nil(t, rets[1])
		}(i)
	}

	wg.Wait()
	assert.True(t, ctrl.Satisfied())
	ctrl.Finish()

	assert.Equal(t, 0, len(rep.errors))
	assert.Equal(t, 0, len(rep.fatals))
}

func TestAdversarial_Gomock_VerificationFailure(t *testing.T) {
	t.Parallel()

	rep := &threadSafeGomockReporter{}
	ctrl := gomock.NewController(rep)

	receiver := "dummy"
	ctrl.RecordCall(receiver, "MustBeCalled").Times(1)

	// Don't call -> Finish() should report Errorf
	ctrl.Finish()

	rep.mu.Lock()
	defer rep.mu.Unlock()
	assert.Equal(t, 1, len(rep.errors))
	assert.Contains(t, rep.errors[0], "missing expected call")
}
