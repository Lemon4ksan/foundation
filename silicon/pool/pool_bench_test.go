// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pool_test

import (
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/pool"
)

func BenchmarkPerCorePool_GetPut_Sequential(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	// Pre-fill shard
	storage.Put(&[64]byte{})

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		obj := storage.Get()
		storage.Put(obj)
	}
}

func BenchmarkPerCorePool_GetPut_Parallel(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	// Pre-fill all shards to capacity
	for range 256 {
		storage.Put(&[64]byte{})
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := storage.Get()
			if obj != nil {
				storage.Put(obj)
			}
		}
	})
}

func BenchmarkPerCorePool_HighConcurrency_Contention(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	for range 256 {
		storage.Put(&[64]byte{})
	}

	b.SetParallelism(64)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := storage.Get()
			if obj != nil {
				storage.Put(obj)
			}
		}
	})
}

func BenchmarkPerCorePool_Contention_Burst128(b *testing.B) {
	storage := pool.NewPerCorePool(func() *[64]byte {
		var buf [64]byte
		return &buf
	})

	for range 256 {
		storage.Put(&[64]byte{})
	}

	b.SetParallelism(128)
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			obj := storage.Get()
			if obj != nil {
				storage.Put(obj)
			}
		}
	})
}

func BenchmarkRequestArena_AllocReset(b *testing.B) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = arena.Alloc(128)
		arena.Reset()
	}
}

func BenchmarkRequestArena_MultipleAllocReset(b *testing.B) {
	arena := pool.AcquireRequestArena()
	defer pool.ReleaseRequestArena(arena)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = arena.Alloc(64)
		_ = arena.Alloc(128)
		_ = arena.Alloc(256)
		_ = arena.Alloc(512)
		arena.Reset()
	}
}

func BenchmarkRequestArena_GetRelease(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		arena := pool.AcquireRequestArena()
		pool.ReleaseRequestArena(arena)
	}
}

func BenchmarkTimerPool_AcquireRelease(b *testing.B) {
	// Pre-warm pool
	t := pool.AcquireTimer(time.Hour)
	pool.ReleaseTimer(t)

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		tm := pool.AcquireTimer(time.Hour)
		pool.ReleaseTimer(tm)
	}
}

func BenchmarkTimerPool_AcquireRelease_Parallel(b *testing.B) {
	// Pre-warm pool
	for range 64 {
		t := pool.AcquireTimer(time.Hour)
		pool.ReleaseTimer(t)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tm := pool.AcquireTimer(time.Hour)
			pool.ReleaseTimer(tm)
		}
	})
}
