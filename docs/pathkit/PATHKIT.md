# Unified Immutable Path Abstraction (`pathkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/pathkit)

`pathkit` provides an immutable, unified Path abstraction bridging remote network URLs (`https://...`), cloud storage URIs (`s3://...`), local OS filepaths (`C:\...` or `/etc/...`), and RFC 8089 `file://` URIs with cross-platform consistency and zero allocations.

## Core Capabilities

1. **Unified Path Type (`pathkit.Path`)**: Represents local paths, network URLs, and cloud URIs within a single immutable value struct.
2. **RFC 8089 File URIs**: Seamless bidirectional conversion between local paths (`C:\dir\file.txt` or `/var/log`) and standard `file:///` URIs.
3. **Smart Joining (`pathkit.Join`)**: Platform-aware path and URI joining without duplicate separators or trailing slashes.
4. **Clean Normalization**: Standardizes separator slashes across Windows (`\`) and Unix (`/`).

## Key APIs & Usage

### 1. Unified Path Handling

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/pathkit"
)

func main() {
    p1 := pathkit.New("https://api.example.com/v1").Join("users", "42")
    fmt.Println(p1.String()) // "https://api.example.com/v1/users/42"

    p2 := pathkit.New("/var/log").Join("app", "stdout.log")
    fmt.Println(p2.String()) // "/var/log/app/stdout.log"
}
```

### 2. File URI Conversions (RFC 8089)

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/pathkit"
)

func main() {
    uri := pathkit.PathToFileURI("C:\\Projects\\app\\config.json")
    fmt.Println(uri) // "file:///C:/Projects/app/config.json"

    localPath, _ := pathkit.FileURIToPath(uri)
    fmt.Println(localPath) // "C:\Projects\app\config.json"
}
```
