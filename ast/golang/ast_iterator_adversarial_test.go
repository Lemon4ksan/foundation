// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/lemon4ksan/foundation/ast/golang"
)

func parseSampleFile(t *testing.T) *ast.File {
	t.Helper()
	src := `package testpkg
import "fmt"
type Config struct { Host string; Port int }
func Run(c *Config) error {
	if c == nil { return nil }
	fmt.Println(c.Host, c.Port)
	return nil
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "sample.go", src, 0)
	if err != nil {
		t.Fatalf("failed to parse sample: %v", err)
	}
	return f
}

func TestEmpirical_ASTGolang_WalkSeq_EarlyTermination(t *testing.T) {
	file := parseSampleFile(t)

	// Nil node handling
	var nilNode ast.Node
	for range golang.WalkSeq(nilNode) {
		t.Fatal("expected no iterations for nil node")
	}

	// 1. Immediate break after 0 iterations (break on first yielded node)
	count := 0
	for n := range golang.WalkSeq(file) {
		_ = n
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 2 iterations
	count = 0
	for n := range golang.WalkSeq(file) {
		_ = n
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// 3. Full traversal count
	totalNodes := 0
	for n := range golang.WalkSeq(file) {
		_ = n
		totalNodes++
	}
	if totalNodes < 10 {
		t.Fatalf("expected >= 10 nodes, got %d", totalNodes)
	}

	// 4. Break mid-iteration
	mid := totalNodes / 2
	visited := 0
	for n := range golang.WalkSeq(file) {
		_ = n
		visited++
		if visited == mid {
			break
		}
	}
	if visited != mid {
		t.Fatalf("expected visited %d, got %d", mid, visited)
	}
}

func BenchmarkWalkSeq_Traverse(b *testing.B) {
	src := `package testpkg
type Data struct { A, B, C int }
func Foo(d *Data) int { return d.A + d.B + d.C }`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "bench.go", src, 0)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for n := range golang.WalkSeq(f) {
			_ = n
		}
	}
}
