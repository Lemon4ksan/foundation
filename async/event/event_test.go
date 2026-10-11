// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

type (
	TestEventA    struct{ Envelope }
	TestEventB    struct{ Envelope }
	TestEventData struct {
		Envelope
		Payload string
	}
)

func TestBus_SubscribeAndPublish(t *testing.T) {
	Envelope{}.isEvent() // safe no-op marker invoke

	b := New()
	defer b.Close()

	sub := b.Subscribe(TestEventA{})

	// Publish a matched event.
	b.Publish(TestEventA{})

	select {
	case ev := <-sub.C():
		assert.IsType(t, TestEventA{}, ev)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout: Event was not delivered to the subscription")
	}
}

func TestBus_ResolveType(t *testing.T) {
	t1 := resolveType(TestEventA{})
	t2 := resolveType(&TestEventA{})
	assert.Equal(t, t1, t2)

	// Ensure resolveType handles nil values safely.
	assert.Nil(t, resolveType(nil))
}

func TestBus_PublishNil(t *testing.T) {
	b := New()
	defer b.Close()

	assert.NotPanics(t, func() {
		b.Publish(nil)
	})
}

func TestBus_SubscribeDuplicates(t *testing.T) {
	b := New()
	defer b.Close()

	sub := b.Subscribe(TestEventA{}, TestEventA{})
	assert.Len(t, sub.types, 1)
}

func TestBus_SubscribeAll(t *testing.T) {
	b := New()
	defer b.Close()

	subAll := b.SubscribeAll()
	b.Publish(TestEventA{})
	b.Publish(TestEventB{})

	for i := range 2 {
		select {
		case <-subAll.C():
			// Event received successfully.
		case <-time.After(100 * time.Millisecond):
			t.Fatalf("Timeout waiting for event %d in SubscribeAll", i)
		}
	}
}

func TestBus_Unsubscribe(t *testing.T) {
	b := New()
	sub := b.Subscribe(TestEventA{})

	// Verify that the subscription is successfully registered.
	b.mu.RLock()
	assert.NotEmpty(t, b.subs[resolveType(TestEventA{})])
	b.mu.RUnlock()

	sub.Unsubscribe()

	// Verify that internal maps are cleaned up.
	b.mu.RLock()
	assert.Empty(t, b.subs)
	assert.Empty(t, b.all)
	b.mu.RUnlock()

	// Verify that the subscription channel is closed.
	_, ok := <-sub.C()
	assert.False(t, ok, "Channel should be closed after unsubscribe")

	// Verify that a second Unsubscribe call is a safe no-op.
	assert.NotPanics(t, func() {
		sub.Unsubscribe()
	})
}

func TestBus_UnsubscribeWhenBusClosed(t *testing.T) {
	b := New()
	sub := b.Subscribe(TestEventA{})
	_ = b.Close()

	// The bus has closed the channel in Close(). Unsubscribe must handle
	// this gracefully and avoid duplicate channel closure panics.
	assert.NotPanics(t, func() {
		sub.Unsubscribe()
	})
}

func TestBus_Close(t *testing.T) {
	b := New()
	sub := b.Subscribe(TestEventA{})
	subAll := b.SubscribeAll()

	err := b.Close()
	assert.NoError(t, err)
	sub.Unsubscribe()

	// Verify that subsequent Close calls are safe and idempotent.
	err = b.Close()
	assert.NoError(t, err)

	// Verify that all active subscription channels are closed.
	_, ok1 := <-sub.C()
	assert.False(t, ok1)

	_, ok2 := <-subAll.C()
	assert.False(t, ok2)

	// Verify that internal registries are nil'ed out.
	assert.True(t, b.closed)
	assert.Nil(t, b.subs)
	assert.Nil(t, b.all)
}

func TestBus_OperationOnClosedBus(t *testing.T) {
	b := New()
	_ = b.Close()

	// Subscribing to a closed bus should return an already closed subscription.
	sub := b.Subscribe(TestEventA{})
	_, ok := <-sub.C()
	assert.False(t, ok)

	subAll := b.SubscribeAll()
	_, okAll := <-subAll.C()
	assert.False(t, okAll)

	// Publishing on a closed bus is a safe no-op.
	assert.NotPanics(t, func() {
		b.Publish(TestEventA{})
	})
}

