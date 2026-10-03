# ADR-0004: c2plan9 Compiler Pipeline and Silicon SIMD Substrate

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: `silicon/*`, `csrc/*`, `cmd/c2plan9`

## 1. Context & Problem Statement

Go's runtime scheduler and garbage collector impose strict constraints on low-level performance optimization:
1. **CGO Overhead**: Calling external C/C++ libraries via CGO introduces stack switching, thread transitions, and runtime coordination costs (~50–100 ns overhead per call), eliminating the throughput gains of micro-kernels.
2. **Missing Vector Compiler Autovectorization**: The standard Go compiler (`gc`) lacks mature autovectorization for AVX2, AVX-512, GFNI, and ARM NEON.
3. **Manual Assembly Burden**: Writing and maintaining thousands of lines of Plan 9 Go assembly by hand is error-prone, hard to refactor, and challenging to debug.

## 2. Decision Drivers

- Zero runtime function call overhead (native Go assembly linkage, no CGO).
- Leverage state-of-the-art vectorizing compilers (Clang/LLVM 18+) with target CPU flags (`-mavx2`, `-mbmi2`, `-O3`).
- Automated translation from compiled ELF machine code directly to Go Plan 9 assembly (`.s`) and type-safe Go stubs.

## 3. Considered Options

1. **Option 1: CGO Bindings**: Link C vector kernels via `import "C"`. (Rejected due to call latency).
2. **Option 2: Pure Hand-Written Go**: Rely only on standard loops and bounds-check hints. (Rejected: loses 3x–15x SIMD speedups).
3. **Option 3: Pure Plan 9 Hand-Written Assembly**: Write `.s` files manually. (Rejected: unmaintainable for complex vector math).
4. **Option 4: In-Tree c2plan9 Ahead-of-Time Pipeline**: Compile C kernels with Clang to ELF objects, then parse machine instructions and generate Plan 9 assembly stubs via `cmd/c2plan9`.

## 4. Decision Outcome

**Chosen Option**: **Option 4: In-Tree c2plan9 Ahead-of-Time Pipeline**.

### 4.1. Architecture & Pipeline

```
 [ csrc/*.c ] ──( clang -O3 )──> [ ELF .o ] ──( cmd/c2plan9 )──> [ *_amd64.s / *.go ]
```

1. Performance-critical SIMD routines reside in `csrc/` (`match.c`, `hash.c`, `base64.c`, `hex.c`, `gfni.c`).
2. Each package utilizing a kernel specifies a `//go:generate` directive:
   ```go
   //go:generate c2plan9 -c ../../csrc/hex.c -o hex_amd64.s -stub hex_amd64.go -pkg hexkit
   ```
3. The generated assembly files (`.s`) and Go stubs are committed to the repository, eliminating any C compiler prerequisite for library consumers.

### 4.2. Go 1.27 `simd/archsimd` Integration

Alongside `c2plan9`, Go 1.27 introduces `GOEXPERIMENT=simd` and native `simd/archsimd` inlined intrinsics. `foundation` adopts these native compiler intrinsics for inlined vector math while preserving `c2plan9` for complex unrolled multi-register pipelines.

## 5. Verification & Testing

- Benchmarks against standard library equivalents confirm 2x–14x throughput improvements (e.g., Hex encoding at 13.0 GB/s, Base64 decoding at 8.9 GB/s).
- Fuzzing suites verify bit-exact parity between SIMD kernels and portable pure-Go fallback routines.
