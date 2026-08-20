// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package keylock provides a generic, thread-safe, striped/key-based locking
// mechanism (also known as a lock stripe or key-based mutex).
//
// It allows separate goroutines to concurrently lock and work with different
// keys, preventing global application lock bottlenecks. Unlike a simple
// map[K]*sync.Mutex, keys are automatically cleaned up via reference
// counting when no goroutines are waiting for or holding them.
//
// # Architecture
//
// The [KeyMutex] stores per-key lock state in an internal map protected by a global
// [sync.Mutex]. Each key entry ([refCounter]) tracks a reference count of waiting
// and holding goroutines. When the reference count drops to zero, the entry is
// atomically removed from the map, preventing memory leaks in long-running
// applications with highly dynamic or infinite key sets.
//
// To prevent deadlocks, the global map lock is always released before any
// goroutine blocks on an individual key-level mutex.
//
// # Error Handling & Safety
//
// [KeyMutex.Unlock] panics if the key is not currently locked or does not exist.
// This catches double-unlock and unlock-without-lock bugs at the point of failure
// rather than silently corrupting application state.
//
// # Example
//
//	package main
//
//	import (
//	    "fmt"
//	    "sync"
//	    "time"
//
//	    "github.com/lemon4ksan/foundation/sync/keylock"
//	)
//
//	func main() {
//	    kl := keylock.New[string]()
//	    var wg sync.WaitGroup
//
//	    wg.Add(2)
//
//	    go func() {
//	        defer wg.Done()
//	        kl.Lock("user-alice")
//	        defer kl.Unlock("user-alice")
//	        fmt.Println("Locked alice")
//	        time.Sleep(50 * time.Millisecond)
//	    }()
//
//	    go func() {
//	        defer wg.Done()
//	        kl.Lock("user-bob")
//	        defer kl.Unlock("user-bob")
//	        fmt.Println("Locked bob - no wait for alice")
//	        time.Sleep(50 * time.Millisecond)
//	    }()
//
//	    wg.Wait()
//	}
package keylock
