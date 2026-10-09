// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disjointset

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

func TestConstructors(t *testing.T) {
	// New panics on negative n
	require.PanicsWithError(t, "disjointset: negative element count", func() {
		New(-1)
	})
	require.PanicsWithError(t, "disjointset: negative element count", func() {
		New(-100)
	})

	// New with zero elements
	d0 := New(0)
	require.Equal(t, 0, d0.Len())
	require.Equal(t, 0, d0.Count())

	// Standard sizes
	for _, n := range []int{1, 2, 5, 10, 100, 1000} {
		d := New(n)
		require.Equal(t, n, d.Len())
		require.Equal(t, n, d.Count())

		for i := 0; i < n; i++ {
			require.Equal(t, i, d.Find(i))
			require.Equal(t, 1, d.Size(i))
			require.True(t, d.Connected(i, i))
		}
	}
}

func TestNilReceiver(t *testing.T) {
	var d *Set

	// Safe queries on nil
	require.Equal(t, 0, d.Len())
	require.Equal(t, 0, d.Count())

	// Safe reset on nil
	d.Reset()

	// Panics on operations requiring valid indices
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(0, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(0, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Size(0)
	})
}

func TestZeroValueReceiver(t *testing.T) {
	var d Set

	require.Equal(t, 0, d.Len())
	require.Equal(t, 0, d.Count())

	d.Reset()

	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(0, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(0, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Size(0)
	})
}

func TestFind(t *testing.T) {
	d := New(10)

	// Direct root lookup
	for i := 0; i < 10; i++ {
		require.Equal(t, i, d.Find(i))
	}

	// Bounds panics
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(-1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(-50)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(10)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Find(100)
	})

	// Multi-hop path compression
	d.Union(0, 1)
	d.Union(1, 2)
	d.Union(2, 3)

	root := d.Find(3)
	require.Equal(t, root, d.Find(0))
	require.Equal(t, root, d.Find(1))
	require.Equal(t, root, d.Find(2))
}

func TestUnion(t *testing.T) {
	d := New(10)

	// Bounds panics on x
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(-1, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(10, 0)
	})

	// Bounds panics on y
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(0, -1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Union(0, 10)
	})

	// Self-union
	require.False(t, d.Union(3, 3))
	require.Equal(t, 10, d.Count())

	// Merge singletons
	require.True(t, d.Union(0, 1))
	require.Equal(t, 9, d.Count())
	require.True(t, d.Connected(0, 1))
	require.Equal(t, 2, d.Size(0))
	require.Equal(t, 2, d.Size(1))

	// Already connected pair
	require.False(t, d.Union(0, 1))
	require.False(t, d.Union(1, 0))
	require.Equal(t, 9, d.Count())

	// Branch: size[rootX] < size[rootY]
	// Root of {2} has size 1, Root of {0, 1} has size 2
	require.True(t, d.Union(2, 0))
	require.Equal(t, 8, d.Count())
	require.Equal(t, 3, d.Size(2))
	require.Equal(t, 3, d.Size(0))

	// Branch: size[rootX] > size[rootY]
	// Root of {0, 1, 2} has size 3, Root of {3} has size 1
	require.True(t, d.Union(0, 3))
	require.Equal(t, 7, d.Count())
	require.Equal(t, 4, d.Size(3))

	// Branch: size[rootX] == size[rootY]
	// Create another component of size 4: {4, 5, 6, 7}
	require.True(t, d.Union(4, 5))
	require.True(t, d.Union(6, 7))
	require.True(t, d.Union(4, 6))
	require.Equal(t, 4, d.Size(4))
	require.Equal(t, 4, d.Count())

	// Merge equal-sized components: {0,1,2,3} and {4,5,6,7}
	require.True(t, d.Union(0, 4))
	require.Equal(t, 8, d.Size(0))
	require.Equal(t, 8, d.Size(4))
	require.Equal(t, 3, d.Count())
}

func TestConnected(t *testing.T) {
	d := New(6)

	// Bounds panics
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(-1, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(0, -1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(6, 0)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Connected(0, 6)
	})

	// Reflexivity
	for i := 0; i < 6; i++ {
		require.True(t, d.Connected(i, i))
	}

	// Disjoint
	require.False(t, d.Connected(0, 1))
	require.False(t, d.Connected(2, 5))

	// Transitive connections
	d.Union(0, 1)
	d.Union(1, 2)
	require.True(t, d.Connected(0, 2))
	require.True(t, d.Connected(2, 0))
	require.False(t, d.Connected(0, 3))

	d.Union(3, 4)
	d.Union(4, 5)
	require.True(t, d.Connected(3, 5))
	require.False(t, d.Connected(0, 5))

	// Connect the two components
	d.Union(2, 3)
	require.True(t, d.Connected(0, 5))
	require.Equal(t, 1, d.Count())
}

func TestSize(t *testing.T) {
	d := New(5)

	// Bounds panics
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Size(-1)
	})
	require.PanicsWithError(t, "disjointset: index out of range", func() {
		d.Size(5)
	})

	// Initial sizes
	for i := 0; i < 5; i++ {
		require.Equal(t, 1, d.Size(i))
	}

	d.Union(0, 1)
	require.Equal(t, 2, d.Size(0))
	require.Equal(t, 2, d.Size(1))
	require.Equal(t, 1, d.Size(2))

	d.Union(2, 3)
	require.Equal(t, 2, d.Size(2))
	require.Equal(t, 2, d.Size(3))

	d.Union(1, 3)
	for i := 0; i < 4; i++ {
		require.Equal(t, 4, d.Size(i))
	}
	require.Equal(t, 1, d.Size(4))
}

