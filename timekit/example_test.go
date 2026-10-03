// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit_test

import (
	"fmt"
	"time"

	"github.com/lemon4ksan/foundation/timekit"
)

func ExampleCoarseNow() {
	// Read current time with single atomic load
	now := timekit.CoarseNow()
	_ = now
}

func ExampleAppendHTTPDate() {
	var buf [32]byte
	// Generates HTTP-date with 0 allocations
	res := timekit.AppendHTTPDate(buf[:0], time.Date(2026, 8, 26, 18, 30, 0, 0, time.UTC))
	fmt.Println(string(res))
	// Output: Wed, 26 Aug 2026 18:30:00 GMT
}

func ExampleStartStopwatch() {
	sw := timekit.StartStopwatch()
	// Execute critical section...
	elapsed := sw.Elapsed()
	_ = elapsed
}
