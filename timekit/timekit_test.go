// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"sync"
	"testing"
	"time"
)

func TestCoarseClock(t *testing.T) {
	now := CoarseNow()
	if now.IsZero() {
		t.Fatalf("expected non-zero coarse time")
	}

	unix := CoarseUnix()
	if unix <= 0 {
		t.Fatalf("expected positive unix timestamp, got %d", unix)
	}

	time.Sleep(5 * time.Millisecond)
	now2 := CoarseNow()
	if now2.Before(now) {
		t.Fatalf("monotonic invariant broken: %v < %v", now2, now)
	}
}

func TestHTTPDate_FormatAndParse(t *testing.T) {
	tm := time.Date(1994, time.November, 6, 8, 49, 37, 0, time.UTC)
	formatted := FormatHTTPDate(tm)

	expected := "Sun, 06 Nov 1994 08:49:37 GMT"
	if formatted != expected {
		t.Fatalf("expected %q, got %q", expected, formatted)
	}

	parsed, err := ParseHTTPDate(formatted)
	if err != nil {
		t.Fatalf("ParseHTTPDate failed: %v", err)
	}

	if !parsed.Equal(tm) {
		t.Fatalf("expected %v, got %v", tm, parsed)
	}
}

func TestRFC3339_Format(t *testing.T) {
	tm := time.Date(2026, time.August, 26, 16, 30, 45, 0, time.UTC)
	formatted := FormatRFC3339(tm)

	expected := "2026-08-26T16:30:45Z"
	if formatted != expected {
		t.Fatalf("expected %q, got %q", expected, formatted)
	}
}

func TestStopwatch(t *testing.T) {
	sw := StartStopwatch()
	time.Sleep(5 * time.Millisecond)

	elapsed := sw.Elapsed()
	if elapsed < 4*time.Millisecond {
		t.Fatalf("expected >= 4ms elapsed, got %v", elapsed)
	}
}

func BenchmarkAppendHTTPDate(b *testing.B) {
	tm := time.Now().UTC()
	buf := make([]byte, 0, HTTPDateLength)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = AppendHTTPDate(buf[:0], tm)
	}
}

func BenchmarkStdTimeFormat_RFC1123(b *testing.B) {
	tm := time.Now().UTC()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = tm.Format(time.RFC1123)
	}
}

func TestEmpirical_Timekit_RangeSeq_EarlyTermination(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * time.Hour)
	step := 1 * time.Hour

	// 1. Break immediately after 0 iterations (first item returns false)
	count := 0
	for tm := range RangeSeq(start, end, step) {
		_ = tm
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 3 items
	count = 0
	for tm := range RangeSeq(start, end, step) {
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
	for tm := range RangeSeq(start, end, step) {
		_ = tm
		count++
	}
	if count != 11 {
		t.Fatalf("expected 11 items, got %d", count)
	}

	// 4. Break mid-iteration
	mid := count / 2
	visited := 0
	for range RangeSeq(start, end, step) {
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
	for range RangeSeq(start, end, 0) {
		t.Fatal("expected no iterations when step == 0")
	}
	for range RangeSeq(start, end, -time.Hour) {
		t.Fatal("expected no iterations when step < 0")
	}

	// Inverted range (start > end)
	for range RangeSeq(end, start, time.Hour) {
		t.Fatal("expected no iterations when start > end")
	}

	// Exact single point (start == end)
	singleCount := 0
	for tm := range RangeSeq(start, start, time.Hour) {
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
		wg.Go(func() {
			c := 0
			for range RangeSeq(start, end, step) {
				c++
			}
			if c != 101 {
				t.Errorf("expected 101 iterations, got %d", c)
			}
		})
	}
	wg.Wait()
}

func BenchmarkRangeSeq_ZeroAlloc(b *testing.B) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(100 * time.Minute)
	step := time.Minute

	b.ReportAllocs()

	for b.Loop() {
		for tm := range RangeSeq(start, end, step) {
			_ = tm
		}
	}
}

func TestReproduce_ISO8601NegativeYearBug(t *testing.T) {
	// ISO8601 formatting breaks on negative years due to modulo operations on negative numbers.
	tm := time.Date(-1, time.January, 1, 0, 0, 0, 0, time.UTC)
	formatted := FormatRFC3339(tm)
	
	if formatted != "0001-01-01T00:00:00Z" {
		t.Fatalf("expected fixed behavior '0001-01-01T00:00:00Z', got %q", formatted)
	}
}