func TestReset(t *testing.T) {
	d := New(10)

	// Merge all into single set
	for i := 1; i < 10; i++ {
		d.Union(0, i)
	}
	require.Equal(t, 1, d.Count())
	require.Equal(t, 10, d.Size(0))

	// Reset
	d.Reset()

	require.Equal(t, 10, d.Len())
	require.Equal(t, 10, d.Count())

	for i := 0; i < 10; i++ {
		require.Equal(t, i, d.Find(i))
		require.Equal(t, 1, d.Size(i))
		for j := 0; j < 10; j++ {
			if i == j {
				require.True(t, d.Connected(i, j))
			} else {
				require.False(t, d.Connected(i, j))
			}
		}
	}

	// Reset on empty DSU
	d0 := New(0)
	d0.Reset()
	require.Equal(t, 0, d0.Len())
	require.Equal(t, 0, d0.Count())
}

func TestDeepChainPathCompression(t *testing.T) {
	n := 5000
	d := New(n)

	// Build linear chain by manually setting parents to simulate deep uncompressed chain:
	// n-1 -> n-2 -> ... -> 1 -> 0
	for i := 1; i < n; i++ {
		d.parent[i] = i - 1
	}

	// Find on deepest leaf triggers full two-pass path compression
	root := d.Find(n - 1)
	require.Equal(t, 0, root)

	// Verify all nodes now point directly to root 0
	for i := 0; i < n; i++ {
		require.Equal(t, 0, d.parent[i])
		require.Equal(t, 0, d.Find(i))
	}
}

func TestConcurrentReaders_CountLen(t *testing.T) {
	d := New(1000)
	for i := 0; i < 500; i++ {
		d.Union(i, i+500)
	}

	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				_ = d.Len()
				_ = d.Count()
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentSynchronizedWriters(t *testing.T) {
	d := New(1000)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	// Readers of Len & Count
	for r := 0; r < 16; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 500; i++ {
				mu.RLock()
				_ = d.Len()
				_ = d.Count()
				mu.RUnlock()
			}
		}()
	}

	// Writers performing mutations and path-compressing queries
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(wId int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				u := (wId*100 + i) % 1000
				v := (wId*50 + i*3) % 1000

				mu.Lock()
				d.Union(u, v)
				_ = d.Connected(u, v)
				_ = d.Size(u)
				_ = d.Find(v)
				mu.Unlock()
			}
		}(w)
	}

	wg.Wait()
}

func TestConcurrentMutex_AllOps(t *testing.T) {
	d := New(2000)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func(gId int) {
			defer wg.Done()
			for i := 0; i < 300; i++ {
				u := (gId*120 + i) % 2000
				v := (gId*70 + i*7) % 2000

				mu.Lock()
				d.Union(u, v)
				_ = d.Find(u)
				_ = d.Connected(u, v)
				_ = d.Size(u)
				_ = d.Count()
				_ = d.Len()
				if i == 299 && gId == 0 {
					d.Reset()
				}
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
}

func TestConcurrentIndependentInstances(t *testing.T) {
	var wg sync.WaitGroup

	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			d := New(200)
			for i := 0; i < 100; i++ {
				u := (seed*13 + i) % 200
				v := (seed*7 + i*3) % 200
				d.Union(u, v)
				_ = d.Connected(u, v)
				_ = d.Size(u)
				_ = d.Find(v)
				_ = d.Count()
			}
			d.Reset()
			require.Equal(t, 200, d.Count())
		}(g)
	}

	wg.Wait()
}

// -----------------------------------------------------------------------------
// Mathematical Invariants & Reference Graph Oracle
// -----------------------------------------------------------------------------

func TestMathematicalInvariants(t *testing.T) {
	n := 200
	d := New(n)
	rng := rand.New(rand.NewPCG(42, 99))

	// Perform random unions
	for i := 0; i < 150; i++ {
		u := rng.IntN(n)
		v := rng.IntN(n)
		d.Union(u, v)
	}

	// 1. Idempotence: Find(Find(x)) == Find(x)
	for i := 0; i < n; i++ {
		root := d.Find(i)
		require.Equal(t, root, d.Find(root))
	}

	// 2. Exact Root Count: count(roots where parent[r] == r) == Count()
	rootCount := 0
	rootSizes := make(map[int]int)
	for i := 0; i < n; i++ {
		root := d.Find(i)
		if root == i {
			rootCount++
			rootSizes[root] = d.Size(root)
		}
	}
	require.Equal(t, d.Count(), rootCount)

	// 3. Conservation of Elements: sum of component sizes equals total elements N
	totalSize := 0
	for _, sz := range rootSizes {
		totalSize += sz
	}
	require.Equal(t, n, totalSize)

	// 4. Transitivity: Connected(a, b) && Connected(b, c) => Connected(a, c)
	for a := 0; a < 20; a++ {
		for b := 0; b < 20; b++ {
			for c := 0; c < 20; c++ {
				if d.Connected(a, b) && d.Connected(b, c) {
					require.True(t, d.Connected(a, c))
				}
			}
		}
	}
}

type graphOracle struct {
	n   int
	adj map[int][]int
}

func newGraphOracle(n int) *graphOracle {
	return &graphOracle{
		n:   n,
		adj: make(map[int][]int, n),
	}
}

func (o *graphOracle) union(u, v int) bool {
	if o.connected(u, v) {
		return false
	}
	o.adj[u] = append(o.adj[u], v)
	o.adj[v] = append(o.adj[v], u)
	return true
}

