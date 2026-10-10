package list

import "testing"

func TestReproduce_AllocLeakOnFreeListReuse(t *testing.T) {
	l := New[int]()
	
	// Add and remove an element to populate the free list
	e := l.PushFront(1)
	l.Remove(e)
	
	// allocs should be 2 (idx 0 is root, idx 1 is the allocated then freed element)
	if l.allocs != 2 {
		t.Fatalf("expected allocs to be 2, got %d", l.allocs)
	}
	
	// Push a new element. It should reuse the element from the free list (idx 1)
	l.PushFront(2)
	
	// allocs should still be 2. If it's 3, the leak is present!
	if l.allocs != 2 {
		t.Fatalf("BUG: allocs leaked! Expected 2, got %d", l.allocs)
	}
}
