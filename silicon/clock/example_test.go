// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clock_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/clock"
)

func ExampleCoarseTime() {
	// Retrieve cached time in background reducing timestamp reads to a single atomic load
	t := clock.CoarseTime()

	if !t.IsZero() {
		fmt.Println("Cached time retrieved successfully")
	}

	// Output:
	// Cached time retrieved successfully
}

func ExampleRDTSC() {
	// Read serialized hardware time-stamp counter (LFENCE + RDTSC)
	c := clock.RDTSC()
	if c > 0 {
		fmt.Println("RDTSC cycle count retrieved")
	}

	// Output:
	// RDTSC cycle count retrieved
}

func ExampleElapsedCycles() {
	// Measure elapsed duration using calibrated cycle counting
	start := clock.RDTSC()
	elapsed := clock.ElapsedCycles(start)
	if elapsed >= 0 {
		fmt.Println("Elapsed cycles duration computed")
	}

	// Output:
	// Elapsed cycles duration computed
}
