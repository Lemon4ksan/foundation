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

// SpinLock implements an adaptive Silicon Spinlock utilizing a Test and Test-and-Set (TTAS)
// busy-wait loop with hardware CPU PAUSE / YIELD instructions.
//
// Zero value of SpinLock is ready to use and represents an unlocked state.
// Aligned to 64 bytes to eliminate False Sharing across CPU L1/L2 cache lines.
type SpinLock struct {
	state uint32
	_     [60]byte
}

// Lock acquires the lock.
//
// It first attempts an immediate Compare-And-Swap (CAS) operation. If contested,
// it enters a TTAS loop using cheap read-only atomic loads and hardware CPU PAUSE instructions,
// falling back to [runtime.Gosched] under prolonged contention to prevent starvation.
func (s *SpinLock) Lock() {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		return
	}

	s.lockSlow()
}

func (s *SpinLock) lockSlow() {
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

// Unlock releases the lock.
// Panics if the spinlock is not currently locked.
func (s *SpinLock) Unlock() {
	if atomic.SwapUint32(&s.state, 0) != 1 {
		panic("spinlock: unlock of unlocked spinlock")
	}
}

// TryLock attempts to acquire the lock without spinning or waiting.
//
// It returns true if the lock was successfully acquired, and false otherwise.
func (s *SpinLock) TryLock() bool {
	return atomic.CompareAndSwapUint32(&s.state, 0, 1)
}
