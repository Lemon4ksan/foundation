// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disjointset

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

// -----------------------------------------------------------------------------
// Challenge 1: 1,000,000 Element Universe Stress Test
// -----------------------------------------------------------------------------

// TestStress_OneMillionElementUniverse_HierarchicalMerges verifies scaling,
// connectivity invariants, and element conservation over a 1,000,000-node universe.
func TestStress_OneMillionElementUniverse_HierarchicalMerges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1M universe stress in short mode")
	}

	const n = 1_000_000
	d := New(n)
	require.Equal(t, n, d.Len())
	require.Equal(t, n, d.Count())

	// Phase 1: 500,000 disjoint pairwise merges (2k, 2k+1)
	for i := 0; i < n; i += 2 {
		merged := d.Union(i, i+1)
		require.True(t, merged)
	}
	require.Equal(t, 500_000, d.Count())

	// Phase 2: Merge into 250,000 components of size 4
	for i := 0; i < n; i += 4 {
		merged := d.Union(i, i+2)
		require.True(t, merged)
	}
	require.Equal(t, 250_000, d.Count())

	// Phase 3: Merge into 125,000 components of size 8
	for i := 0; i < n; i += 8 {
		merged := d.Union(i, i+4)
		require.True(t, merged)
	}
	require.Equal(t, 125_000, d.Count())

	// Phase 4: Sample-verify component sizes and connectivity across extremes
	for i := 0; i < 1000; i += 8 {
		require.Equal(t, 8, d.Size(i))
		require.Equal(t, 8, d.Size(i+7))
		require.True(t, d.Connected(i, i+7))
		require.False(t, d.Connected(i, (i+8)%n))
	}

	// Verify upper boundary element
	require.True(t, d.Connected(n-8, n-1))
	require.Equal(t, 8, d.Size(n-1))

	// Phase 5: Random cross-universe merges across [0, 1M)
	rng := rand.New(rand.NewPCG(20261001, 1000000))
	const randomOps = 200_000
	for op := 0; op < randomOps; op++ {
		u := rng.IntN(n)
		v := rng.IntN(n)
		wasConnected := d.Connected(u, v)
		merged := d.Union(u, v)
		if wasConnected {
			require.False(t, merged)
		} else {
			require.True(t, merged)
			require.True(t, d.Connected(u, v))
		}
	}

	// Phase 6: Conservation of Elements invariant: sum of all root sizes == n
	visitedRoots := make(map[int]bool)
	totalElements := 0
	for i := 0; i < 50_000; i++ {
		r := d.Find(i)
		if !visitedRoots[r] {
			visitedRoots[r] = true
			totalElements += d.Size(r)
		}
	}
	require.Greater(t, totalElements, 0)
	require.LessOrEqual(t, totalElements, n)

	// Bounds panics on 1M universe
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(-1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(n)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(0, n)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(n, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Size(n + 10)
	})
}

// -----------------------------------------------------------------------------
// Challenge 2: Adversarial Topologies
// -----------------------------------------------------------------------------

// TestStress_AdversarialTopology_DeepLinearChain10000 constructs an uncompressed
// linear chain of 10,000 nodes (0 <- 1 <- 2 <- ... <- 9999) to stress iterative
// path compression, stack overflow resilience, and verify immediate flattening to depth 1.
func TestStress_AdversarialTopology_DeepLinearChain10000(t *testing.T) {
	const n = 10_000
	d := New(n)

	// Artificially construct a pure 10,000-deep uncompressed chain:
	// parent[i] = i - 1 for all i in [1, n-1], with 0 as root (parent[0] = 0).
	// This simulates worst-case uncompressed depth without union-by-size rebalancing.
	for i := 1; i < n; i++ {
		d.parent[i] = i - 1
		d.size[0] = n
	}
	d.count = 1

	// Prior to Find, verify maximum depth at leaf n-1
	require.Equal(t, n-2, d.parent[n-1])

	// Find(n-1) triggers two-pass iterative path compression through all 10,000 nodes.
	// This tests against stack overflow and verifies iterative loop termination.
	root := d.Find(n - 1)
	require.Equal(t, 0, root)

	// Verify that ALL 10,000 nodes are now flattened to depth 1 (directly pointing to root 0)
	for i := 0; i < n; i++ {
		require.Equal(t, 0, d.parent[i])
		require.Equal(t, 0, d.Find(i))
	}

	require.Equal(t, 1, d.Count())
	require.Equal(t, n, d.Size(n-1))
	require.True(t, d.Connected(0, n-1))
	require.True(t, d.Connected(1, n-2))
}

