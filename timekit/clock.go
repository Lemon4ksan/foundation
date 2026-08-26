// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"sync/atomic"
	"time"
)

var (
	cachedUnixNano atomic.Int64
)

func init() {
	cachedUnixNano.Store(time.Now().UnixNano())

	// Background ticker updates atomic timestamp with high frequency (~1ms)
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for now := range ticker.C {
			cachedUnixNano.Store(now.UnixNano())
		}
	}()
}

// CoarseNow returns the approximate current [time.Time] with millisecond resolution
// without triggering OS clock system calls.
func CoarseNow() time.Time {
	return time.Unix(0, cachedUnixNano.Load())
}

// CoarseUnix returns the approximate Unix timestamp in seconds.
func CoarseUnix() int64 {
	return cachedUnixNano.Load() / int64(time.Second)
}

// CoarseUnixMilli returns the approximate Unix timestamp in milliseconds.
func CoarseUnixMilli() int64 {
	return cachedUnixNano.Load() / int64(time.Millisecond)
}

// CoarseUnixNano returns the approximate Unix timestamp in nanoseconds.
func CoarseUnixNano() int64 {
	return cachedUnixNano.Load()
}
