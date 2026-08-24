// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package clock_test

import (
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/clock"
)

func TestCoarseClock(t *testing.T) {
	nano := clock.CoarseNowNano()
	if nano <= 0 {
		t.Fatalf("expected non-zero coarse nano, got %d", nano)
	}

	sec := clock.CoarseNowUnix()
	if sec <= 0 {
		t.Fatalf("expected non-zero coarse sec, got %d", sec)
	}

	ct := clock.CoarseTime()
	if ct.IsZero() {
		t.Fatal("expected non-zero coarse time")
	}

	time.Sleep(5 * time.Millisecond)

	newNano := clock.CoarseNowNano()
	if newNano <= nano {
		t.Fatalf("expected coarse nano to advance, before=%d after=%d", nano, newNano)
	}
}

func TestRDTSC(t *testing.T) {
	c1 := clock.RDTSC()
	if c1 == 0 {
		t.Fatal("expected non-zero RDTSC cycles")
	}

	time.Sleep(2 * time.Millisecond)

	c2 := clock.RDTSC()
	if c2 <= c1 {
		t.Fatalf("expected RDTSC cycles to advance: c1=%d, c2=%d", c1, c2)
	}

	dur := clock.ElapsedCycles(c1)
	if dur < time.Millisecond || dur > 50*time.Millisecond {
		t.Fatalf("expected elapsed duration around ~2ms, got %v", dur)
	}
}

func BenchmarkRDTSC(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = clock.RDTSC()
	}
}

func BenchmarkCoarseClock_Nano(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = clock.CoarseNowNano()
	}
}

func BenchmarkCoarseClock_Time(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = clock.CoarseTime()
	}
}

func BenchmarkStandardTimeNow(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		_ = time.Now()
	}
}