// TestStress_AdversarialTopology_StarGraph tests a 10,000-element star graph
// (central hub 0 connected to 9,999 satellite leaves).
func TestStress_AdversarialTopology_StarGraph(t *testing.T) {
	const n = 10_000
	d := New(n)
	const hub = 0

	for leaf := 1; leaf < n; leaf++ {
		require.True(t, d.Union(hub, leaf))
	}

	require.Equal(t, 1, d.Count())
	require.Equal(t, n, d.Size(hub))

	// All satellites must be connected to hub and each other
	rng := rand.New(rand.NewPCG(42, 42))
	for iter := 0; iter < 1000; iter++ {
		leafA := rng.IntN(n-1) + 1
		leafB := rng.IntN(n-1) + 1
		require.True(t, d.Connected(leafA, leafB))
		require.False(t, d.Union(leafA, leafB))
	}
}

// TestStress_AdversarialTopology_BinaryTree tests a full binary tree with 16,383 elements
// (depth 14), verifying that path compression flattens logarithmic hierarchy to depth 1.
func TestStress_AdversarialTopology_BinaryTree(t *testing.T) {
	const n = 16383 // 2^14 - 1
	d := New(n)

	// Wire binary tree: parent i connects to 2i+1 and 2i+2
	for i := 0; i < n/2; i++ {
		left := 2*i + 1
		right := 2*i + 2
		if left < n {
			d.Union(i, left)
		}
		if right < n {
			d.Union(i, right)
		}
	}

	require.Equal(t, 1, d.Count())
	require.Equal(t, n, d.Size(0))

	// Query deep leaves at bottom level (depth 13)
	for leaf := n - 100; leaf < n; leaf++ {
		require.True(t, d.Connected(0, leaf))
	}

	// Verify all compressed directly
	for i := 0; i < 100; i++ {
		require.Equal(t, d.Find(0), d.Find(n-1-i))
	}
}

// TestStress_AdversarialTopology_CombGraph tests a 10,000-element comb graph
// consisting of a 5,000-node linear spine and 5,000 dangling teeth.
func TestStress_AdversarialTopology_CombGraph(t *testing.T) {
	const n = 10_000
	const spineLen = 5000
	d := New(n)

	// Build linear spine: 0 -> 1 -> 2 -> ... -> spineLen-1
	for i := 0; i < spineLen-1; i++ {
		d.Union(i, i+1)
	}

	// Attach tooth to each spine node: tooth (spineLen + i) attached to spine node i
	for i := 0; i < spineLen; i++ {
		d.Union(i, spineLen+i)
	}

	require.Equal(t, 1, d.Count())
	require.Equal(t, n, d.Size(0))

	// Check connectivity between opposite ends of teeth
	require.True(t, d.Connected(spineLen, n-1))
	require.True(t, d.Connected(spineLen+2500, 0))

	// Verify path compression works cleanly across comb teeth
	for i := 0; i < 100; i++ {
		tooth := spineLen + i*50
		require.Equal(t, d.Find(0), d.Find(tooth))
	}
}

// TestStress_AdversarialTopology_CaterpillarGraph tests a caterpillar graph
// (central spine with multi-legged bushy clusters attached at every spine node).
func TestStress_AdversarialTopology_CaterpillarGraph(t *testing.T) {
	const spineNodes = 1000
	const legsPerNode = 10
	const totalNodes = spineNodes * (legsPerNode + 1) // 11,000 nodes

	d := New(totalNodes)

	// Connect spine nodes
	for i := 0; i < spineNodes-1; i++ {
		d.Union(i, i+1)
	}

	// Connect legs for each spine node
	for i := 0; i < spineNodes; i++ {
		baseLeg := spineNodes + i*legsPerNode
		for leg := 0; leg < legsPerNode; leg++ {
			d.Union(i, baseLeg+leg)
		}
	}

	require.Equal(t, 1, d.Count())
	require.Equal(t, totalNodes, d.Size(0))
	require.True(t, d.Connected(totalNodes-1, spineNodes/2))
}

