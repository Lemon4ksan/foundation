// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event_test

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/event"
)

type CustomTestEvent struct {
	event.BaseEvent
	Data string
}

type OtherTestEvent struct {
	event.BaseEvent
	ID int
}

func TestEmpirical_Event_SubscribeTyped_ValueAndPointer(t *testing.T) {
	b := event.New()
	defer b.Close()

	// Subscribe to value type CustomTestEvent
	subVal := event.SubscribeTyped[CustomTestEvent](b, 10)
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
	b := event.New()
	defer b.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	subTyped := event.SubscribeTyped[CustomTestEvent](b)
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
	b := event.New()
	defer b.Close()

	sub := event.SubscribeTyped[CustomTestEvent](b, 100)

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
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range sub.C() {
			consumed++
			if consumed >= 50 {
				sub.Unsubscribe()
				return
			}
		}
	}()

	wg.Wait()
}
