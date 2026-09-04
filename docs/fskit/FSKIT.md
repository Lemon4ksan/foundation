# High-Performance Filesystem Primitives (`fskit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/fskit)

`fskit` provides concurrent multi-threaded directory traversal, cross-platform memory-mapped I/O (`mmap`), and file media heuristics.

## Core Capabilities

1. **Concurrent Directory Walker (`fskit.FastWalk`)**: Multi-threaded parallel file traversal outperforming standard `filepath.Walk` and `filepath.WalkDir` on large directory trees.
2. **Cross-Platform Mmap (`fskit.Mmap`)**: Zero-copy memory mapping of files on Windows (`CreateFileMappingW` / `MapViewOfFile`) and Unix (`mmap` syscall), bypassing user-space read buffers.
3. **Media Sniffing**: Fast header inspection for media format categorization.

## Key APIs & Usage

### 1. Parallel Directory Traversal (`fskit.FastWalk`)

```go
package main

import (
    "fmt"
    "io/fs"

    "github.com/lemon4ksan/foundation/fskit"
)

func main() {
    err := fskit.FastWalk(".", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return nil
        }
        if !d.IsDir() {
            fmt.Println("Found file:", path)
        }
        return nil
    })
    _ = err
}
```

### 2. Memory-Mapped File I/O (`fskit.Mmap`)

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/fskit"
)

func main() {
    // Memory map large dataset
    mapping, err := fskit.Mmap("large_dataset.bin")
    if err != nil {
        return
    }
    defer mapping.Close()

    data := mapping.Bytes()
    fmt.Printf("Mapped %d bytes directly from kernel cache\n", len(data))
}
```
