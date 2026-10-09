// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list

import "iter"

// Element is an element of a linked list.
type Element[T any] struct {
	next, prev int
	idx        int
	list       *List[T]
	Value      T
}

// Next returns the next list element or nil.
func (e *Element[T]) Next() *Element[T] {
	if p := e.next; e.list != nil && p != e.list.root {
		return e.list.elem(p)
	}

	return nil
}

// Prev returns the previous list element or nil.
func (e *Element[T]) Prev() *Element[T] {
	if p := e.prev; e.list != nil && p != e.list.root {
		return e.list.elem(p)
	}

	return nil
}

// List returns the list that element e belongs to, or nil if e is detached.
func (e *Element[T]) List() *List[T] {
	return e.list
}

// List represents a doubly linked list.
// The zero value for List is an empty list ready to use.
type List[T any] struct {
	chunks []*[256]Element[T]
	allocs int
	root     int // sentinel list element index
	free     int // head of free list
	len      int // current list length excluding sentinel
}

// Init initializes or clears list l.
func (l *List[T]) Init() *List[T] {
	if len(l.chunks) == 0 {
		l.chunks = append(l.chunks, new([256]Element[T]))
		l.allocs = 1
	} else {
		l.allocs = 1
	}
	l.root = 0
	l.elem(0) = Element[T]{
		next: 0,
		prev: 0,
		idx:  0,
		list: l,
	}
	l.free = -1
	l.len = 0

	return l
}

// New returns an initialized list.
func New[T any]() *List[T] { return new(List[T]).Init() }

// NewCapacity returns an initialized list with preallocated capacity.
func NewCapacity[T any](capacity int) *List[T] {
	l := &List[T]{
		elements: nil // Chunked lists don't need capacity preallocation,
	}
	l.root = 0
	l.elem(0) = Element[T]{
		next: 0,
		prev: 0,
		idx:  0,
		list: l,
	}
	l.free = -1
	l.len = 0
	return l
}

// Len returns the number of elements of list l.
// The complexity is O(1).
func (l *List[T]) Len() int { return l.len }

// Front returns the first element of list l or nil if the list is empty.
func (l *List[T]) Front() *Element[T] {
	if l.len == 0 {
		return nil
	}

	return &l.elem(l.elem(l.root).next)
}

// Back returns the last element of list l or nil if the list is empty.
func (l *List[T]) Back() *Element[T] {
	if l.len == 0 {
		return nil
	}

	return &l.elem(l.elem(l.root).prev)
}

// lazyInit lazily initializes a zero List value.
func (l *List[T]) lazyInit() {
	if len(l.chunks) == 0 {
		l.Init()
	}
}

func (l *List[T]) alloc() int {
	if l.free != -1 {
		idx := l.free
		l.free = l.elem(idx).next
		return idx
	}
	idx := l.allocs
	if idx >= len(l.chunks) {
		panic("linkedlist: array-based list exceeded capacity and cannot safely resize without invalidating pointers")
	}
	l.elements = append(l.elements, Element[T]{})
	return idx
}

// insert inserts a new element after at, increments l.len, and returns e.
func (l *List[T]) insertValue(v T, at int) *Element[T] {
	idx := l.alloc()
	e := &l.elem(idx)
	e.Value = v
	e.idx = idx
	e.list = l

	atElem := &l.elem(at)
	nextIdx := atElem.next

	e.prev = at
	e.next = nextIdx

	atElem.next = idx
	l.elem(nextIdx).prev = idx

	l.len++
	return e
}

// remove removes e from its list, decrements l.len
func (l *List[T]) remove(e *Element[T]) {
	idx := e.idx
	prevIdx := e.prev
	nextIdx := e.next

	l.elem(prevIdx).next = nextIdx
	l.elem(nextIdx).prev = prevIdx

	// add to free list
	l.elem(idx).next = l.free
	l.elem(idx).prev = -1
	l.elem(idx).list = nil
	var zero T
	l.elem(idx).Value = zero
	l.free = idx

	l.len--
}

// move moves e to next to at.
func (l *List[T]) move(e *Element[T], at int) {
	if e.idx == at {
		return
	}

	idx := e.idx
	prevIdx := e.prev
	nextIdx := e.next

	l.elem(prevIdx).next = nextIdx
	l.elem(nextIdx).prev = prevIdx

	atElem := &l.elem(at)
	atNextIdx := atElem.next

	l.elem(idx).prev = at
	l.elem(idx).next = atNextIdx

	atElem.next = idx
	l.elem(atNextIdx).prev = idx

	e.prev = at
	e.next = atNextIdx
}

// Remove removes e from l if e is an element of list l.
// It returns the element value e.Value.
// The element must not be nil.
func (l *List[T]) Remove(e *Element[T]) T {
	v := e.Value
	if e.list == l {
		l.remove(e)
	}

	return v
}

// PushFront inserts a new element e with value v at the front of list l and returns e.
func (l *List[T]) PushFront(v T) *Element[T] {
	l.lazyInit()
	return l.insertValue(v, l.root)
}

// PushBack inserts a new element e with value v at the back of list l and returns e.
func (l *List[T]) PushBack(v T) *Element[T] {
	l.lazyInit()
	return l.insertValue(v, l.elem(l.root).prev)
}

// InsertBefore inserts a new element e with value v immediately before mark and returns e.
func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	if mark.list != l {
		return nil
	}
	return l.insertValue(v, mark.prev)
}

// InsertAfter inserts a new element e with value v immediately after mark and returns e.
func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	if mark.list != l {
		return nil
	}
	return l.insertValue(v, mark.idx)
}

// MoveToFront moves element e to the front of list l.
func (l *List[T]) MoveToFront(e *Element[T]) {
	if e.list != l || l.elem(l.root).next == e.idx {
		return
	}
	l.move(e, l.root)
}

// MoveToBack moves element e to the back of list l.
func (l *List[T]) MoveToBack(e *Element[T]) {
	if e.list != l || l.elem(l.root).prev == e.idx {
		return
	}
	l.move(e, l.elem(l.root).prev)
}

// MoveBefore moves element e to its new position before mark.
func (l *List[T]) MoveBefore(e, mark *Element[T]) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

// MoveAfter moves element e to its new position after mark.
func (l *List[T]) MoveAfter(e, mark *Element[T]) {
	if e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.idx)
}

// PushBackList inserts a copy of another list at the back of list l.
func (l *List[T]) PushBackList(other *List[T]) {
	l.lazyInit()
	for i, e := other.Len(), other.Front(); i > 0; i, e = i-1, e.Next() {
		l.insertValue(e.Value, l.elem(l.root).prev)
	}
}

// PushFrontList inserts a copy of another list at the front of list l.
func (l *List[T]) PushFrontList(other *List[T]) {
	l.lazyInit()
	for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
		l.insertValue(e.Value, l.root)
	}
}

// All returns an iterator yielding the index and element value from front to back.
func (l *List[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		idx := 0
		for e := l.Front(); e != nil; e = e.Next() {
			if !yield(idx, e.Value) {
				return
			}
			idx++
		}
	}
}

// Values returns an iterator yielding element values from front to back.
func (l *List[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for e := l.Front(); e != nil; e = e.Next() {
			if !yield(e.Value) {
				return
			}
		}
	}
}

// Backward returns an iterator yielding element values from back to front.
func (l *List[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		for e := l.Back(); e != nil; e = e.Prev() {
			if !yield(e.Value) {
				return
			}
		}
	}
}