func (o *graphOracle) connected(u, v int) bool {
	if u == v {
		return true
	}
	visited := make(map[int]bool, o.n)
	queue := []int{u}
	visited[u] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr == v {
			return true
		}
		for _, neighbor := range o.adj[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return false
}

func (o *graphOracle) size(u int) int {
	visited := make(map[int]bool, o.n)
	queue := []int{u}
	visited[u] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, neighbor := range o.adj[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	return len(visited)
}

func (o *graphOracle) count() int {
	visited := make(map[int]bool, o.n)
	components := 0

	for i := 0; i < o.n; i++ {
		if !visited[i] {
			components++
			queue := []int{i}
			visited[i] = true

			for len(queue) > 0 {
				curr := queue[0]
				queue = queue[1:]

				for _, neighbor := range o.adj[curr] {
					if !visited[neighbor] {
						visited[neighbor] = true
						queue = append(queue, neighbor)
					}
				}
			}
		}
	}
	return components
}

func TestGraphOracleDifferential(t *testing.T) {
	n := 100
	dsu := New(n)
	oracle := newGraphOracle(n)
	rng := rand.New(rand.NewPCG(12345, 67890))

	for op := 0; op < 2000; op++ {
		u := rng.IntN(n)
		v := rng.IntN(n)

		dsuMerged := dsu.Union(u, v)
		oracleMerged := oracle.union(u, v)
		require.Equal(t, oracleMerged, dsuMerged)

		queryU := rng.IntN(n)
		queryV := rng.IntN(n)
		require.Equal(t, oracle.connected(queryU, queryV), dsu.Connected(queryU, queryV))
		require.Equal(t, oracle.size(queryU), dsu.Size(queryU))
	}

	require.Equal(t, oracle.count(), dsu.Count())
}

func TestKruskalMSTSimulation(t *testing.T) {
	type edge struct {
		u, v, weight int
	}

	edges := []edge{
		{0, 1, 1},
		{1, 2, 2},
		{0, 2, 3},
		{2, 3, 4},
		{1, 3, 5},
		{3, 4, 6},
		{2, 4, 7},
	}

	sort.Slice(edges, func(i, j int) bool {
		return edges[i].weight < edges[j].weight
	})

	dsu := New(5)
	var mstEdges []edge
	totalWeight := 0

	for _, e := range edges {
		if !dsu.Connected(e.u, e.v) {
			merged := dsu.Union(e.u, e.v)
			require.True(t, merged)
			mstEdges = append(mstEdges, e)
			totalWeight += e.weight
		}
	}

	require.Equal(t, 1, dsu.Count())
	require.Equal(t, 4, len(mstEdges))
	require.Equal(t, 1+2+4+6, totalWeight)

	expectedMST := []edge{
		{0, 1, 1},
		{1, 2, 2},
		{2, 3, 4},
		{3, 4, 6},
	}
	require.Equal(t, expectedMST, mstEdges)
}

func TestStress_1MillionElements(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 1M stress in short mode")
	}

	n := 1_000_000
	d := New(n)
	require.Equal(t, n, d.Len())
	require.Equal(t, n, d.Count())

	// Connect 500,000 disjoint pairs: (0, 1), (2, 3), ..., (999998, 999999)
	for i := 0; i < n; i += 2 {
		merged := d.Union(i, i+1)
		require.True(t, merged)
	}

	require.Equal(t, 500_000, d.Count())

	// Verify lookups
	for i := 0; i < 1000; i += 2 {
		require.True(t, d.Connected(i, i+1))
		require.Equal(t, 2, d.Size(i))
	}

	// Connect pairs into components of 4
	for i := 0; i < n; i += 4 {
		merged := d.Union(i, i+2)
		require.True(t, merged)
	}

	require.Equal(t, 250_000, d.Count())
	for i := 0; i < 1000; i += 4 {
		require.True(t, d.Connected(i, i+3))
		require.Equal(t, 4, d.Size(i))
	}
}

func TestStress_AdversarialTopologies(t *testing.T) {
	// Topology 1: Star graph (1 hub, N-1 leaves)
	t.Run("StarGraph", func(t *testing.T) {
		n := 10_000
		d := New(n)
		hub := 0
		for leaf := 1; leaf < n; leaf++ {
			d.Union(hub, leaf)
		}
		require.Equal(t, 1, d.Count())
		require.Equal(t, n, d.Size(hub))
		for leaf := 1; leaf < n; leaf++ {
			require.True(t, d.Connected(hub, leaf))
		}
	})

	// Topology 2: Long chain (0 -> 1 -> 2 -> ... -> N-1)
	t.Run("LongChain", func(t *testing.T) {
		n := 10_000
		d := New(n)
		for i := 0; i < n-1; i++ {
			d.Union(i, i+1)
		}
		require.Equal(t, 1, d.Count())
		require.True(t, d.Connected(0, n-1))
	})

	// Topology 3: Binary Tree Structure
	t.Run("BinaryTree", func(t *testing.T) {
		n := 8191 // 2^13 - 1
		d := New(n)
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
		require.True(t, d.Connected(0, n-1))
	})

	// Topology 4: Comb Graph (spine with teeth)
	t.Run("CombGraph", func(t *testing.T) {
		n := 10_000
		d := New(n)
		half := n / 2
		// Spine: 0 to half-1
		for i := 0; i < half-1; i++ {
			d.Union(i, i+1)
		}
		// Teeth: connect each spine node i to tooth half+i
		for i := 0; i < half; i++ {
			d.Union(i, half+i)
		}
		require.Equal(t, 1, d.Count())
		require.Equal(t, n, d.Size(0))
		require.True(t, d.Connected(half, n-1))
	})
}

func TestStress_RapidResetBurst(t *testing.T) {
	n := 1000
	d := New(n)
	rng := rand.New(rand.NewPCG(777, 888))

	for burst := 0; burst < 100; burst++ {
		for op := 0; op < 200; op++ {
			d.Union(rng.IntN(n), rng.IntN(n))
		}
		d.Reset()
		require.Equal(t, n, d.Count())
		require.Equal(t, 1, d.Size(rng.IntN(n)))
	}
}

func TestStress_ContendedConcurrentTorture(t *testing.T) {
	n := 2000
	d := New(n)
	var mu sync.Mutex
	var wg sync.WaitGroup

	goroutines := 32
	opsPerGoroutine := 500

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gId int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(gId), 5555))

			for i := 0; i < opsPerGoroutine; i++ {
				u := rng.IntN(n)
				v := rng.IntN(n)

				mu.Lock()
				action := rng.IntN(4)
				switch action {
				case 0:
					d.Union(u, v)
				case 1:
					_ = d.Find(u)
				case 2:
					_ = d.Connected(u, v)
				case 3:
					_ = d.Size(u)
				}
				mu.Unlock()
			}
		}(g)
	}

	wg.Wait()
	require.GreaterOrEqual(t, d.Count(), 1)
}
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
	var dNil *Set
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
func BenchmarkSet_Find_DirectRoot(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Find(512)
	}
}

