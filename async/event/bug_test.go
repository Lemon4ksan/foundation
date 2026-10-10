package event

import (
	"testing"
	"time"
)

func TestReproduce_EmptySubscriptionLeakOnClose(t *testing.T) {
	b := New()
	sub := b.Subscribe() // No event types provided!
	
	// Close the bus. This should close ALL active subscriptions.
	b.Close()
	
	// The channel should be closed, so this should not block.
	select {
	case <-sub.C():
		// Success!
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("BUG: Empty subscription channel was NOT closed by b.Close()!")
	}
}
