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
// It executes in O(1) time.
// It performs zero allocations.
// It is safe for concurrent use.
type Semaphore struct {
	mu      sync.Mutex
	limit   int
	active  int
	waiters []chan struct{}
}

// New creates a [Semaphore] with the given initial limit.
// It executes in O(1) time and allocates memory for the tracking structures.
// It is safe for concurrent use.
func New(initialLimit int) *Semaphore {
	if initialLimit < 0 {
		initialLimit = 0
	}

	return &Semaphore{
		limit: initialLimit,
	}
}

// Acquire attempts to acquire a slot from the semaphore, blocking if the limit is exceeded.
// It executes in O(1) time if uncontended, allocating a channel otherwise.
// It is safe for concurrent use.
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

// Release releases an acquired slot back to the semaphore and wakes up the next waiting goroutine if available.
// It executes in O(1) time and performs zero allocations.
// It is safe for concurrent use.
func (s *Semaphore) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.active > 0 {
		s.active--
		s.notifyWaiters()
	}
}

// Resize dynamically adjusts the maximum allowed limit of the semaphore on the fly.
// It executes in O(W) time where W is the number of woken waiters.
// It performs zero allocations.
// It is safe for concurrent use.
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
