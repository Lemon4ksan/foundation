// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"time"
)

// Stopwatch provides lightweight monotonic latency measurement.
type Stopwatch struct {
	start time.Time
}

// StartStopwatch creates and immediately starts a new [Stopwatch].
func StartStopwatch() Stopwatch {
	return Stopwatch{
		start: time.Now(),
	}
}

// Elapsed returns the duration elapsed since the stopwatch was started.
func (s Stopwatch) Elapsed() time.Duration {
	return time.Since(s.start)
}

// ElapsedMicro returns elapsed time in microseconds.
func (s Stopwatch) ElapsedMicro() int64 {
	return time.Since(s.start).Microseconds()
}

// ElapsedNano returns elapsed time in nanoseconds.
func (s Stopwatch) ElapsedNano() int64 {
	return time.Since(s.start).Nanoseconds()
}

// Reset restarts the stopwatch from the current moment.
func (s *Stopwatch) Reset() {
	s.start = time.Now()
}
