# Foundation Documentation Hub

Welcome to the technical documentation hub for `github.com/lemon4ksan/foundation` — the unified silicon substrate and concurrency runtime for Go.

## Architectural Layout

```text
foundation/
├── silicon/                  // Hardware Substrate (0 B/op, Memory, SIMD)
│   ├── SIMD.md               // 256-bit AVX2/BMI2 vector processing (81.7 GB/s)
│   ├── BYTESCONV.md          // Zero-copy scanning, slicing, and unsafe conversions
│   ├── OFFHEAP.md            // Direct memory slabs bypassing Go GC
│   ├── POOL.md               // Multi-tier memory arenas and perpetual byte storage
│   ├── RINGBUF.md            // Lock-free SPSC / MPMC ring buffers and SoA
│   ├── CLOCK_AND_RAND.md     // Monotonic fast-clock, lock-free fastrand, UUID v7
│   └── TRIE.md               // Compressed radix search trees
│
├── bufkit/                   // High-Performance Memory Buffers
│   └── BUFKIT.md             // Cache-aligned buffers, BufferChain, RingBuffer
│
├── timekit/                  // High-Throughput Time & Dates
│   └── TIMEKIT.md            // CoarseNow, zero-alloc HTTP-date & ISO 8601, Stopwatch
│
├── refkit/                   // Struct & Type Reflection Helpers
│   └── REFKIT.md             // Tag parsing with cache, panic-safe reflection checks
│
├── async/                    // Concurrency & Runtime Orchestration
│   ├── LIFECYCLE.md          // DAG DFS topological service boot & BehaviorRunner
│   ├── EVENT.md              // Type-safe non-blocking event bus
│   ├── TASK.md               // Correlation-ID task manager, timeouts, and futures
│   ├── DEDUP.md              // Single-flight request deduplication & panic isolation
│   ├── FSM.md                // Strictly typed FSM with rollback & DOT export
│   ├── PIPELINE.md           // Concurrent mapping, fan-out/fan-in, DataLoader
│   ├── CONTEXT.md            // Ultra-fast flat-array context (0 B/op, L1 cache)
│   ├── POOL.md               // Auto-scaling goroutine worker pool
│   ├── SCHEDULER.md          // Microsecond precision task scheduler & cron
│   └── LOG.md                // Zero-allocation structured logging
│
├── sync/                     // Tactical Synchronization & Resilience
│   └── SYNC.md               // KeyLock, Limiter (Vegas/Keyed), Breaker, Backoff, Semaphore, Lazy, SpinLock
│
├── generic/                  // Type-Safe Generics & Collections
│   └── GENERIC.md            // Safe[T], Cache[K,V], LRU, Pool, Optional/Result, Slices, Maps, Stream
│
├── io/                       // Streaming I/O & Replayable Buffers
│   └── IO.md                 // ReplayableBody, BytesReader, Stream Limits, Copy Pools
│
└── net/                      // Low-Level Network Protocol Primitives
    └── NET.md                // Header, HPACK, gRPC-Web, Cache-Status, DoH/DoQ, Proxy, Cookie, PSL
```

## Module Index

### Silicon & Memory Substrate (`silicon/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `simd` | [`docs/silicon/SIMD.md`](silicon/SIMD.md) | Vectorized 256-bit AVX2 frame masking at 81.7 GB/s. |
| `bytesconv` | [`docs/silicon/BYTESCONV.md`](silicon/BYTESCONV.md) | Zero-copy string/slice conversions and delimiter tokenization. |
| `offheap` | [`docs/silicon/OFFHEAP.md`](silicon/OFFHEAP.md) | Direct virtual memory slabs eliminating GC scan latency. |
| `pool` | [`docs/silicon/POOL.md`](silicon/POOL.md) | Contiguous bump-allocator arenas with O(1) instant reset. |
| `ringbuf` | [`docs/silicon/RINGBUF.md`](silicon/RINGBUF.md) | Lock-free SPSC / MPMC ring buffers eliminating channel mutex overhead. |
| `clock`/`rand` | [`docs/silicon/CLOCK_AND_RAND.md`](silicon/CLOCK_AND_RAND.md) | Monotonic clock without syscalls and lock-free sortable UUID v7. |
| `trie` | [`docs/silicon/TRIE.md`](silicon/TRIE.md) | Compressed radix prefix trees for URL routing and domain lookup. |

### High-Performance Buffers (`bufkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `bufkit` | [`docs/bufkit/BUFKIT.md`](bufkit/BUFKIT.md) | Cacheline-aligned buffers, scatter-gather BufferChain, and SPSC RingBuffer. |

### High-Throughput Time & Dates (`timekit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `timekit` | [`docs/timekit/TIMEKIT.md`](timekit/TIMEKIT.md) | Coarse atomic clock, zero-alloc HTTP-date / ISO 8601 formatting, stopwatch. |

### Struct & Type Reflection Helpers (`refkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `refkit` | [`docs/refkit/REFKIT.md`](refkit/REFKIT.md) | High-speed struct tag parsing with cache and panic-safe zero-alloc type checks. |

### Concurrency & Runtime Orchestration (`async/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `context` | [`docs/async/CONTEXT.md`](async/CONTEXT.md) | High-performance flat-array `context.Context` (0 B/op, L1 cache, generics). |
| `lifecycle` | [`docs/async/LIFECYCLE.md`](async/LIFECYCLE.md) | Topologically sorted DAG service boot and reverse graceful teardown. |
| `event` | [`docs/async/EVENT.md`](async/EVENT.md) | Type-safe non-blocking event bus preventing slow consumer backpressure. |
| `task` | [`docs/async/TASK.md`](async/TASK.md) | Correlation-ID async task manager with pooled memory and deadlines. |
| `dedup` | [`docs/async/DEDUP.md`](async/DEDUP.md) | Single-flight request coalescing with safe worker panic isolation. |
| `fsm` | [`docs/async/FSM.md`](async/FSM.md) | Strictly typed finite state machines with transactional rollback. |
| `pipeline` | [`docs/async/PIPELINE.md`](async/PIPELINE.md) | Concurrent mapping pipelines with rate limiting and DataLoader batching. |
| `pool` | [`docs/async/POOL.md`](async/POOL.md) | Auto-scaling goroutine worker pool with idle scale-down and futures. |
| `scheduler` | [`docs/async/SCHEDULER.md`](async/SCHEDULER.md) | Microsecond precision task scheduler with recurring interval loops. |
| `log` | [`docs/async/LOG.md`](async/LOG.md) | Zero-allocation structured logging facade with asynchronous flushing. |

### Tactical Synchronization (`sync/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `sync` | [`docs/sync/SYNC.md`](sync/SYNC.md) | Striped KeyLock, Vegas AdaptiveLimiter, CircuitBreaker, Jittered Backoff, and Resizable Semaphore. |

### Type-Safe Generics & Collections (`generic/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `generic` | [`docs/generic/GENERIC.md`](generic/GENERIC.md) | Thread-safe `Safe[T]`, `LRU[K, V]`, `ResourcePool[T]`, in-memory `Cache[K, V]`, monadic `Optional`/`Result`, and lazy `Stream[T]` iterators. |

### Streaming I/O & Replay Buffers (`io/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `io` | [`docs/io/IO.md`](io/IO.md) | Replayable body buffers, allocation-free `BytesReader`, and pooled stream copy helpers. |

### Low-Level Network Protocol Primitives (`net/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `net` | [`docs/net/NET.md`](net/NET.md) | Canonical HTTP headers, HPACK compression, gRPC-Web framing, RFC 9211 Cache-Status, DoH/DoQ/DoT DNS, and Proxy engines. |
