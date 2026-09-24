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
│   ├── RINGBUF.md            // Lock-free SPSC / MPMC ring buffers
│   ├── CLOCK_AND_RAND.md     // Monotonic fast-clock, lock-free fastrand, UUID v7
│   └── TRIE.md               // Compressed radix search trees
│
├── argkit/                   // Command-Line Argument & Flag Parsing
│   └── ARGKIT.md             // POSIX flag interspersing, short flag stacking, typo suggestions
│
├── borrow/                   // Generational Borrow Checker & Arenas
│   └── BORROW.md             // Scope arenas, Safe References (Ref/Mut), Zero UAF
│
├── encoding/                 // Binary Encoding & Fast Serialization
│   ├── bin/BIN.md            // Sequential Reader/Writer, sticky errors, JIT struct codec
│   └── VARINT.md             // QUIC Varint Encoding (SIMD)
│
├── bufkit/                   // High-Performance Memory Buffers
│   └── BUFKIT.md             // Cache-aligned buffers, BufferChain, RingBuffer
│
├── codec/                    // Compression & Format Codecs
│   └── CODEC.md              // Brotli, Zstd, Gzip, Deflate, LZ4, LZMA, Filters (BCJ/Delta/Shuffle), JSON
│
├── crypto/                   // High-Performance Cryptography
│   └── CRYPTO.md             // AEAD, KDF, Signatures
│
├── fskit/                    // High-Throughput Filesystem Primitives
│   └── FSKIT.md              // FastWalk directory traversal, cross-platform mmap
│
├── pathkit/                  // Unified Path & URI Abstraction
│   └── PATHKIT.md            // File URIs (RFC 8089), network URLs, OS paths
│
├── testing/                  // Zero-Dependency Test Suite & Mocks
│   └── TESTING.md            // Assertions, require, mock expectations, gomock
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
├── structures/               // Data Structures
│   └── STRUCTURES.md         // MinHeap, RingBuffer, LinkedList
│
├── generic/                  // Type-Safe Generics & Collections
│   └── GENERIC.md            // Safe[T], Cache[K,V], LRU, Pool, Optional/Result, Slices, Maps, Stream
│
├── text/                     // Text Processing
│   └── TEXT.md               // Casing, Diff, Encoding
│
├── iokit/                    // Streaming I/O & Replayable Buffers
│   └── IOKIT.md              // ReplayableBody, BytesReader, Stream Limits, Copy Pools
│
└── net/                      // Low-Level Network Protocol Primitives
    └── NET.md                // QUIC, URL Engine (urlkit), Proxy, IP, IPC, Host Normalization
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

### Binary Encoding & Layout Serialization (`encoding/`, `borrow/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `encoding/bin` | [`docs/encoding/bin/BIN.md`](encoding/bin/BIN.md) | Sequential zero-allocation binary Reader/Writer, sticky errors, JIT struct serialization. |
| `encoding/varint` | [`docs/encoding/VARINT.md`](encoding/VARINT.md) | QUIC Varint encoding using SIMD intrinsics. |
| `borrow` | [`docs/borrow/BORROW.md`](borrow/BORROW.md) | Generational borrow checker, scope arenas, safe references (Ref/Mut) with zero UAF. |

### High-Performance Buffers (`bufkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `bufkit` | [`docs/bufkit/BUFKIT.md`](bufkit/BUFKIT.md) | Cacheline-aligned buffers, scatter-gather BufferChain, and SPSC RingBuffer. |

### Compression & Codecs (`codec/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `codec` | [`docs/codec/CODEC.md`](codec/CODEC.md) | Brotli, Zstd, Gzip, Deflate, LZ4, LZMA, pre-compression filters (BCJ/Delta/Shuffle), SIMD JSON. |

### Cryptography (`crypto/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `crypto` | [`docs/crypto/CRYPTO.md`](crypto/CRYPTO.md) | AEAD wrappers, KDFs (HKDF, Argon2, PBKDF2), and high-throughput digital signatures. |

### Filesystem Primitives (`fskit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `fskit` | [`docs/fskit/FSKIT.md`](fskit/FSKIT.md) | Parallel multi-threaded directory walking (`FastWalk`), cross-platform mmap. |

### Path & URI Abstraction (`pathkit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `pathkit` | [`docs/pathkit/PATHKIT.md`](pathkit/PATHKIT.md) | Unified immutable Path type, RFC 8089 file:// URIs, clean normalization. |

### Testing & Mocking Toolkit (`testing/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `testing` | [`docs/testing/TESTING.md`](testing/TESTING.md) | Zero-dependency test assertions (`assert`), immediate failure (`require`), method `mock`. |

### High-Throughput Time & Dates (`timekit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `timekit` | [`docs/timekit/TIMEKIT.md`](timekit/TIMEKIT.md) | Coarse atomic clock, zero-alloc HTTP-date / ISO 8601 formatting, stopwatch. |

### Terminal UI & CLI Framework (`tuikit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `tuikit` | [`docs/tuikit/TUIKIT.md`](tuikit/TUIKIT.md) | CLI subcommands, formatted data tables, bordered boxes, progress indicators, ANSI sniffer. |

### Core Types & Text Processing (`types/`, `text/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `types` | [`docs/types/TYPES.md`](types/TYPES.md) | RFC 9562 UUIDv4/v7 with SIMD parsers, zero-allocation dynamic value extraction. |
| `text` | [`docs/text/TEXT.md`](text/TEXT.md) | Text processing: casing, diffs, charset encodings, and stream transformers. |

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

### Type-Safe Generics & Collections (`generic/`, `structures/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `generic` | [`docs/generic/GENERIC.md`](generic/GENERIC.md) | Thread-safe `Safe[T]`, `LRU[K, V]`, `ResourcePool[T]`, in-memory `Cache[K, V]`, monadic `Optional`/`Result`, and lazy `Stream[T]` iterators. |
| `structures` | [`docs/structures/STRUCTURES.md`](structures/STRUCTURES.md) | Zero-alloc, generic data structures like `MinHeap`, `RingBuffer`, and `LinkedList`. |

### Streaming I/O & Replay Buffers (`iokit/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `iokit` | [`docs/iokit/IOKIT.md`](iokit/IOKIT.md) | Replayable body buffers, allocation-free `BytesReader`, and pooled stream copy helpers. |

### Low-Level Network Protocol Primitives (`net/`)

| Module | Documentation | Focus Area |
| :--- | :--- | :--- |
| `net` | [`docs/net/NET.md`](net/NET.md) | QUIC transport (RFC 9000), URL engine (`urlkit`), SOCKS4/5 & HTTP proxies, IP routing, IPC, and Host sanitization. |
