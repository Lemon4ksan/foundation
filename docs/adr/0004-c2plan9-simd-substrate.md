# ADR-0004: External c2plan9 compiler pipeline and SIMD

Status: Accepted
Date: 2026-10-10
Deciders: Lemon4ksan

## Context
We need SIMD vectorization for high-throughput algorithms. Calling C libraries via CGO introduces severe call latency. Writing Plan 9 assembly by hand is error-prone. We developed `c2plan9` to parse machine code and generate Plan 9 assembly (`.s`).

## Decision
`c2plan9` is maintained as an independent external repository (`d:\CodingProjects\c2plan9`), strictly separating build-time compiler tooling from the runtime standard library. Packages in `foundation` use `//go:generate` directives to invoke the external `c2plan9` tool. We commit the generated assembly and stubs so library consumers do not need the toolchain.

## Consequences
We keep the `foundation` codebase clean of AST parsers and compiler logic. The `c2plan9` tool can evolve independently and serve other repositories (like `seal`). We pay by maintaining an external toolchain dependency for developers who need to regenerate SIMD assembly.
