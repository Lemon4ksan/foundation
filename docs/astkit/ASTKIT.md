# Go AST Parser & Code Introspection (`astkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/astkit)

`astkit` provides zero-dependency Go AST parsing, structural inspection, statement slicing, and syntax traversal helpers.

## Core Capabilities

1. **Expression & Statement Parsing**: Parses isolated expressions (`ParseExpr`) and statements (`ParseStmt`) without boilerplate file scaffolding.
2. **Script Parsing**: Parses multi-line statement scripts (`ParseScript`) by wrapping code in synthetic function bodies.
3. **Struct & Method Discovery**: Scans ASTs to extract struct declarations, field tags, types, embedded structures (`FindStructs`), and receiver methods (`FindMethods`).
4. **AST Traversal (`Walk`)**: Depth-first AST inspection visitor.
5. **Code Completeness Check**: Validates whether multi-line code buffers form syntactically complete Go constructs (useful for REPLs and interpreters).

## Key APIs & Usage

### 1. Discover Structs and Fields (`astkit.FindStructs`)

```go
package main

import (
    "fmt"
    "go/parser"
    "go/token"

    "github.com/lemon4ksan/foundation/astkit"
)

func main() {
    src := `
    package schema
    type User struct {
        ID   int64  ` + "`json:\"id\"`" + `
        Name string ` + "`json:\"name\"`" + `
    }`

    fset := token.NewFileSet()
    file, _ := parser.ParseFile(fset, "schema.go", src, 0)

    structs := astkit.FindStructs(file)
    for _, s := range structs {
        fmt.Printf("Struct: %s\n", s.Name)
        for _, f := range s.Fields {
            fmt.Printf("  Field: %s (%s) Tag: %s\n", f.Name, f.Type, f.Tag)
        }
    }
}
```

### 2. Parse Go Expressions and Statements

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/astkit"
)

func main() {
    // Parse single expression
    expr, err := astkit.ParseExpr("a + b * 2")
    if err == nil {
        fmt.Printf("Parsed expression: %T\n", expr)
    }

    // Parse statement block
    stmts, err := astkit.ParseScript("x := 10\nif x > 0 { println(x) }")
    if err == nil {
        fmt.Printf("Parsed %d statements\n", len(stmts))
    }
}
```
