// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list_test

import (
	"math/rand"
	"slices"
	"testing"

	list "github.com/lemon4ksan/foundation/structures/linkedlist"
)

func checkListLen[T any](t *testing.T, l *list.List[T], expectedLen int) {
	t.Helper()
	if l.Len() != expectedLen {
		t.Fatalf("l.Len() = %d, want %d", l.Len(), expectedLen)
	}
}

func checkListPointers[T comparable](t *testing.T, l *list.List[T], expected []T) {
	t.Helper()
	checkListLen(t, l, len(expected))

	// Forward check
	var forward []T
	for e := l.Front(); e != nil; e = e.Next() {
		if e.List() != l {
			t.Fatalf("element %v points to incorrect list %v, want %v", e.Value, e.List(), l)
		}
		forward = append(forward, e.Value)
	}
	if !slices.Equal(forward, expected) {
		t.Fatalf("Forward traversal = %v, want %v", forward, expected)
	}

	// Backward check
	var backward []T
	for e := l.Back(); e != nil; e = e.Prev() {
		backward = append(backward, e.Value)
	}
	slices.Reverse(backward)
	if !slices.Equal(backward, expected) {
		t.Fatalf("Backward traversal reversed = %v, want %v", backward, expected)
	}
}

func TestListBasicOperations(t *testing.T) {
	l := list.New[int]()
	checkListPointers(t, l, nil)

	// PushFront and PushBack
	e1 := l.PushFront(1)
	checkListPointers(t, l, []int{1})
	if e1.List() != l {
		t.Fatalf("e1.List() != l")
	}

	e2 := l.PushBack(2)
	checkListPointers(t, l, []int{1, 2})

	e0 := l.PushFront(0)
	checkListPointers(t, l, []int{0, 1, 2})

	// InsertBefore / InsertAfter
	e1_5 := l.InsertAfter(15, e1)
	checkListPointers(t, l, []int{0, 1, 15, 2})

	l.InsertBefore(5, e1)
	checkListPointers(t, l, []int{0, 5, 1, 15, 2})

	// Remove
	val := l.Remove(e1_5)
	if val != 15 {
		t.Fatalf("Remove returned %d, want 15", val)
	}
	checkListPointers(t, l, []int{0, 5, 1, 2})

	// MoveToFront
	l.MoveToFront(e2)
	checkListPointers(t, l, []int{2, 0, 5, 1})

	// MoveToBack
	l.MoveToBack(e0)
	checkListPointers(t, l, []int{2, 5, 1, 0})

	// MoveBefore
	l.MoveBefore(e0, e2)
	checkListPointers(t, l, []int{0, 2, 5, 1})

	// MoveAfter
	l.MoveAfter(e2, e1)
	checkListPointers(t, l, []int{0, 5, 1, 2})

	// Clear through Init
	l.Init()
	checkListPointers(t, l, nil)
}

func TestListZeroValue(t *testing.T) {
	var l list.List[int]
	if l.Len() != 0 {
		t.Fatalf("zero value list Len() = %d, want 0", l.Len())
	}
	if l.Front() != nil {
		t.Fatalf("zero value list Front() != nil")
	}
	if l.Back() != nil {
		t.Fatalf("zero value list Back() != nil")
	}

	// Iterators on zero value list should not execute
	for range l.Values() {
		t.Fatalf("zero value list Values() yielded element")
	}
	for range l.All() {
		t.Fatalf("zero value list All() yielded element")
	}
	for range l.Backward() {
		t.Fatalf("zero value list Backward() yielded element")
	}

	// First insertion lazily initializes the list
	e := l.PushBack(100)
	if e == nil || e.Value != 100 {
		t.Fatalf("PushBack on zero value list failed")
	}
	checkListPointers(t, &l, []int{100})

	// Lazy init on PushFront
	var l2 list.List[string]
	eStr := l2.PushFront("init")
	if eStr == nil || eStr.Value != "init" {
		t.Fatalf("PushFront on zero value list failed")
	}
	checkListPointers(t, &l2, []string{"init"})
}

