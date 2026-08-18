// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package semaphore

import (
	"context"
	"sync"
)

// Semaphore represents a resizable counting semaphore that coordinates access
// to a shared pool of limited concurrent slots.
//
// A Semaphore is safe for concurrent use by multiple goroutines.
type Semaphore struct {
	mu      sync.Mutex
	limit   int
	active  int
	waiters []chan struct{}
}

// New creates and returns a new [Semaphore] instance with the given initial limit.
//
// If the initialLimit is negative, it is coerced to 0 to prevent state corruption.
func New(initialLimit int) *Semaphore {
	if initialLimit < 0 {
		initialLimit = 0
	}

	return &Semaphore{
		limit: initialLimit,
	}
}

// Acquire attempts to acquire a slot from the semaphore, blocking if the limit is exceeded.
//
// It returns nil on successful acquisition. It returns [context.Canceled] or
// [context.DeadlineExceeded] if the context expires before a slot becomes available.
func (s *Semaphore) Acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	if s.active < s.limit {
		s.active++
		s.mu.Unlock()
		return nil
	}

	ch := make(chan struct{})
	s.waiters = append(s.waiters, ch)
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		s.mu.Lock()

		found := false
		for i, w := range s.waiters {
			if w == ch {
				s.waiters = append(s.waiters[:i], s.waiters[i+1:]...)
				found = true
				break
			}
		}

		s.mu.Unlock()

		if !found {
			// The slot was already allocated to us by notifyWaiters before we processed
			// the cancellation. We must return nil to ensure this slot is eventually released.
			return nil
		}

		return ctx.Err()

	case <-ch:
		return nil
	}
}

// Release releases an acquired slot back to the semaphore and wakes up the next
// waiting goroutine if available.
func (s *Semaphore) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.active--
	s.notifyWaiters()
}

// Resize dynamically adjusts the maximum allowed limit of the semaphore on the fly.
//
// If the newLimit is negative, it is coerced to 0. Resize automatically wakes
// up as many waiting goroutines as the new capacity allows.
func (s *Semaphore) Resize(newLimit int) {
	if newLimit < 0 {
		newLimit = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.limit = newLimit
	s.notifyWaiters()
}

// notifyWaiters wakes up blocked waiters as long as free slots are available.
//
// The caller must hold s.mu.
func (s *Semaphore) notifyWaiters() {
	for s.active < s.limit && len(s.waiters) > 0 {
		ch := s.waiters[0]
		s.waiters = s.waiters[1:]
		s.active++

		close(ch)
	}
}
