<img src="assets/logo.svg" align="right" width="128" height="128" alt="foundation" />

# foundation

English • [Русский](README_RU.md)

[![Go Version](https://img.shields.io/badge/go-1.27%2B-007d9c?logo=go&logoColor=white&style=flat-square)](https://go.dev/)
[![Go Reference](https://img.shields.io/badge/godoc-reference-007d9c?style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue?style=flat-square)](LICENSE)

`foundation` is a Go standard library replacement for systems where the standard library's defaults are too slow or allocate too much. It reverses the compromises made by Go's `net` and `encoding` packages, exchanging simplicity and stability for SIMD acceleration, explicit memory management, and zero-allocation hot paths.

It is used as the base layer for my other network, storage, and automation systems: [`aoni`](https://github.com/lemon4ksan/aoni), [`mach`](https://github.com/lemon4ksan/mach), [`sein`](https://github.com/lemon4ksan/sein), [`g-man`](https://github.com/lemon4ksan/g-man), [`dawn`](https://github.com/lemon4ksan/dawn), and [`seal`](https://github.com/lemon4ksan/seal).

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
* `simd`, `cpukit`: AVX2/BMI2/GFNI vector processing for frame scanning, bit deposit, and Galois field arithmetic.
* `hexkit`, `bytesconv`, `endian`: SIMD hex codec (13 GB/s), zero-copy string/byte conversions, and branchless byte-order primitives.
* `pool`, `ringbuf`, `offheap`: Multi-tiered memory arenas, lock-free SPSC/MPSC ring buffers, and off-heap kernel page buffers/slabs.
* `clock`, `randkit`, `sysnet`, `iouring`, `comkit`: Syscall-free RDTSC/coarse clocks, lock-free PRNG, low-level socket/vectored I/O, Linux `io_uring` shared rings, and Windows COM/WinRT vtable primitives.

### Layer 1: Memory & Structures
* `borrow`: Generational affine ownership and borrow checkers (`Box`, `Cell`, `Ref`, `Mut`).
* `structures`: Cache-friendly typed data structures (`minheap`, `deque`, `ringbuffer`, `bitset`, `disjointset`, `linkedlist`, `trie`).
* `bufkit`, `refkit`: Cacheline-aligned (64B) ring/chain buffers and allocation-free struct tag/field accessors.

### Layer 2: Concurrency
* `sync`: Adaptive Vegas limiters, per-key mutexes (`keylock`), circuit breakers, semaphores, backoff, lazy values, and TTAS spinlocks.
* `async`: Zero-alloc `ctxkit`, topologically sorted `lifecycle`, `fsm`, `dedup`, `pipeline`, `pool`, `rate`, `scheduler`, `task`, and `logkit`.

### Layer 3: Codecs & Cryptography
* `encoding`: SIMD `json`, zero-alloc wire-format `proto` scanner/encoder, QUIC `varint`, AVX2 `base64` (31 GB/s), and binary `bin` serialization.
* `crypto`: Hardware-accelerated AEAD ciphers (`aead`), KDFs (`kdf`), and Ed25519/HMAC digital signatures (`sign`).
* `text`: Zero-alloc identifier `casing` and streaming byte transformations (`transform`).
* `iokit`: Vectored, replayable, and zero-copy I/O helpers.

### Layer 4: Wire & Transport (`net/`)
* `urlkit`: High-throughput SIMD URL view parsing, RFC 3986 path/query escaping, and host sanitization (`CleanHost`).
* `dns/wire`, `dns/svcb`: Layer 4 DNS wire codecs and RFC 9460 SVCB/HTTPS service discovery (including ECH extraction).
* `tls/cert`: Lock-free hot-reloading X.509 certificates and SPKI pinning.
* `ip`: Lock-free IP address rotation (`SourceRotator`), CIDR classification, and IPv6 `/64` subnet generation.

### Layer 5: Tooling & Utilities
* `types`: High-entropy `uuid` (v4/v7), zero-leak `secret`, and lenient `values` coercions.
* `timekit`: Zero-alloc coarse timestamps, stopwatches, and duration formatting.
* `fskit`, `pathkit`: Parallel directory walkers, memory-mapped files (`MmapFile`), atomic file writes, and zero-alloc path traversal.
* `testing`: Zero-dependency assertion and mocking helpers (`assert`, `require`, `mock`, `gomock`).

### External: Consumer Helpers
* `generic`: Monadic types (`Optional`, `Result`), sharded maps, thread-safe `LRU` & TTL `Cache`, `SingleFlight`, and lazy `Stream` (outside the layer graph, for external consumers).

## SIMD Kernels (`c2plan9`)

CGO overhead is avoided by compiling C/LLVM kernels directly into Plan 9 Go Assembler via `c2plan9`.

| Kernel | Description | Latency | Throughput | Allocations |
| :--- | :--- | :--- | :--- | :--- |
| `IndexCRLFCRLFVector` | HTTP header boundary scan (`\r\n\r\n`) | `4.82 ns/op` | 212 GB/s | 0 |
| `ValidUTF8_SWAR` | 64-bit SWAR UTF-8 validator | `5.78 ns/op` | 12.6 GB/s | 0 |
| `Hex.Encode1KB` | AVX2 Hex Encoding | `78.58 ns/op` | 13.0 GB/s | 0 |
| `json.UnmarshalNoCopy` | SIMD JSON Decoder | `916.4 ns/op` | - | 12 |

## 📄 License

This project is licensed under the **BSD 3-Clause License**. See the [LICENSE](LICENSE) file for details.
