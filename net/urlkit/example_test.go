// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package urlkit_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/net/urlkit"
)

func ExampleParseView() {
	u, err := urlkit.ParseView("https://api.internal/v1/users?id=123")
	if err == nil {
		fmt.Println("Host:", u.Host)
		fmt.Println("Path:", u.Path)
	}

	// Output:
	// Host: api.internal
	// Path: /v1/users
}

func ExampleQueryUnescape() {
	unescaped, err := urlkit.QueryUnescape("hello+world%21")
	if err == nil {
		fmt.Println(unescaped)
	}

	// Output:
	// hello world!
}

func ExampleURLView_QuerySeq() {
	v, err := urlkit.ParseView("https://api.internal/search?query=foundation&limit=10")
	if err == nil {
		for k, val := range v.QuerySeq() {
			fmt.Printf("%s: %s\n", k, val)
		}
	}

	// Output:
	// query: foundation
	// limit: 10
}

func ExampleBuildPath() {
	params := map[string]string{"id": "42"}
	path := urlkit.BuildPath("/users/{id}/details", params, nil)
	fmt.Println(path)

	// Output:
	// /users/42/details
}
