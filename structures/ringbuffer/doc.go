// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ringbuffer implements a growable, circular FIFO queue that reuses backing storage.
//
// RingBuffer avoids the constant allocations typical in queue implementations by wrapping around a contiguous backing array. When full, it seamlessly grows its internal capacity. No locking is performed internally, allowing lightweight usage in single-threaded loops.
//
// Key operations include [RingBuffer.PushBack], [RingBuffer.PopFront], and [RingBuffer.PeekFront].
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: growable circular buffer
//
// Rejected compromise: Standard channel primitives have a fixed size and require locking/channel overhead even for single-goroutine queues.
//
// Accepted cost: Lack of thread safety requires external synchronization if used concurrently.
//
// Allocations: amortized
package ringbuffer
