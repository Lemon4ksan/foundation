// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lifecycle

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

// mockLogger implements the Logger interface and records messages for verification in tests.
type mockLogger struct {
	mu        sync.Mutex
	infoMsgs  []string
	errorMsgs []string
	warnMsgs  []string
}

func (l *mockLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.infoMsgs = append(l.infoMsgs, msg)
}

func (l *mockLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.errorMsgs = append(l.errorMsgs, msg)
}

func (l *mockLogger) Warn(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.warnMsgs = append(l.warnMsgs, msg)
}

// testBehavior is a mock implementation of Behavior with lifecycle tracking.
type testBehavior struct {
	name    string
	runFunc func(ctx context.Context) error
	started atomic.Bool
	stopped atomic.Bool
	mu      sync.Mutex
	err     error
}

func (t *testBehavior) Name() string { return t.name }

func (t *testBehavior) Run(ctx context.Context) error {
	t.started.Store(true)
	defer t.stopped.Store(true)

	var err error
	if t.runFunc != nil {
		err = t.runFunc(ctx)
	} else {
		<-ctx.Done()
		err = ctx.Err()
	}

	t.mu.Lock()
	t.err = err
	t.mu.Unlock()

	return err
}

func (t *testBehavior) Err() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}

func TestBehaviorRunner_BasicLifecycle(t *testing.T) {
	runner := NewBehaviorRunner()

	b1 := &testBehavior{name: "b1"}
	b2 := &testBehavior{name: "b2"}

	runner.Register(b1)
	runner.Register(b2)

	assert.Equal(t, 2, runner.Count())

	err := runner.Start(t.Context())
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	assert.True(t, b1.started.Load())
	assert.True(t, b2.started.Load())

	runner.Stop()
}

func TestBehaviorRunner_DuplicateRegister(t *testing.T) {
	runner := NewBehaviorRunner()

	runner.Register(&testBehavior{name: "dup"})
	runner.Register(&testBehavior{name: "dup"})

	assert.Equal(t, 1, runner.Count())
}

func TestBehaviorRunner_RegisterWhileRunning(t *testing.T) {
	logger := &mockLogger{}
	runner := NewBehaviorRunner(WithLogger(logger))

	b1 := &testBehavior{name: "b1"}
	runner.Register(b1)

	err := runner.Start(context.Background())
	require.NoError(t, err)

	defer runner.Stop()

	b2 := &testBehavior{name: "b2"}
	runner.Register(b2)

	assert.Equal(t, 1, runner.Count())

	logger.mu.Lock()
	hasWarn := slices.Contains(logger.warnMsgs, "Cannot register behavior while runner is running")
	logger.mu.Unlock()

	assert.True(t, hasWarn)
}

func TestBehaviorRunner_AlreadyRunning(t *testing.T) {
	runner := NewBehaviorRunner()
	runner.Register(&testBehavior{name: "b1"})

	err := runner.Start(t.Context())
	require.NoError(t, err)

	defer runner.Stop()

	err = runner.Start(t.Context())
	assert.ErrorIs(t, err, ErrAlreadyRunning)
}

func TestBehaviorRunner_FailFast(t *testing.T) {
	runner := NewBehaviorRunner(WithFailFast())

	errBehavior := &testBehavior{
		name: "failer",
		runFunc: func(ctx context.Context) error {
			return errors.New("boom")
		},
	}

	waiting := &testBehavior{name: "waiter"}

	runner.Register(errBehavior)
	runner.Register(waiting)

	err := runner.Start(t.Context())
	require.NoError(t, err)

	runner.Stop()

	assert.ErrorIs(t, waiting.Err(), context.Canceled)
}

func TestBehaviorRunner_StopWithoutStart(t *testing.T) {
	runner := NewBehaviorRunner()
	runner.Register(&testBehavior{name: "b1"})

	assert.NotPanics(t, func() {
		runner.Stop()
	})
}

func TestBehaviorRunner_EmptyStop(t *testing.T) {
	runner := NewBehaviorRunner()

	err := runner.Start(context.Background())
	require.NoError(t, err)

	runner.Stop()
}

func TestBehaviorRunner_WithLogger(t *testing.T) {
	logger := &mockLogger{}
	runner := NewBehaviorRunner(WithLogger(logger))

	errBehavior := &testBehavior{
		name: "failer",
		runFunc: func(_ context.Context) error {
			return errors.New("boom")
		},
	}

	normalBehavior := &testBehavior{name: "normal"}

	runner.Register(errBehavior)
	runner.Register(normalBehavior)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := runner.Start(ctx)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	runner.Stop()

	logger.mu.Lock()
	defer logger.mu.Unlock()

	assert.NotEmpty(t, logger.infoMsgs)
	assert.NotEmpty(t, logger.errorMsgs)
}

func TestAsService_Adapter(t *testing.T) {
	beh := &testBehavior{name: "my-behavior"}
	svc := AsService(beh)

	assert.Equal(t, "my-behavior", svc.Name())

	err := svc.Init(t.Context())
	assert.NoError(t, err)

	err = svc.Start(t.Context())
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.True(t, beh.started.Load())

	err = svc.Stop(t.Context())
	assert.ErrorIs(t, err, context.Canceled)
	assert.True(t, beh.stopped.Load())
}
