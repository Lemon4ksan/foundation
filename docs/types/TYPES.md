# Core Types: High-Performance UUID & Dynamic Values (`types`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/types)

The `types` package group provides RFC 9562 UUIDv4 and UUIDv7 identifiers with SIMD validation and parsing, alongside zero-allocation dynamic value conversions.

## Architectural Components

```text
foundation/types/
├── uuid/                # RFC 9562 UUID v4 / UUID v7 generator, SIMD validator & parser
└── values/              # High-speed generic value conversion and structured extraction
```

## Core Capabilities

1. **RFC 9562 UUIDv7**: Monotonically sortable, time-ordered UUIDs suitable for database primary keys without B-Tree fragmentation.
2. **SIMD UUID Validation & Parsing (`uuid.Parse`)**: Hardware-accelerated byte scanning and hexadecimal decoding directly into 16-byte arrays without heap allocation.
3. **Buffer Appending (`uuid.Append` / `uuid.Format`)**: Formats 36-byte canonical representation (`xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`) directly into existing buffers.

## Key APIs & Usage

### 1. Generating & Formatting UUID v7

```go
package main

import (
    "fmt"
    "time"

    "github.com/lemon4ksan/foundation/types/uuid"
)

func main() {
    // Generate UUIDv7 from timestamp
    id := uuid.NewV7(time.Now())

    // Format directly into 36-byte stack buffer (0 allocs)
    var buf [36]byte
    str := id.Format(&buf)
    fmt.Println("UUID v7:", str)

    // Fast SIMD-accelerated parsing
    parsed, err := uuid.Parse(str)
    if err == nil {
        fmt.Println("Parsed:", parsed.String())
    }
}
```
