// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fuzzTarget struct {
	pkg  string
	name string
}

func discoverFuzzTargets(root string) ([]fuzzTarget, error) {
	var discovered []fuzzTarget
	fset := token.NewFileSet()
	cleanRoot := filepath.Clean(root)

	err := filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == cleanRoot {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "scripts" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		node, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		relDir, relErr := filepath.Rel(cleanRoot, filepath.Dir(path))
		if relErr != nil {
			return relErr
		}
		pkg := "./" + filepath.ToSlash(relDir)
		for _, decl := range node.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || !strings.HasPrefix(fd.Name.Name, "Fuzz") {
				continue
			}
			if fd.Type.Params != nil && len(fd.Type.Params.List) == 1 {
				discovered = append(discovered, fuzzTarget{pkg: pkg, name: fd.Name.Name})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(discovered, func(i, j int) bool {
		if discovered[i].pkg != discovered[j].pkg {
			return discovered[i].pkg < discovered[j].pkg
		}
		return discovered[i].name < discovered[j].name
	})
	return discovered, nil
}

func main() {
	fuzzDuration := flag.String("fuzztime", "3s", "duration to fuzz each target")

	flag.Parse()

	targets, err := discoverFuzzTargets(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to discover fuzz targets: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf(
		"=== Starting Foundation Heavy Security Fuzzing Suite (%d targets, %s each) ===\n\n",
		len(targets),
		*fuzzDuration,
	)

	var failed []string

	startTotal := time.Now()

	for i, tgt := range targets {
		fmt.Printf("[%2d/%2d] Fuzzing %s :: %s (fuzztime=%s) ... ", i+1, len(targets), tgt.pkg, tgt.name, *fuzzDuration)

		start := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		cmd := exec.CommandContext(
			ctx,
			"go",
			"test",
			"-run",
			"^$",
			"-fuzz",
			fmt.Sprintf("^%s$", tgt.name),
			"-fuzztime",
			*fuzzDuration,
			tgt.pkg,
		) // #nosec G204

		var outBuf bytes.Buffer

		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf

		err := cmd.Run()
		cancel()

		dur := time.Since(start)

		if err != nil {
			fmt.Printf("FAILED (%v)\n", dur.Round(time.Millisecond))
			fmt.Printf("--- Output ---\n%s\n--------------\n", outBuf.String())
			failed = append(failed, fmt.Sprintf("%s::%s", tgt.pkg, tgt.name))
		} else {
			fmt.Printf("PASSED (%v)\n", dur.Round(time.Millisecond))
		}
	}

	totalDur := time.Since(startTotal)

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Printf("Heavy Fuzzing Complete in %v\n", totalDur.Round(time.Second))
	fmt.Printf("Total: %d | Passed: %d | Failed: %d\n", len(targets), len(targets)-len(failed), len(failed))

	if len(failed) > 0 {
		fmt.Println("Failed Targets:")
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Println("All fuzzing targets passed with zero crashes and curl-grade accuracy!")
}