func TestBus_DropOnFullBuffer(t *testing.T) {
	b := New()
	defer b.Close()

	// The default buffer size for specific subscriptions is 128.
	sub := b.Subscribe(TestEventA{})

	// Publish 129 events. The 129th should be dropped.
	for range 129 {
		b.Publish(TestEventA{})
	}

	// Drain the 128 queued events.
	count := 0
	for range 128 {
		<-sub.C()

		count++
	}

	// Verify that the 129th event was dropped and the channel does not block.
	select {
	case <-sub.C():
		t.Error("Expected 129th event to be dropped, but received it")
	default:
		// Correct non-blocking drop behavior.
	}

	assert.Equal(t, 128, count)
}

func TestBus_DirectSendToClosedSub(t *testing.T) {
	b := New()
	defer b.Close()

	sub := b.Subscribe(TestEventA{})
	sub.closed.Store(true) // Simulate concurrent unsubscription.

	// Should handle the closed status gracefully without blocking or panicking.
	assert.NotPanics(t, func() {
		b.directSend(sub, TestEventA{})
	})
}

func TestBus_ConcurrentUsage(t *testing.T) {
	b := New()

	var wg sync.WaitGroup

	publishers := 10
	subscribers := 10
	iterations := 50

	wg.Add(publishers + subscribers)

	// Concurrent Publishers
	for range publishers {
		go func() {
			defer wg.Done()

			for range iterations {
				b.Publish(TestEventData{Payload: "data"})
			}
		}()
	}

	// Concurrent Subscribers
	for range subscribers {
		go func() {
			defer wg.Done()

			sub := b.Subscribe(TestEventData{})
			defer sub.Unsubscribe()

			timeout := time.After(500 * time.Millisecond)

			received := 0
			for received < 10 {
				select {
				case _, ok := (<-sub.C()):
					if !ok {
						// Safe exit if the channel is closed concurrently by Close().
						return
					}

					received++

				case <-timeout:
					return
				}
			}
		}()
	}

	// Chaos goroutine: randomly close the bus during concurrent read/write operations.
	go func() {
		time.Sleep(10 * time.Millisecond)

		_ = b.Close()
	}()

	wg.Wait()
}

func TestBus_EnvelopeAndClosedSub(t *testing.T) {
	var ev Envelope
	ev.isEvent()

	ctx := ev.Context()
	assert.NotNil(t, ctx)
	assert.Equal(t, context.Background(), ctx)

	customCtx := context.WithValue(context.Background(), "key", "value") //nolint:staticcheck
	ev.SetContext(customCtx)
	assert.Equal(t, customCtx, ev.Context())

	b := New()
	_ = b.Close()

	sub := b.Subscribe(TestEventA{})
	assert.True(t, sub.closed.Load())
	_, open := <-sub.C()
	assert.False(t, open)

	subAll := b.SubscribeAll()
	assert.True(t, subAll.closed.Load())
	_, openAll := <-subAll.C()
	assert.False(t, openAll)

	b2 := New()
	defer b2.Close()

	subNil := b2.Subscribe(nil, TestEventA{}, nil)
	assert.Len(t, subNil.types, 1)
}

func TestBus_SetOnDropped(t *testing.T) {
	b := New()
	defer b.Close()

	var droppedEvent Event
	b.SetOnDropped(func(e Event, subID uint64) {
		droppedEvent = e
	})

	sub := b.Subscribe(TestEventA{})
	// Fill sub buffer (buffer size is 128)
	for range 128 {
		b.Publish(TestEventA{})
	}

	// 129th event should overflow and trigger SetOnDropped callback
	overflowEvent := TestEventA{}
	b.Publish(overflowEvent)

	assert.NotNil(t, droppedEvent)
	assert.IsType(t, TestEventA{}, droppedEvent)

	// Clean up sub
	sub.Unsubscribe()
}

type CustomTestEvent struct {
	Envelope
	Data string
}

type OtherTestEvent struct {
	Envelope
	ID int
}

