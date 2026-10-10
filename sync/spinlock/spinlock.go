// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package spinlock implements an adaptive Silicon Spinlock utilizing TTAS (Test and Test-and-Set),
// hardware CPU PAUSE micro-spinning, exponential backoff, and single-cycle Store-Release.
package spinlock

import (
	"runtime"
	"sync/atomic"
)

// Lock represents an adaptive Silicon Spinlock utilizing a Test and Test-and-Set (TTAS) busy-wait loop with hardware CPU PAUSE or YIELD instructions.
//
// The zero value is empty and ready to use.
type Lock struct {
	state uint32
	_     [60]byte
}

// Lock acquires the spinlock.
//
// It executes in amortized O(1) time and does not allocate memory.
// It is safe for concurrent use.
func (s *Lock) Lock() {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		return
	}

	s.lockSlow()
}

func (s *Lock) lockSlow() {
	backoff := 1

	for {
		// Slow Path (TTAS): Spin on a cheap read-only atomic load (Shared MESI state)
		// to prevent cache line invalidation across CPU cores.
		for atomic.LoadUint32(&s.state) == 1 {
			if backoff < 32 {
				procYield(uint32(backoff * 4))
				backoff <<= 1
			} else {
				runtime.Gosched()
			}
		}

		if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
			return
		}
	}
}

// Unlock releases the spinlock.
//
// Panics if the spinlock is not currently locked.
// It executes in amortized O(1) time and does not allocate memory.
// It is safe for concurrent use.
func (s *Lock) Unlock() {
	if atomic.SwapUint32(&s.state, 0) != 1 {
		panic("spinlock: unlock of unlocked spinlock")
	}
}

// TryLock attempts to acquire the lock without spinning or waiting.
//
// It executes in amortized O(1) time and does not allocate memory.
// It is safe for concurrent use.
func (s *Lock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&s.state, 0, 1)
}
