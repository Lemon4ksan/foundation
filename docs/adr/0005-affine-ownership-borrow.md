# ADR-0005: Runtime generational affine ownership and borrow exclusivity

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

High-throughput, zero-allocation Go applications rely heavily on memory pooling (`sync.Pool`, custom ring buffers, or slab allocators) to eliminate garbage collection overhead. However, in concurrent environments, manual memory recycling introduces severe safety vulnerabilities:

1. **Use-After-Free (UAF):** A goroutine returns a pooled buffer to an allocator, but another concurrent worker or pipeline stage retains an aliased slice pointer. When the buffer is recycled and assigned to an unrelated operation, the stale reader or writer reads or overwrites active data.
2. **Double-Free Corruptions:** Returning the same pooled slot twice corrupts the free-list or internal pool queue, leading to duplicate simultaneous leases of the same physical memory chunk.
3. **Data Races and Mutable Aliasing:** Go's runtime does not enforce the "Aliasing XOR Mutability" invariant. Multiple goroutines can concurrently mutate identical memory slices without compiler warnings.
4. **Silent Failure Mode:** Unlike languages with compile-time borrow checkers (e.g. Rust), Go provides no language-level lifetime tracking. When memory pooling goes wrong, the failure does not manifest as an immediate crash at the callsite. Instead, it appears as intermittent, impossible-to-reproduce data corruption, authentication bypasses, or panic storms deep within unrelated subsystems.

Attempting to avoid this through defensive slice cloning (`bytes.Clone`, `append([]byte(nil), ...)`) destroys zero-allocation throughput.

## Decision

We introduce the **`borrow`** package in Layer 1 ([ADR-0002](0002-layering-and-isolation.md)), providing runtime-enforced affine ownership, borrow exclusivity, and generational lifetime verification with zero heap allocations (`zero` allocation tier, [ADR-0003](0003-allocation-budgets.md)).

### Core Architectural Mechanisms

1. **Affine Single Ownership (`Box[T]`):**
   - An owned resource is encapsulated inside `Box[T]`.
   - A `Box[T]` must be explicitly consumed, moved, or released back to its backing allocator.
   - Calling `Box.Move()` transfers ownership to a new container and increments the underlying slot generation, instantly invalidating any handles that were borrowed prior to the move.

2. **Aliasing XOR Mutability (Dynamic RWLock Semantics):**
   - **Shared Immutable Borrow (`Ref[T]`):** Acquired via `Box.Borrow()`. Multiple concurrent readers can acquire shared references simultaneously.
   - **Exclusive Mutable Borrow (`Mut[T]`):** Acquired via `Box.BorrowMut()`. Only a single writer can borrow the slot.
   - Attempting to acquire an exclusive borrow while active readers exist, or acquiring any borrow while an exclusive borrow is active, returns a descriptive error or triggers an immediate panic.
   - **Downgrade Support:** An exclusive borrow can be safely downgraded into a shared immutable borrow using `Mut.Freeze()`.

3. **64-Bit Generational Counters:**
   - Every physical memory slot and lease token carries a monotonic 64-bit generation counter.
   - When a slot is recycled, moved, or freed, its generation counter advances.
   - Every access through `Ref[T]`, `Mut[T]`, or zero-copy `Bytes` validates the token's generation against the slot's current generation.
   - Any stale access triggers an immediate, deterministic panic (`ErrUseAfterFree` or `ErrDoubleFree`), converting non-deterministic data corruption into an instant, localized failure with an informative stack trace.

4. **Lexical Arenas & Scoped Lifetimes (`Scope`):**
   - `Scope` enables lexical stack-like memory allocation (`Scope.AllocBytes`, `borrow.Alloc`).
   - Slices allocated within a scope are bound to that scope's lifetime.
   - Individual resources can be released early via `Bytes.Release()`, and exiting the scope arena automatically and atomically recycles all allocated slots.

5. **Single-Cycle Microarchitectural Verification:**
   - Generation and state validation are implemented via fast bitmask and equality operations (`CMPQ` / `TESTQ`).
   - The validation compiles down to single-cycle machine instructions with predictable branch prediction, ensuring sub-nanosecond check overhead.

## Consequences

### What gets easier:
- **Zero-Allocation Safety:** Teams can aggressively pool memory across high-concurrency pipelines (e.g. vectored I/O queues, streaming protocol decoders) without the risk of silent UAF or double-free corruption.
- **Immediate Bug Localization:** Concurrency and lifecycle violations fail fast at the exact point of the invalid access rather than corrupting memory silently.
- **No Language Forks:** Rust-grade ownership discipline is achieved within standard Go 1.27+ language semantics without proprietary compilers or runtime patches.

### What gets harder / what we pay:
- Ergonomics require explicit lifecycle discipline (`defer ref.Release()`, `defer mut.Release()`, or `defer scope.Close()`).
- Microarchitectural cost of 1–2 CPU cycles per handle dereference to validate the 64-bit generation word.
- Developers must design APIs around ownership handoffs rather than passing unmanaged raw byte slices across goroutine boundaries.

## Alternatives Considered

- **Standard `sync.Pool` Without Guards:** Rejected. Offers zero protection against use-after-free, double-free, or simultaneous mutation races. In production network services, this consistently causes catastrophic data corruption.
- **Defensive Memory Copying:** Rejected. Copying every buffer to prevent aliasing defeats the core mission of `foundation` as a zero-allocation systems substrate.
- **Static Analysis & Linters:** Rejected as insufficient. Go linters cannot track dynamic slice aliasing and memory lifetime transitions across channels, goroutine pools, and asynchronous event loops without full inter-procedural escape analysis.
