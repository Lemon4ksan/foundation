// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit

import (
	"iter"
	"reflect"
	"sync"
)

var (
	staticIndex2 [16][16][]int
	staticIndex3 [8][8][8][]int
)

func init() {
	for i := 0; i < 16; i++ {
		for j := 0; j < 16; j++ {
			staticIndex2[i][j] = []int{i, j}
		}
	}
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			for k := 0; k < 8; k++ {
				staticIndex3[i][j][k] = []int{i, j, k}
			}
		}
	}
}

type nameEntry struct {
	name  string
	index int
}

type visibleWalker struct {
	entries  []nameEntry
	visiting []reflect.Type
	fields   []reflect.StructField
	index    []int
}

func newWalker() *visibleWalker {
	return &visibleWalker{
		entries:  make([]nameEntry, 0, 32),
		visiting: make([]reflect.Type, 0, 8),
		fields:   make([]reflect.StructField, 0, 32),
		index:    make([]int, 0, 8),
	}
}

var walkerCache struct {
	mu      sync.Mutex
	walkers []*visibleWalker
}

func getWalker() *visibleWalker {
	walkerCache.mu.Lock()
	n := len(walkerCache.walkers)
	if n > 0 {
		w := walkerCache.walkers[n-1]
		walkerCache.walkers = walkerCache.walkers[:n-1]
		walkerCache.mu.Unlock()
		return w
	}
	walkerCache.mu.Unlock()
	return newWalker()
}

func putWalker(w *visibleWalker) {
	w.entries = w.entries[:0]
	w.visiting = w.visiting[:0]
	w.fields = w.fields[:0]
	w.index = w.index[:0]
	walkerCache.mu.Lock()
	if len(walkerCache.walkers) < 32 {
		walkerCache.walkers = append(walkerCache.walkers, w)
	}
	walkerCache.mu.Unlock()
}

func (w *visibleWalker) isVisiting(t reflect.Type) bool {
	for _, v := range w.visiting {
		if v == t {
			return true
		}
	}
	return false
}

func (w *visibleWalker) findEntry(name string) (int, int, bool) {
	for i := range w.entries {
		if w.entries[i].name == name {
			return i, w.entries[i].index, true
		}
	}
	return -1, 0, false
}

func (w *visibleWalker) walk(t reflect.Type) {
	if w.isVisiting(t) {
		return
	}
	w.visiting = append(w.visiting, t)
	num := t.NumField()
	for i := 0; i < num; i++ {
		f := t.Field(i)
		w.index = append(w.index, i)
		add := true
		entryIdx, oldIndex, ok := w.findEntry(f.Name)
		if ok {
			old := &w.fields[oldIndex]
			switch {
			case len(w.index) == len(old.Index):
				old.Name = ""
				add = false
			case len(w.index) < len(old.Index):
				old.Name = ""
				w.entries[entryIdx].index = len(w.fields)
			default:
				add = false
			}
		}
		if add {
			switch {
			case len(w.index) == 1:
				f.Index = t.Field(w.index[0]).Index
			case len(w.index) == 2 && w.index[0] < 16 && w.index[1] < 16:
				f.Index = staticIndex2[w.index[0]][w.index[1]]
			case len(w.index) == 3 && w.index[0] < 8 && w.index[1] < 8 && w.index[2] < 8:
				f.Index = staticIndex3[w.index[0]][w.index[1]][w.index[2]]
			default:
				f.Index = append([]int(nil), w.index...)
			}
			if !ok {
				w.entries = append(w.entries, nameEntry{name: f.Name, index: len(w.fields)})
			}
			w.fields = append(w.fields, f)
		}
		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				w.walk(ft)
			}
		}
		w.index = w.index[:len(w.index)-1]
	}
	w.visiting = w.visiting[:len(w.visiting)-1]
}

// VisibleFields returns an iterator over visible fields of struct type t,
// matching the visibility and ordering semantics of [reflect.VisibleFields]
// without allocating slices or walker structures on the heap.
//
// For structs without anonymous/embedded fields, VisibleFields takes an optimized
// fast path that yields fields directly using runtime static indices, executing with
// 0 heap allocations. For structs with embedded fields, VisibleFields resolves
// shadowing and depth rules using pooled walker buffers.
//
// If t is nil or resolves to a non-struct type, VisibleFields returns an empty sequence.
func VisibleFields(t reflect.Type) iter.Seq[reflect.StructField] {
	return func(yield func(reflect.StructField) bool) {
		if t == nil {
			return
		}
		target := DerefType(t)
		if target.Kind() != reflect.Struct {
			return
		}

		hasAnonymous := false
		n := target.NumField()
		for i := 0; i < n; i++ {
			if target.Field(i).Anonymous {
				hasAnonymous = true
				break
			}
		}

		if !hasAnonymous {
			for i := 0; i < n; i++ {
				if !yield(target.Field(i)) {
					return
				}
			}
			return
		}

		w := getWalker()
		w.walk(target)
		for i := range w.fields {
			f := &w.fields[i]
			if f.Name == "" {
				continue
			}
			if !yield(*f) {
				putWalker(w)
				return
			}
		}
		putWalker(w)
	}
}
