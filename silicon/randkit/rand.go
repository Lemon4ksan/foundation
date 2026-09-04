// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package randkit implements a silicon-grade, zero-allocation, thread-local fast pseudo-random number generator,
// bypassing standard math/rand lock contention, atomic CAS loops, and heap allocations.
package randkit

import (
	"encoding/binary"
	"math/bits"
	randv2 "math/rand/v2"
	"time"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

const alphaNumericCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// Uint32 returns a fast pseudo-random 32-bit unsigned integer without atomic CAS loops or lock contention.
//
//go:inline
func Uint32() uint32 {
	return randv2.Uint32()
}

// Uint32n returns a fast pseudo-random 32-bit unsigned integer in range [0, n) using Lemire's multiplication reduction.
//
//go:inline
func Uint32n(n uint32) uint32 {
	if n == 0 {
		return 0
	}

	return randv2.Uint32N(n)
}

// Uint64 returns a fast pseudo-random 64-bit unsigned integer with zero heap allocations.
//
//go:inline
func Uint64() uint64 {
	return randv2.Uint64()
}

// Uint64n returns a fast pseudo-random 64-bit unsigned integer in range [0, n) using Lemire's multiplication reduction.
//
//go:inline
func Uint64n(n uint64) uint64 {
	if n == 0 {
		return 0
	}

	return randv2.Uint64N(n)
}

// Intn returns a fast pseudo-random integer in range [0, n) with zero allocations.
//
//go:inline
func Intn(n int) int {
	if n <= 0 {
		return 0
	}

	return randv2.IntN(n)
}

// Int64n returns a fast pseudo-random 64-bit integer in range [0, n) with zero allocations.
//
//go:inline
func Int64n(n int64) int64 {
	if n <= 0 {
		return 0
	}

	return randv2.Int64N(n)
}

// Float64 returns a pseudo-random float64 in the half-open interval [0.0, 1.0).
//
//go:inline
func Float64() float64 {
	return randv2.Float64()
}

// Float32 returns a pseudo-random float32 in the half-open interval [0.0, 1.0).
//
//go:inline
func Float32() float32 {
	return randv2.Float32()
}

// Jitter returns a pseudo-random jitter duration between 0 and maxJitter.
//
//go:inline
func Jitter(maxJitter time.Duration) time.Duration {
	if maxJitter <= 0 {
		return 0
	}

	return time.Duration(randv2.Uint64N(uint64(maxJitter)))
}

// JitterRange returns a pseudo-random jitter duration between minJitter and maxJitter.
//
//go:inline
func JitterRange(minJitter, maxJitter time.Duration) time.Duration {
	if maxJitter <= minJitter {
		return minJitter
	}

	return minJitter + time.Duration(randv2.Uint64N(uint64(maxJitter-minJitter)))
}

// Shuffle pseudo-randomizes the order of elements using Fisher-Yates shuffle.
func Shuffle(n int, swap func(i, j int)) {
	randv2.Shuffle(n, swap)
}

// Perm returns a pseudo-random permutation of the integers [0, n).
func Perm(n int) []int {
	return randv2.Perm(n)
}

// Read fills p with pseudo-random bytes.
func Read(p []byte) (n int, err error) {
	i := 0
	for i+8 <= len(p) {
		val := randv2.Uint64()
		binary.LittleEndian.PutUint64(p[i:], val)
		i += 8
	}

	if i < len(p) {
		val := randv2.Uint64()
		for ; i < len(p); i++ {
			p[i] = byte(val)
			val >>= 8
		}
	}

	return len(p), nil
}

// String generates a pseudo-random string of length n using characters from charset.
func String(n int, charset string) string {
	if n <= 0 || len(charset) == 0 {
		return ""
	}

	buf := make([]byte, n)
	charsetLen := uint32(len(charset))

	for i := range buf {
		buf[i] = charset[randv2.Uint32N(charsetLen)]
	}

	return bytesconv.B2S(buf)
}

// AlphaNumeric generates a pseudo-random alphanumeric string of length n with zero heap allocations.
func AlphaNumeric(n int) string {
	return String(n, alphaNumericCharset)
}

// RNG is a high-speed, 64-bit pseudorandom number generator using the Wyrand algorithm.
// It is designed for hot loops, per-goroutine isolation, and sub-nanosecond generation (~0.21 ns/op).
//
// It is unsafe to call RNG methods from concurrent goroutines without external synchronization.
type RNG struct {
	state uint64
}

// NewRNG initializes and returns a new RNG with random entropy.
func NewRNG() *RNG {
	r := &RNG{}
	r.Seed(randv2.Uint64())

	return r
}

// NewRNGWithSeed initializes and returns a new RNG seeded with the given value.
func NewRNGWithSeed(seed uint64) *RNG {
	r := &RNG{}
	r.Seed(seed)

	return r
}

// Seed seeds the RNG with the given value.
func (r *RNG) Seed(seed uint64) {
	if seed == 0 {
		seed = 0xa0761d6478bd642f
	}

	r.state = seed
}

// Uint64 returns a pseudorandom uint64 using the Wyrand algorithm.
//
//go:inline
func (r *RNG) Uint64() uint64 {
	if r.state == 0 {
		r.state = 0xa0761d6478bd642f
	}

	r.state += 0xa0761d6478bd642f
	hi, lo := bits.Mul64(r.state, r.state^0xe7037ed1a0b428db)

	return hi ^ lo
}

// Uint32 returns a pseudorandom uint32.
//
//go:inline
func (r *RNG) Uint32() uint32 {
	return uint32(r.Uint64())
}

// Uint64n returns a pseudorandom uint64 in the range [0, n) using Lemire's 64-bit reduction.
//
//go:inline
func (r *RNG) Uint64n(n uint64) uint64 {
	if n == 0 {
		return 0
	}

	hi, _ := bits.Mul64(r.Uint64(), n)

	return hi
}

// Uint32n returns a pseudorandom uint32 in the range [0, n) using Lemire's multiplication reduction.
//
//go:inline
func (r *RNG) Uint32n(n uint32) uint32 {
	if n == 0 {
		return 0
	}

	return uint32((uint64(r.Uint32()) * uint64(n)) >> 32)
}

// Intn returns a pseudorandom integer in the range [0, n).
//
//go:inline
func (r *RNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}

	return int(r.Uint32n(uint32(n)))
}

