// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spinlock

import (
	"runtime"
	"sync/atomic"
)

// SpinLock implements a fast spinlock utilizing a Test and Test-and-Set (TTAS)
// busy-wait loop.
//
// Zero value of SpinLock is ready to use and represents an unlocked state.
type SpinLock struct {
	state uint32
}

// Lock acquires the lock.
//
// It first attempts an immediate Compare-And-Swap (CAS) operation. If the lock
// is contested, it enters a spinning read-only loop, yielding the processor via
// [runtime.Gosched] to prevent processor starvation while avoiding constant
// cache invalidation writes.
func (s *SpinLock) Lock() {
	for {
		// Fast Path: Attempt to acquire the lock immediately.
		if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
			return
		}

		// Slow Path (TTAS): Spin on a cheap read-only atomic load
		// to prevent cache line invalidation across CPU cores.
		for atomic.LoadUint32(&s.state) == 1 {
			runtime.Gosched()
		}
	}
}

// Unlock releases the lock.
//
// Unlock panics if the spinlock is not currently locked.
func (s *SpinLock) Unlock() {
	if !atomic.CompareAndSwapUint32(&s.state, 1, 0) {
		panic("spinlock: unlock of unlocked spinlock")
	}
}

// TryLock attempts to acquire the lock without spinning or waiting.
//
// It returns true if the lock was successfully acquired, and false otherwise.
func (s *SpinLock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&s.state, 0, 1)
}
