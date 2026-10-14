# Copyright (c) 2026 Lemon4ksan All rights reserved.
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

import re

with open('linkedlist.go', 'r', encoding='utf-8') as f:
    code = f.read()

# 1. GC Leak in remove
remove_old = '''	l.elements[idx].next = l.free
	l.elements[idx].prev = -1
	l.elements[idx].list = nil
	l.free = idx'''

remove_new = '''	var zero T
	l.elem(idx).next = l.free
	l.elem(idx).prev = -1
	l.elem(idx).list = nil
	l.elem(idx).Value = zero
	l.free = idx'''
code = code.replace(remove_old, remove_new)

# 2. Struct change
struct_old = '''type List[T any] struct {
	elements []Element[T]
	root     int // sentinel list element index
	free     int // head of free list
	len      int // current list length excluding sentinel
}'''

struct_new = '''const chunkSize = 256

type listChunk[T any] [chunkSize]Element[T]

// List represents a doubly linked list.
// The zero value for List is an empty list ready to use.
type List[T any] struct {
	chunks []*listChunk[T]
	allocs int
	root   int // sentinel list element index
	free   int // head of free list
	len    int // current list length excluding sentinel
}

func (l *List[T]) elem(idx int) *Element[T] {
	return &l.chunks[idx/chunkSize][idx%chunkSize]
}'''
code = code.replace(struct_old, struct_new)

# 3. Targeted replaces
code = code.replace('&e.list.elements[p]', 'e.list.elem(p)')
code = code.replace('&l.elements[l.elements[l.root].next]', 'l.elem(l.elem(l.root).next)')
code = code.replace('&l.elements[l.elements[l.root].prev]', 'l.elem(l.elem(l.root).prev)')
code = code.replace('l.elements[l.root].prev', 'l.elem(l.root).prev')
code = code.replace('l.elements[l.root].next', 'l.elem(l.root).next')
code = code.replace('l.elements[0] = Element[T]', '*l.elem(0) = Element[T]')
code = code.replace('len(l.elements)', 'l.allocs')

# Replace l.elements[xxx] -> l.elem(xxx)
def repl(m):
    return f"l.elem({m.group(1)})"
code = re.sub(r'l\.elements\[([^\]]+)\]', repl, code)

# Fix Init
init_old = '''func (l *List[T]) Init() *List[T] {
	if cap(l.elements) == 0 {
		l.elements = make([]Element[T], 1, 16)
	} else {
		l.elements = l.elements[:1]
	}
	l.root = 0'''
init_new = '''func (l *List[T]) Init() *List[T] {
	if len(l.chunks) == 0 {
		l.chunks = append(l.chunks, new(listChunk[T]))
	}
	l.allocs = 1
	l.root = 0'''
code = code.replace(init_old, init_new)

# Fix NewCapacity
cap_old = '''func NewCapacity[T any](capacity int) *List[T] {
	l := &List[T]{
		elements: make([]Element[T], 1, capacity+1),
	}
	l.root = 0'''
cap_new = '''func NewCapacity[T any](capacity int) *List[T] {
	chunksCount := (capacity + 1 + chunkSize - 1) / chunkSize
	chunks := make([]*listChunk[T], chunksCount)
	for i := range chunks {
		chunks[i] = new(listChunk[T])
	}
	l := &List[T]{
		chunks: chunks,
		allocs: 1,
	}
	l.root = 0'''
code = code.replace(cap_old, cap_new)

# Fix lazyInit
lazy_old = '''func (l *List[T]) lazyInit() {
	if l.allocs == 0 {
		l.Init()
	}
}'''
lazy_new = '''func (l *List[T]) lazyInit() {
	if len(l.chunks) == 0 {
		l.Init()
	}
}'''
code = code.replace(lazy_old, lazy_new)

# Fix alloc
alloc_old = '''func (l *List[T]) alloc() int {
	if l.free != -1 {
		idx := l.free
		l.free = l.elem(idx).next
		return idx
	}
	idx := l.allocs
	if idx >= cap(l.elements) {
		panic("linkedlist: array-based list exceeded capacity and cannot safely resize without invalidating pointers")
	}
	l.elements = append(l.elements, Element[T]{})
	return idx
}'''
alloc_new = '''func (l *List[T]) alloc() int {
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
}'''
code = code.replace(alloc_old, alloc_new)

# insertValue fixes
code = code.replace('e := &l.elements[idx]', 'e := l.elem(idx)')
code = code.replace('atElem := &l.elements[at]', 'atElem := l.elem(at)')

with open('linkedlist.go', 'w', encoding='utf-8') as f:
    f.write(code)