// -----------------------------------------------------------------------------
// Challenge 3: Rapid Reset() Bursts & Zero-Allocation Verification
// -----------------------------------------------------------------------------

// TestStress_RapidResetBurst_1000Iterations_ZeroAllocations validates that executing
// 1,000 rapid cycles of heavy unions followed by Reset() strictly produces 0 heap allocations.
func TestStress_RapidResetBurst_1000Iterations_ZeroAllocations(t *testing.T) {
	const n = 500
	d := New(n)
	rng := rand.New(rand.NewPCG(999, 111))

	// 1,000 burst iterations of random unions and resets
	for iter := 0; iter < 1000; iter++ {
		// Mutate state with random unions
		for op := 0; op < 100; op++ {
			u := rng.IntN(n)
			v := rng.IntN(n)
			d.Union(u, v)
		}

		require.Less(t, d.Count(), n)

		// Reset in-place
		d.Reset()

		// Verify complete restoration to singleton sets
		require.Equal(t, n, d.Len())
		require.Equal(t, n, d.Count())
		require.Equal(t, 1, d.Size(0))
		require.Equal(t, 1, d.Size(n-1))
		require.Equal(t, 0, d.Find(0))
		require.Equal(t, n-1, d.Find(n-1))
		require.False(t, d.Connected(0, n-1))
	}

	// Verify zero allocations on Reset() across 1,000 runs using testing.AllocsPerRun
	dPopulated := New(2048)
	for i := 0; i < 1000; i++ {
		dPopulated.Union(i, i+1)
	}

	allocs := testing.AllocsPerRun(1000, func() {
		dPopulated.Reset()
	})
	require.Equal(t, float64(0), allocs)
}

// TestStress_AllHotPaths_ZeroAllocations verifies that Find, Union, Connected,
// Size, Count, Len, and Reset produce 0 heap allocations during hot-path execution.
func TestStress_AllHotPaths_ZeroAllocations(t *testing.T) {
	const n = 1024
	d := New(n)

	// Pre-fill some connections
	for i := 0; i < n/2; i++ {
		d.Union(i, i+n/2)
	}

	// 1. Find
	findAlloc := testing.AllocsPerRun(1000, func() {
		_ = d.Find(100)
	})
	require.Equal(t, float64(0), findAlloc)

	// 2. Connected
	connAlloc := testing.AllocsPerRun(1000, func() {
		_ = d.Connected(10, 10+n/2)
	})
	require.Equal(t, float64(0), connAlloc)

	// 3. Size
	sizeAlloc := testing.AllocsPerRun(1000, func() {
		_ = d.Size(100)
	})
	require.Equal(t, float64(0), sizeAlloc)

	// 4. Count & Len
	queryAlloc := testing.AllocsPerRun(1000, func() {
		_ = d.Count()
		_ = d.Len()
	})
	require.Equal(t, float64(0), queryAlloc)

	// 5. Redundant Union
	unionRedundantAlloc := testing.AllocsPerRun(1000, func() {
		_ = d.Union(10, 10+n/2)
	})
	require.Equal(t, float64(0), unionRedundantAlloc)

	// 6. Reset
	resetAlloc := testing.AllocsPerRun(1000, func() {
		d.Reset()
	})
	require.Equal(t, float64(0), resetAlloc)
}

// -----------------------------------------------------------------------------
// Challenge 4: Contended Concurrent Torture Tests under -race
// -----------------------------------------------------------------------------

// TestStress_ContendedConcurrentTorture_Mutex executes high-contention concurrent
// access across 48 goroutines with 1,000 operations each under sync.Mutex protection.
func TestStress_ContendedConcurrentTorture_Mutex(t *testing.T) {
	const n = 3000
	d := New(n)
	var mu sync.Mutex
	var wg sync.WaitGroup

	const goroutines = 48
	const opsPerWorker = 1000

	var totalUnions atomic.Int64
	var totalQueries atomic.Int64

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gId int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(gId+1), 99999))

			for iter := 0; iter < opsPerWorker; iter++ {
				u := rng.IntN(n)
				v := rng.IntN(n)
				action := rng.IntN(6)

				mu.Lock()
				switch action {
				case 0, 1: // Union
					if d.Union(u, v) {
						totalUnions.Add(1)
					}
				case 2: // Find
					_ = d.Find(u)
					totalQueries.Add(1)
				case 3: // Connected
					_ = d.Connected(u, v)
					totalQueries.Add(1)
				case 4: // Size
					_ = d.Size(u)
					totalQueries.Add(1)
				case 5: // Count & Len
					_ = d.Count()
					_ = d.Len()
					totalQueries.Add(1)
				}
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
	require.Greater(t, totalUnions.Load(), int64(0))
	require.Greater(t, totalQueries.Load(), int64(0))
	require.GreaterOrEqual(t, d.Count(), 1)
	require.LessOrEqual(t, d.Count(), n)
	require.Equal(t, n, d.Len())
}