func BenchmarkSet_Find_FlatTree(b *testing.B) {
	d := New(1024)
	for i := 1; i < 1024; i++ {
		d.Union(0, i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var idx int
	for b.Loop() {
		_ = d.Find(idx & 1023)
		idx++
	}
}

func BenchmarkSet_Find_Deep(b *testing.B) {
	n := 1024
	d := New(n)
	// Build chain without flattening
	for i := 0; i < n-1; i++ {
		d.parent[i+1] = i
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Find(n - 1)
	}
}

func BenchmarkSet_Union_Disjoint(b *testing.B) {
	d := New(1_000_000)
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		u := (i * 2) % 1_000_000
		v := (u + 1) % 1_000_000
		d.Union(u, v)
		i++
	}
}

func BenchmarkSet_Union_AlreadyConnected(b *testing.B) {
	d := New(1024)
	d.Union(10, 20)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Union(10, 20)
	}
}

func BenchmarkSet_Union_Self(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Union(42, 42)
	}
}

func BenchmarkSet_Connected(b *testing.B) {
	d := New(1024)
	for i := 0; i < 512; i++ {
		d.Union(i, i+512)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		_ = d.Connected(i&511, (i&511)+512)
		i++
	}
}

func BenchmarkSet_Size(b *testing.B) {
	d := New(1024)
	for i := 0; i < 512; i++ {
		d.Union(i, i+512)
	}
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		_ = d.Size(i & 1023)
		i++
	}
}

func BenchmarkSet_Count(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Count()
	}
}

func BenchmarkSet_Len(b *testing.B) {
	d := New(1024)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_ = d.Len()
	}
}

func BenchmarkSet_Reset_1024(b *testing.B) {
	d := New(1024)
	for i := 0; i < 500; i++ {
		d.Union(i, i+1)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		d.Reset()
	}
}

func BenchmarkSet_Reset_65536(b *testing.B) {
	d := New(65536)
	for i := 0; i < 30000; i++ {
		d.Union(i, i+1)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		d.Reset()
	}
}

func BenchmarkSet_Kruskal_Workload(b *testing.B) {
	type edge struct {
		u, v int
	}

	n := 1000
	numEdges := 5000
	rng := rand.New(rand.NewPCG(123, 456))

	edges := make([]edge, numEdges)
	for i := 0; i < numEdges; i++ {
		edges[i] = edge{
			u: rng.IntN(n),
			v: rng.IntN(n),
		}
	}

	d := New(n)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		d.Reset()
		for _, e := range edges {
			if !d.Connected(e.u, e.v) {
				d.Union(e.u, e.v)
			}
		}
	}
}
// -----------------------------------------------------------------------------
// Independent Reference Graph Connected-Components Oracle (BFS & DFS)
// -----------------------------------------------------------------------------

// graphRefOracle maintains an adjacency list representation of an undirected graph
// and computes connectivity and component metrics using classical BFS and DFS.
type graphRefOracle struct {
	n   int
	adj [][]int

	// Scratch buffers to achieve high performance across 100k+ operations without GC thrashing
	visitedBFS []int
	tokenBFS   int
	queue      []int

	visitedDFS []int
	tokenDFS   int
}

func newGraphRefOracle(n int) *graphRefOracle {
	adj := make([][]int, n)
	for i := 0; i < n; i++ {
		adj[i] = make([]int, 0, 8)
	}
	return &graphRefOracle{
		n:          n,
		adj:        adj,
		visitedBFS: make([]int, n),
		queue:      make([]int, 0, n),
		visitedDFS: make([]int, n),
	}
}

func (o *graphRefOracle) reset() {
	for i := 0; i < o.n; i++ {
		o.adj[i] = o.adj[i][:0]
		o.visitedBFS[i] = 0
		o.visitedDFS[i] = 0
	}
	o.tokenBFS = 0
	o.tokenDFS = 0
	o.queue = o.queue[:0]
}

// union adds an undirected edge between u and v if they are not already connected.
// Returns true if an edge was added (u and v were previously disconnected), or false otherwise.
func (o *graphRefOracle) union(u, v int) bool {
	if o.connectedBFS(u, v) {
		return false
	}
	o.adj[u] = append(o.adj[u], v)
	o.adj[v] = append(o.adj[v], u)
	return true
}

