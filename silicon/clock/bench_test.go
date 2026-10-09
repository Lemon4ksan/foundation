package clock

import (
	"math"
	"runtime"
	"sync"
	"testing"
)

// Merged from doc_test.go

func TestRDTSC_MonotonicProgression(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	prev := rdtsc()
	for i := 0; i < 100_000; i++ {
		curr := rdtsc()
		if curr < prev {
			t.Fatalf("RDTSC monotonic progression violated: prev=%d, curr=%d at iteration %d", prev, curr, i)
		}
		prev = curr
	}
}

func TestRDTSC_MonotonicProgression_Concurrent(t *testing.T) {
	const goroutines = 8
	const iters = 20_000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			prev := rdtsc()
			for i := 0; i < iters; i++ {
				curr := rdtsc()
				if curr < prev {
					t.Errorf("goroutine %d: RDTSC decreased: prev=%d, curr=%d (iter %d)", id, prev, curr, i)
					return
				}
				prev = curr
			}
		}(g)
	}

	wg.Wait()
}

func TestRDTSC_CalibrationStability(t *testing.T) {
	// 1. Verify initial calibration performed in init()
	scaled := tscTicksPerNano.Load()
	if scaled == 0 {
		t.Fatal("tscTicksPerNano is 0 after init")
	}

	freqGHz := float64(scaled) / 1024.0
	t.Logf("Initial calibrated frequency: %.4f GHz (scaled=%d)", freqGHz, scaled)

	if freqGHz < 1.0 || freqGHz > 6.0 {
		t.Fatalf("Calibrated frequency %.4f GHz is outside realistic range [1.0 GHz, 6.0 GHz]", freqGHz)
	}

	// 2. Perform repeated calibrations with 100ms spin-wait and verify stability across runs
	const runs = 3
	samples := make([]float64, 0, runs+1)
	samples = append(samples, freqGHz)

	for i := 0; i < runs; i++ {
		calibrateRDTSC()
		s := tscTicksPerNano.Load()
		sampleGHz := float64(s) / 1024.0
		t.Logf("Calibration run %d: %.4f GHz (scaled=%d)", i+1, sampleGHz, s)

		if sampleGHz < 1.0 || sampleGHz > 6.0 {
			t.Fatalf("Calibration run %d: %.4f GHz is outside realistic range [1.0 GHz, 6.0 GHz]", i+1, sampleGHz)
		}
		samples = append(samples, sampleGHz)
	}

	// 3. Check stability: variance across 100ms spin-wait calibration runs should be tight (< 5%)
	var sum float64
	for _, s := range samples {
		sum += s
	}
	avg := sum / float64(len(samples))
	for i, s := range samples {
		pctDiff := math.Abs(s-avg) / avg * 100.0
		if pctDiff > 5.0 {
			t.Fatalf(
				"Calibration sample %d (%.4f GHz) deviated %.2f%% from avg (%.4f GHz), exceeds 5%% threshold",
				i, s, pctDiff, avg,
			)
		}
	}
}

func TestElapsedCycles_Positive(t *testing.T) {
	// Verify that serialized RDTSC via LFENCE guarantees positive elapsed cycle measurement
	for i := 0; i < 100_000; i++ {
		start := rdtsc()
		dur := ElapsedCycles(start)
		if dur <= 0 {
			t.Fatalf("iteration %d: expected positive elapsed duration, got %v", i, dur)
		}
	}
}

func TestElapsedCycles_SmallWorkload(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		start := rdtsc()
		// Micro-workload: 10 nanosleep or simple arithmetic
		var x uint64
		for j := 0; j < 100; j++ {
			x += uint64(j)
		}
		dur := ElapsedCycles(start)
		if dur <= 0 {
			t.Fatalf(
				"iteration %d: expected positive elapsed duration with workload, got %v (workload result: %d)",
				i, dur, x,
			)
		}
	}
}

func TestElapsedCycles_BoundaryEdgeCases(t *testing.T) {
	// Future start cycles timestamp should return 0 duration safely
	if d := ElapsedCycles(^uint64(0)); d != 0 {
		t.Fatalf("expected 0 for future start cycles, got %v", d)
	}

	if d := CyclesToDuration(0); d != 0 {
		t.Fatalf("expected 0 for 0 cycles, got %v", d)
	}
}
