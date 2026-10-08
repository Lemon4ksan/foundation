// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matchfinder_test

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/brotli/matchfinder"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

// verifyMatchesOracle verifies LZ77 reconstruction fidelity and ensures all match offsets are valid.
func verifyMatchesOracle(t *testing.T, src []byte, matches []matchfinder.Match, finderName string) {
	t.Helper()

	if len(src) == 0 {
		return
	}

	reconstructed := make([]byte, 0, len(src))
	pos := 0

	for i, m := range matches {
		if m.Unmatched > 0 {
			require.Truef(t, pos+m.Unmatched <= len(src),
				"%s: match %d unmatched %d overflows src at pos %d", finderName, i, m.Unmatched, pos)
			reconstructed = append(reconstructed, src[pos:pos+m.Unmatched]...)
			pos += m.Unmatched
		}

		if m.Length > 0 {
			require.Truef(t, m.Distance > 0, "%s: match %d invalid non-positive distance %d", finderName, i, m.Distance)
			require.Truef(t, m.Distance <= len(reconstructed),
				"%s: match %d distance %d exceeds reconstructed length %d", finderName, i, m.Distance, len(reconstructed))

			start := len(reconstructed) - m.Distance
			for j := 0; j < m.Length; j++ {
				reconstructed = append(reconstructed, reconstructed[start+j])
			}
			pos += m.Length
		}
	}

	require.Equalf(t, len(src), pos, "%s: total matched bytes %d != input len %d", finderName, pos, len(src))
	require.Equalf(t, src, reconstructed, "%s: reconstructed bytes mismatch", finderName)
}

func allFinders() map[string]func() matchfinder.Finder {
	return map[string]func() matchfinder.Finder{
		"Bargain1":   func() matchfinder.Finder { return &matchfinder.Bargain1{} },
		"Bargain2":   func() matchfinder.Finder { return &matchfinder.Bargain2{} },
		"Bargain3":   func() matchfinder.Finder { return &matchfinder.Bargain3{} },
		"M0":         func() matchfinder.Finder { return &matchfinder.M0{} },
		"M0_Lazy":    func() matchfinder.Finder { return &matchfinder.M0{Lazy: true} },
		"M4":         func() matchfinder.Finder { return &matchfinder.M4{} },
		"M4_Chained": func() matchfinder.Finder { return &matchfinder.M4{ChainLength: 8, DistanceBitCost: 16} },
		"Pathfinder": func() matchfinder.Finder { return &matchfinder.Pathfinder{} },
		"Trio":       func() matchfinder.Finder { return &matchfinder.Trio{} },
		"ZDFast":     func() matchfinder.Finder { return &matchfinder.ZDFast{} },
		"ZFast":      func() matchfinder.Finder { return &matchfinder.ZFast{} },
		"ZM":         func() matchfinder.Finder { return &matchfinder.ZM{} },
		"NoFinder":   func() matchfinder.Finder { return matchfinder.NoFinder{} },
		"AutoReset":  func() matchfinder.Finder { return &matchfinder.AutoReset{Finder: &matchfinder.ZFast{}} },
	}
}

// TestFinder_DeterminismAndReset verifies that resetting a Finder makes it produce identical matches
// on repeat runs, and that a fresh instance matches a reset instance byte-for-byte.
func TestFinder_DeterminismAndReset(t *testing.T) {
	t.Parallel()

	testData := [][]byte{
		[]byte("Deterministic matchfinder verification string."),
		bytes.Repeat([]byte("0123456789ABCDEF"), 256),
		bytes.Repeat([]byte{0x00}, 2048),
		bytes.Repeat([]byte{0xAA, 0x55}, 1024),
	}

	for fname, factory := range allFinders() {
		fname, factory := fname, factory
		t.Run(fname, func(t *testing.T) {
			t.Parallel()

			for idx, data := range testData {
				idx, data := idx, data
				t.Run(fmt.Sprintf("Payload_%d", idx), func(t *testing.T) {
					// Instance A: run once
					instA := factory()
					instA.Reset()
					matchesA1 := instA.FindMatches(nil, data)

					// Verify oracle reconstruction
					verifyMatchesOracle(t, data, matchesA1, fname)

					// Instance A: reset and run again
					instA.Reset()
					matchesA2 := instA.FindMatches(nil, data)
					assert.Equalf(t, matchesA1, matchesA2, "%s: matches non-deterministic after Reset", fname)

					// Instance B: fresh instance
					instB := factory()
					matchesB := instB.FindMatches(nil, data)
					assert.Equalf(t, matchesA1, matchesB, "%s: fresh instance differs from reset instance", fname)
				})
			}
		})
	}
}

// TestFinder_NoFinderStrictContract verifies that NoFinder always emits exactly one Match containing all literals.
func TestFinder_NoFinderStrictContract(t *testing.T) {
	t.Parallel()

	nf := matchfinder.NoFinder{}
	nf.Reset() // verify no panic

	payloads := [][]byte{
		nil,
		{},
		[]byte("hello world"),
		bytes.Repeat([]byte("repetitive pattern"), 100),
		make([]byte, 65536),
	}

	for _, p := range payloads {
		// Nil destination
		matches := nf.FindMatches(nil, p)
		assert.Equal(t, 1, len(matches))
		assert.Equal(t, len(p), matches[0].Unmatched)
		assert.Equal(t, 0, matches[0].Length)
		assert.Equal(t, 0, matches[0].Distance)

		// Pre-populated destination
		existing := []matchfinder.Match{{Unmatched: 5, Length: 0, Distance: 0}}
		res := nf.FindMatches(existing, p)
		assert.Equal(t, 2, len(res))
		assert.Equal(t, existing[0], res[0])
		assert.Equal(t, len(p), res[1].Unmatched)
		assert.Equal(t, 0, res[1].Length)
		assert.Equal(t, 0, res[1].Distance)
	}
}

// TestFinder_ConcurrentStress verifies thread-safety and race condition freedom across all Finders.
func TestFinder_ConcurrentStress(t *testing.T) {
	t.Parallel()

	randPayload := make([]byte, 8192)
	_, _ = rand.Read(randPayload)

	for fname, factory := range allFinders() {
		fname, factory := fname, factory
		t.Run(fname, func(t *testing.T) {
			t.Parallel()

			const concurrency = 6
			const iterations = 8

			var wg sync.WaitGroup
			wg.Add(concurrency)

			for c := 0; c < concurrency; c++ {
				go func() {
					defer wg.Done()
					finder := factory()
					for iter := 0; iter < iterations; iter++ {
						finder.Reset()
						matches := finder.FindMatches(nil, randPayload)
						verifyMatchesOracle(t, randPayload, matches, fname)
					}
				}()
			}

			wg.Wait()
		})
	}
}
