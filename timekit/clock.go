// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"iter"
	"time"

	"github.com/lemon4ksan/foundation/silicon/clock"
)

// CoarseNow returns the approximate current [time.Time] with millisecond resolution, retaining monotonic clock readings without triggering OS clock system calls.
//
//go:inline
func CoarseNow() time.Time {
	return clock.CoarseTime()
}

// CoarseUnix returns the approximate Unix timestamp in seconds.
//
//go:inline
func CoarseUnix() int64 {
	return clock.CoarseNowUnix()
}

// CoarseUnixMilli returns the approximate Unix timestamp in milliseconds.
//
//go:inline
func CoarseUnixMilli() int64 {
	return clock.CoarseNowNano() / 1e6
}

// CoarseUnixNano returns the approximate Unix timestamp in nanoseconds.
//
//go:inline
func CoarseUnixNano() int64 {
	return clock.CoarseNowNano()
}

// RangeSeq returns an iterator yielding timestamps from start up to end (inclusive if exact match), advancing by step on each iteration.
//
//go:inline
func RangeSeq(start, end time.Time, step time.Duration) iter.Seq[time.Time] {
	return func(yield func(time.Time) bool) {
		if step <= 0 {
			return
		}
		for curr := start; !curr.After(end); curr = curr.Add(step) {
			if !yield(curr) {
				return
			}
		}
	}
}