func TestEmpirical_Event_SubscribeTyped_ValueAndPointer(t *testing.T) {
	b := New()
	defer b.Close()

	// Subscribe to value type CustomTestEvent
	subVal := b.SubscribeTyped[CustomTestEvent](10)
	defer subVal.Unsubscribe()

	// Publish value
	b.Publish(CustomTestEvent{Data: "value-payload"})

	select {
	case ev := <-subVal.C():
		if ev.Data != "value-payload" {
			t.Fatalf("expected value-payload, got %q", ev.Data)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for typed value event")
	}

	// Publish pointer to CustomTestEvent — SubscribeTyped should unwrap pointer to value
	b.Publish(&CustomTestEvent{Data: "pointer-payload"})

	select {
	case ev := <-subVal.C():
		if ev.Data != "pointer-payload" {
			t.Fatalf("expected pointer-payload, got %q", ev.Data)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for unwrapped pointer event")
	}

	// Publish irrelevant event: should NOT be delivered
	b.Publish(OtherTestEvent{ID: 123})
	select {
	case ev := <-subVal.C():
		t.Fatalf("unexpected event received on typed sub: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// Expected: no event delivered
	}
}

func TestEmpirical_Event_EventsSeq_EarlyTerminationAndNoLeak(t *testing.T) {
	b := New()
	defer b.Close()

	ctx := t.Context()

	beforeG := runtime.NumGoroutine()

	// 1. Raw Subscription EventsSeq early break
	subRaw := b.Subscribe(CustomTestEvent{})
	go func() {
		for i := range 10 {
			b.Publish(CustomTestEvent{Data: "seq"})
			time.Sleep(2 * time.Millisecond)
			_ = i
		}
	}()

	rawCount := 0
	for ev := range subRaw.EventsSeq(ctx) {
		_ = ev
		rawCount++
		if rawCount == 2 {
			break // early termination
		}
	}
	if rawCount != 2 {
		t.Fatalf("expected rawCount 2, got %d", rawCount)
	}
	subRaw.Unsubscribe()

	// 2. TypedSubscription EventsSeq early break
	subTyped := b.SubscribeTyped[CustomTestEvent]()
	go func() {
		for range 10 {
			b.Publish(CustomTestEvent{Data: "typed-seq"})
			time.Sleep(2 * time.Millisecond)
		}
	}()

	typedCount := 0
	for ev := range subTyped.EventsSeq(ctx) {
		_ = ev
		typedCount++
		if typedCount == 3 {
			break // early termination
		}
	}
	if typedCount != 3 {
		t.Fatalf("expected typedCount 3, got %d", typedCount)
	}
	subTyped.Unsubscribe()

	// Verify goroutines after unsubscribe
	time.Sleep(50 * time.Millisecond)
	runtime.GC()
	afterG := runtime.NumGoroutine()
	if afterG > beforeG+2 {
		t.Fatalf("goroutines leaked in event iteration: before=%d, after=%d", beforeG, afterG)
	}
}

func TestEmpirical_Event_SubscribeTyped_ConcurrentPublishAndUnsubscribe(t *testing.T) {
	b := New()
	defer b.Close()

	sub := b.SubscribeTyped[CustomTestEvent](100)

	var wg sync.WaitGroup
	// 5 concurrent publishers
	for p := range 5 {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			for i := range 50 {
				b.Publish(CustomTestEvent{Data: "msg"})
				if i%10 == 0 {
					runtime.Gosched()
				}
			}
		}(p)
	}

	// 1 consumer goroutine
	consumed := 0
	wg.Go(func() {
		for range sub.C() {
			consumed++
			if consumed >= 50 {
				sub.Unsubscribe()
				return
			}
		}
	})

	wg.Wait()
}

type utilTestEvent struct {
	Envelope
	Data string
}

func TestSubscribeTo_DispatchesTypedEvent(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx := t.Context()

	received := make(chan string, 1)

	subCancel := SubscribeTo[*utilTestEvent](ctx, bus, func(ev *utilTestEvent) {
		received <- ev.Data
	})
	defer subCancel()

	bus.Publish(&utilTestEvent{Data: "hello_event"})

	select {
	case data := <-received:
		assert.Equal(t, "hello_event", data)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event delivery")
	}
}

func TestSubscribeTo_CancelStopsDelivery(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx := t.Context()

	var count atomic.Int32

	subCancel := SubscribeTo[*utilTestEvent](ctx, bus, func(ev *utilTestEvent) {
		count.Add(1)
	})

	bus.Publish(&utilTestEvent{Data: "first"})

	require.Eventually(t, func() bool {
		return count.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)

	probeReceived := make(chan struct{}, 1)

	probeCancel := SubscribeTo[*utilTestEvent](ctx, bus, func(ev *utilTestEvent) {
		if ev.Data == "second" {
			select {
			case probeReceived <- struct{}{}:
			default:
			}
		}
	})
	defer probeCancel()

	subCancel()

	bus.Publish(&utilTestEvent{Data: "second"})

	select {
	case <-probeReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for probe event to confirm dispatch")
	}

	assert.Equal(t, int32(1), count.Load())
}

func TestSubscribeTo_ContextCancellation(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())

	var count atomic.Int32
	SubscribeTo[*utilTestEvent](ctx, bus, func(ev *utilTestEvent) {
		count.Add(1)
	})

	bus.Publish(&utilTestEvent{Data: "e1"})
	require.Eventually(t, func() bool {
		return count.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)

	probeReceived := make(chan struct{}, 1)

	probeCancel := SubscribeTo[*utilTestEvent](context.Background(), bus, func(ev *utilTestEvent) {
		if ev.Data == "e2" {
			select {
			case probeReceived <- struct{}{}:
			default:
			}
		}
	})
	defer probeCancel()

	cancel()

	bus.Publish(&utilTestEvent{Data: "e2"})

	select {
	case <-probeReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for probe event to confirm dispatch")
	}

	assert.Equal(t, int32(1), count.Load())
}

func TestSubscribeTo_NilSafety(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Nil bus
	c1 := SubscribeTo[*utilTestEvent](ctx, nil, func(ev *utilTestEvent) {})
	assert.NotNil(t, c1)
	c1() // no panic

	// Nil handler
	bus := New()
	defer bus.Close()

	c2 := SubscribeTo[*utilTestEvent](ctx, bus, nil)
	assert.NotNil(t, c2)
	c2() // no panic
}

func TestSubscribeTo_IdempotentCancel(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx := context.Background()
	cancel := SubscribeTo[*utilTestEvent](ctx, bus, func(ev *utilTestEvent) {})

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			cancel()
		})
	}

	wg.Wait()
}