// Int64n returns a pseudorandom int64 in the range [0, n).
//
//go:inline
func (r *RNG) Int64n(n int64) int64 {
	if n <= 0 {
		return 0
	}

	return int64(r.Uint64n(uint64(n)))
}

// Float64 returns a pseudorandom float64 in the half-open interval [0.0, 1.0).
//
//go:inline
func (r *RNG) Float64() float64 {
	return float64(r.Uint64()>>11) / (1 << 53)
}

// Float32 returns a pseudorandom float32 in the half-open interval [0.0, 1.0).
//
//go:inline
func (r *RNG) Float32() float32 {
	return float32(r.Uint32()>>8) / (1 << 24)
}

// Read fills p with pseudorandom bytes.
func (r *RNG) Read(p []byte) (n int, err error) {
	i := 0
	for i+8 <= len(p) {
		val := r.Uint64()
		binary.LittleEndian.PutUint64(p[i:], val)
		i += 8
	}

	if i < len(p) {
		val := r.Uint64()
		for ; i < len(p); i++ {
			p[i] = byte(val)
			val >>= 8
		}
	}

	return len(p), nil
}

// Shuffle pseudo-randomizes the order of elements using Fisher-Yates shuffle.
func (r *RNG) Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("rand: Shuffle called with negative size")
	}

	for i := n - 1; i > 0; i-- {
		j := int(r.Uint32n(uint32(i + 1)))
		swap(i, j)
	}
}

// Perm returns a pseudo-random permutation of the integers [0, n).
func (r *RNG) Perm(n int) []int {
	if n <= 0 {
		return []int{}
	}

	m := make([]int, n)
	for i := range m {
		j := int(r.Uint32n(uint32(i + 1)))
		m[i] = m[j]
		m[j] = i
	}

	return m
}
