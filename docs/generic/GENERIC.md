# Type-Safe Generics & Functional Utilities (`generic/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/generic)

`generic/` provides a zero-dependency, high-performance, and type-safe generic toolkit for Go. It eliminates repetitive boilerplate, manual type assertions, and slow runtime reflection across high-throughput application paths.

## 1. Architectural Structure

```text
foundation/generic/
├── types.go        // Pointer manipulation, null-coalescing, ternary operators
├── slices.go       // Zero-alloc slice operations, chunking, filtering, and indexing
├── lazy.go         // Go 1.23+ iter.Seq lazy transformation streams (O(1) space)
├── monads.go       // Swift-inspired monadic containers (Optional, Result, TypedResult)
├── collections.go  // Generic Set, TTL in-memory Cache, and ThreadSafe maps
├── concurrency.go  // ParallelMap, Futures, SingleFlight, and Resilient Retry
└── sync.go         // Generic Safe[T] thread-safe container, Mutate, and Read
```

## 2. Core Primitives & Capabilities

### A. Thread-Safe State Isolation (`generic.Safe[T]`)
`Safe[T]` encapsulates an arbitrary data type behind an optimized RWMutex, guaranteeing safe concurrent reads and transactional mutations without race conditions.

```go
// Encapsulate state:
sessionCache := generic.NewSafe(make(map[string]UserSession))

// Read safely:
sessionCache.Read(func(sessions map[string]UserSession) {
    session, ok = sessions["sess_123"]
})

// Mutate transactionally:
sessionCache.Mutate(func(sessions *map[string]UserSession) {
    (*sessions)["sess_123"] = newSession
})
```

### B. In-Memory Generic Cache with TTL (`generic.Cache[K, V]`)
A lock-striped, concurrent generic cache with automatic expiration and item eviction.

```go
cache := generic.NewCache[string, AccountData](5 * time.Minute)

// Set item with TTL:
cache.Set("acc_42", account, 10 * time.Minute)

// O(1) fast lookup:
if account, ok := cache.Get("acc_42"); ok {
    fmt.Println("Found account:", account.Name)
}
```

### C. Monadic Safety (`Optional[T]` & `Result[T]`)
Deterministic error handling and state encapsulation for channels, batch outputs, and PATCH endpoints:

```go
// 1. Partial updates (JSON PATCH):
type UpdateUserDTO struct {
    Name  generic.Optional[string] `json:"name"`
    Email generic.Optional[string] `json:"email"`
}

// 2. Channel transfer without one-off structs:
results := make(chan generic.Result[UserData], 10)
results <- generic.Success(data)
results <- generic.Failure[UserData](err)
```

### D. Lazy Pipeline Sequences (`iter.Seq` in Go 1.23+)
Process large collections without allocating intermediate slices in memory:

```go
numbers := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

// Define pipeline without allocating memory:
pipeline := generic.FilterLazy(
    generic.MapLazy(generic.ToSeq(numbers), func(n int) int { return n * 2 }),
    func(n int) bool { return n > 10 },
)

// Collect to slice with a single allocation:
filtered := generic.ToSlice(pipeline) // []int{12, 14, 16, 18, 20}
```

### E. Declarative Functional Options (`generic.ApplyOptions`)
Uniform, zero-allocation functional configuration pattern:

```go
type ServerConfig struct {
    Port    int
    Timeout time.Duration
}

type ServerOption = generic.OptionFunc[ServerConfig]

func WithPort(p int) ServerOption {
    return func(c *ServerConfig) { c.Port = p }
}

// Apply options cleanly:
cfg := ServerConfig{Port: 8080, Timeout: 30 * time.Second}
generic.ApplyOptions(&cfg, WithPort(9090))
```

## 3. Benchmarks & Allocation Profile

| Operation | Standard Approach | `foundation/generic` | Memory Overhead |
| :--- | :---: | :---: | :---: |
| **Slice Chunking (10k items)** | Manual loops + allocs | `generic.Chunked` | **0 allocs** (shared slice backing) |
| **Parallel Map (1,000 tasks)** | Goroutines + Channels | `generic.ParallelMap` | **1 alloc** (pooled workers) |
| **Option/Result Box/Unbox** | `interface{}` boxing | `generic.Result[T]` | **0 B/op** (Inline value type) |
| **ThreadSafe Read (`Safe[T]`)** | `sync.Mutex` contention | `Safe.Read` | **0 B/op** (RWMutex lock-elision) |
