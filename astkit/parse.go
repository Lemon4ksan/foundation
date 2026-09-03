// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astkit

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// ParseExpr parses a single Go expression string using the standard library parser.
func ParseExpr(src string) (ast.Expr, error) {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return nil, fmt.Errorf("empty expression")
	}
	return parser.ParseExpr(trimmed)
}

// ParseStmt parses a single Go statement (e.g. "x := 10", "if x > 0 { return x }").
func ParseStmt(src string) (ast.Stmt, error) {
	stmts, err := ParseScript(src)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, fmt.Errorf("empty statement")
	}
	return stmts[0], nil
}

// ParseScript wraps statements in a synthetic function body and parses them into a list of [ast.Stmt].
func ParseScript(src string) ([]ast.Stmt, error) {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return nil, nil
	}

	// Wrap in a synthetic package and function
	code := fmt.Sprintf("package main\nfunc __powned_eval__() {\n%s\n}", trimmed)
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", code, parser.AllErrors)
	if err != nil {
		// Try parsing as a raw expression fallback
		if expr, exprErr := parser.ParseExpr(trimmed); exprErr == nil {
			return []ast.Stmt{&ast.ExprStmt{X: expr}}, nil
		}
		return nil, cleanParseError(err)
	}

	if len(file.Decls) == 0 {
		return nil, nil
	}

	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || fn.Body == nil {
		return nil, fmt.Errorf("failed to parse statement body")
	}

	return fn.Body.List, nil
}

// IsComplete checks whether a block of Go code is syntactically complete
// or requires additional lines (e.g. open braces, open parentheses, unclosed strings).
func IsComplete(code string) bool {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" {
		return true
	}

	// 1. Quick balance check for delimiters
	var parenCount, braceCount, bracketCount int
	inString := false
	inRawString := false
	var prev byte

	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]

		if inRawString {
			if ch == '`' {
				inRawString = false
			}
			continue
		}

		if inString {
			if ch == '"' && prev != '\\' {
				inString = false
			}
			prev = ch
			continue
		}

		switch ch {
		case '`':
			inRawString = true
		case '"':
			inString = true
		case '(':
			parenCount++
		case ')':
			parenCount--
		case '{':
			braceCount++
		case '}':
			braceCount--
		case '[':
			bracketCount++
		case ']':
			bracketCount--
		}
		prev = ch
	}

	if inString || inRawString || parenCount > 0 || braceCount > 0 || bracketCount > 0 {
		return false
	}

	// 2. Check if ends with dangling operators that imply continuation
	last := strings.TrimRight(trimmed, " \t\r\n")
	if strings.HasSuffix(last, ",") ||
		strings.HasSuffix(last, "+") ||
		strings.HasSuffix(last, "-") ||
		strings.HasSuffix(last, "*") ||
		strings.HasSuffix(last, "/") ||
		strings.HasSuffix(last, "|") ||
		strings.HasSuffix(last, "&&") ||
		strings.HasSuffix(last, "||") {
		return false
	}

	return true
}

func cleanParseError(err error) error {
	msg := err.Error()
	// Strip synthetic filename and line offset if present
	if idx := strings.Index(msg, "__powned_eval__"); idx != -1 {
		msg = msg[idx:]
	}
	return fmt.Errorf("%s", msg)
}
