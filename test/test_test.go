package test

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	stdurl "net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/idna"

	"github.com/lemon4ksan/foundation/async/dedup"
	"github.com/lemon4ksan/foundation/async/event"
	"github.com/lemon4ksan/foundation/async/lifecycle"
	"github.com/lemon4ksan/foundation/async/pool"
	"github.com/lemon4ksan/foundation/async/rate"
	"github.com/lemon4ksan/foundation/async/task"
	"github.com/lemon4ksan/foundation/net/netutil"
	"github.com/lemon4ksan/foundation/net/urlkit"
	"github.com/lemon4ksan/foundation/silicon/clock"
	"github.com/lemon4ksan/foundation/sync/breaker"
	"github.com/lemon4ksan/foundation/sync/keylock"
	"github.com/lemon4ksan/foundation/sync/limiter"
	"github.com/lemon4ksan/foundation/sync/semaphore"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
	"github.com/lemon4ksan/foundation/text/encoding/charmap"
	"github.com/lemon4ksan/foundation/text/encoding/htmlindex"
)

// -----------------------------------------------------------------------------
// 1. URL Engine: Standard net/url.Parse vs Foundation CRC32 Sharded URL Cache
// -----------------------------------------------------------------------------

const testRawURL = "https://api.gateway.internal:8443/v2/telemetry/events?session_id=9876543210&region=eu-central-1&trace=true#anchor"

