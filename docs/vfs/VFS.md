# Virtual Filesystem & Path Safety Defenses (`vfs`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/vfs)

`vfs` provides a standard `io/fs.FS` virtual filesystem abstraction over arbitrary entry providers, Zip Slip / Tar Slip directory traversal defenses, resource limits, and hierarchical tree builders.

## Core Capabilities

1. **Standard `fs.FS` Implementation (`vfs.FS`)**: Implements `fs.FS`, `fs.StatFS`, `fs.ReadFileFS`, and `fs.ReadDirFS` over in-memory or custom entry backends.
2. **Path Sanitization & Traversal Defenses (`vfs.SanitizePath`)**: Neutralizes path traversal attacks (Zip Slip / Tar Slip) and prevents escaping designated base directories.
3. **Extraction & Size Limits (`vfs.ExtractionLimits`)**: Guards against decompression bombs, excessive file counts, and malicious symlinks during archive extraction.
4. **Virtual File Tree (`vfs.Tree`)**: Builds in-memory hierarchical directory representations from flat lists of paths.

## Key APIs & Usage

### 1. Safe Path Resolution & Extraction Defense

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/vfs"
)

func main() {
    // Malicious entry from an archive
    untrustedPath := "../../etc/passwd"
    
    cleanPath, ok := vfs.SanitizePath("/safe/extract/dir", untrustedPath)
    if !ok {
        fmt.Println("Path traversal attack detected and blocked!")
        return
    }

    fmt.Println("Safe target path:", cleanPath)
}
```

### 2. Standard `io/fs.FS` over Virtual Entries

```go
package main

import (
    "io/fs"

    "github.com/lemon4ksan/foundation/vfs"
)

func readVirtualFS(vfsFS *vfs.FS) error {
    // Standard Go 1.16+ fs.FS read
    data, err := fs.ReadFile(vfsFS, "config/settings.json")
    if err != nil {
        return err
    }
    _ = data
    return nil
}
```