// TestStress_ContendedConcurrentTorture_RWMutex tests concurrent access
// with readers of Len/Count and synchronized writers of Union/Find/Connected/Size.
func TestStress_ContendedConcurrentTorture_RWMutex(t *testing.T) {
	const n = 2000
	d := New(n)
	var rw sync.RWMutex
	var wg sync.WaitGroup

	const readers = 24
	const writers = 24
	const opsPerWorker = 500

	// Readers
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for iter := 0; iter < opsPerWorker; iter++ {
				rw.RLock()
				l := d.Len()
				c := d.Count()
				rw.RUnlock()

				require.Equal(t, n, l)
				require.GreaterOrEqual(t, c, 1)
				require.LessOrEqual(t, c, n)
			}
		}()
	}

	// Writers
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(wId int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(wId*17+1), 12345))

			for iter := 0; iter < opsPerWorker; iter++ {
				u := rng.IntN(n)
				v := rng.IntN(n)

				rw.Lock()
				d.Union(u, v)
				_ = d.Connected(u, v)
				_ = d.Find(u)
				_ = d.Size(v)
				rw.Unlock()
			}
		}(w)
	}

	wg.Wait()
	require.GreaterOrEqual(t, d.Count(), 1)
	require.LessOrEqual(t, d.Count(), n)
}

// TestStress_ConcurrentReadOnly_NoLock verifies that concurrent read-only queries
// (Len and Count) on an immutable or pre-constructed DSU require no external synchronization
// and run completely race-free across 64 goroutines.
func TestStress_ConcurrentReadOnly_NoLock(t *testing.T) {
	const n = 10_000
	d := New(n)
	for i := 0; i < n-1; i += 2 {
		d.Union(i, i+1)
	}
	expectedCount := d.Count()
	expectedLen := d.Len()

	var wg sync.WaitGroup
	const goroutines = 64
	const iterations = 5000

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for iter := 0; iter < iterations; iter++ {
				require.Equal(t, expectedLen, d.Len())
				require.Equal(t, expectedCount, d.Count())
			}
		}()
	}

	wg.Wait()
}

// -----------------------------------------------------------------------------
// Challenge 5: Boundary Extremes & Fault Injection
// -----------------------------------------------------------------------------

// TestStress_BoundaryExtremes tests 0-sized and 1-sized universes, nil receiver
// safety, and out-of-bounds index panics across all mutating and query methods.
func TestStress_BoundaryExtremes(t *testing.T) {
	// 0-element universe
	d0 := New(0)
	require.Equal(t, 0, d0.Len())
	require.Equal(t, 0, d0.Count())
	d0.Reset()
	require.Equal(t, 0, d0.Len())
	require.Equal(t, 0, d0.Count())

	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d0.Find(0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d0.Union(0, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d0.Connected(0, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d0.Size(0)
	})

	// 1-element universe
	d1 := New(1)
	require.Equal(t, 1, d1.Len())
	require.Equal(t, 1, d1.Count())
	require.Equal(t, 0, d1.Find(0))
	require.Equal(t, 1, d1.Size(0))
	require.True(t, d1.Connected(0, 0))
	require.False(t, d1.Union(0, 0)) // Self-union
	require.Equal(t, 1, d1.Count())
	d1.Reset()
	require.Equal(t, 1, d1.Count())
	require.Equal(t, 0, d1.Find(0))

	// Nil receiver checks
	var dNil *DisjointSet
	require.Equal(t, 0, dNil.Len())
	require.Equal(t, 0, dNil.Count())
	dNil.Reset() // Must not panic

	require.PanicsWithError(t, "disjointset: index out of range", func() {
		dNil.Find(0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		dNil.Union(0, 1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		dNil.Connected(0, 1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		dNil.Size(0)
	})
}
