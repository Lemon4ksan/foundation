# foundation

Silicon substrate and concurrency runtime for Go 1.27+. Consolidates hardware-accelerated memory primitives, native `simd/archsimd` compiler intrinsics, LLVM-compiled SIMD kernels, and zero-allocation concurrency orchestration into a unified architecture.

[Documentation](https://pkg.go.dev/github.com/lemon4ksan/foundation) | [License (BSD-3-Clause)](LICENSE)

```bash
go get github.com/lemon4ksan/foundation
```

## Architecture: Native SIMD & c2plan9

`foundation` eliminates CGO overhead by combining Go 1.27 `simd/archsimd` inlined compiler intrinsics with a pure C/LLVM to Plan 9 Go Assembler pipeline (`cmd/c2plan9`). Vector instructions run directly on hardware registers with zero memory allocations.

Critical C kernels are compiled via Clang/LLVM, and the ELF64 machine code is translated into native Plan 9 Assembler (`.s`) supporting x86-64 (AVX2/BMI2) and ARM64 (NEON):

```text
foundation/
├── csrc/                           # C/LLVM vector kernels (match.c, hash.c, base64.c, etc.)
└── cmd/c2plan9/                    # C/LLVM -> Plan 9 Assembler compiler (AMD64 / ARM64)
```

Compilation directives are defined per package:

```go
package hexkit

//go:generate c2plan9 -c ../../csrc/hex.c -o hex_amd64.s -stub hex_amd64.go -pkg hexkit
```

To rebuild all kernels:
```bash
go generate ./...
```

## Benchmarks

*Environment: Intel Core i5-12400F, Go 1.27.*

### Protocol Scanning & SIMD Primitives (`silicon/simd`)

| Kernel | Description | Execution Time | Memory Throughput | Allocations |
| :--- | :--- | :--- | :--- | :--- |
| `IndexCRLFCRLFVector` | HTTP header boundary scan (`\r\n\r\n`) | 4.82 ns/op | 212.49 GB/s | 0 allocs |
| `FindMatchLengthVector` | LZ77 long-match scanner (Brotli/Zstd) | 6.14 ns/op | 41.66 GB/s | 0 allocs |
| `ScanByteVector` | 256-bit unrolled single-byte scanner | 16.33 ns/op | 62.70 GB/s | 0 allocs |
| `Hash64Vector` | 64-bit AVX2 bulk hashing | 59.69 ns/op | 17.16 GB/s | 0 allocs |
| `ValidUTF8_SWAR` | 64-bit SWAR UTF-8 validator | 5.78 ns/op | 12.63 GB/s | 0 allocs |

### Hex & Base64 Codecs (vs Standard Library)

| Operation | `foundation` (AVX2) | `encoding/` (Stdlib) | Speedup / Throughput |
| :--- | :--- | :--- | :--- |
| `Hex.Encode1KB` | 78.58 ns/op | 408.20 ns/op | 5.2x (13.03 GB/s) |
| `Hex.Encode16` | 4.86 ns/op | 7.10 ns/op | 1.5x (0 allocs) |
| `AppendToLower1KB` | 29.93 ns/op | 412.10 ns/op | 13.8x (34.58 GB/s) |
| `URL.Unescape1KB` | 804.80 ns/op | 3210.00 ns/op | 4.0x (1.27 GB/s) |

### JSON Parsing (`codec/json`)

| Benchmark | `foundation` (SIMD) | `encoding/json` | Allocated Memory | Speedup |
| :--- | :--- | :--- | :--- | :--- |
| `UnmarshalNoCopy` | 916.4 ns/op | 2028.0 ns/op | 347 B/op (12 allocs) | 2.21x |
| `Unmarshal` | 1041.0 ns/op | 2028.0 ns/op | 392 B/op (17 allocs) | 1.95x |
| `MarshalTo` | 417.7 ns/op | 433.1 ns/op | 192 B/op (2 allocs) | 1.04x |

### UUID Formatting & Parsing (`types/uuid`)

| Operation | Execution Time | Allocations | Details |
| :--- | :--- | :--- | :--- |
| `UUID.Format` | 19.48 ns/op | 0 allocs | 36-char hex+dash buffer formatting |
| `UUID.Append` | 19.81 ns/op | 0 allocs | Direct `[]byte` appending |
| `uuid.Parse` | 31.96 ns/op | 0 allocs | Vectorized hex validation & decoding |

## Package Index

### 1. Hardware Substrate (`silicon/`, `bufkit/`, `binkit/`)
* **`simd`**: AVX2/BMI2 vector processing for frame scanning and match lengths.
* **`hexkit`**: SIMD hex encoder/decoder (13.0 GB/s).
* **`bytesconv`**: Vector casing, Base64 codecs, zero-copy converters, tokenizers (31.8 GB/s).
* **`offheap`**: Unmanaged direct memory slabs bypassing Go GC.
* **`pool`**: Multi-tiered memory arenas, perpetual byte storage, lock-free object pools.
* **`ringbuf`**: Lock-free SPSC / MPMC ring buffers, Structure-of-Arrays (SoA) layout.
* **`clock` & `randkit`**: Syscall-free monotonic clock, lock-free PRNG, UUIDv7.
* **`trie`**: Compressed radix search trees.
* **`bufkit`**: Cacheline-aligned (64B) buffers, scatter-gather `BufferChain`, SPSC `RingBuffer`.
* **`binkit`**: Sequential zero-alloc binary Reader/Writer, JIT struct codecs.

### 2. Codecs & Filesystem (`codec/`, `fskit/`, `pathkit/`, `vfs/`, `iokit/`)
* **`codec`**: Multi-algorithm compression (`brotli`, `zstd`, `gzip`, `flate`, `lz4`, `lzma`, `fse`, `huff0`), filters (`bcj`, `delta`, `shuffle`), SIMD JSON.
* **`fskit`**: Multi-threaded directory walking (`FastWalk`), cross-platform memory-mapped I/O (`Mmap`).
* **`pathkit`**: Immutable Path type, RFC 8089 `file://` URIs, path normalization.
* **`vfs`**: `io/fs.FS` integration with Zip Slip / Tar Slip defenses and resource limits.
* **`iokit`**: Replayable body buffers, zero-alloc `BytesReader`, pooled stream copies.

### 3. CLI & AST Tooling (`argkit/`, `astkit/`, `tuikit/`, `testkit/`)
* **`argkit`**: POSIX flag parsing, short flag stacking (`-la`), attached values, Levenshtein suggestions.
* **`astkit`**: Zero-dependency Go AST inspection, struct field/tag extraction, method discovery.
* **`tuikit`**: Terminal UI framework, subcommand routing, auto-aligned tables, ANSI TrueColor.
* **`testkit`**: Zero-dependency assertion (`assert`), termination (`require`), and method expectation (`mock`).

### 4. Concurrency Orchestration (`async/`)
* **`ctxkit`**: Flat-array, L1-cache resident `context.Context` (0 allocs).
* **`lifecycle`**: Topologically sorted DAG service boot, health monitoring, graceful teardown.
* **`event`**: Type-safe, non-blocking asynchronous event bus.
* **`task`**: Asynchronous task manager with correlation IDs, timeouts, futures.
* **`dedup`**: Single-flight request deduplication with isolated panic boundaries.
* **`fsm`**: Compile-time type-safe finite state machines with transactional rollback.
* **`pipeline`**: Concurrent worker pipelines, token-bucket rate limiting, DataLoader batching.
* **`pool`**: Auto-scaling goroutine worker pools with idle scale-down and panic recovery.
* **`scheduler`**: Microsecond-precision recurring task schedulers and cron runners.
* **`logkit`**: Zero-allocation structured logging facade, asynchronous flushing.

### 5. Synchronization & Generics (`sync/`, `generic/`)
* **`sync`**: Striped key-based locks, Vegas adaptive limiters, circuit breakers, jittered backoff.
* **`generic`**: Thread-safe `Safe[T]`, `LRU[K, V]` cache, `ResourcePool[T]`, in-memory TTL `Cache[K, V]`, monadic `Optional`/`Result`, lazy `Stream[T]` (`iter.Seq`).

### 6. Network Primitives (`net/`)
* **`net/http/header`**: Canonical HTTP constants, pseudo-headers, zero-allocation header map parser.
* **`net/urlkit`**: CRC32 sharded URL cache, path variable expansion, query param appending.
* **`net`**: HPACK compression, gRPC-Web framing, RFC 9211 Cache-Status, DoH/DoQ/DoT DNS, Proxy connectors.

### 7. Types (`text/`, `types/`)
* **`text/htmlkit`**: Zero-allocation HTML entity unescaping.
* **`types/uuid`**: RFC 9562 UUIDv4/v7 generators, SIMD formatting and parsing.
* **`types/values`**: Type conversions and structured extraction.
