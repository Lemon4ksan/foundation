# ADR-0001: Foundation as Go Standard Library v2.0

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: Foundation-wide (all packages)

## 1. Context & Problem Statement

The official Go standard library (`stdlib`) is designed for maximum accessibility, unconditional backward compatibility (Go 1 promise), and broad portability across dozens of architectures and operating systems. These foundational design tenets require structural compromises:
1. Interfaces and signatures favor ergonomic convenience over zero-allocation guarantees (e.g., returning slices or interfaces allocated on the heap).
2. Algorithms target the lowest common denominator of hardware instructions, avoiding direct SIMD vector registers or unrolled assembly pipelines.
3. Evolution is constrained: API flaws or suboptimal structures cannot be removed or altered without breaking the Go 1 promise.

High-throughput systems (game engines, low-latency network proxies, cryptographic runtimes, storage WAL engines) frequently hit CPU and memory bottlenecks when relying on standard library primitives (`sync.Pool`, `encoding/json`, `net/http`, `compress/flate`, `io.ReadAll`).

## 2. Decision Drivers

- Extreme throughput and mechanical sympathy on Tier-1 hardware (amd64 / arm64).
- Deterministic memory management with zero heap allocations on hot paths.
- Freedom to refine and optimize APIs within `v0.x` without multi-year committee cycles.
- Direct alignment with the needs of four core consumer projects: `aoni`, `sein`, `dawn`, `seal`.

## 3. Considered Options

1. **Option 1: Ad-hoc Utility Repositories**: Create scattered packages inside each consumer codebase as bottlenecks appear.
2. **Option 2: Fork the Go Compiler and Standard Library**: Maintain a private runtime/stdlib fork.
3. **Option 3: Foundation as Go Standard Library 2.0**: A unified substrate repository providing modern, high-performance replacements for standard packages, explicitly documenting every rejected compromise.

## 4. Decision Outcome

**Chosen Option**: **Option 3: Foundation as Go Standard Library 2.0**.

`foundation` serves as a first-class, drop-in capable systems substrate that re-evaluates stdlib compromises for systems programming.

### 4.1. Core Rules of Engagement

1. **Rejected Compromise Mandate**: Every package in `foundation` must state its stdlib counterpart, the specific compromise rejected, the accepted cost, and the allocation tier in its `doc.go`:
   ```go
   // Stdlib counterpart: <stdlib pkg or "none — fills gap: ...">
   // Rejected compromise: <compromise rejected>
   // Accepted cost: <tradeoff paid>
   // Allocations: zero | amortized | bounded(N/op) | unconstrained
   ```
2. **API Autonomy**: While standard interfaces (`io.Reader`, `io.Writer`, `net.Conn`, `iter.Seq`) are favored where zero-overhead contracts permit, `foundation` maintains API autonomy and is not bound to mimic flawed stdlib signatures.
3. **Strict Layering & Non-Goals**: Domain-specific Layer 7 protocols (HTTP/1-3, HPACK, QPACK, Steam wire protocols) are strictly excluded from `foundation` and relegated to specialized repositories (such as `mach`).

### 4.2. Consequences

- **Positive**:
  - Unified mental model across all consumer projects.
  - Predictable zero-allocation and hardware-accelerated runtime behavior.
  - Transparent documentation of design tradeoffs in every package.
- **Negative / Costs**:
  - Requires compiler compatibility with Go 1.27+.
  - Consumers must accommodate breaking API improvements prior to `v1.0`.

## 5. Verification & Testing

- Architectural tests in `root_test.go` verify that package doc headers contain standard library comparison blocks and valid allocation tier declarations.
- CI benchmarks with `-benchmem` verify zero unintended allocations on hot paths.
