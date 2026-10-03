# ADR-0005: Runtime generational affine ownership

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
Zero-allocation buffer pooling in high-concurrency Go systems is dangerous. Returning a buffer to a `sync.Pool` while a concurrent goroutine still retains a slice reference causes silent memory corruption. Go offers no language-level guarantee to prevent use-after-free or double-free errors.

## Decision
We enforce single exclusive ownership and read-write borrow exclusivity in pure Go using generational runtime handles in the `borrow` package.

We wrap pooled buffers in structs with 64-bit generational counters. When a buffer is recycled, its generation increments. Dereferencing a stale handle panics immediately.

## Consequences
We prevent silent memory corruption and turn concurrency bugs into deterministic panics. We pay a tiny overhead for bitmask checks, which compile down to single-cycle CPU instructions.
