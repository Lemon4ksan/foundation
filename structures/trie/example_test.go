// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package trie_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/structures/trie"
)

func ExampleRadixTree() {
	tree := trie.New[string]()
	tree.Insert("/api/v1", "v1_root")
	tree.Insert("/api/v1/users", "users_handler")

	prefix, handler, ok := tree.LongestPrefix("/api/v1/users/42")
	fmt.Println(ok, prefix, handler)
	// Output: true /api/v1/users users_handler
}
