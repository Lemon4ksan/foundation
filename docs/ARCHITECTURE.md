# Architecture & Engineering

`foundation` is a unified, high-performance silicon substrate and tactical concurrency runtime for Go. It consolidates hardware-accelerated memory and SIMD primitives with service orchestration and data structures into a cohesive vertical architecture.

## Architectural Decomposition

```mermaid
graph TD
    subgraph Silicon["Silicon & Memory Substrate (foundation/silicon, foundation/bufkit)"]
        SIMD["simd (256-bit AVX2/BMI2 & ARM64 NEON)"]
        MEM["offheap, pool (Direct Slabs, Arenas, Perpetual Storage)"]
        BUFKIT["bufkit (AlignedBuffer, BufferChain, RingBuffer)"]
        CONV["bytesconv, hex (Zero-Copy Slicers, SIMD Codecs)"]
        TIME["timekit, clock, rand (Monotonic Fast-Clock, FastRand, UUID)"]
        STRUCT["ringbuf, trie (Lock-Free Ring Buffers, Radix Trie)"]
    end

    subgraph Primitives["Language & Reflection Primitives (foundation/refkit, foundation/generic)"]
        REF["refkit (Cached Tag Parser, Zero-Alloc Checks)"]
        GEN["generic (Safe[T], LRU Cache, ResourcePool, Stream iter.Seq)"]
    end

    subgraph Async["Concurrency Runtime (foundation/async, foundation/sync)"]
        LIFE["lifecycle (DAG Topological Boot / Teardown)"]
        TASK["task (Jobs / Correlation IDs / Futures)"]
        DEDUP["dedup (SingleFlight Coalescing & Panic Isolation)"]
        FSM["fsm (Compile-Time Safe State Machine)"]
        PIPE["pipeline (Worker Pipelines & Throttling)"]
        POOL["pool (Auto-Scaling Worker Pool)"]
        EVENT["event (Type-Safe Event Backbone)"]
        SCHED["scheduler (Precision Task Scheduler & Cron)"]
        SYNC["sync (KeyLock, Limiter, Breaker, Backoff, Semaphore)"]
        LOG["log (Zero-Alloc Structured Logger)"]
    end

    Primitives --> Silicon
    Async --> Silicon
```

### Pillar 1: `silicon/` & `bufkit/` (Hardware Substrate)
* **Design Goal**: Nanosecond compute, zero garbage collection overhead, direct register utilization.
* **Constraints**: Zero external dependencies, pure Go runtime + Plan 9 Go Assembly, zero heap allocations on hot paths.

### Pillar 2: `async/` & `sync/` (Tactical Concurrency Runtime)
* **Design Goal**: Orchestrating goroutine worker pools, DAG service lifecycles, type-safe events, and resilient synchronization without race conditions or memory leaks.
* **Constraints**: Generics-first (`[T any, K comparable]`), context-aware cancellation, panic boundary isolation, zero reflection in hot execution loops.

## Package Directory Layout

| Domain | Submodule | Documentation | Purpose |
| :--- | :--- | :--- | :--- |
| **`silicon/`** | `simd` | [`docs/silicon/SIMD.md`](silicon/SIMD.md) | 256-bit AVX2/BMI2 vector processing (81.7 GB/s masking). |
| | `bytesconv` | [`docs/silicon/BYTESCONV.md`](silicon/BYTESCONV.md) | Zero-copy string/slice conversions, fast scanning and slicing. |
| | `offheap` | [`docs/silicon/OFFHEAP.md`](silicon/OFFHEAP.md) | Direct unmanaged memory slabs avoiding Go GC overhead. |
| | `pool` | [`docs/silicon/POOL.md`](silicon/POOL.md) | Multi-tiered memory arenas, perpetual byte storage, object pools. |
| | `ringbuf` | [`docs/silicon/RINGBUF.md`](silicon/RINGBUF.md) | Lock-free SPSC / MPMC ring buffers and Structure-of-Arrays (SoA). |
| | `clock`, `randkit` | [`docs/silicon/CLOCK_AND_RAND.md`](silicon/CLOCK_AND_RAND.md) | Monotonic fast-clock and lock-free fast pseudo-random / UUID v4/v7. |
| | `hexkit` | [`docs/silicon/BYTESCONV.md`](silicon/BYTESCONV.md) | Vectorized AVX2 hexadecimal encoding and decoding. |
| | `trie` | [`docs/silicon/TRIE.md`](silicon/TRIE.md) | Zero-allocation prefix and radix search trees. |
| **`bufkit/`** | `bufkit` | [`docs/bufkit/BUFKIT.md`](bufkit/BUFKIT.md) | Cacheline-aligned buffers, scatter-gather BufferChain, and SPSC RingBuffer. |
| **`timekit/`** | `timekit` | [`docs/timekit/TIMEKIT.md`](timekit/TIMEKIT.md) | Coarse atomic clock, zero-alloc HTTP-date / ISO 8601 formatting, stopwatch. |
| **`refkit/`** | `refkit` | [`docs/refkit/REFKIT.md`](refkit/REFKIT.md) | High-speed struct tag parsing with cache and panic-safe reflection checks. |
| **`async/`** | `ctxkit` | [`docs/async/CTXKIT.md`](async/CTXKIT.md) | Flat-array zero-alloc L1-resident `context.Context` replacement. |
| | `lifecycle` | [`docs/async/LIFECYCLE.md`](async/LIFECYCLE.md) | Topologically sorted DAG service boot and background loop runners. |
| | `event` | [`docs/async/EVENT.md`](async/EVENT.md) | Type-safe non-blocking event bus with reflection-free dispatch. |
| | `task` | [`docs/async/TASK.md`](async/TASK.md) | Asynchronous task tracking with Correlation IDs and Futures. |
| | `dedup` | [`docs/async/DEDUP.md`](async/DEDUP.md) | SingleFlight request deduplication with isolated panic propagation. |
| | `fsm` | [`docs/async/FSM.md`](async/FSM.md) | Strongly typed finite state machines with transactional rollback. |
| | `pipeline` | [`docs/async/PIPELINE.md`](async/PIPELINE.md) | Concurrent mapping pipelines with rate-limiting and fan-out/fan-in. |
| | `pool` | [`docs/async/POOL.md`](async/POOL.md) | Dynamic auto-scaling goroutine worker pool with futures. |
| | `scheduler` | [`docs/async/SCHEDULER.md`](async/SCHEDULER.md) | Microsecond-precision recurring task schedulers and cron runners. |
| | `logkit` | [`docs/async/LOGKIT.md`](async/LOGKIT.md) | High-performance zero-allocation structured logger facade. |
| **`sync/`** | `sync` | [`docs/sync/SYNC.md`](sync/SYNC.md) | Striped key-based locks, Vegas limiters, breakers, backoff, and semaphores. |
| **`generic/`** | `generic` | [`docs/generic/GENERIC.md`](generic/GENERIC.md) | Thread-safe `Safe[T]`, `LRU[K, V]`, `ResourcePool[T]`, `Cache[K, V]`, `Stream[T]`. |
| **`iokit/`** | `iokit` | [`docs/iokit/IOKIT.md`](iokit/IOKIT.md) | Replayable body buffers, allocation-free `BytesReader`, and copy pools. |
| **`net/`** | `net` | [`docs/net/NET.md`](net/NET.md) | Canonical HTTP headers, URL parsing (`urlkit`), HPACK compression, gRPC-Web, Cache-Status, DNS. |
