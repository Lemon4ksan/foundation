# Foundation

### Silicon Substrate & Tactical Concurrency Runtime for Go (Go 1.27+)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation)
[![License](https://img.shields.io/github/license/lemon4ksan/foundation?style=flat-square)](LICENSE)

`foundation` is a high-performance substrate and concurrency runtime for Go 1.27+. It consolidates hardware-accelerated memory primitives, native `simd/archsimd` compiler intrinsics, LLVM-compiled SIMD kernels, and zero-allocation concurrency orchestration into a unified architecture.

```bash
go get github.com/lemon4ksan/foundation
```

## Benchmarks (Intel Core i5-12400F, Go 1.27)

`foundation` combines native **Go 1.27 `simd/archsimd` inlined compiler intrinsics** (with 4-way loop unrolling) and a pure C/LLVM to Plan 9 Go Assembler compiler pipeline (`cmd/c2plan9`), eliminating CGO overhead and running vector instructions directly on hardware registers with 0 memory allocations:

### 1. Protocol Scanning & SIMD Primitives (`silicon/simd`)

| Kernel | Description | Execution Time | Memory Throughput | Allocations |
| :--- | :--- | :--- | :--- | :--- |
| `IndexCRLFCRLFVector` | HTTP header boundary scan (`\r\n\r\n` inlined `archsimd`) | 4.82 ns/op | 212.49 GB/s | 0 allocs |
| `FindMatchLengthVector` | LZ77 long-match scanner (Brotli / Zstd) | 6.14 ns/op | 41.66 GB/s | 0 allocs |
| `ScanByteVector` | 256-bit unrolled single-byte scanner (`:`, `,`, `"`) | 16.33 ns/op | 62.70 GB/s | 0 allocs |
| `Hash64Vector` | 64-bit AVX2 bulk hashing | 59.69 ns/op | 17.16 GB/s | 0 allocs |
| `ValidUTF8_SWAR` | 64-bit SWAR UTF-8 validator | 5.78 ns/op | 12.63 GB/s | 0 allocs |

### 2. Hex & Base64 Codecs vs Go Standard Library

| Operation | `foundation` (AVX2) | Standard Library | Speedup / Throughput |
| :--- | :--- | :--- | :--- |
| `Hex.Encode1KB` | 78.58 ns/op | 408.20 ns/op | 5.2x faster (13.03 GB/s) |
| `Hex.Encode16` | 4.86 ns/op | 7.10 ns/op | 1.5x faster (0 allocs) |
| `AppendToLower1KB` | 29.93 ns/op | 412.10 ns/op | 13.8x faster (34.58 GB/s) |
| `URL.Unescape1KB` | 804.80 ns/op | 3210.00 ns/op | 4.0x faster (1.27 GB/s) |

### 3. High-Throughput JSON Parsing (`codec/json`)

| Benchmark | `foundation` (SIMD) | `encoding/json` | Allocated Memory | Speedup |
| :--- | :--- | :--- | :--- | :--- |
| `UnmarshalNoCopy` | 916.4 ns/op | 2028.0 ns/op | 347 B/op (12 allocs) | 2.21x faster |
| `Unmarshal` | 1041.0 ns/op | 2028.0 ns/op | 392 B/op (17 allocs) | 1.95x faster |
| `MarshalTo` | 417.7 ns/op | 433.1 ns/op | 192 B/op (2 allocs) | 1.04x faster |

### 4. UUID Formatting & Parsing (`types/uuid`)

| Operation | Time per Op | Allocations | Details |
| :--- | :--- | :--- | :--- |
| `UUID.Format` | 19.48 ns/op | 0 allocs | 36-char hex+dash buffer formatting |
| `UUID.Append` | 19.81 ns/op | 0 allocs | Direct `[]byte` appending |
| `uuid.Parse` | 31.96 ns/op | 0 allocs | Vectorized hex validation & decoding |

## Native SIMD & c2plan9 Architecture

`foundation` compiles performance-critical C kernels using Clang/LLVM and translates ELF64 machine code into native Plan 9 Go Assembler (`.s`) supporting both x86-64 (AVX2/BMI2) and ARM64 (NEON):

```text
foundation/
├── csrc/                           # Pure C/LLVM vector kernels
│   ├── equalfold.c                 # Case-folding vector comparisons
│   ├── fastscan.c                  # CRLFCRLF & byte boundary scanners
│   ├── match.c                     # LZ77 match length vector finder
│   ├── hash.c                      # AVX2 bulk hashing
│   ├── hex.c                       # Vectorized hexadecimal codec
│   ├── uuid.c                      # Unrolled zero-branch UUID formatter & parser
│   ├── json.c                      # SIMD whitespace and string escape scanners
│   ├── casing.c                    # 32-byte vector ASCII ToLower/ToUpper
│   ├── base64.c                    # Hardware Base64 codec
│   └── urlencode.c                 # Vector URL percent-unescape scanner
│
└── cmd/c2plan9/                    # Automatic C/LLVM -> Plan 9 Assembler tool (AMD64 / ARM64)
```

### Adding or Recompiling Kernels
Every package defines a standard `generate.go` directive:

```go
package hexkit

//go:generate c2plan9 -c ../../csrc/hex.c -o hex_amd64.s -stub hex_amd64.go -pkg hexkit
```

To recompile all kernels across the entire repository:
```bash
go generate ./...
```

## Architecture & Packages

### 1. Silicon & Hardware Substrate (`silicon/`, `bufkit/`, `binkit/`)
* **`simd`**: AVX2/BMI2 vector processing for frame scanning and match lengths.
* **`hexkit`**: 13.0 GB/s SIMD hex encoder and decoder.
* **`bytesconv`**: 31.8 GB/s vector casing, Base64 codecs, zero-copy converters, and tokenizers.
* **`offheap`**: Unmanaged direct memory slabs bypassing the Go GC.
* **`pool`**: Multi-tiered memory arenas, perpetual byte storage, and lock-free object pools.
* **`ringbuf`**: Lock-free SPSC / MPMC ring buffers and Structure-of-Arrays (SoA) layout.
* **`clock` & `randkit`**: Monotonic fast-clock without syscalls, lock-free PRNG, and UUIDv7.
* **`trie`**: Compressed radix search trees for high-speed prefix routing.
* **`bufkit`**: Cacheline-aligned (64B) buffers, scatter-gather `BufferChain`, and SPSC `RingBuffer`.
* **`binkit`**: Sequential zero-allocation binary Reader/Writer with sticky errors and JIT struct codecs.

### 2. Core Codecs & Filesystem (`codec/`, `fskit/`, `pathkit/`, `vfs/`, `iokit/`)
* **`codec`**: Multi-algorithm compression (`brotli`, `zstd`, `gzip`, `flate`, `lz4`, `lzma`, `fse`, `huff0`), pre-compression filters (`bcj`, `delta`, `shuffle`), and SIMD JSON.
* **`fskit`**: High-performance multi-threaded directory walking (`FastWalk`) and cross-platform memory-mapped I/O (`Mmap`).
* **`pathkit`**: Unified immutable Path type, RFC 8089 `file://` URIs, and cross-platform path normalization.
* **`vfs`**: Standard `io/fs.FS` integration with Zip Slip / Tar Slip path defenses and extraction resource limits.
* **`iokit`**: Replayable body buffers, allocation-free `BytesReader`, and pooled stream copy helpers.

### 3. CLI & Developer Tooling (`argkit/`, `astkit/`, `tuikit/`, `testkit/`)
* **`argkit`**: POSIX flag parsing with interspersing, short flag stacking (`-la`), attached values, and Levenshtein typo suggestions.
* **`astkit`**: Zero-dependency Go AST inspection, struct field/tag extraction, method discovery, and expression parsing.
* **`tuikit`**: Terminal UI framework, subcommand routing, auto-aligned tables, bordered boxes, badges, progress bars, and ANSI TrueColor detection.
* **`testkit`**: Zero-dependency assertion (`assert`), immediate termination (`require`), and method expectation (`mock`) toolkit.

### 4. Concurrency & Runtime Orchestration (`async/`)
* **`ctxkit`**: Flat-array, L1-cache resident `context.Context` with zero allocations.
* **`lifecycle`**: Topologically sorted DAG service boot, health monitoring, and graceful teardown.
* **`event`**: Type-safe, non-blocking asynchronous event bus.
* **`task`**: Asynchronous task manager with correlation IDs, timeouts, and futures.
* **`dedup`**: Single-flight request deduplication with isolated panic boundaries.
* **`fsm`**: Compile-time type-safe finite state machines with transactional rollback.
* **`pipeline`**: Concurrent worker pipelines with token-bucket rate limiting and DataLoader batching.
* **`pool`**: Auto-scaling goroutine worker pools with idle scale-down and panic recovery.
* **`scheduler`**: Microsecond-precision recurring task schedulers and cron runners.
* **`logkit`**: Zero-allocation structured logging facade with asynchronous flushing.

### 5. Tactical Synchronization & Generics (`sync/`, `generic/`)
* **`sync`**: Striped key-based locks, Vegas adaptive limiters, circuit breakers, and jittered backoff.
* **`generic`**: Thread-safe `Safe[T]`, `LRU[K, V]` cache, `ResourcePool[T]`, in-memory TTL `Cache[K, V]`, monadic `Optional`/`Result`, and lazy `Stream[T]` (`iter.Seq`) pipelines.

### 6. Low-Level Network Protocol Primitives (`net/`)
* **`net/http/header`**: Canonical HTTP header constants, pseudo-headers, MIME media types, and zero-allocation header map parser.
* **`net/urlkit`**: CRC32 sharded URL cache, path variable expansion, and fast query param appending.
* **`net`**: HPACK compression, gRPC-Web framing, RFC 9211 Cache-Status, DoH/DoQ/DoT DNS, and Proxy connectors.

### 7. Core Types (`text/`, `types/`)
* **`text/htmlkit`**: Zero-allocation HTML entity unescaping.
* **`types/uuid`**: RFC 9562 UUIDv4 and UUIDv7 generators with SIMD formatting and parsing.
* **`types/values`**: High-speed type conversions and structured extraction.

## License

BSD-3-Clause License. See [LICENSE](LICENSE) for details.
