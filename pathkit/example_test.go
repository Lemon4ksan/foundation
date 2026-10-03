// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/pathkit"
)

func ExampleNew() {
	p1 := pathkit.New("https://api.example.com/v1").Join("users", "42")
	fmt.Println(p1.String())

	p2 := pathkit.New("/var/log").Join("app", "stdout.log")
	fmt.Println(p2.String())
	// Output:
	// https://api.example.com/v1/users/42
	// /var/log/app/stdout.log
}

func ExamplePathToURI() {
	uri := pathkit.PathToURI("C:\\Projects\\app\\config.json")
	fmt.Println(uri)

	localPath, _ := pathkit.URIToPath(uri)
	fmt.Println(localPath)
	// Output:
	// file:///C:/Projects/app/config.json
	// C:\Projects\app\config.json
}
