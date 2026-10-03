# foundation

[![Go Version](https://img.shields.io/badge/go-1.27%2B-007d9c?logo=go&logoColor=white&style=flat-square)](https://go.dev/)
[![Go Reference](https://img.shields.io/badge/godoc-reference-007d9c?style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue?style=flat-square)](LICENSE)

`foundation` is a Go standard library replacement for systems where the standard library's defaults are too slow or allocate too much. It reverses the compromises made by Go's `net` and `encoding` packages, exchanging simplicity and stability for SIMD acceleration, explicit memory management, and zero-allocation hot paths.

It is used as the base layer for my other network and storage systems: [`aoni`](https://github.com/lemon4ksan/aoni), [`sein`](https://github.com/lemon4ksan/sein), [`mach`](https://github.com/lemon4ksan/mach), and [`seal`](https://github.com/lemon4ksan/seal).

## Design tradeoffs

Read the full reasoning in [docs/VISION.md](docs/VISION.md) and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

| Standard library compromise | What foundation does instead | What you pay |
| :--- | :--- | :--- |
| API frozen forever | Breaking changes are documented in the CHANGELOG. | You have to pin versions and upgrade on purpose. |
| Same code on every CPU | Hardware SIMD (AVX2/NEON), assembly via `c2plan9`. | Primarily targets amd64 and arm64. |
| One signature fits all | Specialized `Append*` and `dst []byte` variants. | Larger API surface, steeper learning curve. |
| Allocations are acceptable | Zero-alloc contracts, off-heap memory, `borrow`. | You have to manage buffer lifetimes manually. |

## Rules

Development in `foundation` follows strict Architecture Decision Records ([ADRs](docs/adr/README.md)):

1. **Layering ([ADR-0002](docs/adr/0002-layering-and-isolation.md))**: The codebase is split into strict L0-L5 layers. Imports only point down. The `silicon/` package is isolated from everything else. `generic` is for external consumers only.
2. **Allocation Budgets ([ADR-0003](docs/adr/0003-allocation-budgets.md))**: Every package documents its allocation tier (`zero`, `amortized`, `bounded`, `unconstrained`), verified by `alloc_test.go`.
3. **No L7 Protocols ([ADR-0006](docs/adr/0006-l7-boundary-and-mach.md))**: Application protocols like HTTP, HPACK, QPACK, and gRPC do not belong here. They belong in `mach` or the consuming project.
4. **Documentation ([ADR-0008](docs/adr/0008-doc-go-canonical-truth.md))**: API documentation is in `doc.go` and `example_test.go` only.

## Layout

### Layer 0: Hardware Substrate (`silicon/`)
Autonomous packages with zero internal dependencies.
* `simd`: AVX2/BMI2 vector processing for frame scanning and match lengths.
* `hexkit`, `bytesconv`: SIMD codecs (13 GB/s hex, 31 GB/s Base64).
* `pool`, `ringbuf`: Multi-tiered memory arenas, lock-free object pools.
* `clock`, `randkit`: Syscall-free monotonic clocks, lock-free PRNG.

### Layer 1: Memory & Base Primitives
* `sync`: Adaptive Vegas limiters, strip locks, circuit breakers.
* `bufkit`, `borrow`: Cacheline-aligned (64B) buffers and generational ownership arenas.

### Layer 2: Structures & Formats
* `structures`: Zero-allocation typed data structures (`minheap`, `deque`, `ringbuffer`).
* `codec`: Streaming `brotli`, `zstd`, `gzip`, `lz4`, `lzma`, `fse`, `huff0`, and SIMD `json`.
* `crypto`: Hardware-accelerated AEAD ciphers and KDFs.

### Layer 3: Networking & Concurrency
* `async`: Zero-alloc `ctxkit`, topologically sorted `lifecycle`, `fsm`, `dedup`, and `logkit`.
* `net`: Zero-alloc `ip` / `ipc` manipulation, `quic` transport, and proxy dialers.

### Layer 4 & 5: Ecosystem & Userland
* `generic`: Monadic types (`Optional`), thread-safe `LRU`, lazy `Stream`.
* `argkit`, `tuikit`: Terminal UI frameworks and POSIX flag parsing.
* `fskit`, `pathkit`: Multi-threaded walkers and immutable URIs.

## SIMD Kernels (`c2plan9`)

CGO overhead is avoided by compiling C/LLVM kernels directly into Plan 9 Go Assembler via [`c2plan9`](cmd/c2plan9/doc.go).

| Kernel | Description | Latency | Throughput | Allocations |
| :--- | :--- | :--- | :--- | :--- |
| `IndexCRLFCRLFVector` | HTTP header boundary scan (`\r\n\r\n`) | `4.82 ns/op` | 212 GB/s | 0 |
| `ValidUTF8_SWAR` | 64-bit SWAR UTF-8 validator | `5.78 ns/op` | 12.6 GB/s | 0 |
| `Hex.Encode1KB` | AVX2 Hex Encoding | `78.58 ns/op` | 13.0 GB/s | 0 |
| `json.UnmarshalNoCopy` | SIMD JSON Decoder | `916.4 ns/op` | - | 12 |

---
*For contributing guidelines, see [docs/VISION.md](docs/VISION.md).*
