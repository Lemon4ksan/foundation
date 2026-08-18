# Monotonic Clock & Lock-Free Rand (`silicon/clock`, `silicon/rand`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/silicon/clock)

`silicon/clock` and `silicon/rand` provide nanosecond-precision monotonic time tracking without OS syscall overhead, lock-free fast pseudo-random generators, and zero-allocation UUID v4/v7 builders.

## Motivation & Problem Context

Querying system time via `time.Now()` on millions of operations incurs operating system vDSO and syscall overhead. Concurrently, standard pseudo-random number generators utilize global mutexes that bottleneck under multi-core concurrency, and standard UUID generation libraries produce heap allocations on every identifier generated. Lock-free, thread-local algorithms provide nanosecond-level time and identifier generation with zero memory allocation.

## Comparison

### Standard Implementation (Global Mutexes & Allocations)

```go
// Global lock contention across goroutines
num := rand.Intn(1000)

// Allocates 16 bytes on heap
id := uuid.New().String()
```

### Foundation Implementation (Lock-Free & Zero-Alloc)

```go
// Thread-local lock-free XorShift (0 mutexes)
num := rand.Uint32n(1000)

// Zero-allocation UUID v7 directly into buffer
var buf [36]byte
id := rand.UUIDv7Into(&buf)
```

## Architecture & Mechanics

* **Lock-Free PRNG**: Uses high-performance Xorshift / Wyhash algorithms seeded per CPU core to completely eliminate mutex contention.
* **Cached Timestamp Tickers**: For high-concurrency loops where microsecond precision is sufficient, `clock.NowCached()` updates timestamps in background goroutines, reducing timestamp reads to a single atomic load ($< 1\text{ ns}$).

## Practical Recipes

### 1. Zero-Alloc UUID v7 for Database Primary Keys

*Rationale*: UUID v7 is time-ordered (sortable), making it ideal for primary keys in PostgreSQL/SQLite without B-Tree page fragmentation.

```go
package main

import (
	"fmt"

	"github.com/lemon4ksan/foundation/silicon/rand"
)

func main() {
	var uuidBuf [36]byte
	uuidStr := rand.UUIDv7Into(&uuidBuf)

	fmt.Println("Time-sortable UUID v7:", uuidStr)
}
```
