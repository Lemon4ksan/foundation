// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylock

import (
	"sync"
	"testing"
	"time"
)

func TestRepro_KeyMutex_UnlockUnheldKey(t *testing.T) {
	km := New[string]()

	km.Lock("key1")

	var wg sync.WaitGroup
	wg.Add(1)
	blocked := make(chan struct{})

	go func() {
		defer wg.Done()
		close(blocked)
		km.Lock("key1")
		km.Unlock("key1")
	}()

	<-blocked
	// Allow background goroutine to enter km.Lock and wait on ref.mu.Lock()
	time.Sleep(20 * time.Millisecond)

	// Legitimate unlock
	km.Unlock("key1")

	// At this point, key1 is registered because the background goroutine is waking up or active.
	// However, if the first caller erroneously calls Unlock("key1") again,
	// it must panic with "foundation/keylock: unlock of unlocked key" instead of corrupting ref.count or crashing with sync.Mutex panic.
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when unlocking unheld key")
		}
		expected := "foundation/keylock: unlock of unlocked key"
		if r != expected {
			t.Fatalf("expected panic %q, got %v", expected, r)
		}
		wg.Wait()
	}()

	km.Unlock("key1")
}
