// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"

	"github.com/lemon4ksan/foundation/async/event"
)

type testEvent struct {
	event.BaseEvent
	Data string
}

func TestSubscribeTo_DispatchesTypedEvent(t *testing.T) {
	t.Parallel()

	bus := event.New()
	defer bus.Close()

	ctx := t.Context()

	received := make(chan string, 1)

	subCancel := event.SubscribeTo[*testEvent](ctx, bus, func(ev *testEvent) {
		received <- ev.Data
	})
	defer subCancel()

	bus.Publish(&testEvent{Data: "hello_event"})

	select {
	case data := <-received:
		assert.Equal(t, "hello_event", data)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event delivery")
	}
}

func TestSubscribeTo_CancelStopsDelivery(t *testing.T) {
	t.Parallel()

	bus := event.New()
	defer bus.Close()

	ctx := t.Context()

	var count atomic.Int32

	subCancel := event.SubscribeTo[*testEvent](ctx, bus, func(ev *testEvent) {
		count.Add(1)
	})

	bus.Publish(&testEvent{Data: "first"})

	require.Eventually(t, func() bool {
		return count.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)

	probeReceived := make(chan struct{}, 1)

	probeCancel := event.SubscribeTo[*testEvent](ctx, bus, func(ev *testEvent) {
		if ev.Data == "second" {
			select {
			case probeReceived <- struct{}{}:
			default:
			}
		}
	})
	defer probeCancel()

	subCancel()

	bus.Publish(&testEvent{Data: "second"})

	select {
	case <-probeReceived:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for probe event to confirm dispatch")
	}

	assert.Equal(t, int32(1), count.Load())
}

func TestSubscribeTo_ContextCancellation(t *testing.T) {
	t.Parallel()

	bus := event.New()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())

	var count atomic.Int32
	event.SubscribeTo[*testEvent](ctx, bus, func(ev *testEvent) {
		count.Add(1)
	})

	bus.Publish(&testEvent{Data: "e1"})
	require.Eventually(t, func() bool {
		return count.Load() == 1
	}, 2*time.Second, 10*time.Millisecond)

	probeReceived := make(chan struct{}, 1)

	probeCancel := event.SubscribeTo[*testEvent](context.Background(), bus, func(ev *testEvent) {
		if ev.Data == "e2" {
			select {
			case probeReceived <- struct{}{}:
			default:
			}
		}
	})
	defer probeCancel()

	cancel()

	bus.Publish(&testEvent{Data: "e2"})

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
	c1 := event.SubscribeTo[*testEvent](ctx, nil, func(ev *testEvent) {})
	assert.NotNil(t, c1)
	c1() // no panic

	// Nil handler
	bus := event.New()
	defer bus.Close()

	c2 := event.SubscribeTo[*testEvent](ctx, bus, nil)
	assert.NotNil(t, c2)
	c2() // no panic
}

func TestSubscribeTo_IdempotentCancel(t *testing.T) {
	t.Parallel()

	bus := event.New()
	defer bus.Close()

	ctx := context.Background()
	cancel := event.SubscribeTo[*testEvent](ctx, bus, func(ev *testEvent) {})

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

	bus := event.New()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	seq := event.SubscribeSeq[*testEvent](ctx, bus)

	go func() {
		bus.Publish(&testEvent{Data: "msg1"})
		bus.Publish(&testEvent{Data: "msg2"})
		bus.Publish(&testEvent{Data: "msg3"})
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

	bus := event.New()
	defer bus.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	seq := event.SubscribeSeq[*testEvent](ctx, bus)

	go func() {
		bus.Publish(&testEvent{Data: "event_1"})
		bus.Publish(&testEvent{Data: "event_2"})
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

	bus := event.New()
	defer bus.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	count := 0

	go func() {
		for range event.SubscribeSeq[*testEvent](ctx, bus) {
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

	bus := event.New()

	done := make(chan struct{})
	count := 0

	go func() {
		for range event.SubscribeSeq[*testEvent](context.Background(), bus) {
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
	for range event.SubscribeSeq[*testEvent](context.Background(), nil) {
		count++
	}

	assert.Equal(t, 0, count)
}

func TestSubscribe_ParallelAndRaceDetector(t *testing.T) {
	t.Parallel()

	bus := event.New()
	defer bus.Close()

	ctx := t.Context()

	var receivedCount atomic.Int64

	numSubs := 5
	cancels := make([]func(), numSubs)

	for i := range numSubs {
		cancels[i] = event.SubscribeTo[*testEvent](ctx, bus, func(ev *testEvent) {
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
				bus.Publish(&testEvent{Data: "data"})
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