func TestListCrossListSafety(t *testing.T) {
	l1 := list.New[int]()
	l2 := list.New[int]()

	e1 := l1.PushBack(1)
	e2 := l2.PushBack(2)

	// InsertBefore with element from another list must return nil and leave list untouched
	if res := l1.InsertBefore(100, e2); res != nil {
		t.Fatalf("InsertBefore with element from another list returned non-nil %v", res)
	}
	checkListPointers(t, l1, []int{1})

	// InsertAfter with element from another list must return nil
	if res := l1.InsertAfter(200, e2); res != nil {
		t.Fatalf("InsertAfter with element from another list returned non-nil %v", res)
	}
	checkListPointers(t, l1, []int{1})

	// MoveToFront with element from another list must be a no-op
	l1.MoveToFront(e2)
	checkListPointers(t, l1, []int{1})
	checkListPointers(t, l2, []int{2})

	// MoveToBack with element from another list must be a no-op
	l1.MoveToBack(e2)
	checkListPointers(t, l1, []int{1})
	checkListPointers(t, l2, []int{2})

	// MoveBefore with element from another list must be a no-op
	l1.MoveBefore(e1, e2)
	checkListPointers(t, l1, []int{1})
	l1.MoveBefore(e2, e1)
	checkListPointers(t, l1, []int{1})

	// MoveAfter with element from another list must be a no-op
	l1.MoveAfter(e1, e2)
	checkListPointers(t, l1, []int{1})
	l1.MoveAfter(e2, e1)
	checkListPointers(t, l1, []int{1})

	// Remove element from another list returns value but does not remove from either list
	remVal := l1.Remove(e2)
	if remVal != 2 {
		t.Fatalf("Remove returned %d, want 2", remVal)
	}
	checkListPointers(t, l1, []int{1})
	checkListPointers(t, l2, []int{2})
	if e2.List() != l2 {
		t.Fatalf("e2 was detached unexpectedly from l2")
	}
}

func TestListSelfOperationsAndNoops(t *testing.T) {
	// Single element list operations
	l := list.New[int]()
	e := l.PushBack(1)

	l.MoveToFront(e)
	checkListPointers(t, l, []int{1})

	l.MoveToBack(e)
	checkListPointers(t, l, []int{1})

	l.MoveBefore(e, e)
	checkListPointers(t, l, []int{1})

	l.MoveAfter(e, e)
	checkListPointers(t, l, []int{1})

	// Multi-element list operations
	a := l.PushBack(2)
	b := l.PushBack(3)
	checkListPointers(t, l, []int{1, 2, 3})

	// Moving front element to front is a no-op
	l.MoveToFront(e)
	checkListPointers(t, l, []int{1, 2, 3})

	// Moving back element to back is a no-op
	l.MoveToBack(b)
	checkListPointers(t, l, []int{1, 2, 3})

	// Moving element relative to itself is a no-op
	l.MoveBefore(a, a)
	checkListPointers(t, l, []int{1, 2, 3})
	l.MoveAfter(a, a)
	checkListPointers(t, l, []int{1, 2, 3})

	// MoveBefore where element is already before mark triggers move early return (e.idx == at)
	l.MoveBefore(e, a)
	checkListPointers(t, l, []int{1, 2, 3})

	// MoveAfter where element is already after mark
	l.MoveAfter(b, a)
	checkListPointers(t, l, []int{1, 2, 3})
}

func TestListPushListConcatenation(t *testing.T) {
	l1 := list.NewWithCapacity[string](12)
	l1.PushBack("a")
	l1.PushBack("b")

	l2 := list.NewWithCapacity[string](6)
	l2.PushBack("c")
	l2.PushBack("d")

	// PushBackList with non-empty list
	l1.PushBackList(l2)
	checkListPointers(t, l1, []string{"a", "b", "c", "d"})
	checkListPointers(t, l2, []string{"c", "d"})

	// PushFrontList with non-empty list
	l3 := list.NewWithCapacity[string](6)
	l3.PushBack("x")
	l3.PushBack("y")

	l1.PushFrontList(l3)
	checkListPointers(t, l1, []string{"x", "y", "a", "b", "c", "d"})

	// Empty list concatenation is a no-op
	emptyList := list.New[string]()
	l1.PushBackList(emptyList)
	checkListPointers(t, l1, []string{"x", "y", "a", "b", "c", "d"})
	l1.PushFrontList(emptyList)
	checkListPointers(t, l1, []string{"x", "y", "a", "b", "c", "d"})

	// Self concatenation at back
	lSelf := list.NewWithCapacity[int](10)
	lSelf.PushBack(1)
	lSelf.PushBack(2)
	lSelf.PushBackList(lSelf)
	checkListPointers(t, lSelf, []int{1, 2, 1, 2})

	// Self concatenation at front
	lSelfFront := list.NewWithCapacity[int](10)
	lSelfFront.PushBack(1)
	lSelfFront.PushBack(2)
	lSelfFront.PushFrontList(lSelfFront)
	checkListPointers(t, lSelfFront, []int{1, 2, 1, 2})
}

