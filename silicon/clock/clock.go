// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package clock provides high-performance coarse atomic time and hardware RDTSC cycle measurement utilities.
//
// Architectural Concept & Mechanical Sympathy:
// System calls to time.Now() invoke kernel vDSO routines, which incur branch prediction overhead,
// registers saving, and CPU pipeline stalls on extreme-throughput hot paths (1M-2M+ RPS).
//
// This package provides two complimentary hardware-sympathetic clocks:
//  1. Coarse Clock: A dedicated 1ms background ticker atomically updating an [atomic.Int64],
//     reducing time and timeout checks to a sub-nanosecond L1 cache load (0 B/op).
//  2. Hardware RDTSC: Direct single-instruction CPU cycle counter (RDTSC on x86_64 / CNTVCT_EL0 on ARM64)
//     for sub-nanosecond latency benchmarking and timer deltas.
package clock

import (
	"sync/atomic"
	"time"
)

var (
	coarseUnixNano atomic.Int64
	coarseUnixSec  atomic.Int64

	// tscTicksPerNano stores the calibrated number of RDTSC ticks per nanosecond scaled by 1024.
	tscTicksPerNano atomic.Uint64
)

func init() {
	now := time.Now()
	coarseUnixNano.Store(now.UnixNano())
	coarseUnixSec.Store(now.Unix())

	// Calibrate RDTSC against OS monotonic clock
	calibrateRDTSC()

	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for t := range ticker.C {
			coarseUnixNano.Store(t.UnixNano())
			coarseUnixSec.Store(t.Unix())
		}
	}()
}

func calibrateRDTSC() {
	startTSC := rdtsc()
	startTime := time.Now()
	time.Sleep(5 * time.Millisecond)
	endTSC := rdtsc()
	elapsedNanos := time.Since(startTime).Nanoseconds()

	if elapsedNanos > 0 && endTSC > startTSC {
		ticks := endTSC - startTSC
		// Scale by 1024 for fixed-point integer arithmetic precision
		scaled := (ticks << 10) / uint64(elapsedNanos)
		if scaled > 0 {
			tscTicksPerNano.Store(scaled)
			return
		}
	}

	// Fallback to 2.5 GHz default if calibration failed
	tscTicksPerNano.Store(2560) // 2.5 * 1024
}

// RDTSC returns the current raw CPU hardware Time-Stamp Counter (1-2 CPU cycles).
// On x86_64 this executes the RDTSC instruction; on ARM64 it reads CNTVCT_EL0.
//
//go:inline
//go:nosplit
func RDTSC() uint64 {
	return rdtsc()
}

// Cycles returns the current hardware CPU cycle count (alias for [RDTSC]).
//
//go:inline
//go:nosplit
func Cycles() uint64 {
	return rdtsc()
}

// CyclesToDuration converts a delta in CPU cycles into a [time.Duration] using calibrated fixed-point arithmetic.
func CyclesToDuration(cycles uint64) time.Duration {
	scale := tscTicksPerNano.Load()
	if scale == 0 {
		return 0
	}
	nanos := (cycles << 10) / scale
	return time.Duration(nanos)
}

// ElapsedCycles returns the elapsed duration since the recorded start CPU cycle timestamp.
func ElapsedCycles(startCycles uint64) time.Duration {
	now := rdtsc()
	if now <= startCycles {
		return 0
	}
	return CyclesToDuration(now - startCycles)
}

// CoarseNowNano returns the current Unix timestamp in nanoseconds with ~1ms coarse resolution.
// It executes a single sub-nanosecond atomic load from L1 cache instead of invoking vDSO time.Now().
//
//go:inline
//go:nosplit
func CoarseNowNano() int64 {
	return coarseUnixNano.Load()
}

// CoarseNowUnix returns the current Unix timestamp in seconds with ~1ms coarse resolution.
//
//go:inline
//go:nosplit
func CoarseNowUnix() int64 {
	return coarseUnixSec.Load()
}

// CoarseTime returns the current coarse timestamp as [time.Time] with ~1ms resolution.
//
//go:inline
//go:nosplit
func CoarseTime() time.Time {
	return time.Unix(0, coarseUnixNano.Load())
}

// CoarseSince returns the elapsed duration since t calculated against the coarse clock.
func CoarseSince(t time.Time) time.Duration {
	return time.Duration(coarseUnixNano.Load() - t.UnixNano())
}
