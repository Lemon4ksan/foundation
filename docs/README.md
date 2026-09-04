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
├── argkit/                   // Command-Line Argument & Flag Parsing
│   └── ARGKIT.md             // POSIX flag interspersing, short flag stacking, typo suggestions
│
├── astkit/                   // Go AST Code Inspection & Script Parsing
│   └── ASTKIT.md             // AST traversal, struct/method introspection, expression parsing
│
├── binkit/                   // Binary Encoding & Fast Serialization
│   └── BINKIT.md             // Sequential Reader/Writer, sticky errors, JIT struct codec
│
├── bufkit/                   // High-Performance Memory Buffers
│   └── BUFKIT.md             // Cache-aligned buffers, BufferChain, RingBuffer
│
├── codec/                    // Compression & Format Codecs
│   └── CODEC.md              // Brotli, Zstd, Gzip, Deflate, LZ4, LZMA, Filters (BCJ/Delta/Shuffle), JSON
│
├── fskit/                    // High-Throughput Filesystem Primitives
│   └── FSKIT.md              // FastWalk directory traversal, cross-platform mmap
│
├── pathkit/                  // Unified Path & URI Abstraction
│   └── PATHKIT.md            // File URIs (RFC 8089), network URLs, OS paths
│
├── testkit/                  // Zero-Dependency Test Suite & Mocks
│   └── TESTKIT.md            // Assertions, require, mock expectations, gomock
│
├── timekit/                  // High-Throughput Time & Dates
│   └── TIMEKIT.md            // CoarseNow, zero-alloc HTTP-date & ISO 8601, Stopwatch
│
├── tuikit/                   // Terminal UI & CLI App Framework
│   └── TUIKIT.md             // App routing, table formatting, bordered boxes, badges, ANSI probe
│
├── types/                    // Core Types & Identifiers
│   └── TYPES.md              // RFC 9562 UUIDv4/v7 with SIMD parsing, dynamic values
│
├── vfs/                      // Virtual Filesystem & Security Defenses
│   └── VFS.md                // io/fs.FS implementation, Zip Slip defense, extraction limits
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
│   ├── CTXKIT.md             // Ultra-fast flat-array context (0 B/op, L1 cache)
│   ├── POOL.md               // Auto-scaling goroutine worker pool
│   ├── SCHEDULER.md          // Microsecond precision task scheduler & cron
│   └── LOGKIT.md             // Zero-allocation structured logging
│
├── sync/                     // Tactical Synchronization & Resilience
│   └── SYNC.md               // KeyLock, Limiter (Vegas/Keyed), Breaker, Backoff, Semaphore, Lazy, SpinLock
│
├── generic/                  // Type-Safe Generics & Collections
│   └── GENERIC.md            // Safe[T], Cache[K,V], LRU, Pool, Optional/Result, Slices, Maps, Stream
│
├── iokit/                    // Streaming I/O & Replayable Buffers
│   └── IOKIT.md              // ReplayableBody, BytesReader, Stream Limits, Copy Pools
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
| `clock`/`randkit` | [`docs/silicon/CLOCK_AND_RAND.md`](silicon/CLOCK_AND_RAND.md) | Monotonic clock without syscalls and lock-free sortable UUID v7. |
| `hexkit` | [`docs/silicon/BYTESCONV.md`](silicon/BYTESCONV.md) | Vectorized AVX2 hexadecimal encoding and decoding. |
| `trie` | [`docs/silicon/TRIE.md`](silicon/TRIE.md) | Compressed radix prefix trees for URL routing and domain lookup. |

### CLI Arguments & Flag Parsing (`argkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `argkit` | [`docs/argkit/ARGKIT.md`](argkit/ARGKIT.md) | POSIX flag interspersing, short flag stacking (`-la`), attached values, typo suggestions. |

### AST Inspection & Code Analysis (`astkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `astkit` | [`docs/astkit/ASTKIT.md`](astkit/ASTKIT.md) | Go AST traversal, struct tag extraction, method inspection, statement parsing. |

### Binary Encoding & Layout Serialization (`binkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `binkit` | [`docs/binkit/BINKIT.md`](binkit/BINKIT.md) | Sequential zero-allocation binary Reader/Writer, sticky errors, JIT struct serialization. |

### High-Performance Buffers (`bufkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `bufkit` | [`docs/bufkit/BUFKIT.md`](bufkit/BUFKIT.md) | Cacheline-aligned buffers, scatter-gather BufferChain, and SPSC RingBuffer. |

### Compression & Codecs (`codec/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `codec` | [`docs/codec/CODEC.md`](codec/CODEC.md) | Brotli, Zstd, Gzip, Deflate, LZ4, LZMA, pre-compression filters (BCJ/Delta/Shuffle), SIMD JSON. |

