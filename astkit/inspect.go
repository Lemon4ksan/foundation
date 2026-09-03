// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package astkit

import (
	"go/ast"
	"strings"
)

// StructMeta holds metadata about a Go struct declaration.
type StructMeta struct {
	Name   string
	Doc    string
	Fields []FieldMeta
}

// FieldMeta holds metadata about a struct field.
type FieldMeta struct {
	Name string
	Type string
	Tag  string
}

// MethodMeta holds metadata about a method declared on a receiver type.
type MethodMeta struct {
	Receiver string
	Name     string
	Doc      string
}

// Walk traverses an AST in depth-first order, invoking fn for each node.
// If fn returns false, Walk does not visit the node's children.
func Walk(node ast.Node, fn func(ast.Node) bool) {
	if node == nil || fn == nil {
		return
	}
	ast.Inspect(node, fn)
}

// FindStructs discovers all struct type declarations in the provided file AST.
func FindStructs(file *ast.File) []StructMeta {
	if file == nil {
		return nil
	}

	var results []StructMeta

	Walk(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		st, ok := typeSpec.Type.(*ast.StructType)
		if !ok || st.Fields == nil {
			return true
		}

		meta := StructMeta{
			Name: typeSpec.Name.Name,
		}
		if typeSpec.Doc != nil {
			meta.Doc = strings.TrimSpace(typeSpec.Doc.Text())
		}

		for _, f := range st.Fields.List {
			tagVal := ""
			if f.Tag != nil {
				tagVal = strings.Trim(f.Tag.Value, "`")
			}

			typeStr := exprToString(f.Type)

			if len(f.Names) == 0 {
				// Embedded field
				meta.Fields = append(meta.Fields, FieldMeta{
					Name: typeStr,
					Type: typeStr,
					Tag:  tagVal,
				})
			} else {
				for _, name := range f.Names {
					meta.Fields = append(meta.Fields, FieldMeta{
						Name: name.Name,
						Type: typeStr,
						Tag:  tagVal,
					})
				}
			}
		}

		results = append(results, meta)
		return false
	})

	return results
}

// FindMethods discovers all methods with receiver types in the provided file AST.
func FindMethods(file *ast.File) []MethodMeta {
	if file == nil {
		return nil
	}

	var results []MethodMeta

	Walk(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return true
		}

		recvType := exprToString(fn.Recv.List[0].Type)
		meta := MethodMeta{
			Receiver: recvType,
			Name:     fn.Name.Name,
		}
		if fn.Doc != nil {
			meta.Doc = strings.TrimSpace(fn.Doc.Text())
		}

		results = append(results, meta)
		return false
	})

	return results
}

func exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + exprToString(e.X)
	case *ast.SelectorExpr:
		return exprToString(e.X) + "." + e.Sel.Name
	case *ast.ArrayType:
		return "[]" + exprToString(e.Elt)
	default:
		return ""
	}
}