func TestListIterators(t *testing.T) {
	// Empty list iterators
	emptyList := list.New[int]()
	for range emptyList.Values() {
		t.Fatalf("emptyList.Values() yielded item")
	}
	for range emptyList.All() {
		t.Fatalf("emptyList.All() yielded item")
	}
	for range emptyList.Backward() {
		t.Fatalf("emptyList.Backward() yielded item")
	}

	l := list.NewWithCapacity[int](8)
	for i := 1; i <= 5; i++ {
		l.PushBack(i * 10)
	}

	// Values()
	var vals []int
	for v := range l.Values() {
		vals = append(vals, v)
	}
	if !slices.Equal(vals, []int{10, 20, 30, 40, 50}) {
		t.Fatalf("Values() = %v, want [10 20 30 40 50]", vals)
	}

	// All()
	var indices []int
	var allVals []int
	for idx, v := range l.All() {
		indices = append(indices, idx)
		allVals = append(allVals, v)
	}
	if !slices.Equal(indices, []int{0, 1, 2, 3, 4}) || !slices.Equal(allVals, []int{10, 20, 30, 40, 50}) {
		t.Fatalf("All() = idx %v, vals %v", indices, allVals)
	}

	// Backward()
	var bwd []int
	for v := range l.Backward() {
		bwd = append(bwd, v)
	}
	if !slices.Equal(bwd, []int{50, 40, 30, 20, 10}) {
		t.Fatalf("Backward() = %v, want [50 40 30 20 10]", bwd)
	}

	// Early break from Values()
	count := 0
	for range l.Values() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("Values early break count = %d, want 2", count)
	}

	// Early break from All()
	countAll := 0
	for range l.All() {
		countAll++
		if countAll == 3 {
			break
		}
	}
	if countAll != 3 {
		t.Fatalf("All early break count = %d, want 3", countAll)
	}

	// Early break from Backward()
	countBwd := 0
	for range l.Backward() {
		countBwd++
		if countBwd == 2 {
			break
		}
	}
	if countBwd != 2 {
		t.Fatalf("Backward early break count = %d, want 2", countBwd)
	}
}

func TestListTableDriven(t *testing.T) {
	type step struct {
		op   string
		val  int
		want []int
	}

	tests := []struct {
		name  string
		steps []step
	}{
		{
			name: "FIFO queue operations",
			steps: []step{
				{op: "push_back", val: 10, want: []int{10}},
				{op: "push_back", val: 20, want: []int{10, 20}},
				{op: "push_back", val: 30, want: []int{10, 20, 30}},
				{op: "pop_front", want: []int{20, 30}},
				{op: "pop_front", want: []int{30}},
				{op: "pop_front", want: []int{}},
			},
		},
		{
			name: "LIFO stack operations",
			steps: []step{
				{op: "push_front", val: 1, want: []int{1}},
				{op: "push_front", val: 2, want: []int{2, 1}},
				{op: "push_front", val: 3, want: []int{3, 2, 1}},
				{op: "pop_front", want: []int{2, 1}},
				{op: "pop_front", want: []int{1}},
				{op: "pop_front", want: []int{}},
			},
		},
		{
			name: "Interleaved insertions and movements",
			steps: []step{
				{op: "push_back", val: 1, want: []int{1}},
				{op: "push_back", val: 3, want: []int{1, 3}},
				{op: "insert_before_3", val: 2, want: []int{1, 2, 3}},
				{op: "insert_after_3", val: 4, want: []int{1, 2, 3, 4}},
				{op: "move_to_front_4", want: []int{4, 1, 2, 3}},
				{op: "move_to_back_1", want: []int{4, 2, 3, 1}},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := list.NewWithCapacity[int](16)
			for idx, s := range tc.steps {
				switch s.op {
				case "push_back":
					l.PushBack(s.val)
				case "push_front":
					l.PushFront(s.val)
				case "pop_front":
					if f := l.Front(); f != nil {
						l.Remove(f)
					}
				case "insert_before_3":
					for e := l.Front(); e != nil; e = e.Next() {
						if e.Value == 3 {
							l.InsertBefore(s.val, e)
							break
						}
					}
				case "insert_after_3":
					for e := l.Front(); e != nil; e = e.Next() {
						if e.Value == 3 {
							l.InsertAfter(s.val, e)
							break
						}
					}
				case "move_to_front_4":
					for e := l.Front(); e != nil; e = e.Next() {
						if e.Value == 4 {
							l.MoveToFront(e)
							break
						}
					}
				case "move_to_back_1":
					for e := l.Front(); e != nil; e = e.Next() {
						if e.Value == 1 {
							l.MoveToBack(e)
							break
						}
					}
				default:
					t.Fatalf("unknown step op %q", s.op)
				}
				checkListPointers(t, l, s.want)
				_ = idx
			}
		})
	}
}

