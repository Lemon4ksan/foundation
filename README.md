# Foundation

### Silicon Substrate & Concurrency Runtime for Go

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation)
[![License](https://img.shields.io/github/license/lemon4ksan/foundation?style=flat-square)](LICENSE)

> *"As soon as bytes must be processed on silicon, it happens with 0 allocations, at maximum hardware line speed, with zero type drift, and absolute architectural clarity."*

`foundation` is a unified, high-performance silicon substrate and tactical concurrency runtime for Go. It consolidates hardware-accelerated memory and SIMD primitives with industrial-grade service orchestration into a single, cohesive foundation following the Apple-style vertical integration model.

## Architectural Pillars

`foundation` is strictly partitioned into two foundational domains with zero external framework dependencies:

### 1. Silicon & Memory Substrate (`silicon/`)
Hardware-adjacent primitives engineered for zero memory allocations and line-rate throughput:
* **`simd`**: 256-bit AVX2/BMI2 vector processing for frame masking at 81.7 GB/s.
* **`bytesconv`**: Zero-copy string-to-byte conversions, scanners, and slice tokenizers.
* **`offheap`**: Unmanaged direct memory slabs bypassing the Go garbage collector.
* **`pool`**: Multi-tiered memory arenas, perpetual byte storage, and object pools.
* **`ringbuf`**: Lock-free SPSC / MPMC ring buffers and Structure-of-Arrays (SoA) layout.
* **`clock` & `rand`**: Monotonic fast-clock without syscalls, lock-free fastrand, and sortable UUID v7.
* **`trie`**: Compressed radix search trees for high-speed prefix routing.

### 2. Concurrency & Runtime Orchestration (`async/`)
Predictable, resilient primitives for goroutine governance and synchronization:
* **`lifecycle`**: Topologically sorted DAG service boot, health monitoring, and graceful teardown.
* **`event`**: Type-safe, non-blocking asynchronous event bus.
* **`task`**: Asynchronous task manager with correlation IDs, context timeouts, and futures.
* **`dedup`**: Single-flight request deduplication with isolated panic boundaries.
* **`fsm`**: Compile-time type-safe finite state machines with transactional rollback.
* **`pipeline`**: Concurrent worker pipelines with token-bucket rate limiting and DataLoader batching.
* **`pool`**: Auto-scaling goroutine worker pools with idle scale-down and panic recovery.
* **`scheduler`**: Microsecond-precision recurring task schedulers and cron runners.
* **`sync`**: Striped key-based locks, Vegas adaptive limiters, circuit breakers, and jittered backoff.
* **`log`**: Zero-allocation structured logging facade with asynchronous flushing.

## Installation

```bash
go get github.com/lemon4ksan/foundation
```

## Documentation

Detailed architecture specifications and practical recipes are located in the [`docs/`](docs/README.md) directory:
* [Architecture & Engineering Manifesto](docs/ARCHITECTURE.md)
* [Async Runtime Reference](docs/README.md#concurrency--runtime-orchestration-async)
* [Silicon Substrate Reference](docs/README.md#silicon--memory-substrate-silicon)

## License

BSD-3-Clause License. See [LICENSE](LICENSE) for details.
