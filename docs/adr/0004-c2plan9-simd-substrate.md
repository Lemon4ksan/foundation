# ADR-0004: External c2plan9 compiler pipeline and SIMD vectorization substrate

Status: Accepted  
Date: 2026-10-10  
Deciders: Lemon4ksan  

## Context

`foundation` is engineered as a high-performance standard library substrate for latency-critical network and data systems ([ADR-0001](0001-stdlib-v2.md)). Core primitives — including Base64 encoding/decoding, JSON string scanning, LEB128/varint decoding, case-insensitive ASCII comparison, and memory hashing — demand hardware SIMD acceleration (AVX2, AVX-512, NEON, GFNI) to saturate memory bus bandwidth and achieve line-rate throughput.

However, achieving silicon-grade vectorization in Go presents a classic systems dilemma:

1. **The CGO Latency Tax:** Invoking external C libraries via CGO introduces an unavoidable 40–150 ns overhead per invocation due to runtime stack transitions, OS thread handoffs (`entersyscall`/`exitsyscall`), and scheduler contention. For microsecond- and nanosecond-scale micro-kernels, this overhead completely nullifies the algorithmic gains of SIMD instructions.
2. **The Plan 9 Assembly Bottleneck:** Hand-writing complex vector kernels in Go's idiosyncratic Plan 9 assembly syntax is brittle, error-prone, and difficult to verify. Plan 9 assembly lacks high-level register allocators, instruction scheduling, and typed stack frame management.
3. **Toolchain Bitrot & ABI Incompatibilities:** Legacy translation tools (such as `c2goasm` or `asm2plan9s`) are unmaintained and target outdated Go internal ABIs, failing to preserve 32-byte dynamic stack alignment (`ANDQ $-32, SP`) required for aligned AVX operations (`VMOVDQA`).

To resolve this, we created **`c2plan9`**, a dedicated compiler pipeline that compiles C intrinsics via Clang/LLVM, decodes the resulting machine code, and synthesizes clean, Go-compatible Plan 9 assembly (`.s`) and companion stubs (`.go`).

## Decision

We establish the following architectural rules for SIMD code generation and maintenance:

1. **Independent Compiler Toolchain Repository:**  
   `c2plan9` is maintained as a sovereign external repository ([`github.com/lemon4ksan/c2plan9`](https://github.com/lemon4ksan/c2plan9)). This preserves strict boundary isolation ([ADR-0002](0002-layering-and-isolation.md)): `foundation`'s codebase remains strictly a runtime systems substrate and never absorbs Clang invocation drivers, LLVM AST parsers, or machine code disassemblers.

2. **Portable ELF Object Intermediate Representation:**  
   The compilation pipeline forces `-target x86_64-pc-linux-gnu` (for AMD64) and `-target aarch64-linux-gnu` (for ARM64) across all host operating systems (Linux, macOS, Windows). This avoids platform-dependent object file quirks (PE addends, Mach-O symbol mangling, and section layout differences).

3. **Go ABI Conformance and Stack Alignment Preservation:**  
   - The emitter retains dynamic stack alignment instructions (`ANDQ $-32, SP`) and accounts for alignment padding in frame sizing, ensuring that aligned vector operations (`VMOVDQA`) never trigger general protection faults (#GP).
   - Typed return registers (`float32`, `float64`, `m128`, `m256`) are mapped directly into Go return frames (`VMOVDQU`, `FMOVQ`).

4. **Committed Generated Artifacts & Zero Runtime Dependencies:**  
   Packages in `foundation` (e.g. `silicon/simd`, `silicon/bytesconv`, `silicon/hexkit`, `encoding/base64`, `types/uuid`) invoke `c2plan9` through standard `//go:generate` directives pointing to local C sources in `csrc/`. All generated assembly files (`*_amd64.s`, `*_arm64.s`) and Go companion stubs are committed into version control. End consumers of `foundation` require only standard `go build` and zero external C toolchain dependencies.

## Consequences

### What gets easier:
- **Zero-Latency Execution:** SIMD kernels execute as native Go functions with zero CGO overhead, zero heap allocation, and full register-based Go internal ABI compatibility.
- **Maintainability & Expressiveness:** Complex vector algorithms are written in clean, readable C using standard compiler intrinsics (`immintrin.h`, `arm_neon.h`), benefiting from LLVM's world-class instruction scheduler, peephole optimizer, and register allocation.
- **Substrate Hygiene:** `foundation` runtime remains 100% pure Go without build-time compiler bloat.

### What gets harder / what we pay:
- Developers modifying C source kernels in `csrc/` need Clang/LLVM installed locally (or via WSL on Windows) to re-run `go generate ./...`.
- Generated Plan 9 assembly files must be continuously audited and fuzzed (`scripts/fuzz_all.go`) across platforms to ensure cross-architecture instruction parity.

## Alternatives Considered

- **Pure Go Implementations Only:** Rejected. For algorithms like 64-byte vector pattern matching, GF(2^8) Galois field arithmetic, and vectorized Base64 encoding, software fallback loops are 5x to 15x slower than dedicated AVX2/AVX-512/NEON instructions.
- **CGO Dynamic/Static Linking:** Rejected. The 40–150 ns CGO boundary tax destroys performance for short-lived operations processing buffers under 4 KB.
- **Monolithic In-Tree Tooling (`cmd/c2plan9` inside `foundation`):** Rejected. Violates ADR-0002 and ADR-0006 by introducing build-time C compiler orchestrators and external architecture disassemblers into the foundational runtime library.