func TestListPropertyBasedRandomOps(t *testing.T) {
	const maxCap = 80
	l := list.NewWithCapacity[int](maxCap)
	var model []int

	rng := rand.New(rand.NewSource(42))

	for step := range 600 {
		op := rng.Intn(9)
		switch op {
		case 0: // PushFront
			if len(model) < maxCap {
				val := rng.Intn(1000)
				l.PushFront(val)
				model = append([]int{val}, model...)
			}
		case 1: // PushBack
			if len(model) < maxCap {
				val := rng.Intn(1000)
				l.PushBack(val)
				model = append(model, val)
			}
		case 2: // InsertBefore
			if len(model) > 0 && len(model) < maxCap {
				targetIdx := rng.Intn(len(model))
				val := rng.Intn(1000)
				curr := l.Front()
				for range targetIdx {
					curr = curr.Next()
				}
				l.InsertBefore(val, curr)
				model = slices.Insert(model, targetIdx, val)
			}
		case 3: // InsertAfter
			if len(model) > 0 && len(model) < maxCap {
				targetIdx := rng.Intn(len(model))
				val := rng.Intn(1000)
				curr := l.Front()
				for range targetIdx {
					curr = curr.Next()
				}
				l.InsertAfter(val, curr)
				model = slices.Insert(model, targetIdx+1, val)
			}
		case 4: // MoveToFront
			if len(model) > 1 {
				targetIdx := rng.Intn(len(model))
				curr := l.Front()
				for range targetIdx {
					curr = curr.Next()
				}
				l.MoveToFront(curr)
				val := model[targetIdx]
				model = slices.Delete(model, targetIdx, targetIdx+1)
				model = append([]int{val}, model...)
			}
		case 5: // MoveToBack
			if len(model) > 1 {
				targetIdx := rng.Intn(len(model))
				curr := l.Front()
				for range targetIdx {
					curr = curr.Next()
				}
				l.MoveToBack(curr)
				val := model[targetIdx]
				model = slices.Delete(model, targetIdx, targetIdx+1)
				model = append(model, val)
			}
		case 6: // MoveBefore
			if len(model) > 1 {
				i1 := rng.Intn(len(model))
				i2 := rng.Intn(len(model))
				var e1, e2 *list.Element[int]
				curr := l.Front()
				for i := 0; i < len(model); i++ {
					if i == i1 {
						e1 = curr
					}
					if i == i2 {
						e2 = curr
					}
					curr = curr.Next()
				}
				l.MoveBefore(e1, e2)
				if i1 != i2 {
					val := model[i1]
					model = slices.Delete(model, i1, i1+1)
					insertPos := i2
					if i1 < i2 {
						insertPos = i2 - 1
					}
					model = slices.Insert(model, insertPos, val)
				}
			}
		case 7: // MoveAfter
			if len(model) > 1 {
				i1 := rng.Intn(len(model))
				i2 := rng.Intn(len(model))
				var e1, e2 *list.Element[int]
				curr := l.Front()
				for i := 0; i < len(model); i++ {
					if i == i1 {
						e1 = curr
					}
					if i == i2 {
						e2 = curr
					}
					curr = curr.Next()
				}
				l.MoveAfter(e1, e2)
				if i1 != i2 {
					val := model[i1]
					model = slices.Delete(model, i1, i1+1)
					insertPos := i2 + 1
					if i1 < i2 {
						insertPos = i2
					}
					model = slices.Insert(model, insertPos, val)
				}
			}
		case 8: // Remove
			if len(model) > 0 {
				targetIdx := rng.Intn(len(model))
				curr := l.Front()
				for range targetIdx {
					curr = curr.Next()
				}
				l.Remove(curr)
				model = slices.Delete(model, targetIdx, targetIdx+1)
			}
		}

		// Periodic verification of all invariants
		if step%20 == 0 || step == 599 {
			checkListPointers(t, l, model)
		}
	}
}

// -----------------------------------------------------------------------------
// Zero-Allocation Benchmarks
// -----------------------------------------------------------------------------

func BenchmarkList_PushBack(b *testing.B) {
	l := list.NewWithCapacity[int](b.N)
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		l.PushBack(i)
		i++
	}
}

func BenchmarkList_PushFront(b *testing.B) {
	l := list.NewWithCapacity[int](b.N)
	b.ReportAllocs()
	b.ResetTimer()

	var i int
	for b.Loop() {
		l.PushFront(i)
		i++
	}
}

func BenchmarkList_PushBackRemove(b *testing.B) {
	l := list.NewWithCapacity[int](16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		e := l.PushBack(42)
		l.Remove(e)
	}
}

