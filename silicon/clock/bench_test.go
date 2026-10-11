package clock

import (
	"math"
	"runtime"
	"sync"
	"testing"
)

func TestRDTSC_MonotonicProgression(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	prev := rdtsc()
	for i := range 100_000 {
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

	for g := range goroutines {
		go func(id int) {
			defer wg.Done()
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()

			prev := rdtsc()
			for i := range iters {
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

	// x86 invariant TSC runs at core frequencies [1.0, 6.0] GHz.
	// ARM64 CNTVCT_EL0 runs at fixed reference frequencies [0.01, 1.0] GHz (e.g. 24 MHz on Apple Silicon).
	minFreq, maxFreq := 1.0, 6.0
	if runtime.GOARCH == "arm64" {
		minFreq, maxFreq = 0.01, 1.0
	}

	if freqGHz < minFreq || freqGHz > maxFreq {
		t.Fatalf(
			"Calibrated frequency %.4f GHz is outside realistic range [%.2f GHz, %.2f GHz]",
			freqGHz, minFreq, maxFreq,
		)
	}

	// 2. Perform repeated calibrations with 100ms spin-wait and verify stability across runs
	const runs = 3
	samples := make([]float64, 0, runs+1)
	samples = append(samples, freqGHz)

	for i := range runs {
		calibrateRDTSC()
		s := tscTicksPerNano.Load()
		sampleGHz := float64(s) / 1024.0
		t.Logf("Calibration run %d: %.4f GHz (scaled=%d)", i+1, sampleGHz, s)

		if sampleGHz < minFreq || sampleGHz > maxFreq {
			t.Fatalf(
				"Calibration run %d: %.4f GHz is outside realistic range [%.2f GHz, %.2f GHz]",
				i+1, sampleGHz, minFreq, maxFreq,
			)
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
	// Verify that serialized RDTSC guarantees non-negative elapsed duration
	for i := range 100_000 {
		start := rdtsc()
		dur := ElapsedCycles(start)
		if dur < 0 {
			t.Fatalf("iteration %d: expected non-negative elapsed duration, got %v", i, dur)
		}
		// On amd64, serialized LFENCE guarantees positive cycle progression
		if runtime.GOARCH == "amd64" && dur <= 0 {
			t.Fatalf("iteration %d: expected positive elapsed duration on amd64, got %v", i, dur)
		}
	}
}

func TestElapsedCycles_SmallWorkload(t *testing.T) {
	for i := range 10_000 {
		start := rdtsc()
		// Micro-workload exceeding hardware clock tick resolution (> 42ns on 24MHz ARM64)
		var x uint64
		for j := range 2000 {
			x += uint64(j)
		}
		dur := ElapsedCycles(start)
		if dur <= 0 {
			// Spin-wait briefly if sampled within the exact same timer tick
			for range 50 {
				if dur = ElapsedCycles(start); dur > 0 {
					break
				}
			}
			if dur <= 0 {
				t.Fatalf(
					"iteration %d: expected positive elapsed duration with workload, got %v (workload result: %d)",
					i, dur, x,
				)
			}
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
