# Foundation Architecture Guide

## 1. Overview & Vision
`foundation` is a high-performance systems foundation library for Go (targeting Go 1.27+), engineered for low-latency networking, hardware-accelerated processing, zero-allocation streaming, and concurrency safety.

The repository is organized into five layered subsystems:

```
+-------------------------------------------------------------------------+
| Layer 5: Tooling & Utilities (scripts, cmd/c2plan9, types, timekit)     |
+-------------------------------------------------------------------------+
| Layer 4: Wire Protocols & Networking (net/*, http/zerocopy, quic, hpack)|
+-------------------------------------------------------------------------+
| Layer 3: Codecs & Cryptography (codec/*, crypto/*, encoding/*, text/*)  |
+-------------------------------------------------------------------------+
| Layer 2: Concurrency & Synchronization (async/*, sync/*)                |
+-------------------------------------------------------------------------+
| Layer 1: Hardware, Silicon & Memory (silicon/*, borrow, structures/*)   |
+-------------------------------------------------------------------------+
```

---

## 2. Core Architectural Principles

### 2.1. Mechanical Sympathy (Hardware Awareness)
- **Cache-Line Alignment (`cpu.CacheLinePad`)**:
  In multi-core workloads, cache-line contention (**false sharing**) can degrade throughput by an order of magnitude. Primitives like `silicon/pool.PerPStorage` use 64-byte padding (`cpu.CacheLinePad`) between CPU core shards to guarantee independent L1/L2 cache ownership.
- **Power-of-Two Bitmasking**:
  Index hashing avoids expensive CPU integer division (`id % N`), relying strictly on bitwise masks (`id & mask`), requiring precalculated power-of-two shard capacities.
- **Bounds Check Elimination (BCE)**:
  Looping operations across vectors (e.g., `silicon/simd/gfni.MultiplyGF2P8Vector`) establish slice boundary invariants upfront (`_ = dst[n-1]`), allowing the Go compiler to omit bounds check instructions (`runtime.panicIndex`) inside the loop body.

### 2.2. Zero-Allocation Lifetime Contracts
- **Array-Backed Collections**:
  Unlike standard library pointer-chasing structures (`container/list`), collections such as `structures/linkedlist.List[T]` are flat array-backed (`[]Element[T]`). Elements are referenced by array indices (`int`), preserving cache locality and recycling slots through an intrusive free list without garbage collection overhead.
- **Push Iterators (`iter.Seq`, `iter.Seq2`)**:
  Rather than allocating slices of results, collections and stream parsers (e.g., `net/quic/varint.DecodeSeq`, `structures/linkedlist.Values`) use native push iterators. Iteration executes within the caller's call stack with $O(1)$ stack allocation, immediate early exit on `break`, and zero heap pressure.

### 2.3. Affine Ownership & Borrow Checking (`borrow`)
Go does not provide compile-time borrow checking like Rust. The `borrow` package introduces runtime affine semantics:
- **`Box[T]`**: Exclusive single ownership of heap-allocated or pooled objects.
- **`Ref[T]` / `Mut[T]`**: Enforces *Aliasing XOR Mutability* (multiple concurrent readers or exactly one exclusive writer).
- **Generational Lifetimes**: Handles carry a generation ID. Releasing or recycling a buffer increments the generational counter; any stale reference attempting access triggers a deterministic panic, preventing silent Use-After-Free (UAF) corruptions.

### 2.4. Dual-Mutex Coordination (`async/fsm`)
Transitions in stateful systems separate serialization from state querying:
- `transMu sync.Mutex`: Serializes state transitions, preventing interleaving of before/after hooks.
- `mu sync.RWMutex`: Protects internal maps and values, enabling lockless or shared-lock reads (`CurrentState()`, `Validate()`) while long-running user hooks execute outside the lock.

---

## 3. Subsystem Index

| Package / Layer | Primary Role | Key Primitives |
|---|---|---|
| `silicon/pool` | Per-core sharded memory caches | `PerPStorage[T]`, `RequestArena` |
| `silicon/simd/gfni` | Galois Field $GF(2^8)$ arithmetic | `MultiplyGF2P8`, `MultiplyGF2P8Vector` |
| `silicon/randkit` | Timing-safe cryptographic entropy | `SecureBytes`, `ConstantTimeEqual` |
| `structures/linkedlist` | Zero-alloc array-backed list | `List[T]`, `Values()`, `All()` |
| `structures/ringbuffer` | Lock-free / low-overhead ring | `RingBuffer[T]`, `Values()` |
| `sync/lazy` | Lockless atomic double-checked lazy init | `Lazy[T]`, `Get()`, `Reset()` |
| `async/fsm` | Generic, thread-safe finite state machine | `FSM[State, Event]`, `Transition` |
| `async/pipeline` | Stream processing pipelines | `Pipeline[In, Out]`, `ProcessSeq` |
| `net/quic/varint` | RFC 9000 variable-length integers | `EncodeVarintSlice`, `DecodeSeq` |
| `net/http/zerocopy` | High-throughput HTTP wire parsing | `Request`, `Response`, `Args`, `URI` |
| `net/hpack` | RFC 7541 HTTP/2 header compression | `HeaderField`, `Encoder`, `Decoder` |
| `borrow` | Linear ownership and lifetime verification | `Box[T]`, `Ref[T]`, `Mut[T]`, `Scoped` |
