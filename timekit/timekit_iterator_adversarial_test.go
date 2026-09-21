// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit_test

import (
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/timekit"
)

func TestEmpirical_Timekit_RangeSeq_EarlyTermination(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Hour)
	step := 1 * time.Hour

	// 1. Break immediately after 0 iterations (first item returns false)
	count := 0
	for tm := range timekit.RangeSeq(start, end, step) {
		_ = tm
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 3 items
	count = 0
	for tm := range timekit.RangeSeq(start, end, step) {
		_ = tm
		count++
		if count == 3 {
			break
		}
	}
	if count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}

	// 3. Full traversal (0 to 10 inclusive = 11 items)
	count = 0
	for tm := range timekit.RangeSeq(start, end, step) {
		_ = tm
		count++
	}
	if count != 11 {
		t.Fatalf("expected 11 items, got %d", count)
	}

	// 4. Break mid-iteration
	mid := count / 2
	visited := 0
	for range timekit.RangeSeq(start, end, step) {
		visited++
		if visited == mid {
			break
		}
	}
	if visited != mid {
		t.Fatalf("expected visited %d, got %d", mid, visited)
	}
}

func TestEmpirical_Timekit_RangeSeq_EdgeCases(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Hour)

	// Invalid step <= 0
	for range timekit.RangeSeq(start, end, 0) {
		t.Fatal("expected no iterations when step == 0")
	}
	for range timekit.RangeSeq(start, end, -time.Hour) {
		t.Fatal("expected no iterations when step < 0")
	}

	// Inverted range (start > end)
	for range timekit.RangeSeq(end, start, time.Hour) {
		t.Fatal("expected no iterations when start > end")
	}

	// Exact single point (start == end)
	singleCount := 0
	for tm := range timekit.RangeSeq(start, start, time.Hour) {
		if !tm.Equal(start) {
			t.Fatalf("expected %v, got %v", start, tm)
		}
		singleCount++
	}
	if singleCount != 1 {
		t.Fatalf("expected exactly 1 item for start == end, got %d", singleCount)
	}
}

func TestEmpirical_Timekit_RangeSeq_Concurrent(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(100 * time.Minute)
	step := time.Minute

	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := 0
			for range timekit.RangeSeq(start, end, step) {
				c++
			}
			if c != 101 {
				t.Errorf("expected 101 iterations, got %d", c)
			}
		}()
	}
	wg.Wait()
}

func BenchmarkRangeSeq_ZeroAlloc(b *testing.B) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(100 * time.Minute)
	step := time.Minute

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for tm := range timekit.RangeSeq(start, end, step) {
			_ = tm
		}
	}
}