func TestSubscribeSeq_StreamsEvents(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	seq := SubscribeSeq[*utilTestEvent](ctx, bus)

	go func() {
		bus.Publish(&utilTestEvent{Data: "msg1"})
		bus.Publish(&utilTestEvent{Data: "msg2"})
		bus.Publish(&utilTestEvent{Data: "msg3"})
	}()

	var received []string
	for ev := range seq {
		received = append(received, ev.Data)
		if len(received) == 3 {
			break
		}
	}

	assert.Equal(t, []string{"msg1", "msg2", "msg3"}, received)
}

func TestSubscribeSeq_EarlyBreakCleansUp(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	seq := SubscribeSeq[*utilTestEvent](ctx, bus)

	go func() {
		bus.Publish(&utilTestEvent{Data: "event_1"})
		bus.Publish(&utilTestEvent{Data: "event_2"})
	}()

	count := 0
	for range seq {
		count++
		break // Break immediately after 1 event
	}

	assert.Equal(t, 1, count)
}

func TestSubscribeSeq_ContextCancellationStopsLoop(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	count := 0

	go func() {
		for range SubscribeSeq[*utilTestEvent](ctx, bus) {
			count++
		}

		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for iterator loop to stop on context cancellation")
	}

	assert.Equal(t, 0, count)
}

func TestSubscribeSeq_BusClosedStopsLoop(t *testing.T) {
	t.Parallel()

	bus := New()

	done := make(chan struct{})
	count := 0

	go func() {
		for range SubscribeSeq[*utilTestEvent](context.Background(), bus) {
			count++
		}

		close(done)
	}()

	_ = bus.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for iterator loop to stop on bus close")
	}

	assert.Equal(t, 0, count)
}

func TestSubscribeSeq_NilBus(t *testing.T) {
	t.Parallel()

	count := 0
	for range SubscribeSeq[*utilTestEvent](context.Background(), nil) {
		count++
	}

	assert.Equal(t, 0, count)
}