func BenchmarkCompare_URL_Parse_Std(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		u, err := stdurl.Parse(testRawURL)
		if err != nil || u.Host == "" {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_URL_Parse_Foundation(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		u, err := urlkit.Parse(testRawURL)
		if err != nil || u.Host == "" {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 2. Clock & Timestamping: Standard time.Now() vs Foundation CoarseClock
// -----------------------------------------------------------------------------

func BenchmarkCompare_Clock_Std_TimeNow(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		t := time.Now()
		if t.IsZero() {
			b.Fatal("zero time")
		}
	}
}

func BenchmarkCompare_Clock_Foundation_CoarseClock(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		t := clock.CoarseNowNano()
		if t == 0 {
			b.Fatal("zero time")
		}
	}
}

// -----------------------------------------------------------------------------
// 3. Rate Limiting: Foundation Rate Limiter (Token Bucket Allow vs Sometimes)
// -----------------------------------------------------------------------------

func BenchmarkCompare_Rate_TokenBucket_Allow(b *testing.B) {
	lim := rate.NewLimiter(rate.Inf, 1000000000)
	b.ReportAllocs()

	for b.Loop() {
		if !lim.Allow() {
			b.Fatal("rate limit exceeded")
		}
	}
}

func BenchmarkCompare_Rate_Sometimes_FastPath(b *testing.B) {
	s := &rate.Sometimes{Interval: time.Hour}
	b.ReportAllocs()

	for b.Loop() {
		s.Do(func() {})
	}
}

// -----------------------------------------------------------------------------
// 4. Web Encodings: WhatWG Charset Resolver & Windows-1251 Transcoding
// -----------------------------------------------------------------------------

func BenchmarkCompare_Encoding_WhatWG_Lookup(b *testing.B) {
	charsets := []string{"windows-1251", "utf-8", "shift_jis", "gbk", "iso-8859-1", "euc-kr"}
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		enc, err := htmlindex.Get(charsets[i%len(charsets)])
		if err != nil || enc == nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_Encoding_Windows1251_Transcode(b *testing.B) {
	sampleWin1251 := []byte(
		"\xcf\xf0\xe8\xe2\xe5\xf2, \xcc\xe8\xf0! 1234567890 Тестирование производительности кремния",
	)
	dec := charmap.Windows1251.NewDecoder()
	b.ReportAllocs()

	for b.Loop() {
		out, err := dec.Bytes(sampleWin1251)
		if err != nil || len(out) == 0 {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_Encoding_Windows1251_Transform_ZeroAlloc(b *testing.B) {
	sampleWin1251 := []byte(
		"\xcf\xf0\xe8\xe2\xe5\xf2, \xcc\xe8\xf0! 1234567890 Тестирование производительности кремния",
	)
	dec := charmap.Windows1251.NewDecoder()
	dst := make([]byte, 256)
	b.ReportAllocs()

	for b.Loop() {
		_, _, err := dec.Transform(dst, sampleWin1251, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 5. Host Sanitization & CleanHost (RFC 3492 / 5891)
// -----------------------------------------------------------------------------

const (
	internationalDomain = "президент.рф"
	asciiDomain         = "api.gateway.internal"
)

func BenchmarkCompare_CleanHost_Unicode(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		res := netutil.CleanHost(internationalDomain)
		if !strings.HasPrefix(res, "xn--") {
			b.Fatal("unexpected result")
		}
	}
}

func BenchmarkCompare_CleanHost_ASCII_FastPath(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		res := netutil.CleanHost(asciiDomain)
		if res != asciiDomain {
			b.Fatal("unexpected result")
		}
	}
}

func BenchmarkCompare_IDNA_ToASCII(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		res, err := idna.ToASCII(internationalDomain)
		if err != nil || !strings.HasPrefix(res, "xn--") {
			b.Fatal(err)
		}
	}
}

// DownstreamMock represents a simulated flaky downstream service.
type DownstreamMock struct {
	shouldFail   atomic.Bool
	latency      time.Duration
	callCount    atomic.Uint64
	failCount    atomic.Uint64
	successCount atomic.Uint64
}

func (d *DownstreamMock) Call(ctx context.Context) (string, error) {
	d.callCount.Add(1)

	if d.latency > 0 {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(d.latency):
		}
	}

	if d.shouldFail.Load() {
		d.failCount.Add(1)
		return "", errors.New("downstream error: service unavailable")
	}

	d.successCount.Add(1)

	return "ok", nil
}

// OrderProcessedEvent is sent over the event.
type OrderProcessedEvent struct {
	event.Envelope
	OrderID uint64
	Result  string
	Err     error
}

// TestScenario_PipelineDeadlockAndResilience builds a massive concurrent integration pipeline.
// It combines:
// 1. event.Bus (events)
// 2. pool.Pool (worker scaling + task queue)
// 3. sync/semaphore (dynamic resource gating)
// 4. sync/breaker (circuit breaker isolation)
// 5. task.Tracker (correlation tracking)
// 6. dedup.Group (request deduplication)
//
// The goal is to aggressively trigger race conditions, deadlocks, and context cancellation leakages.
func TestScenario_PipelineDeadlockAndResilience(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize the components.
	eventBus := event.New()
	defer eventBus.Close()

	workerPool := pool.New[struct{}](pool.Config{
		MinWorkers:  5,
		MaxWorkers:  50,
		IdleTimeout: 100 * time.Millisecond,
		QueueLimit:  2000,
	})
	defer workerPool.Close()

	resourceGate := semaphore.New(10) // Initial limit is 10

	downstream := &DownstreamMock{
		latency: 1 * time.Millisecond,
	}

	circuitBreaker := breaker.New[string](breaker.Config{
		FailureThreshold: 0.3, // Trip if 30% of requests fail
		Cooldown:         50 * time.Millisecond,
		MinRequests:      5,
	})

	jobTracker := task.NewTracker[uint64, string](2000)
	defer jobTracker.Close()

	dedupGroup := &dedup.Group[uint64, string]{}

	// Metrics.
	var (
		processedCount atomic.Uint64
		cancelledCount atomic.Uint64
		failedCount    atomic.Uint64
	)

	// Subscribe to OrderProcessedEvent on the event bus with ample buffer for parallel burst load.
	sub := eventBus.SubscribeBuffer(2048, OrderProcessedEvent{})

	subDone := make(chan struct{})
	go func() {
		defer close(subDone)

		for ev := range sub.C() {
			event, ok := ev.(OrderProcessedEvent)
			if !ok {
				continue
			}

			if event.Err != nil {
				if errors.Is(event.Err, context.Canceled) {
					cancelledCount.Add(1)
				} else {
					failedCount.Add(1)
				}
			} else {
				processedCount.Add(1)
			}
		}
	}()

	// Spawn a background goroutine that continuously resizes the semaphore to stress-test Acquire/Release locks.
	var stopSemaphoreChurn atomic.Bool
	go func() {
		rng := rand.New(rand.NewSource(42))
		for !stopSemaphoreChurn.Load() {
			time.Sleep(10 * time.Millisecond)

			newLimit := rng.Intn(30) + 1
			resourceGate.Resize(newLimit)
		}
	}()

	// Spawn a background goroutine that toggles downstream failures to trip the circuit breaker.
	var stopBreakerChurn atomic.Bool
	go func() {
		for !stopBreakerChurn.Load() {
			time.Sleep(150 * time.Millisecond)
			downstream.shouldFail.Store(true)
			time.Sleep(50 * time.Millisecond)
			downstream.shouldFail.Store(false)
		}
	}()

	// Simulate high concurrent client load.
	numRequests := 1000

	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := range numRequests {
		orderID := uint64(i%100) + 1 // High probability of duplicate OrderIDs for deduplication stress

		go func(id uint64, reqIdx int) {
			defer wg.Done()

			// Decide randomly whether the client aborts early (concurrency cancellation stress).
			reqCtx := ctx

			var cancelReq context.CancelFunc
			if reqIdx%7 == 0 {
				reqCtx, cancelReq = context.WithTimeout(ctx, time.Duration(reqIdx%5)*time.Millisecond)
				defer cancelReq()
			}

			// Add a correlation tracking job in Job Manager.
			jobID := uint64(reqIdx)

			err := jobTracker.Add(jobID, nil, task.WithContext[string](reqCtx))
			if err != nil {
				eventBus.Publish(OrderProcessedEvent{OrderID: id, Err: err})
				return
			}

			// Submit the processing pipeline to the worker pool.
			_, err = workerPool.Submit(reqCtx, func(poolCtx context.Context) (struct{}, error) {
				res, err := dedupGroup.Do(poolCtx, id, func(workerCtx context.Context) (string, error) {
					// Acquire slot from the resizable semaphore.
					if err := resourceGate.Acquire(workerCtx); err != nil {
						return "", err
					}

					defer resourceGate.Release()

					// Execute call protected by Circuit Breaker.
					return circuitBreaker.Do(workerCtx, func(cbCtx context.Context) (string, error) {
						return downstream.Call(cbCtx)
					})
				})

				// Resolve the job back to the client thread.
				jobTracker.Resolve(poolCtx, jobID, res, err)

				return struct{}{}, nil
			})
			if err != nil {
				// Worker pool queue is full or pool closed.
				jobTracker.Remove(t.Context(), jobID)
				eventBus.Publish(OrderProcessedEvent{OrderID: id, Err: err})
				return
			}

			// Client awaits response from job manager.
			res, err := jobTracker.WaitFor(reqCtx, jobID)
			eventBus.Publish(OrderProcessedEvent{OrderID: id, Result: res, Err: err})
		}(orderID, i)
	}

	wg.Wait()

	// Wait briefly for all published events to be processed by the subscriber.
	deadline := time.Now().Add(5 * time.Second)
	for processedCount.Load()+cancelledCount.Load()+failedCount.Load() < uint64(numRequests) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}

	// Stop background churn goroutines.
	stopSemaphoreChurn.Store(true)
	stopBreakerChurn.Store(true)

	// Ensure sub is cleaned up.
	sub.Unsubscribe()
	<-subDone

	// Logging test metrics for evaluation.
	t.Logf("Stress Pipeline Stats:\n"+
		"  Total Requests: %d\n"+
		"  Successfully Processed: %d\n"+
		"  Cancelled Requests: %d\n"+
		"  Failed Requests (including Circuit Breaker trips): %d\n"+
		"  Downstream Calls: %d (Success: %d, Fail: %d)",
		numRequests,
		processedCount.Load(),
		cancelledCount.Load(),
		failedCount.Load(),
		downstream.callCount.Load(),
		downstream.successCount.Load(),
		downstream.failCount.Load(),
	)

	// In high-concurrency tests, we verify no panic/deadlock occurred and work completed.
	assert.GreaterOrEqual(t, processedCount.Load()+cancelledCount.Load()+failedCount.Load(), uint64(numRequests))
}

// TestScenario_KeyedLimiterAndKeyLockStress checks for:
// 1. Memory leaks in sync/limiter.KeyedLimiter (ensuring unused limiters get swept up).
// 2. Deadlocks & state corruption in sync/keylock.KeyMutex (ensuring refcount behaves).
func TestScenario_KeyedLimiterAndKeyLockStress(t *testing.T) {
	// Create a Keyed with 10ms TTL and rate of 100 requests/sec.
	keyLimiter := limiter.NewKeyed[string](rate.Limit(100), 2, 10*time.Millisecond)
	defer keyLimiter.Close()

	keyMutex := keylock.New[string]()

	numGoroutines := 100
	numIterations := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// We lock/limit across a set of keys.
	// We want high contention on some keys, and complete churn on others.
	for g := range numGoroutines {
		go func(gIdx int) {
			defer wg.Done()

			rng := rand.New(rand.NewSource(int64(gIdx)))

			for range numIterations {
				var key string
				if gIdx%2 == 0 {
					// High contention: all even goroutines fight for 3 keys
					key = fmt.Sprintf("contended-%d", rng.Intn(3))
				} else {
					// High churn: all odd goroutines request completely unique keys
					key = fmt.Sprintf("churn-%d-%d", gIdx, rng.Intn(10000))
				}

				// Wait on keyed rate limiter.
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				err := keyLimiter.Wait(ctx, key)

				cancel()

				if err != nil {
					// Timeout is expected under heavy rate limits
					continue
				}

				// Lock the key.
				keyMutex.Lock(key)
				// Perform short sleep to stress lock duration.
				time.Sleep(100 * time.Microsecond)
				keyMutex.Unlock(key)
			}
		}(g)
	}

	wg.Wait()

	// Wait 25ms to allow TTL sweeper to clean up churn keys in KeyedLimiter.
	time.Sleep(25 * time.Millisecond)

	// Explicitly verify that the internal map has been cleared of all expired keys.
	assert.Equal(t, 0, keyLimiter.Len())

	// Since we used 10,000+ unique churn keys, verify that the internal map
	// doesn't leak memory. Let's make sure the number of active limiters is small.
	// Since we cannot inspect internal maps directly, we verify that subsequent requests work
	// and no deadlock occurred.
	assert.NotPanics(t, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := keyLimiter.Wait(ctx, "fresh-key")
		assert.NoError(t, err)
	})

	// Check keylock integrity: verify no keys are locked permanently.
	// If lock/unlock reference counts are correct, keyMutex should be completely empty/clean.
	assert.Empty(t, keyMutex.Keys())
}

// MockLifecycleService represents a mock component registered with lifecycle.Orchestrator.
type MockLifecycleService struct {
	name         string
	dependencies []string
	initErr      error
	startErr     error
	initCalled   atomic.Bool
	startCalled  atomic.Bool
	stopCalled   atomic.Bool
	behaviorLog  []string
	mu           sync.Mutex
	runner       *lifecycle.BehaviorRunner
	onStop       func(string)
}

func (m *MockLifecycleService) Name() string           { return m.name }
func (m *MockLifecycleService) Dependencies() []string { return m.dependencies }

func (m *MockLifecycleService) Init(ctx context.Context) error {
	m.initCalled.Store(true)

	if m.initErr != nil {
		return m.initErr
	}

	return nil
}

func (m *MockLifecycleService) Start(ctx context.Context) error {
	m.startCalled.Store(true)

	if m.startErr != nil {
		return m.startErr
	}

	m.runner = lifecycle.NewBehaviorRunner()
	m.runner.Register(&mockBehavior{
		name: m.name + "-behavior",
		log: func(s string) {
			m.mu.Lock()
			m.behaviorLog = append(m.behaviorLog, s)
			m.mu.Unlock()
		},
	})

	return m.runner.Start(ctx)
}

func (m *MockLifecycleService) Stop(ctx context.Context) error {
	m.stopCalled.Store(true)

	if m.runner != nil {
		m.runner.Stop()
	}

	if m.onStop != nil {
		m.onStop(m.name)
	}

	return nil
}

type mockBehavior struct {
	name string
	log  func(string)
}

func (mb *mockBehavior) Name() string { return mb.name }
func (mb *mockBehavior) Run(ctx context.Context) error {
	mb.log("started")
	defer mb.log("stopped")

	<-ctx.Done()

	return ctx.Err()
}

// TestScenario_LifecycleOrchestrationRollback tests:
// 1. Dependency-aware initialization and start sequence.
// 2. Cascade stopping of behaviors inside lifecycle runner.
// 3. Rollback of started services in reverse order when a middle service fails to start.
func TestScenario_LifecycleOrchestrationRollback(t *testing.T) {
	// Build a service graph:
	// db -> cache (depends on db) -> web (depends on cache)
	// queue -> worker (depends on queue)
	// web and worker are both leaf nodes.
	var (
		stopHistoryMu sync.Mutex
		stopHistory   []string
	)

	recordStop := func(name string) {
		stopHistoryMu.Lock()

		stopHistory = append(stopHistory, name)
		stopHistoryMu.Unlock()
	}

	db := &MockLifecycleService{name: "db", onStop: recordStop}
	cache := &MockLifecycleService{name: "cache", dependencies: []string{"db"}, onStop: recordStop}
	web := &MockLifecycleService{name: "web", dependencies: []string{"cache"}, onStop: recordStop}
	queue := &MockLifecycleService{name: "queue", onStop: recordStop}
	worker := &MockLifecycleService{name: "worker", dependencies: []string{"queue"}, onStop: recordStop}

	// We inject a start failure in the "web" service.
	web.startErr = errors.New("web server port binding failed")

	orch := lifecycle.NewOrchestrator()
	orch.Register(db)
	orch.Register(cache)
	orch.Register(web)
	orch.Register(queue)
	orch.Register(worker)

	ctx := context.Background()

	// 1. Initialize all. Must succeed since topology is correct.
	err := orch.InitAll(ctx)
	require.NoError(t, err)

	assert.True(t, db.initCalled.Load())
	assert.True(t, cache.initCalled.Load())
	assert.True(t, web.initCalled.Load())
	assert.True(t, queue.initCalled.Load())
	assert.True(t, worker.initCalled.Load())

	// 2. Start all. This should fail on "web", initiating rollback on all started components.
	err = orch.StartAll(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "web server port binding failed")

	// 3. Verify Rollback behavior:
	// A service must be stopped if and only if it was started.
	assert.Equal(t, db.startCalled.Load(), db.stopCalled.Load())
	assert.Equal(t, cache.startCalled.Load(), cache.stopCalled.Load())
	assert.Equal(t, queue.startCalled.Load(), queue.stopCalled.Load())
	assert.Equal(t, worker.startCalled.Load(), worker.stopCalled.Load())

	// Verify the reverse topological order of stop invocations:
	indexOf := func(name string) int {
		stopHistoryMu.Lock()
		defer stopHistoryMu.Unlock()

		for i, h := range stopHistory {
			if h == name {
				return i
			}
		}

		return -1
	}

	// If cache was started (and stopped), it must stop before db.
	if cache.startCalled.Load() {
		idxCache := indexOf("cache")
		idxDB := indexOf("db")

		assert.GreaterOrEqual(t, idxCache, 0)
		assert.GreaterOrEqual(t, idxDB, 0)
		assert.Less(t, idxCache, idxDB)
	}

	// If worker was started (and stopped), it must stop before queue.
	if worker.startCalled.Load() {
		idxWorker := indexOf("worker")
		idxQueue := indexOf("queue")

		assert.GreaterOrEqual(t, idxWorker, 0)
		assert.GreaterOrEqual(t, idxQueue, 0)
		assert.Less(t, idxWorker, idxQueue)
	}

	// Verify that the internal behaviors in mockBehavior were also stopped correctly.
	db.mu.Lock()
	assert.Contains(t, db.behaviorLog, "started")
	assert.Contains(t, db.behaviorLog, "stopped")
	db.mu.Unlock()

	cache.mu.Lock()
	assert.Contains(t, cache.behaviorLog, "started")
	assert.Contains(t, cache.behaviorLog, "stopped")
	cache.mu.Unlock()
}
