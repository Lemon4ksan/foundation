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

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type mockService struct {
	name       string
	deps       []string
	initErr    error
	startErr   error
	stopErr    error
	initCalls  int
	startCalls int
	stopCalls  int
	mu         sync.Mutex
}

func (m *mockService) Name() string {
	return m.name
}

func (m *mockService) Init(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initCalls++

	return m.initErr
}

func (m *mockService) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startCalls++

	return m.startErr
}

func (m *mockService) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopCalls++

	return m.stopErr
}

func (m *mockService) Dependencies() []string {
	return m.deps
}

type mockSimpleService struct {
	name       string
	initCalls  int
	startCalls int
	stopCalls  int
}

func (m *mockSimpleService) Name() string {
	return m.name
}

func (m *mockSimpleService) Init(ctx context.Context) error {
	m.initCalls++
	return nil
}

func (m *mockSimpleService) Start(ctx context.Context) error {
	m.startCalls++
	return nil
}

func (m *mockSimpleService) Stop(ctx context.Context) error {
	m.stopCalls++
	return nil
}

func TestOrchestrator_HappyPath(t *testing.T) {
	o := NewOrchestrator()

	s1 := &mockService{name: "db"}
	s2 := &mockService{name: "cache", deps: []string{"db"}}
	s3 := &mockService{name: "api", deps: []string{"db", "cache"}}
	s4 := &mockSimpleService{name: "simple"} // Doesn't implement Dependent

	o.Register(s1)
	o.Register(s2)
	o.Register(s3)
	o.Register(s4)

	// Verify All contains all of them
	all := o.All()
	if len(all) != 4 {
		t.Fatalf("expected 4 registered services, got %d", len(all))
	}

	ctx := context.Background()

	// Init
	if err := o.InitAll(ctx); err != nil {
		t.Fatalf("unexpected error during InitAll: %v", err)
	}

	// Verify ordering: simple could be anywhere, but db must be before cache and api, cache must be before api
	// Let's inspect o.ordered directly
	var dbIdx, cacheIdx, apiIdx int
	for i, s := range o.ordered {
		switch s.Name() {
		case "db":
			dbIdx = i
		case "cache":
			cacheIdx = i
		case "api":
			apiIdx = i
		}
	}

	if dbIdx > cacheIdx || dbIdx > apiIdx || cacheIdx > apiIdx {
		t.Errorf("incorrect topological order: db index %d, cache index %d, api index %d", dbIdx, cacheIdx, apiIdx)
	}

	// Verify init was called on all
	if s1.initCalls != 1 || s2.initCalls != 1 || s3.initCalls != 1 || s4.initCalls != 1 {
		t.Errorf("expected each service to be initialized once")
	}

	// Start
	if err := o.StartAll(ctx); err != nil {
		t.Fatalf("unexpected error during StartAll: %v", err)
	}

	// StartAll again should be a no-op
	if err := o.StartAll(ctx); err != nil {
		t.Fatalf("unexpected error during second StartAll: %v", err)
	}

	if s1.startCalls != 1 || s2.startCalls != 1 || s3.startCalls != 1 || s4.startCalls != 1 {
		t.Errorf("expected each service to be started once")
	}

	// Stop
	if err := o.StopAll(ctx); err != nil {
		t.Fatalf("unexpected error during StopAll: %v", err)
	}

	if s1.stopCalls != 1 || s2.stopCalls != 1 || s3.stopCalls != 1 || s4.stopCalls != 1 {
		t.Errorf("expected each service to be stopped once")
	}
}

func TestOrchestrator_CircularDependency(t *testing.T) {
	o := NewOrchestrator()

	s1 := &mockService{name: "s1", deps: []string{"s2"}}
	s2 := &mockService{name: "s2", deps: []string{"s1"}}

	o.Register(s1)
	o.Register(s2)

	err := o.InitAll(context.Background())
	if err == nil {
		t.Fatal("expected error due to circular dependency, got nil")
	}
}

func TestOrchestrator_MissingDependency(t *testing.T) {
	o := NewOrchestrator()

	s1 := &mockService{name: "s1", deps: []string{"missing"}}

	o.Register(s1)

	err := o.InitAll(context.Background())
	if err == nil {
		t.Fatal("expected error due to missing dependency, got nil")
	}
}

func TestOrchestrator_InitFailure(t *testing.T) {
	o := NewOrchestrator()

	errDummy := errors.New("dummy init error")
	s1 := &mockService{name: "s1", initErr: errDummy}

	o.Register(s1)

	err := o.InitAll(context.Background())
	if !errors.Is(err, errDummy) {
		t.Fatalf("expected error %v, got %v", errDummy, err)
	}
}

func TestOrchestrator_StartFailure(t *testing.T) {
	o := NewOrchestrator()

	s1 := &mockService{name: "s1"}
	s2 := &mockService{name: "s2", deps: []string{"s1"}, startErr: errors.New("s2 start failed")}

	o.Register(s1)
	o.Register(s2)

	if err := o.InitAll(context.Background()); err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}

	err := o.StartAll(context.Background())
	if err == nil {
		t.Fatal("expected start error, got nil")
	}

	// Since s2 failed to start, s1 (which started successfully before s2) should have been stopped
	s1.mu.Lock()
	defer s1.mu.Unlock()

	if s1.startCalls != 1 {
		t.Errorf("expected s1 to be started, got %d calls", s1.startCalls)
	}

	if s1.stopCalls != 1 {
		t.Errorf("expected s1 to be stopped after s2 start failure, got %d calls", s1.stopCalls)
	}
}

func TestOrchestrator_StartBeforeInit(t *testing.T) {
	o := NewOrchestrator()
	s1 := &mockService{name: "s1"}
	o.Register(s1)

	// Since InitAll was never called, StartAll should return ErrNotInitialized.
	if err := o.StartAll(context.Background()); !errors.Is(err, ErrNotInitialized) {
		t.Fatalf("expected ErrNotInitialized, got: %v", err)
	}

	if s1.startCalls != 0 {
		t.Errorf("expected start to not be called on s1, got %d calls", s1.startCalls)
	}
}

func TestOrchestrator_ConcurrentRegister(t *testing.T) {
	o := NewOrchestrator()

	var wg sync.WaitGroup

	numGoroutines := 100

	wg.Add(numGoroutines)

	for i := range numGoroutines {
		go func(idx int) {
			defer wg.Done()

			o.Register(&mockService{name: string(rune(idx))})
			_ = o.All()
		}(i)
	}

	wg.Wait()
}

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

	ctx := t.Context()

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
