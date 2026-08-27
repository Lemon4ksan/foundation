// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package randkit_test

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/randkit"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestGlobalRand_Basic(t *testing.T) {
	t.Parallel()

	u32 := randkit.Uint32()
	_ = u32

	u64 := randkit.Uint64()
	_ = u64

	assert.Equal(t, uint32(0), randkit.Uint32n(0))
	assert.Equal(t, uint64(0), randkit.Uint64n(0))
	assert.Equal(t, 0, randkit.Intn(0))
	assert.Equal(t, int64(0), randkit.Int64n(0))

	for i := 0; i < 1000; i++ {
		u32n := randkit.Uint32n(100)
		assert.Less(t, u32n, uint32(100))

		u64n := randkit.Uint64n(1000)
		assert.Less(t, u64n, uint64(1000))

		in := randkit.Intn(50)
		assert.True(t, in >= 0 && in < 50)

		i64n := randkit.Int64n(500)
		assert.True(t, i64n >= 0 && i64n < 500)

		f64 := randkit.Float64()
		assert.True(t, f64 >= 0.0 && f64 < 1.0)

		f32 := randkit.Float32()
		assert.True(t, f32 >= 0.0 && f32 < 1.0)
	}

	assert.Equal(t, time.Duration(0), randkit.Jitter(0))
	jitter := randkit.Jitter(500 * time.Millisecond)
	assert.True(t, jitter >= 0 && jitter < 500*time.Millisecond)

	assert.Equal(t, 100*time.Millisecond, randkit.JitterRange(100*time.Millisecond, 50*time.Millisecond))
	jr := randkit.JitterRange(100*time.Millisecond, 500*time.Millisecond)
	assert.True(t, jr >= 100*time.Millisecond && jr < 500*time.Millisecond)
}

func TestGlobalRand_PermShuffleReadString(t *testing.T) {
	t.Parallel()

	assert.Empty(t, randkit.Perm(0))
	p := randkit.Perm(10)
	require.Len(t, p, 10)
	seen := make(map[int]bool)
	for _, v := range p {
		assert.True(t, v >= 0 && v < 10)
		seen[v] = true
	}
	assert.Len(t, seen, 10)

	orig := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	shuffled := make([]int, len(orig))
	copy(shuffled, orig)
	randkit.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	assert.Len(t, shuffled, 10)

	buf := make([]byte, 32)
	n, err := randkit.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 32, n)
	assert.NotEqual(t, make([]byte, 32), buf)

	// Odd length Read
	bufOdd := make([]byte, 13)
	nOdd, errOdd := randkit.Read(bufOdd)
	require.NoError(t, errOdd)
	require.Equal(t, 13, nOdd)

	assert.Empty(t, randkit.String(0, "abc"))
	assert.Empty(t, randkit.String(10, ""))
	s := randkit.String(16, "abc")
	require.Len(t, s, 16)
	for _, c := range s {
		assert.Contains(t, "abc", string(c))
	}

	an := randkit.AlphaNumeric(24)
	require.Len(t, an, 24)
}

func TestRNG_Local(t *testing.T) {
	t.Parallel()

	r1 := randkit.NewRNG()
	require.NotNil(t, r1)

	r2 := randkit.NewRNGWithSeed(42)
	require.NotNil(t, r2)
	v1 := r2.Uint64()

	r3 := randkit.NewRNGWithSeed(42)
	v2 := r3.Uint64()
	assert.Equal(t, v1, v2, "Identical seed must yield identical sequence")

	var r4 randkit.RNG // zero value
	assert.NotZero(t, r4.Uint64())

	assert.Equal(t, uint32(0), r2.Uint32n(0))
	assert.Equal(t, uint64(0), r2.Uint64n(0))
	assert.Equal(t, 0, r2.Intn(0))
	assert.Equal(t, int64(0), r2.Int64n(0))

	for i := 0; i < 500; i++ {
		assert.Less(t, r2.Uint32n(100), uint32(100))
		assert.Less(t, r2.Uint64n(1000), uint64(1000))
		in := r2.Intn(50)
		assert.True(t, in >= 0 && in < 50)
		i64n := r2.Int64n(500)
		assert.True(t, i64n >= 0 && i64n < 500)
		f64 := r2.Float64()
		assert.True(t, f64 >= 0.0 && f64 < 1.0)
		f32 := r2.Float32()
		assert.True(t, f32 >= 0.0 && f32 < 1.0)
	}

	assert.Empty(t, r2.Perm(0))
	perm := r2.Perm(10)
	require.Len(t, perm, 10)

	buf := make([]byte, 21)
	n, err := r2.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 21, n)

	arr := []int{1, 2, 3, 4, 5}
	r2.Shuffle(len(arr), func(i, j int) {
		arr[i], arr[j] = arr[j], arr[i]
	})
	assert.Len(t, arr, 5)
}

func TestConcurrent_Safety(t *testing.T) {
	t.Parallel()

	const goroutines = 16
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				_ = randkit.Uint32()
				_ = randkit.Uint32n(1000)
				_ = randkit.Uint64()
				_ = randkit.Uint64n(10000)
				_ = randkit.Intn(500)
				_ = randkit.Int64n(5000)
				_ = randkit.Float64()
				_ = randkit.Jitter(time.Second)
				_ = randkit.UUIDv7()
			}
		}()
	}

	wg.Wait()
}

var benchSink uint64

func BenchmarkGlobalUint32n(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var s uint32
		for pb.Next() {
			s += randkit.Uint32n(1e6)
		}
		atomic.AddUint64(&benchSink, uint64(s))
	})
}

func BenchmarkGlobalUint64n(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		var s uint64
		for pb.Next() {
			s += randkit.Uint64n(1e6)
		}
		atomic.AddUint64(&benchSink, s)
	})
}

func BenchmarkRNGUint64n(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		r := randkit.NewRNG()
		var s uint64
		for pb.Next() {
			s += r.Uint64n(1e6)
		}
		atomic.AddUint64(&benchSink, s)
	})
}

func BenchmarkRNGUint32n(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		r := randkit.NewRNG()
		var s uint32
		for pb.Next() {
			s += r.Uint32n(1e6)
		}
		atomic.AddUint64(&benchSink, uint64(s))
	})
}
