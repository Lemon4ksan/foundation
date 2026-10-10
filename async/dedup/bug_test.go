package dedup

import (
	"context"
	"testing"
	"time"
)

func TestReproduce_OldWorkerDeletesNewCall(t *testing.T) {
	g := &Group[string, string]{}
	
	ctx1, cancel1 := context.WithCancel(context.Background())
	
	worker1Started := make(chan struct{})
	worker1Finish := make(chan struct{})
	
	// Thread 1 initiates first call
	go func() {
		g.Do(ctx1, "key", func(ctx context.Context) (string, error) {
			worker1Started <- struct{}{}
			<-worker1Finish // wait until we allow worker 1 to finish
			return "worker1", nil
		})
	}()
	
	<-worker1Started
	
	// Cancel Thread 1. This deletes g.calls["key"]
	cancel1()
	
	// Give it a tiny bit of time to process cancellation
	time.Sleep(50 * time.Millisecond)
	
	worker2Started := make(chan struct{})
	worker2Finish := make(chan struct{})
	
	// Thread 2 initiates second call
	go func() {
		g.Do(context.Background(), "key", func(ctx context.Context) (string, error) {
			worker2Started <- struct{}{}
			<-worker2Finish
			return "worker2", nil
		})
	}()
	
	<-worker2Started
	
	// Now worker 1 finishes! It will incorrectly delete worker 2's call from g.calls!
	worker1Finish <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	
	// Thread 3 initiates third call. It SHOULD join worker 2, but because of the bug, it will start worker 3!
	worker3Started := make(chan struct{})
	worker3Finish := make(chan struct{})
	go func() {
		g.Do(context.Background(), "key", func(ctx context.Context) (string, error) {
			worker3Started <- struct{}{}
			<-worker3Finish
			return "worker3", nil
		})
	}()
	
	// If worker 3 started, deduplication was broken!
	select {
	case <-worker3Started:
		worker2Finish <- struct{}{}
		worker3Finish <- struct{}{}
		t.Fatalf("BUG CONFIRMED: worker 3 started, meaning deduplication was broken!")
	case <-time.After(200 * time.Millisecond):
		// PASSED! It joined worker 2.
		worker2Finish <- struct{}{}
	}
}

func reproduce_PanicSwallowedOnInitiatorCancel(t *testing.T) {
	g := &Group[string, string]{}
	ctx, cancel := context.WithCancel(context.Background())
	
	started := make(chan struct{})
	waiterBlock := make(chan struct{})
	
	// Thread 2 (waiter)
	go func() {
		<-started
		_, err := g.Do(context.Background(), "key2", nil)
		if err != ErrWorkerPanicked {
			t.Errorf("expected ErrWorkerPanicked, got %v", err)
		}
	}()
	
	// Thread 1 (initiator)
	go func() {
		g.Do(ctx, "key2", func(c context.Context) (string, error) {
			started <- struct{}{}
			<-waiterBlock // Wait until we cancel the initiator
			panic("boom")
		})
	}()
	
	<-started // wait for worker to start
	cancel() // cancel initiator
	
	// Give it a tiny bit of time to process cancellation
	time.Sleep(50 * time.Millisecond)
	
	waiterBlock <- struct{}{} // let worker panic
	time.Sleep(50 * time.Millisecond)
	
	// If we reach here without the whole test process crashing with an unrecovered panic,
	// the panic was swallowed!
}
