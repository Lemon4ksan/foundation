// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package argkit_test

import (
	"flag"
	"fmt"

	"github.com/lemon4ksan/foundation/argkit"
)

func ExampleParseInterspersedFlags() {
	fs := flag.NewFlagSet("app", flag.ContinueOnError)
	verbose := fs.Bool("v", false, "verbose output")
	all := fs.Bool("a", false, "process all items")
	output := fs.String("output", "", "output file path")

	// args: ["file1.txt", "-va", "--output=result.json", "file2.txt"]
	args := []string{"file1.txt", "-va", "--output=result.json", "file2.txt"}
	posArgs, err := argkit.ParseInterspersedFlags(fs, args)
	if err != nil {
		return
	}

	fmt.Printf("Verbose: %v, All: %v, Output: %s\n", *verbose, *all, *output)
	fmt.Printf("Positional files: %v\n", posArgs)
	// Output:
	// Verbose: true, All: true, Output: result.json
	// Positional files: [file1.txt file2.txt]
}

func ExampleStringSliceFlag() {
	fs := flag.NewFlagSet("tool", flag.ContinueOnError)
	var includes argkit.StringSliceFlag
	fs.Var(&includes, "include", "directories or files to include (repeatable)")

	// Pass multiple times: --include=src --include=pkg
	_ = fs.Parse([]string{"--include=src", "--include=pkg"})
}