func BenchmarkList_PushFrontRemove(b *testing.B) {
	l := list.NewWithCapacity[int](16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		e := l.PushFront(42)
		l.Remove(e)
	}
}

func BenchmarkList_MoveToFrontMoveToBack(b *testing.B) {
	l := list.NewWithCapacity[int](1024)
	for i := range 1024 {
		l.PushBack(i)
	}
	target := l.Back()
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		l.MoveToFront(target)
		l.MoveToBack(target)
	}
}

func BenchmarkList_IteratorsValues(b *testing.B) {
	l := list.NewWithCapacity[int](1000)
	for i := range 1000 {
		l.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sum := 0
		for v := range l.Values() {
			sum += v
		}
		_ = sum
	}
}

func BenchmarkList_IteratorsAll(b *testing.B) {
	l := list.NewWithCapacity[int](1000)
	for i := range 1000 {
		l.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sum := 0
		for idx, v := range l.All() {
			sum += idx + v
		}
		_ = sum
	}
}

func BenchmarkList_IteratorsBackward(b *testing.B) {
	l := list.NewWithCapacity[int](1000)
	for i := range 1000 {
		l.PushBack(i)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		sum := 0
		for v := range l.Backward() {
			sum += v
		}
		_ = sum
	}
}

func TestList_ChunkRecyclingAllocations(t *testing.T) {
	l := list.New[int]()

	allocs := testing.AllocsPerRun(10, func() {
		for i := range 1000 {
			e := l.PushBack(i)
			l.Remove(e)
		}

		e := l.PushBack(1)
		l.Remove(e)
	})

	if allocs > 0 {
		t.Fatalf("expected 0 heap allocations for chunk recycling, got %f allocations/run", allocs)
	}
}

func TestList_MemoryRetention_Init(t *testing.T) {
	type largeObj struct {
		data [1024]byte
	}

	l := list.New[*largeObj]()
	for range 500 {
		l.PushBack(&largeObj{})
	}

	if l.Len() != 500 {
		t.Fatalf("expected 500 items, got %d", l.Len())
	}
	if l.Cap() < 500 {
		t.Fatalf("expected Cap >= 500, got %d", l.Cap())
	}

	// Re-initialize list: must drop all 500 pointer references
	l.Init()
	if l.Len() != 0 {
		t.Fatalf("expected 0 items after Init, got %d", l.Len())
	}

	// ShrinkToFit
	l.ShrinkToFit()
	if l.Cap() != 256 {
		t.Fatalf("expected Cap == 256 after ShrinkToFit, got %d", l.Cap())
	}
}

func TestList_NilSafety(t *testing.T) {
	l := list.New[int]()
	l.PushBack(1)
	l.PushBack(2)

	// Remove nil
	if v := l.Remove(nil); v != 0 {
		t.Fatalf("expected 0 from Remove(nil), got %d", v)
	}

	// Move methods with nil
	l.MoveToFront(nil)
	l.MoveToBack(nil)
	l.MoveBefore(nil, nil)
	l.MoveAfter(nil, nil)

	// Insert methods with nil mark
	if e := l.InsertBefore(99, nil); e != nil {
		t.Fatal("expected nil from InsertBefore with nil mark")
	}
	if e := l.InsertAfter(99, nil); e != nil {
		t.Fatal("expected nil from InsertAfter with nil mark")
	}

	// Push*List with nil
	l.PushBackList(nil)
	l.PushFrontList(nil)

	if l.Len() != 2 {
		t.Fatalf("expected len 2, got %d", l.Len())
	}
}

func TestList_IteratorZeroAllocs(t *testing.T) {
	l := list.New[int]()
	for i := range 100 {
		l.PushBack(i)
	}

	allocsValues := testing.AllocsPerRun(10, func() {
		sum := 0
		for v := range l.Values() {
			sum += v
		}
		_ = sum
	})
	if allocsValues > 0 {
		t.Errorf("Values() had %f allocs, expected 0", allocsValues)
	}

	allocsBackward := testing.AllocsPerRun(10, func() {
		sum := 0
		for v := range l.Backward() {
			sum += v
		}
		_ = sum
	})
	if allocsBackward > 0 {
		t.Errorf("Backward() had %f allocs, expected 0", allocsBackward)
	}

	allocsAll := testing.AllocsPerRun(10, func() {
		sum := 0
		for idx, v := range l.All() {
			sum += idx + v
		}
		_ = sum
	})
	if allocsAll > 0 {
		t.Errorf("All() had %f allocs, expected 0", allocsAll)
	}
}
