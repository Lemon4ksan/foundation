# Struct & Type Reflection Helpers (`refkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/refkit)

`refkit` provides zero-allocation reflection inspection helpers and high-speed struct tag parsing with built-in caching for serialization, validation, and dependency injection engines.

## Motivation & Architecture

Standard Go reflection (`reflect.TypeOf`, `field.Tag.Lookup`, type assertions) frequently introduces performance bottlenecks in schema builders and serialization libraries due to string parsing overhead and interface allocations.

`refkit` optimizes introspection with:

* **`ParseTag` & `GetTag`**: Parses complex struct tags (e.g. `json:"user_id,omitempty,min=1,max=100"`) into structured `Tag` types with support for key-value extraction and numeric conversion.
* **`IsNil`, `IsZero`, `IsStruct`, `IsCollection`**: Panic-safe, zero-allocation type checking across all Go kinds.

## Key APIs & Usage

### 1. Struct Tag Parsing & Option Extraction

```go
package main

import (
    "fmt"
    "reflect"

    "github.com/lemon4ksan/foundation/refkit"
)

type User struct {
    Age int `json:"age,omitempty" validate:"min=18,max=120"`
}

func main() {
    field, _ := reflect.TypeOf(User{}).FieldByName("Age")

    tag := refkit.GetTag(field, "validate")
    minAge, ok := tag.GetInt("min")
    fmt.Printf("Min age required: %d (found: %v)\n", minAge, ok)
}
```

### 2. Panic-Safe Reflection Checks

```go
package main

import (
    "fmt"
    "reflect"

    "github.com/lemon4ksan/foundation/refkit"
)

func main() {
    var ch chan int
    val := reflect.ValueOf(ch)

    // Checks nil status safely without panicking on unsupported kinds
    if refkit.IsNil(val) {
        fmt.Println("Channel is nil")
    }
}
```
