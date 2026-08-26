// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
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

	for i := 0; i < b.N; i++ {
		_ = AppendHTTPDate(buf[:0], tm)
	}
}

func BenchmarkStdTimeFormat_RFC1123(b *testing.B) {
	tm := time.Now().UTC()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = tm.Format(time.RFC1123)
	}
}
