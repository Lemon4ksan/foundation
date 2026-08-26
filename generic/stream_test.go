// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"reflect"
	"testing"
)

func TestStream_Pipeline(t *testing.T) {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	result := FromSlice(nums).
		Filter(func(n int) bool { return n%2 == 0 }).
		Map(func(n int) int { return n * 10 }).
		Take(3).
		Collect()

	expected := []int{20, 40, 60}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestStream_ChunkAndDistinct(t *testing.T) {
	items := []string{"a", "b", "a", "c", "b", "d"}

	distinct := FromSlice(items).
		Distinct(func(s string) any { return s }).
		Collect()

	expected := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(distinct, expected) {
		t.Fatalf("expected %v, got %v", expected, distinct)
	}

	chunks := ChunkStream(FromSlice([]int{1, 2, 3, 4, 5}), 2).
		Collect()

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if !reflect.DeepEqual(chunks[0], []int{1, 2}) || !reflect.DeepEqual(chunks[2], []int{5}) {
		t.Fatalf("unexpected chunk content: %v", chunks)
	}
}

func TestLRU_GetPutEvict(t *testing.T) {
	lru := NewLRU[string, int](2)

	lru.Put("a", 1)
	lru.Put("b", 2)

	if val, ok := lru.Get("a"); !ok || val != 1 {
		t.Fatalf("expected 1, got %v", val)
	}

	// Insert third item, should evict "b" (since "a" was accessed)
	lru.Put("c", 3)

	if _, ok := lru.Get("b"); ok {
		t.Fatalf("expected 'b' to be evicted")
	}

	if val, ok := lru.Get("c"); !ok || val != 3 {
		t.Fatalf("expected 'c' to exist")
	}

	if val, ok := lru.Get("a"); !ok || val != 1 {
		t.Fatalf("expected 'a' to exist")
	}
}

func TestSlicePool(t *testing.T) {
	sp := NewSlicePool[int](16)

	s := sp.Get()
	if cap(s) < 16 {
		t.Fatalf("expected cap >= 16, got %d", cap(s))
	}

	s = append(s, 1, 2, 3)
	sp.Put(s)

	s2 := sp.Get()
	if len(s2) != 0 {
		t.Fatalf("expected reset slice len=0, got %d", len(s2))
	}
}
