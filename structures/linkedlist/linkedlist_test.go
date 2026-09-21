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

func TestListCapacityExhaustionAndRecycling(t *testing.T) {
	// Create list with bounded capacity of 3 items
	l := list.NewCapacity[int](3)
	e1 := l.PushBack(10)
	e2 := l.PushBack(20)
	e3 := l.PushBack(30)
	checkListPointers(t, l, []int{10, 20, 30})

	// 4th insertion must panic due to capacity exhaustion
	var didPanic bool
	var panicMsg any
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				panicMsg = r
			}
		}()
		l.PushBack(40)
	}()

	if !didPanic {
		t.Fatalf("expected capacity exhaustion panic, but PushBack succeeded")
	}
	expectedMsg := "linkedlist: array-based list exceeded capacity and cannot safely resize without invalidating pointers"
	if panicMsg != expectedMsg {
		t.Fatalf("panic message = %q, want %q", panicMsg, expectedMsg)
	}

	// Remove middle element: releases slot into free list
	remVal := l.Remove(e2)
	if remVal != 20 {
		t.Fatalf("Remove returned %d, want 20", remVal)
	}
	checkListPointers(t, l, []int{10, 30})

	// Now insert 40: should reuse the recycled free slot without panicking
	e4 := l.PushBack(40)
	if e4.Value != 40 {
		t.Fatalf("e4.Value = %d, want 40", e4.Value)
	}
	checkListPointers(t, l, []int{10, 30, 40})

	// Remove all elements in various orders (head, tail, middle)
	l.Remove(e1)
	checkListPointers(t, l, []int{30, 40})
	l.Remove(e4)
	checkListPointers(t, l, []int{30})
	l.Remove(e3)
	checkListPointers(t, l, nil)

	// Re-fill recycled list to its capacity
	l.PushBack(1)
	l.PushBack(2)
	l.PushBack(3)
	checkListPointers(t, l, []int{1, 2, 3})
}

func TestListDefaultCapacityExhaustion(t *testing.T) {
	// Default constructor creates cap=16 (1 sentinel + 15 user elements)
	l := list.New[int]()
	for i := 0; i < 15; i++ {
		l.PushBack(i)
	}
	if l.Len() != 15 {
		t.Fatalf("l.Len() = %d, want 15", l.Len())
	}

	// 16th insertion must trigger capacity exhaustion panic
	var didPanic bool
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		l.PushBack(999)
	}()

	if !didPanic {
		t.Fatalf("expected capacity panic on 16th insert into default New() list")
	}
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

func TestListRemovedElementHandling(t *testing.T) {
	l := list.New[int]()
	e := l.PushBack(42)

	val := l.Remove(e)
	if val != 42 {
		t.Fatalf("Remove returned %d, want 42", val)
	}

	if e.List() != nil {
		t.Fatalf("removed element List() != nil")
	}
	if e.Next() != nil {
		t.Fatalf("removed element Next() != nil")
	}
	if e.Prev() != nil {
		t.Fatalf("removed element Prev() != nil")
	}

	// Repeated removal is a safe no-op
	repeatVal := l.Remove(e)
	if repeatVal != 42 {
		t.Fatalf("repeated Remove returned %d, want 42", repeatVal)
	}
	checkListLen(t, l, 0)

	// Inserting relative to detached element returns nil
	if res := l.InsertBefore(100, e); res != nil {
		t.Fatalf("InsertBefore on detached element returned %v, want nil", res)
	}
	if res := l.InsertAfter(200, e); res != nil {
		t.Fatalf("InsertAfter on detached element returned %v, want nil", res)
	}

	// Moving detached element is a no-op
	l.MoveToFront(e)
	l.MoveToBack(e)
	checkListLen(t, l, 0)
}

func TestListPushListConcatenation(t *testing.T) {
	l1 := list.NewCapacity[string](12)
	l1.PushBack("a")
	l1.PushBack("b")

	l2 := list.NewCapacity[string](6)
	l2.PushBack("c")
	l2.PushBack("d")

	// PushBackList with non-empty list
	l1.PushBackList(l2)
	checkListPointers(t, l1, []string{"a", "b", "c", "d"})
	checkListPointers(t, l2, []string{"c", "d"})

	// PushFrontList with non-empty list
	l3 := list.NewCapacity[string](6)
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
	lSelf := list.NewCapacity[int](10)
	lSelf.PushBack(1)
	lSelf.PushBack(2)
	lSelf.PushBackList(lSelf)
	checkListPointers(t, lSelf, []int{1, 2, 1, 2})

	// Self concatenation at front
	lSelfFront := list.NewCapacity[int](10)
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

	l := list.NewCapacity[int](8)
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
			l := list.NewCapacity[int](16)
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
	l := list.NewCapacity[int](maxCap)
	var model []int

	rng := rand.New(rand.NewSource(42))

	for step := 0; step < 600; step++ {
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
				for i := 0; i < targetIdx; i++ {
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
				for i := 0; i < targetIdx; i++ {
					curr = curr.Next()
				}
				l.InsertAfter(val, curr)
				model = slices.Insert(model, targetIdx+1, val)
			}
		case 4: // MoveToFront
			if len(model) > 1 {
				targetIdx := rng.Intn(len(model))
				curr := l.Front()
				for i := 0; i < targetIdx; i++ {
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
				for i := 0; i < targetIdx; i++ {
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
				for i := 0; i < targetIdx; i++ {
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
	l := list.NewCapacity[int](b.N)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.PushBack(i)
	}
}

func BenchmarkList_PushFront(b *testing.B) {
	l := list.NewCapacity[int](b.N)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		l.PushFront(i)
	}
}

func BenchmarkList_PushBackRemove(b *testing.B) {
	l := list.NewCapacity[int](16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		e := l.PushBack(42)
		l.Remove(e)
	}
}

func BenchmarkList_PushFrontRemove(b *testing.B) {
	l := list.NewCapacity[int](16)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		e := l.PushFront(42)
		l.Remove(e)
	}
}

func BenchmarkList_MoveToFrontMoveToBack(b *testing.B) {
	l := list.NewCapacity[int](1024)
	for i := 0; i < 1024; i++ {
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
	l := list.NewCapacity[int](1000)
	for i := 0; i < 1000; i++ {
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
	l := list.NewCapacity[int](1000)
	for i := 0; i < 1000; i++ {
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
	l := list.NewCapacity[int](1000)
	for i := 0; i < 1000; i++ {
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
