# Architecture notes

This file covers the techniques that show up across many packages. Anything about a single package belongs in that package's `doc.go`. The layer stack and import rules are in [ADR-0002](adr/0002-layering-and-isolation.md); the allocation tiers are in [ADR-0003](adr/0003-allocation-budgets.md). They are not repeated here.

## Mechanical sympathy

Some of the code is written for the CPU and not for the reader. Three habits recur:

- **Padding against false sharing.** When several cores update neighbouring fields, they fight over one cache line and throughput collapses. Per-core structures such as `silicon/pool.PerPStorage` put a `cpu.CacheLinePad` between shards so each core owns its line.
- **Masks instead of modulo.** Shard and ring indexes use `i & mask`, which means capacities are powers of two. Integer division is slow, and a mask costs one instruction.
- **Bounds-check elimination.** Hot loops prove the slice length once before the loop (`_ = dst[n-1]`), so the compiler drops the per-element check. `silicon/simd/gfni.MultiplyGF2P8Vector` is an example.

## Keeping allocations out of the hot path

- **Flat storage instead of pointers.** `structures/linkedlist.List[T]` keeps its elements in one slice and links them by index, with a free list for reused slots. `container/list` allocates a node per element and chases pointers on every step.
- **Push iterators instead of result slices.** Collections and parsers hand out `iter.Seq` and `iter.Seq2` (`linkedlist.List.Values`, `net/quic/varint.DecodeSeq`). The loop body runs in the caller's frame, `break` stops the producer straight away, and nothing is allocated for the result.
- **Caller-supplied destinations.** Encoders expose `Append*` or slice-filling forms next to any form that returns a new slice. When you compare secrets, use `silicon/randkit.ConstantTimeEqual(a, b)`, not `subtle.ConstantTimeCompare([]byte(a), []byte(b))`, which allocates twice.

## Ownership and `borrow`

Go cannot check borrows at compile time, so `borrow` does it at run time. `Box[T]` is the single owner of an object. `Ref[T]` and `Mut[T]` enforce "many readers or one writer". Every handle carries a generation number that goes up when the buffer is released or recycled, so a stale handle panics on first use instead of silently reading someone else's data. See [ADR-0005](adr/0005-affine-ownership-borrow.md).

## State machines (`async/fsm`)

A transition has to be atomic with respect to other transitions, yet readers should not wait while a long hook runs. `fsm` uses two locks. `transMu` serialises transitions so that before/after hooks of two transitions never interleave. A separate `sync.RWMutex` protects the maps and current state, so `CurrentState()` and `Validate()` read under a shared lock while user hooks run outside it.

## Which primitive to pick

### Pools: `sync.Pool` or `silicon/pool.PerPStorage`

Use `sync.Pool` for general-purpose reuse. It lets the GC drop idle objects, which is what you want when memory pressure matters more than latency.

Use `PerPStorage` under heavy multi-core load (over roughly 100k operations a second), when CAS contention on a shared head is the problem, or when buffers should survive GC cycles (network buffers, scratch space).

### Lazy values: `sync.Once` or `sync/lazy.Lazy[T]`

`sync.Once` is right for a one-time, permanent, void action. `Lazy[T]` caches a value and an error, can be reset, and gives readers a lock-free fast path after the first call.

### Lists: `container/list` or `structures/linkedlist.List[T]`

`container/list` is fine when you need to interoperate with code that uses it. Reach for `List[T]` when allocation per push or pop and cache locality matter; it also has `Values()` and `All()` iterators.

## Habits that keep code allocation-free

Encode in place (shown with `net/quic/varint`):

```go
// allocates a []byte
buf := varint.EncodeVarint(val)

// fills the caller's buffer, no allocation
n := varint.EncodeVarintSlice(val, scratch[offset:])
```

Iterate without building slices:

```go
for item := range collection.Values() {
    // ...
}
```

Benchmarks for each package are run with `go test -bench . -benchmem ./<package>`; numbers are not kept in the docs because they go stale.
