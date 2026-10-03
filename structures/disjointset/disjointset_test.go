// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disjointset

import (
	"math/rand/v2"
	"sort"
	"sync"
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
	var d *DisjointSet

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
	var d DisjointSet

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
