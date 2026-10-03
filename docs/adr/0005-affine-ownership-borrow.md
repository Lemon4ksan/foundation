# ADR-0005: Runtime Generational Affine Ownership (`borrow`)

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: `borrow`, `codec/compress/*`, `bufkit`

## 1. Context & Problem Statement

Go's automatic garbage collection simplifies application programming but presents severe hazards when attempting zero-allocation buffer pooling in high-concurrency systems:
1. **Silent Use-After-Free (UAF)**: If a buffer is returned to `sync.Pool` while a concurrent goroutine still retains a slice reference, subsequent pool re-use causes data races and silent memory corruption.
2. **Double Free / Double Recycle**: Returning the same pooled buffer twice corrupts pool free-lists.
3. **Aliasing XOR Mutability**: Go offers no language-level guarantee that exclusive mutable access prevents simultaneous concurrent reads.

## 2. Decision Drivers

- Enforce Rust-like affine type semantics (single exclusive ownership, read-write borrow exclusivity) in pure Go.
- Prevent Use-After-Free and Double-Free without sacrificing zero-allocation runtime performance.
- Deterministic panic on corruption rather than silent memory overwrites.

## 3. Considered Options

1. **Option 1: Unprotected Manual Slices (`[]byte`)**: Rely purely on developer discipline. (High risk of silent concurrency bugs).
2. **Option 2: Compiler Modifications**: Require a modified Go toolchain. (Impractical and breaks standard toolchains).
3. **Option 3: Generational Runtime Affine Handles (`borrow.Box[T]`)**: Lightweight struct handles wrapping values with 64-bit generational counters and active borrow tracker bitmasks.

## 4. Decision Outcome

**Chosen Option**: **Option 3: Generational Runtime Affine Handles**.

### 4.1. Core Primitives in `borrow`

- **`Box[T]`**: Represents exclusive single ownership of a value or pooled buffer. Dropping or transferring a `Box` invalidates the previous handle.
- **`Ref[T]`**: Shared immutable borrow. Multiple concurrent `Ref` handles may coexist.
- **`Mut[T]`**: Exclusive mutable borrow. Panics if any active `Ref` or `Mut` already exists for that generation.
- **Generational Lifetimes**: When a buffer is recycled, its generational tag is monotonically incremented. Any stale reference from an older generation immediately panics upon dereference.

### 4.2. Zero-Overhead Inline Assembly

Handle verification is carefully structured with bounds-check elimination and inlined bitmask checks, compiling down to single-cycle CPU instructions.

## 5. Verification & Testing

- Race tests (`go test -race ./borrow/...`) verify thread safety under concurrent contention.
- Exhaustive chaos and adversarial tests verify that intentional UAF or double-free operations reliably trigger deterministic panics.
