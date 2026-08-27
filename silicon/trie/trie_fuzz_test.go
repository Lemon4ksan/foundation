// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package trie_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/silicon/trie"
)

func FuzzRadixTree(f *testing.F) {
	f.Add("/api/v1/users", "/api/v1/users/123", 100)
	f.Add("apple", "applesauce", 1)
	f.Add("", "test", 0)

	f.Fuzz(func(t *testing.T, key1, key2 string, val int) {
		tree := trie.New[int]()
		tree.Insert(key1, val)

		gotVal, exists := tree.Get(key1)
		if !exists || gotVal != val {
			t.Fatalf("tree.Get(%q) failed: got (%d, %v), want (%d, true)", key1, gotVal, exists, val)
		}

		_, _ = tree.Get(key2)
		_, _, _ = tree.LongestPrefix(key2)
	})
}
