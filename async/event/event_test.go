// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testkit/assert"
)

type (
	TestEventA    struct{ BaseEvent }
	TestEventB    struct{ BaseEvent }
	TestEventData struct {
		BaseEvent
		Payload string
	}
)

func TestBus_SubscribeAndPublish(t *testing.T) {
	BaseEvent{}.isEvent() // safe no-op marker invoke

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

func TestBus_BaseEventAndClosedSub(t *testing.T) {
	var ev BaseEvent
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
	for i := 0; i < 128; i++ {
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