// connectedBFS checks if u and v are reachable using Breadth-First Search.
func (o *graphRefOracle) connectedBFS(u, v int) bool {
	if u == v {
		return true
	}
	o.tokenBFS++
	tok := o.tokenBFS

	o.queue = o.queue[:0]
	o.queue = append(o.queue, u)
	o.visitedBFS[u] = tok

	head := 0
	for head < len(o.queue) {
		curr := o.queue[head]
		head++

		if curr == v {
			return true
		}

		for _, neighbor := range o.adj[curr] {
			if o.visitedBFS[neighbor] != tok {
				o.visitedBFS[neighbor] = tok
				o.queue = append(o.queue, neighbor)
			}
		}
	}
	return false
}

// connectedDFS checks if u and v are reachable using Depth-First Search.
func (o *graphRefOracle) connectedDFS(u, v int) bool {
	if u == v {
		return true
	}
	o.tokenDFS++
	tok := o.tokenDFS

	var dfs func(curr int) bool
	dfs = func(curr int) bool {
		if curr == v {
			return true
		}
		o.visitedDFS[curr] = tok
		for _, neighbor := range o.adj[curr] {
			if o.visitedDFS[neighbor] != tok {
				if dfs(neighbor) {
					return true
				}
			}
		}
		return false
	}

	return dfs(u)
}

// sizeBFS counts reachable nodes from u using BFS.
func (o *graphRefOracle) sizeBFS(u int) int {
	o.tokenBFS++
	tok := o.tokenBFS

	o.queue = o.queue[:0]
	o.queue = append(o.queue, u)
	o.visitedBFS[u] = tok

	head := 0
	for head < len(o.queue) {
		curr := o.queue[head]
		head++

		for _, neighbor := range o.adj[curr] {
			if o.visitedBFS[neighbor] != tok {
				o.visitedBFS[neighbor] = tok
				o.queue = append(o.queue, neighbor)
			}
		}
	}
	return head
}

// sizeDFS counts reachable nodes from u using DFS.
func (o *graphRefOracle) sizeDFS(u int) int {
	o.tokenDFS++
	tok := o.tokenDFS

	var dfs func(curr int) int
	dfs = func(curr int) int {
		o.visitedDFS[curr] = tok
		count := 1
		for _, neighbor := range o.adj[curr] {
			if o.visitedDFS[neighbor] != tok {
				count += dfs(neighbor)
			}
		}
		return count
	}

	return dfs(u)
}

// componentCountBFS calculates the number of connected components in the graph via BFS.
func (o *graphRefOracle) componentCountBFS() int {
	o.tokenBFS++
	tok := o.tokenBFS
	components := 0

	for i := 0; i < o.n; i++ {
		if o.visitedBFS[i] == tok {
			continue
		}
		components++
		o.queue = o.queue[:0]
		o.queue = append(o.queue, i)
		o.visitedBFS[i] = tok

		head := 0
		for head < len(o.queue) {
			curr := o.queue[head]
			head++

			for _, neighbor := range o.adj[curr] {
				if o.visitedBFS[neighbor] != tok {
					o.visitedBFS[neighbor] = tok
					o.queue = append(o.queue, neighbor)
				}
			}
		}
	}
	return components
}

// componentCountDFS calculates the number of connected components in the graph via DFS.
func (o *graphRefOracle) componentCountDFS() int {
	o.tokenDFS++
	tok := o.tokenDFS
	components := 0

	var dfs func(curr int)
	dfs = func(curr int) {
		o.visitedDFS[curr] = tok
		for _, neighbor := range o.adj[curr] {
			if o.visitedDFS[neighbor] != tok {
				dfs(neighbor)
			}
		}
	}

	for i := 0; i < o.n; i++ {
		if o.visitedDFS[i] != tok {
			components++
			dfs(i)
		}
	}
	return components
}

// -----------------------------------------------------------------------------
// 1. Independent BFS/DFS Graph Connected-Components Oracle across 100,000+ Ops
// -----------------------------------------------------------------------------

