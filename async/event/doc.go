// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package event implements a thread-safe, non-blocking, type-based event bus
// for asynchronous in-process communication.
//
// It provides a decoupled architecture where independent components can publish
// and subscribe to typed events without direct compile-time dependencies.
// Events are routed dynamically based on their [reflect.Type]. To ensure
// consistent routing, both pointer and value representations of the same struct
// resolve to the same underlying type.
//
// The bus prioritizes system availability and low latency. Event delivery is
// strictly non-blocking: if a subscriber's channel buffer becomes full,
// incoming events are silently dropped for that subscriber to prevent slow
// consumers from creating backpressure on publishers.
//
// Safe publishing acquires a read lock to guarantee no channel is closed during send,
// preventing panics. Unsubscribing acquires a write lock and uses double-close checks.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: an asynchronous pub/sub backbone
//
// Rejected compromise: typical simple pub/sub blocks the publisher if a subscriber is slow.
//
// Accepted cost: events are dropped if consumers cannot keep up.
//
// Allocations: amortized
package event