### Filesystem Primitives (`fskit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `fskit` | [`docs/fskit/FSKIT.md`](fskit/FSKIT.md) | Parallel multi-threaded directory walking (`FastWalk`), cross-platform mmap. |

### Path & URI Abstraction (`pathkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `pathkit` | [`docs/pathkit/PATHKIT.md`](pathkit/PATHKIT.md) | Unified immutable Path type, RFC 8089 file:// URIs, clean normalization. |

### Testing & Mocking Toolkit (`testkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `testkit` | [`docs/testkit/TESTKIT.md`](testkit/TESTKIT.md) | Zero-dependency test assertions (`assert`), immediate failure (`require`), method `mock`. |

### High-Throughput Time & Dates (`timekit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `timekit` | [`docs/timekit/TIMEKIT.md`](timekit/TIMEKIT.md) | Coarse atomic clock, zero-alloc HTTP-date / ISO 8601 formatting, stopwatch. |

### Terminal UI & CLI Framework (`tuikit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `tuikit` | [`docs/tuikit/TUIKIT.md`](tuikit/TUIKIT.md) | CLI subcommands, formatted data tables, bordered boxes, progress indicators, ANSI sniffer. |

### Core Types (`types/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `types` | [`docs/types/TYPES.md`](types/TYPES.md) | RFC 9562 UUIDv4/v7 with SIMD parsers, zero-allocation dynamic value extraction. |

### Virtual Filesystem & Path Security (`vfs/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `vfs` | [`docs/vfs/VFS.md`](vfs/VFS.md) | Standard io/fs.FS integration, Zip Slip / Tar Slip traversal defenses, extraction limits. |

### Struct & Type Reflection Helpers (`refkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `refkit` | [`docs/refkit/REFKIT.md`](refkit/REFKIT.md) | High-speed struct tag parsing with cache and panic-safe zero-alloc type checks. |

### Concurrency & Runtime Orchestration (`async/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `ctxkit` | [`docs/async/CTXKIT.md`](async/CTXKIT.md) | High-performance flat-array `context.Context` (0 B/op, L1 cache, generics). |
| `lifecycle` | [`docs/async/LIFECYCLE.md`](async/LIFECYCLE.md) | Topologically sorted DAG service boot and reverse graceful teardown. |
| `event` | [`docs/async/EVENT.md`](async/EVENT.md) | Type-safe non-blocking event bus preventing slow consumer backpressure. |
| `task` | [`docs/async/TASK.md`](async/TASK.md) | Correlation-ID async task manager with pooled memory and deadlines. |
| `dedup` | [`docs/async/DEDUP.md`](async/DEDUP.md) | Single-flight request coalescing with safe worker panic isolation. |
| `fsm` | [`docs/async/FSM.md`](async/FSM.md) | Strictly typed finite state machines with transactional rollback. |
| `pipeline` | [`docs/async/PIPELINE.md`](async/PIPELINE.md) | Concurrent mapping pipelines with rate limiting and DataLoader batching. |
| `pool` | [`docs/async/POOL.md`](async/POOL.md) | Auto-scaling goroutine worker pool with idle scale-down and futures. |
| `scheduler` | [`docs/async/SCHEDULER.md`](async/SCHEDULER.md) | Microsecond precision task scheduler with recurring interval loops. |
| `logkit` | [`docs/async/LOGKIT.md`](async/LOGKIT.md) | Zero-allocation structured logging facade with asynchronous flushing. |

### Tactical Synchronization (`sync/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `sync` | [`docs/sync/SYNC.md`](sync/SYNC.md) | Striped KeyLock, Vegas AdaptiveLimiter, CircuitBreaker, Jittered Backoff, and Resizable Semaphore. |

### Type-Safe Generics & Collections (`generic/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `generic` | [`docs/generic/GENERIC.md`](generic/GENERIC.md) | Thread-safe `Safe[T]`, `LRU[K, V]`, `ResourcePool[T]`, in-memory `Cache[K, V]`, monadic `Optional`/`Result`, and lazy `Stream[T]` iterators. |

### Streaming I/O & Replay Buffers (`iokit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `iokit` | [`docs/iokit/IOKIT.md`](iokit/IOKIT.md) | Replayable body buffers, allocation-free `BytesReader`, and pooled stream copy helpers. |

### Low-Level Network Protocol Primitives (`net/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `net` | [`docs/net/NET.md`](net/NET.md) | Canonical HTTP headers, URL parsing (`urlkit`), HPACK compression, gRPC-Web framing, RFC 9211 Cache-Status, DoH/DoQ/DoT DNS, and Proxy engines. |
