# High-Throughput Time & Dates (`timekit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/timekit)

`timekit` provides high-throughput, zero-allocation coarse monotonic clocks, branchless date formatting (HTTP-date, RFC 3339, ISO 8601), and high-precision stopwatch utilities for performance-critical server backends.

## Motivation & Architecture

In high-concurrency servers processing hundreds of thousands of requests per second, invoking `time.Now()` for every log line, HTTP header, or rate-limit bucket triggers operating system vDSO and syscall overhead.

`timekit` resolves this with:

* **`CoarseNow`**: Lock-free atomic monotonic clock continuously refreshed by a background ticker.
* **`AppendHTTPDate`**: Zero-allocation RFC 7231 / RFC 9110 HTTP-date serializer.
* **`AppendRFC3339` / `AppendISO8601`**: Branchless zero-allocation timestamp generators.
* **`Stopwatch`**: High-precision monotonic timer for nanosecond-level latency tracing.

## Key APIs & Usage

### 1. Zero-Syscall Coarse Monotonic Clock

```go
package main

import (
    "fmt"
    "time"

    "github.com/lemon4ksan/foundation/timekit"
)

func main() {
    // Read current time with single atomic load (< 1 ns)
    now := timekit.CoarseNow()
    fmt.Println("Coarse timestamp:", now.Format(time.RFC3339))
}
```

### 2. Zero-Allocation HTTP-Date Formatting

```go
package main

import (
    "fmt"
    "time"

    "github.com/lemon4ksan/foundation/timekit"
)

func main() {
    var buf [32]byte
    // Generates "Wed, 26 Aug 2026 18:30:00 GMT" with 0 allocations
    res := timekit.AppendHTTPDate(buf[:0], time.Now().UTC())
    fmt.Println(string(res))
}
```

### 3. Nanosecond Latency Stopwatch

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/timekit"
)

func main() {
    sw := timekit.NewStopwatch()
    // Execute critical section...
    elapsed := sw.Elapsed()
    fmt.Printf("Elapsed duration: %v\n", elapsed)
}
```
