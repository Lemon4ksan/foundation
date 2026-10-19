// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package list

import "iter"

// Element represents an element of a linked list.
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

// List represents a doubly linked list utilizing chunked memory pooling.
//
// The zero value is an empty list ready for use.
// Detached element pointers may alias newly inserted elements due to memory reuse.
type List[T any] struct {
	chunks []*listChunk[T]
	allocs int
	root   int // sentinel list element index
	free   int // head of free list
	len    int // current list length excluding sentinel
}

// Init initializes or clears the list.
func (l *List[T]) Init() *List[T] {
	for _, chunk := range l.chunks {
		if chunk != nil {
			clear(chunk[:])
		}
	}
	if len(l.chunks) == 0 {
		l.chunks = append(l.chunks, new(listChunk[T]))
	}
	l.allocs = 1
	l.root = 0
	*l.elem(0) = Element[T]{
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

// NewWithCapacity returns an initialized list with preallocated capacity.
func NewWithCapacity[T any](capacity int) *List[T] {
	chunksCount := (capacity + 1 + chunkSize - 1) / chunkSize
	chunks := make([]*listChunk[T], chunksCount)
	for i := range chunks {
		chunks[i] = new(listChunk[T])
	}
	l := &List[T]{
		chunks: chunks,
		allocs: 1,
	}
	l.root = 0
	*l.elem(0) = Element[T]{
		next: 0,
		prev: 0,
		idx:  0,
		list: l,
	}
	l.free = -1
	l.len = 0
	return l
}

// Len returns the number of elements of the list.
func (l *List[T]) Len() int { return l.len }

// Cap returns the total element capacity allocated across all chunks.
func (l *List[T]) Cap() int {
	return len(l.chunks) * chunkSize
}

// ShrinkToFit releases unused chunks to allow GC reclamation.
// If the list is empty, it shrinks chunk allocation down to 1 chunk.
// If the list is non-empty, it trims trailing unallocated chunks.
func (l *List[T]) ShrinkToFit() *List[T] {
	if l.len == 0 {
		if len(l.chunks) > 1 {
			clear(l.chunks[1:])
			l.chunks = l.chunks[:1]
		}
		if len(l.chunks) == 1 {
			clear(l.chunks[0][:])
		} else {
			l.chunks = append(l.chunks, new(listChunk[T]))
		}
		l.allocs = 1
		l.free = -1
		l.root = 0
		*l.elem(0) = Element[T]{
			next: 0,
			prev: 0,
			idx:  0,
			list: l,
		}
		return l
	}

	needed := max((l.allocs+chunkSize-1)/chunkSize, 1)
	if needed < len(l.chunks) {
		clear(l.chunks[needed:])
		l.chunks = l.chunks[:needed]
	}
	return l
}

// Front returns the first element of the list, or nil if the list is empty.
func (l *List[T]) Front() *Element[T] {
	if l.len == 0 {
		return nil
	}

	return l.elem(l.elem(l.root).next)
}

// Back returns the last element of the list, or nil if the list is empty.
func (l *List[T]) Back() *Element[T] {
	if l.len == 0 {
		return nil
	}

	return l.elem(l.elem(l.root).prev)
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
	if idx/chunkSize >= len(l.chunks) {
		l.chunks = append(l.chunks, new(listChunk[T]))
	}

	l.allocs++
	return idx
}

// insert inserts a new element after at, increments l.len, and returns e.
func (l *List[T]) insertValue(v T, at int) *Element[T] {
	idx := l.alloc()
	e := l.elem(idx)
	e.Value = v
	e.idx = idx
	e.list = l

	atElem := l.elem(at)
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

	atElem := l.elem(at)
	atNextIdx := atElem.next

	l.elem(idx).prev = at
	l.elem(idx).next = atNextIdx

	atElem.next = idx
	l.elem(atNextIdx).prev = idx

	e.prev = at
	e.next = atNextIdx
}

// Remove removes e from the list if it is an element, returning its value.
func (l *List[T]) Remove(e *Element[T]) T {
	if e == nil {
		var zero T
		return zero
	}
	v := e.Value
	if e.list == l {
		l.remove(e)
	}
	return v
}

// PushFront inserts a new element with value v at the front of the list and returns it.
func (l *List[T]) PushFront(v T) *Element[T] {
	l.lazyInit()
	return l.insertValue(v, l.root)
}

// PushBack inserts a new element with value v at the back of the list and returns it.
func (l *List[T]) PushBack(v T) *Element[T] {
	l.lazyInit()
	return l.insertValue(v, l.elem(l.root).prev)
}

// InsertBefore inserts a new element with value v immediately before mark and returns it.
func (l *List[T]) InsertBefore(v T, mark *Element[T]) *Element[T] {
	if mark == nil || mark.list != l {
		return nil
	}
	return l.insertValue(v, mark.prev)
}

// InsertAfter inserts a new element with value v immediately after mark and returns it.
func (l *List[T]) InsertAfter(v T, mark *Element[T]) *Element[T] {
	if mark == nil || mark.list != l {
		return nil
	}
	return l.insertValue(v, mark.idx)
}

// MoveToFront moves element e to the front of the list.
func (l *List[T]) MoveToFront(e *Element[T]) {
	if e == nil || e.list != l || l.elem(l.root).next == e.idx {
		return
	}
	l.move(e, l.root)
}

// MoveToBack moves element e to the back of the list.
func (l *List[T]) MoveToBack(e *Element[T]) {
	if e == nil || e.list != l || l.elem(l.root).prev == e.idx {
		return
	}
	l.move(e, l.elem(l.root).prev)
}

// MoveBefore moves element e to its new position before mark.
func (l *List[T]) MoveBefore(e, mark *Element[T]) {
	if e == nil || mark == nil || e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.prev)
}

// MoveAfter moves element e to its new position after mark.
func (l *List[T]) MoveAfter(e, mark *Element[T]) {
	if e == nil || mark == nil || e.list != l || e == mark || mark.list != l {
		return
	}
	l.move(e, mark.idx)
}

// PushBackList inserts a copy of another list at the back of the list.
func (l *List[T]) PushBackList(other *List[T]) {
	if other == nil || other.len == 0 {
		return
	}
	l.lazyInit()
	for i, e := other.Len(), other.Front(); i > 0 && e != nil; i, e = i-1, e.Next() {
		l.insertValue(e.Value, l.elem(l.root).prev)
	}
}

// PushFrontList inserts a copy of another list at the front of the list.
func (l *List[T]) PushFrontList(other *List[T]) {
	if other == nil || other.len == 0 {
		return
	}
	l.lazyInit()
	for i, e := other.Len(), other.Back(); i > 0 && e != nil; i, e = i-1, e.Prev() {
		l.insertValue(e.Value, l.root)
	}
}

// All returns an iterator yielding the index and element value from front to back.
func (l *List[T]) All() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		if l == nil || l.len == 0 {
			return
		}
		idx := 0
		for cur := l.elem(l.root).next; cur != l.root; {
			el := l.elem(cur)
			next := el.next
			if !yield(idx, el.Value) {
				return
			}
			idx++
			cur = next
		}
	}
}

// Values returns an iterator yielding element values from front to back.
func (l *List[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		if l == nil || l.len == 0 {
			return
		}
		for cur := l.elem(l.root).next; cur != l.root; {
			el := l.elem(cur)
			next := el.next
			if !yield(el.Value) {
				return
			}
			cur = next
		}
	}
}

// Backward returns an iterator yielding element values from back to front.
func (l *List[T]) Backward() iter.Seq[T] {
	return func(yield func(T) bool) {
		if l == nil || l.len == 0 {
			return
		}
		for cur := l.elem(l.root).prev; cur != l.root; {
			el := l.elem(cur)
			prev := el.prev
			if !yield(el.Value) {
				return
			}
			cur = prev
		}
	}
}
