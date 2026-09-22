# Foundation Performance & Zero-Allocation Guide

## 1. Choosing the Right Primitives

### 1.1. Memory Pooling: `sync.Pool` vs `silicon/pool.PerPStorage`
- **Use `sync.Pool` when**:
  - The object pool is general-purpose and object sizes are moderate.
  - GC draining of idle objects during stop-the-world / mark phase is desirable to reclaim heap memory.
- **Use `silicon/pool.PerPStorage` when**:
  - You have high concurrency with heavy multi-core throughput (>100k ops/sec).
  - You want to eliminate CAS contention on shared pool heads.
  - You need false-sharing prevention via `cpu.CacheLinePad`.
  - Object retention across GC cycles is desired (e.g., dedicated network buffers, scratch pads).

### 1.2. Lazy Initialization: `sync.Once` vs `sync/lazy.Lazy[T]`
- **Use `sync.Once` when**:
  - The operation is void and permanent (cannot be reset or re-initialized).
- **Use `sync/lazy.Lazy[T]` when**:
  - You need to cache a typed value `T` and error `error`.
  - You require dynamic `Reset()` functionality for stale cache invalidation.
  - Concurrent readers must enjoy a $O(1)$ lockless atomic fast path (`atomic.Pointer[result[T]]`) without acquiring a mutex.

### 1.3. Lists: `container/list` vs `structures/linkedlist.List[T]`
- **Use standard `container/list` when**:
  - You require standard library interop or legacy code compatibility.
- **Use `structures/linkedlist.List[T]` when**:
  - Performance and memory footprint matter.
  - Zero heap allocation per push/pop is required (`NewCapacity`).
  - CPU cache locality is critical (contiguous memory slice vs scattered pointers).
  - Modern Go iterators (`for v := range list.Values()`) are preferred over element pointer walking.

---

## 2. Zero-Allocation Engineering Guidelines

### 2.1. Bounds Check Elimination (BCE)
The Go compiler inserts slice bounds checks (`panicIndex`) whenever accessing `slice[i]`. In performance-critical loops:
```go
// Upfront invariant:
_ = dst[n-1]
for i := range n {
    dst[i] = op(src[i]) // Compiler eliminates bounds check on dst[i]
}
```

### 2.2. In-Place Encoding
Whenever encoding wire formats, avoid returning newly allocated slices (`[]byte`). Instead, provide slice-writing variants accepting a caller-provided destination:
```go
// Allocating pattern:
buf := varint.EncodeVarint(val) // Allocates []byte

// Zero-allocation pattern:
n := varint.EncodeVarintSlice(val, scratch[offset:]) // 0 allocs
```

### 2.3. Constant-Time Operations
When comparing cryptographic secrets, tokens, or hashes, never convert strings to byte slices via `[]byte(str)`, which creates heap allocations:
```go
// Bad: causes 2 heap allocations
subtle.ConstantTimeCompare([]byte(a), []byte(b))

// Good: strictly 0 allocs/op
randkit.ConstantTimeEqual(a, b)
```

### 2.4. Iteration without Slices
Avoid creating slice intermediate copies for traversals:
```go
// Instead of returning []T:
for item := range collection.Values() {
    // Process item
}
```
If the consumer exits early with `break`, push iterators cease immediately with no residual resource leaks.
