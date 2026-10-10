// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package semaphore

import (
	"context"
	"sync"
)

// Semaphore represents a dynamically resizable counting semaphore that coordinates access
// to a shared pool of concurrent slots.
//
// The zero value represents a limit of 0 and is ready for use.
// All exported methods are safe for concurrent use.
type Semaphore struct {
	mu      sync.Mutex
	limit   int
	active  int
	waiters []chan struct{}
}

// New creates a [Semaphore] with the specified initial limit.
//
// Complexity: O(1) time.
func New(initialLimit int) *Semaphore {
	if initialLimit < 0 {
		initialLimit = 0
	}

	return &Semaphore{
		limit: initialLimit,
	}
}

// Acquire requests a slot from the semaphore, blocking until one is available or the context is canceled.
//
// Complexity: O(1) time if uncontended. Allocates memory for a channel if blocking is required.
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
			// the cancellation. Release the allocated slot back to the pool and propagate ctx.Err().
			s.Release()

			return ctx.Err()
		}

		return ctx.Err()

	case <-ch:
		return nil
	}
}

// Release returns an acquired slot to the semaphore, waking the next blocked goroutine if one exists.
//
// Complexity: O(1) time.
func (s *Semaphore) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active > 0 {
		s.active--
		s.notifyWaiters()
	}
}

// Resize updates the maximum allowed limit of the semaphore.
//
// Complexity: O(W) time where W is the number of waiting goroutines woken by the increased limit.
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
