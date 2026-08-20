// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package spinlock provides a lightweight CAS-based spin lock.
//
// It is designed for very short critical sections where the cost of goroutine
// parking and unparking (as in [sync.Mutex]) exceeds the cost of spinning.
//
// # Optimization: Test and Test-and-Set (TTAS)
//
// To prevent severe CPU cache line invalidation and bus lock contention across
// multiple processor cores (Cache Thrashing), [SpinLock.Lock] implements the
// Test and Test-and-Set (TTAS) optimization.
//
// Instead of continuously executing expensive bus-locking write operations ([sync/atomic.CompareAndSwapUint32])
// in a loop, waiting goroutines spin on a cheap, read-only [sync/atomic.LoadUint32] operation.
// This allows the CPU cores to keep the cache line in "Shared" state, avoiding
// bus thrashing until the lock is actually released.
//
// # Panic Safety
//
// [SpinLock.Unlock] strictly validates the locked state before releasing.
// Attempting to unlock an already unlocked spinlock will trigger a panic immediately,
// catching double-unlock and unlock-without-lock bugs at the point of failure.
//
// # When to Use
//
// Use SpinLock when the critical section is very short (nanoseconds to low
// microseconds), contention is low, and the lock is rarely held for long.
// For longer critical sections or high contention, prefer [sync.Mutex] which
// parks the goroutine instead of burning CPU cycles.
//
// # Example
//
//	package main
//
//	import (
//	    "fmt"
//
//	    "github.com/lemon4ksan/foundation/sync/spinlock"
//	)
//
//	func main() {
//	    var mu spinlock.SpinLock
//
//	    mu.Lock()
//	    fmt.Println("locked")
//	    mu.Unlock()
//
//	    if mu.TryLock() {
//	        fmt.Println("try-lock succeeded")
//	        mu.Unlock()
//	    }
//	}
package spinlock