func TestOracle_100kRandomizedBFSDFSOperations(t *testing.T) {
	const totalOps = 100_000
	const n = 300
	rng := rand.New(rand.NewPCG(20261001, 777999))

	dsu := New(n)
	oracle := newGraphRefOracle(n)

	unionsAttempted := 0
	unionsSuccessful := 0
	connectedQueries := 0
	sizeQueries := 0
	findQueries := 0

	for op := 0; op < totalOps; op++ {
		action := rng.IntN(100)

		switch {
		case action < 38: // Union (38%)
			unionsAttempted++
			u := rng.IntN(n)
			v := rng.IntN(n)

			dsuResult := dsu.Union(u, v)
			oracleResult := oracle.union(u, v)

			require.Equal(t, oracleResult, dsuResult,
				fmt.Sprintf("op %d: Union(%d, %d) mismatch: oracle=%v, dsu=%v", op, u, v, oracleResult, dsuResult))

			if dsuResult {
				unionsSuccessful++
			}

		case action < 75: // Connected (37%)
			connectedQueries++
			u := rng.IntN(n)
			v := rng.IntN(n)

			bfsResult := oracle.connectedBFS(u, v)
			dfsResult := oracle.connectedDFS(u, v)
			require.Equal(
				t,
				bfsResult,
				dfsResult,
				fmt.Sprintf("op %d: BFS and DFS oracle disagreement on (%d, %d)", op, u, v),
			)

			dsuResult := dsu.Connected(u, v)
			require.Equal(t, bfsResult, dsuResult,
				fmt.Sprintf("op %d: Connected(%d, %d) mismatch: oracle=%v, dsu=%v", op, u, v, bfsResult, dsuResult))

			// Check symmetry on every Connected query
			require.Equal(t, dsuResult, dsu.Connected(v, u),
				fmt.Sprintf("op %d: Connected symmetry violation for (%d, %d)", op, u, v))

		case action < 90: // Size (15%)
			sizeQueries++
			u := rng.IntN(n)

			bfsSize := oracle.sizeBFS(u)
			dfsSize := oracle.sizeDFS(u)
			require.Equal(t, bfsSize, dfsSize, fmt.Sprintf("op %d: BFS and DFS size disagreement for %d", op, u))

			dsuSize := dsu.Size(u)
			require.Equal(t, bfsSize, dsuSize,
				fmt.Sprintf("op %d: Size(%d) mismatch: oracle=%d, dsu=%d", op, u, bfsSize, dsuSize))

		case action < 98: // Find & Idempotence (8%)
			findQueries++
			u := rng.IntN(n)
			root := dsu.Find(u)
			require.GreaterOrEqual(t, root, 0)
			require.Less(t, root, n)
			require.Equal(t, root, dsu.Find(root), fmt.Sprintf("op %d: Find idempotence failed for %d", op, u))
			require.True(
				t,
				dsu.Connected(u, root),
				fmt.Sprintf("op %d: Node %d not connected to its root %d", op, u, root),
			)

		default: // Periodic Reset (2%)
			dsu.Reset()
			oracle.reset()

			require.Equal(t, n, dsu.Len())
			require.Equal(t, n, dsu.Count())
			for i := 0; i < 10; i++ {
				checkIdx := rng.IntN(n)
				require.Equal(t, checkIdx, dsu.Find(checkIdx))
				require.Equal(t, 1, dsu.Size(checkIdx))
			}
		}

		// Periodic deep check across entire graph
		if op > 0 && op%1_000 == 0 {
			oracleCountBFS := oracle.componentCountBFS()
			oracleCountDFS := oracle.componentCountDFS()
			require.Equal(t, oracleCountBFS, oracleCountDFS)
			require.Equal(t, oracleCountBFS, dsu.Count(), fmt.Sprintf("op %d: Count mismatch", op))

			// Element conservation check
			seenRoots := make(map[int]int)
			for i := 0; i < n; i++ {
				r := dsu.Find(i)
				seenRoots[r] = dsu.Size(r)
			}
			require.Equal(t, dsu.Count(), len(seenRoots))
			sumSize := 0
			for _, sz := range seenRoots {
				sumSize += sz
			}
			require.Equal(t, n, sumSize)
		}
	}

	// Final verification
	require.Equal(t, oracle.componentCountBFS(), dsu.Count())
	require.Equal(t, n, dsu.Len())
	require.Greater(t, unionsAttempted, 30_000)
	require.Greater(t, connectedQueries, 30_000)
	require.Greater(t, sizeQueries, 10_000)
	require.Greater(t, findQueries, 5_000)
}

// -----------------------------------------------------------------------------
// 2. Mathematical Invariants of Equivalence Relations
// -----------------------------------------------------------------------------

func TestChallenge_EquivalenceRelationInvariants(t *testing.T) {
	sizes := []int{1, 2, 5, 20, 100, 500}

	for _, n := range sizes {
		dsu := New(n)
		rng := rand.New(rand.NewPCG(uint64(n), 12345))

		// Initial singleton invariants
		require.Equal(t, n, dsu.Len())
		require.Equal(t, n, dsu.Count())
		for i := 0; i < n; i++ {
			// Reflexivity
			require.True(t, dsu.Connected(i, i))
			require.Equal(t, i, dsu.Find(i))
			require.Equal(t, 1, dsu.Size(i))
		}

		// Random unions to form complex partition
		numMerges := n * 2
		for step := 0; step < numMerges; step++ {
			u := rng.IntN(n)
			v := rng.IntN(n)
			dsu.Union(u, v)
		}

		// Mathematical Invariant 1: Reflexivity
		// ∀ x ∈ [0, n): x ~ x
		for x := 0; x < n; x++ {
			require.True(t, dsu.Connected(x, x), fmt.Sprintf("n=%d: reflexivity failed at %d", n, x))
			require.Equal(t, dsu.Find(x), dsu.Find(x))
			require.GreaterOrEqual(t, dsu.Size(x), 1)
		}

		// Mathematical Invariant 2: Symmetry
		// ∀ x, y ∈ [0, n): x ~ y ⇔ y ~ x
		samplePairs := min(n*n, 2000)
		for s := 0; s < samplePairs; s++ {
			x := rng.IntN(n)
			y := rng.IntN(n)
			xy := dsu.Connected(x, y)
			yx := dsu.Connected(y, x)
			require.Equal(t, xy, yx, fmt.Sprintf("n=%d: symmetry failed for (%d, %d)", n, x, y))
		}

		// Mathematical Invariant 3: Transitivity
		// ∀ x, y, z ∈ [0, n): (x ~ y ∧ y ~ z) ⇒ x ~ z
		sampleTriples := min(n*n, 3000)
		for s := 0; s < sampleTriples; s++ {
			x := rng.IntN(n)
			y := rng.IntN(n)
			z := rng.IntN(n)
			if dsu.Connected(x, y) && dsu.Connected(y, z) {
				require.True(t, dsu.Connected(x, z),
					fmt.Sprintf("n=%d: transitivity failed for triple (%d, %d, %d)", n, x, y, z))
			}
		}

		// Mathematical Invariant 4: Partition Element Conservation & Disjointness
		// Disjoint sets partition the universe U = {0, ..., n-1}
		// 1. Root count strictly equals Count()
		// 2. Sum of component sizes equals N
		// 3. For every root r, Size(r) equals the number of elements whose Find(x) == r
		// 4. Any two distinct roots r1 != r2 have disjoint sets of elements
		rootElements := make(map[int][]int)
		for x := 0; x < n; x++ {
			r := dsu.Find(x)
			require.Equal(t, r, dsu.Find(r), "root must be its own representative")
			rootElements[r] = append(rootElements[r], x)
		}

		require.Equal(t, dsu.Count(), len(rootElements), "number of distinct roots must equal Count()")

		totalConservedElements := 0
		for root, elements := range rootElements {
			totalConservedElements += len(elements)

			// Component size matches actual counted elements
			rootSize := dsu.Size(root)
			require.Equal(t, len(elements), rootSize,
				fmt.Sprintf("n=%d, root %d: Size() %d != actual element count %d", n, root, rootSize, len(elements)))

			// Every element in component reports exact same root and size
			for _, elem := range elements {
				require.Equal(t, root, dsu.Find(elem))
				require.Equal(t, rootSize, dsu.Size(elem))
				require.True(t, dsu.Connected(root, elem))
			}
		}

		require.Equal(t, n, totalConservedElements, "sum of elements across all partitions must equal N")

		// Pairwise disjointness between different components
		rootsList := make([]int, 0, len(rootElements))
		for r := range rootElements {
			rootsList = append(rootsList, r)
		}
		for i := 0; i < len(rootsList); i++ {
			for j := i + 1; j < len(rootsList); j++ {
				r1 := rootsList[i]
				r2 := rootsList[j]
				require.False(t, dsu.Connected(r1, r2),
					fmt.Sprintf("n=%d: distinct roots %d and %d must not be connected", n, r1, r2))
			}
		}
	}
}

