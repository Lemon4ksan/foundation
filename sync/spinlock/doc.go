// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package spinlock provides a lightweight CAS-based spin lock.
//
// It is designed for very short critical sections where the cost of goroutine
// parking and unparking exceeds the cost of spinning. To prevent severe CPU
// cache line invalidation and bus lock contention across multiple processor cores,
// it implements the Test and Test-and-Set (TTAS) optimization.
//
// # Compared to the standard library
//
// Stdlib counterpart: sync.Mutex
//
// Rejected compromise: sync.Mutex incurs goroutine parking and context switching overhead.
//
// Accepted cost: Burning CPU cycles while waiting, strictly for very short critical sections.
//
// Allocations: zero
package spinlock
