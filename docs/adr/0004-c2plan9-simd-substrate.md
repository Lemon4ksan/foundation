# ADR-0004: c2plan9 compiler pipeline and SIMD substrate

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
We need SIMD vectorization for high-throughput algorithms. Calling C libraries via CGO introduces severe call latency. Writing Plan 9 assembly by hand is error-prone and hard to maintain. The Go compiler lacks mature autovectorization.

## Decision
We compile C kernels (`csrc/*.c`) with Clang to ELF objects, then parse the machine code and generate Plan 9 assembly (`.s`) and Go stubs using our `cmd/c2plan9` tool. Packages use `//go:generate` directives to run this pipeline. We commit the generated assembly and stubs so library consumers do not need a C compiler.

We also use native `simd/archsimd` intrinsics for simple inlined vector math, keeping `c2plan9` for complex unrolled multi-register pipelines.

## Consequences
We get massive throughput improvements with zero function call overhead and no CGO requirement. We pay by maintaining the `c2plan9` parser tool and fuzzing the kernels against pure-Go fallbacks.
