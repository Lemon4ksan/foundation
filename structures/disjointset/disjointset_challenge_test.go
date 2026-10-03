// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package disjointset

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

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
	var dNil *DisjointSet
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
	// Verify that multiple goroutines operating on distinct DisjointSet instances
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
