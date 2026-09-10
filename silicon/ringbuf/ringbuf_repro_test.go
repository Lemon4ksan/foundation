// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuf

import (
	"testing"
	"time"
)

func TestRepro_NewRingBuffer_NegativeCapacityHangs(t *testing.T) {
	done := make(chan struct{})

	go func() {
		// Passing negative capacity should not hang in an infinite loop
		rb := NewRingBuffer[int](-1)
		if rb.capacity < 2 {
			t.Errorf("expected capacity >= 2, got %d", rb.capacity)
		}
		close(done)
	}()

	select {
	case <-done:
		// Succeeded without hanging
	case <-time.After(1 * time.Second):
		t.Fatal("CRITICAL: NewRingBuffer(-1) hung in infinite bitshift loop!")
	}
}