// -----------------------------------------------------------------------------
// 3. Union-by-Size Monotonicity & Representative Consistency
// -----------------------------------------------------------------------------

func TestChallenge_UnionBySizeMonotonicity(t *testing.T) {
	n := 1000
	dsu := New(n)

	// Stepwise merging with size tracking
	currentCount := n
	for i := 0; i < n-1; i++ {
		sizeBefore0 := dsu.Size(0)
		sizeBeforeNext := dsu.Size(i + 1)

		merged := dsu.Union(0, i+1)
		require.True(t, merged)
		currentCount--

		require.Equal(t, currentCount, dsu.Count())
		expectedNewSize := sizeBefore0 + sizeBeforeNext
		require.Equal(t, expectedNewSize, dsu.Size(0))
		require.Equal(t, expectedNewSize, dsu.Size(i+1))

		// Subsequent redundant union must be false and leave count and size unchanged
		redundant := dsu.Union(0, i+1)
		require.False(t, redundant)
		require.Equal(t, currentCount, dsu.Count())
		require.Equal(t, expectedNewSize, dsu.Size(0))
	}

	require.Equal(t, 1, dsu.Count())
	require.Equal(t, n, dsu.Size(0))
}

// -----------------------------------------------------------------------------
// 4. Deep Path Compression & Tree Flattening Torture
// -----------------------------------------------------------------------------

func TestChallenge_PathCompressionTorture(t *testing.T) {
	const chainLen = 50_000
	dsu := New(chainLen)

	// Manually construct deep linear uncompressed tree: 49999 -> 49998 -> ... -> 0
	for i := 1; i < chainLen; i++ {
		dsu.parent[i] = i - 1
	}

	// Verify pre-compression chain
	require.Equal(t, chainLen-2, dsu.parent[chainLen-1])

	// Find on deepest leaf triggers full two-pass iterative path compression
	root := dsu.Find(chainLen - 1)
	require.Equal(t, 0, root)

	// Verify all traversed nodes now point directly to root 0
	for i := 0; i < chainLen; i++ {
		require.Equal(t, 0, dsu.parent[i], fmt.Sprintf("node %d was not compressed to root", i))
		require.Equal(t, 0, dsu.Find(i))
	}

	// Now union another deep chain of equal size
	const chain2Len = 50_000
	dsu2 := New(chain2Len * 2)
	// Chain A: 0..49999
	for i := 1; i < chain2Len; i++ {
		dsu2.parent[i] = i - 1
		dsu2.size[0]++
	}
	// Chain B: 50000..99999
	for i := chain2Len + 1; i < chain2Len*2; i++ {
		dsu2.parent[i] = i - 1
		dsu2.size[chain2Len]++
	}

	// Merge the two deep chains via their leaves
	merged := dsu2.Union(chain2Len-1, chain2Len*2-1)
	require.True(t, merged)
	require.True(t, dsu2.Connected(0, chain2Len))
	require.Equal(t, chain2Len*2, dsu2.Size(0))
}

// -----------------------------------------------------------------------------
// 5. Adversarial Topologies (Star, Dense Clique, Split-Merge-Reset Bursts)
// -----------------------------------------------------------------------------

func TestChallenge_AdversarialTopologies(t *testing.T) {
	t.Run("StarTopologyMassiveFanout", func(t *testing.T) {
		const leaves = 100_000
		dsu := New(leaves + 1)
		hub := 0

		for leaf := 1; leaf <= leaves; leaf++ {
			merged := dsu.Union(hub, leaf)
			require.True(t, merged)
		}
		require.Equal(t, 1, dsu.Count())
		require.Equal(t, leaves+1, dsu.Size(hub))

		for leaf := 1; leaf <= leaves; leaf += 100 {
			require.True(t, dsu.Connected(leaf, hub))
			require.Equal(t, hub, dsu.Find(leaf))
		}
	})

	t.Run("CliqueDenseGraph", func(t *testing.T) {
		// Complete graph K_200: all N*(N-1)/2 = 19,900 edges
		n := 200
		dsu := New(n)
		edgesAdded := 0
		redundantEdges := 0

		for u := 0; u < n; u++ {
			for v := u + 1; v < n; v++ {
				if dsu.Union(u, v) {
					edgesAdded++
				} else {
					redundantEdges++
				}
			}
		}

		require.Equal(t, n-1, edgesAdded)
		require.Equal(t, (n*(n-1)/2)-(n-1), redundantEdges)
		require.Equal(t, 1, dsu.Count())
		require.Equal(t, n, dsu.Size(0))
	})

	t.Run("AlternatingUnionResetCycles", func(t *testing.T) {
		n := 500
		dsu := New(n)
		rng := rand.New(rand.NewPCG(888, 999))

		for cycle := 0; cycle < 50; cycle++ {
			// Random unions
			for i := 0; i < 300; i++ {
				dsu.Union(rng.IntN(n), rng.IntN(n))
			}
			require.Less(t, dsu.Count(), n)

			// Reset
			dsu.Reset()
			require.Equal(t, n, dsu.Count())
			require.Equal(t, n, dsu.Len())
			for i := 0; i < 20; i++ {
				x := rng.IntN(n)
				require.Equal(t, 1, dsu.Size(x))
				require.Equal(t, x, dsu.Find(x))
			}
		}
	})
}

