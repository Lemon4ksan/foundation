// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package js provides a lightweight, fluent Abstract Syntax Tree (AST) model
// and code generator for JavaScript and TypeScript programs.
//
// # Architecture
//
// The package models JavaScript source code as a hierarchy of strongly typed AST nodes
// conforming to the [Node] interface, divided into statements ([Stmt]) and expressions ([Expr]):
//   - Statements: Variable declarations ([VarDecl]), function declarations ([FuncDecl]),
//     control flow statements ([IfStmt], [ReturnStmt]), block statements ([BlockStmt]),
//     and raw statements ([RawStmt]).
//   - Expressions: Identifiers ([Ident]), literals ([Literal]), binary operations ([BinaryExpr]),
//     call expressions ([CallExpr]), member accesses ([MemberExpr]), object literals ([ObjectExpr]),
//     array literals ([ArrayExpr]), and arrow functions ([ArrowFunc]).
//
// # Design Rationale
//
// Generating JavaScript / TypeScript source code using raw string concatenation is error-prone,
// susceptible to syntax malformations, and difficult to format cleanly. This package provides:
//   - A fluent builder API ([Const], [Let], [Func], [Call], [Obj], [Arr], [Return]) for assembling AST structures.
//   - High-performance, indentation-aware code formatting via [Format].
//   - Seamless conversion of Go scalar and structured primitives into JavaScript expressions via [ToExpr].
//
// # Concurrency Guarantees
//
// AST nodes are pure data structures without internal synchronization. Concurrent reads of
// immutable node graphs (such as multiple goroutines calling [Format] on the same [Program])
// are thread-safe. Concurrent modifications of AST nodes or mutating a graph while another
// goroutine formats it is unsafe and requires external synchronization.
//
// # Example
//
//	package main
//
//	import (
//		"fmt"
//
//		"github.com/lemon4ksan/foundation/ast/js"
//	)
//
//	func main() {
//		prog := js.NewProgram(
//			js.Const("multiplier", 2),
//			js.Func("double", []string{"x"},
//				js.Return(js.Mul(js.Id("x"), js.Id("multiplier"))),
//			),
//		)
//
//		code, err := js.Format(prog)
//		if err != nil {
//			panic(err)
//		}
//		fmt.Println(code)
//	}
package js
