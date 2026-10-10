// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lazy provides a thread-safe lazy initializer with reset support.
//
// The [Value] container caches the result of its initialization function
// after the first call. All subsequent [Value.Get] calls return the cached
// result without re-executing the function. [Value.Reset] clears the cache,
// allowing the next Get to re-initialize. If the initialization function
// returns an error, it is cached and returned on all subsequent Get calls
// until [Value.Reset] is called, allowing callers to retry initialization.
//
// # Compared to the standard library
//
// Stdlib counterpart: sync.Once
//
// Rejected compromise: sync.Once does not support resetting and re-initialization.
//
// Accepted cost: extra 8 bytes of memory overhead for the lockless atomic pointer.
//
// Allocations: zero
package lazy
