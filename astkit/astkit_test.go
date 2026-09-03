// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astkit_test

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/lemon4ksan/foundation/astkit"
)

func TestASTKit_ParseExpr(t *testing.T) {
	expr, err := astkit.ParseExpr("1 + 2 * 3")
	if err != nil {
		t.Fatalf("ParseExpr failed: %v", err)
	}
	if expr == nil {
		t.Fatalf("expected non-nil expr")
	}
}

func TestASTKit_ParseScript(t *testing.T) {
	script := `
		x := 42
		y := x * 2
		println(y)
	`
	stmts, err := astkit.ParseScript(script)
	if err != nil {
		t.Fatalf("ParseScript failed: %v", err)
	}
	if len(stmts) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(stmts))
	}
}

func TestASTKit_IsComplete(t *testing.T) {
	tests := []struct {
		code     string
		complete bool
	}{
		{"1 + 2", true},
		{"x := 10", true},
		{"if x > 0 {", false},
		{"fn(1, 2,", false},
		{"\"unclosed string", false},
		{"`raw unclosed", false},
		{"x + ", false},
		{"x |", false},
		{"x &&", false},
		{"if x > 0 { return 1 }", true},
	}

	for _, tt := range tests {
		got := astkit.IsComplete(tt.code)
		if got != tt.complete {
			t.Errorf("IsComplete(%q) = %v; want %v", tt.code, got, tt.complete)
		}
	}
}

func TestASTKit_FindStructsAndMethods(t *testing.T) {
	src := `
	package example
	// User document
	type User struct {
		Name string ` + "`json:\"name\"`" + `
		Age  int
	}
	func (u *User) Greet() string {
		return "Hello"
	}
	`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	structs := astkit.FindStructs(file)
	if len(structs) != 1 || structs[0].Name != "User" {
		t.Fatalf("expected 1 struct named User, got %+v", structs)
	}
	if len(structs[0].Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(structs[0].Fields))
	}

	methods := astkit.FindMethods(file)
	if len(methods) != 1 || methods[0].Name != "Greet" {
		t.Fatalf("expected 1 method named Greet, got %+v", methods)
	}
}