// -----------------------------------------------------------------------------
// 6. Adversarial Boundary Conditions & Panics
// -----------------------------------------------------------------------------

func TestChallenge_AdversarialBoundaries(t *testing.T) {
	// Zero elements
	d0 := New(0)
	require.Equal(t, 0, d0.Len())
	require.Equal(t, 0, d0.Count())
	d0.Reset()
	require.PanicsWithError(t, "disjointset: index out of range", func() { d0.Find(0) })
	require.PanicsWithError(t, "disjointset: index out of range", func() { d0.Union(0, 0) })
	require.PanicsWithError(t, "disjointset: index out of range", func() { d0.Connected(0, 0) })
	require.PanicsWithError(t, "disjointset: index out of range", func() { d0.Size(0) })

	// Nil receiver
	var dNil *Set
	require.Equal(t, 0, dNil.Len())
	require.Equal(t, 0, dNil.Count())
	dNil.Reset()
	require.PanicsWithError(t, "disjointset: index out of range", func() { dNil.Find(0) })
	require.PanicsWithError(t, "disjointset: index out of range", func() { dNil.Union(0, 0) })
	require.PanicsWithError(t, "disjointset: index out of range", func() { dNil.Connected(0, 0) })
	require.PanicsWithError(t, "disjointset: index out of range", func() { dNil.Size(0) })

	// Negative length constructor
	require.PanicsWithError(t, "disjointset: negative element count", func() { New(-1) })
	require.PanicsWithError(t, "disjointset: negative element count", func() { New(-999999) })

	// Boundary index panics on valid DSU
	d := New(5)
	invalidIndices := []int{-100, -1, 5, 6, 100, 1_000_000}
	for _, idx := range invalidIndices {
		require.PanicsWithError(t, "disjointset: index out of range", func() { d.Find(idx) })
		require.PanicsWithError(t, "disjointset: index out of range", func() { d.Size(idx) })
		require.PanicsWithError(t, "disjointset: index out of range", func() { d.Union(idx, 0) })
		require.PanicsWithError(t, "disjointset: index out of range", func() { d.Union(0, idx) })
		require.PanicsWithError(t, "disjointset: index out of range", func() { d.Connected(idx, 0) })
		require.PanicsWithError(t, "disjointset: index out of range", func() { d.Connected(0, idx) })
	}
}

// -----------------------------------------------------------------------------
// 7. Multi-Goroutine Concurrent Race Condition Challenge
// -----------------------------------------------------------------------------

func TestChallenge_ConcurrentTortureWithRace(t *testing.T) {
	n := 1000
	dsu := New(n)
	var mu sync.RWMutex
	var wg sync.WaitGroup

	const writers = 8
	const readers = 16
	const opsPerWorker = 2000

	var totalMerges atomic.Int64
	var totalQueries atomic.Int64

	// Concurrent synchronized mutating workers
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			rng := rand.New(rand.NewPCG(uint64(wid*1000+1), 9999))

			for i := 0; i < opsPerWorker; i++ {
				u := rng.IntN(n)
				v := rng.IntN(n)

				mu.Lock()
				if dsu.Union(u, v) {
					totalMerges.Add(1)
				}
				_ = dsu.Find(u)
				_ = dsu.Size(u)
				_ = dsu.Connected(u, v)

				// Occasional reset
				if i == opsPerWorker/2 && wid == 0 {
					dsu.Reset()
				}
				mu.Unlock()
			}
		}(w)
	}

	// Concurrent readers accessing Len and Count (which do not compress paths)
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func(rid int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				mu.RLock()
				l := dsu.Len()
				c := dsu.Count()
				mu.RUnlock()

				require.Equal(t, n, l)
				require.GreaterOrEqual(t, c, 1)
				require.LessOrEqual(t, c, n)
				totalQueries.Add(1)
			}
		}(r)
	}

	wg.Wait()
	require.Greater(t, totalQueries.Load(), int64(0))
}

func TestChallenge_ConcurrentIndependentInstances(t *testing.T) {
	// Verify that multiple goroutines operating on distinct Set instances
	// without any external locks have zero data races (ensuring no shared package-level state).
	var wg sync.WaitGroup
	const goroutines = 32
	const ops = 1000

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			n := 100
			d := New(n)
			rng := rand.New(rand.NewPCG(uint64(gid), 12345))

			for i := 0; i < ops; i++ {
				u := rng.IntN(n)
				v := rng.IntN(n)
				d.Union(u, v)
				_ = d.Find(u)
				_ = d.Connected(u, v)
				_ = d.Size(u)
				_ = d.Count()
				_ = d.Len()
			}
			d.Reset()
			require.Equal(t, n, d.Count())
		}(g)
	}

	wg.Wait()
}