func TestSubscribe_ParallelAndRaceDetector(t *testing.T) {
	t.Parallel()

	bus := New()
	defer bus.Close()

	ctx := t.Context()

	var receivedCount atomic.Int64

	numSubs := 5
	cancels := make([]func(), numSubs)

	for i := range numSubs {
		cancels[i] = SubscribeTo[*utilTestEvent](ctx, bus, func(ev *utilTestEvent) {
			receivedCount.Add(1)
		}, 1024)
	}

	var wg sync.WaitGroup

	publishers := 10
	eventsPerPublisher := 50

	for p := range publishers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for range eventsPerPublisher {
				bus.Publish(&utilTestEvent{Data: "data"})
			}
		}(p)
	}

	wg.Wait()

	expectedTotal := int64(numSubs * publishers * eventsPerPublisher)
	require.Eventually(t, func() bool {
		return receivedCount.Load() == expectedTotal
	}, 3*time.Second, 10*time.Millisecond)

	for _, c := range cancels {
		c()
	}
}

func TestEnvelope_Methods(t *testing.T) {
	var be Envelope
	be.isEvent()
	ctx := be.Context()
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}
	customCtx := t.Context()
	be.SetContext(customCtx)
	if be.Context() != customCtx {
		t.Fatal("expected set context")
	}
}

func TestStandaloneEventsSeq(t *testing.T) {
	b := New()
	defer b.Close()

	sub := b.SubscribeBuffer(10, CustomTestEvent{})
	defer sub.Unsubscribe()

	b.Publish(CustomTestEvent{Data: "hello"})

	ctx := t.Context()
	count := 0
	for ev := range sub.EventsSeq(ctx) {
		if ev.(CustomTestEvent).Data == "hello" {
			count++
			break
		}
	}
	if count != 1 {
		t.Fatalf("expected 1 event, got %d", count)
	}
}

func TestEventsSeq_ContextCancellation(t *testing.T) {
	b := New()
	defer b.Close()

	sub := b.SubscribeBuffer(10, CustomTestEvent{})
	defer sub.Unsubscribe()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // canceled upfront

	for range sub.EventsSeq(ctx) {
		t.Fatal("should not yield")
	}

	typedSub := b.SubscribeTyped[CustomTestEvent](10)
	defer typedSub.Unsubscribe()

	for range typedSub.EventsSeq(ctx) {
		t.Fatal("should not yield")
	}
}

func TestEvent_CoverageEdgeCases(t *testing.T) {
	// 1. Envelope methods
	var env Envelope
	env.SetContext(context.Background())
	assert.NotNil(t, env.Context())
	env.isEvent()

	// 2. SubscribeTyped on closed bus
	closedBus := New()
	closedBus.Close()
	tsClosed := closedBus.SubscribeTyped[CustomTestEvent](10)
	require.NotNil(t, tsClosed)
	tsClosed.Unsubscribe()

	// 3. SubscribeTyped for value type when pointer event is published
	bus := New()
	defer bus.Close()
	subVal := bus.SubscribeTyped[CustomTestEvent](10)
	defer subVal.Unsubscribe()

	bus.Publish(&CustomTestEvent{Data: "ptr-to-val"})
	select {
	case ev := <-subVal.C():
		assert.Equal(t, "ptr-to-val", ev.Data)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}

	// 4. EventsSeq early exit
	subRaw := bus.Subscribe(CustomTestEvent{})
	defer subRaw.Unsubscribe()
	bus.Publish(CustomTestEvent{Data: "early-raw"})
	for range subRaw.EventsSeq(context.Background()) {
		break
	}

	subTyped2 := bus.SubscribeTyped[CustomTestEvent]()
	defer subTyped2.Unsubscribe()
	bus.Publish(CustomTestEvent{Data: "early-typed"})
	for range subTyped2.EventsSeq(context.Background()) {
		break
	}

	// 5. SubscribeTo nil edge cases
	cancelNil := SubscribeTo[*utilTestEvent](nil, nil, nil)
	cancelNil()

	cancelNilH := SubscribeTo[*utilTestEvent](context.Background(), bus, nil)
	cancelNilH()

	cancelNilCtx := SubscribeTo[*utilTestEvent](nil, bus, func(ev *utilTestEvent) {})
	cancelNilCtx()

	// 6. SubscribeSeq nil and pre-canceled ctx
	go func() {
		time.Sleep(10 * time.Millisecond)
		bus.Publish(&utilTestEvent{})
	}()
	for range SubscribeSeq[*utilTestEvent](nil, bus) {
		break
	}

	canceledCtx, cancelC := context.WithCancel(context.Background())
	cancelC()
	for range SubscribeSeq[*utilTestEvent](canceledCtx, bus) {
		break
	}
}
