---
name: foundation-guidelines
description: Strict coding, testing, performance, and multi-agent interaction guidelines for the foundation library
trigger: always_on
---

# Foundation Library AI Guidelines

When writing, modifying, or refactoring code in this repository, AI agents **MUST** strictly adhere to the following rules:

## 0. Agent History & Context
- The `.agents/` directory (specifically subfolders like `.agents/sentinel/`, and files like `PROJECT.md`, `GATE_STATUS.md`, and `ORIGINAL_REQUEST.md`) contains the comprehensive history, architectural decisions, and audit logs of previous agents.
- You **MUST** consult and cross-reference these logs to avoid violating established standards, breaking the architecture, and to understand the project's historical context before making significant changes.

## 1. Testing and Coverage (100% Coverage)
- Any new code or modifications must be accompanied by comprehensive unit or table-driven tests.
- Package test coverage must remain at **100%** (statement coverage) after your changes.
- No dummy or fake tests: tests must genuinely verify logic, branches, and edge cases.

## 2. Concurrency Safety (Race Safety)
- All concurrent primitives, pools, workers, and asynchronous pipelines must be completely thread-safe.
- Always verify code using `go test -race`. There must be absolutely no warnings (0 race warnings).

## 3. Zero-Allocations
- Code on "hot" paths (parsing, codecs, structures, SIMD, crypto) **must not** allocate memory on the heap.
- Always use the `-benchmem` flag in benchmarks and aim for `0 B/op` and `0 allocs/op`.
- Use sync pools (`sync.Pool` or equivalents) and reuse buffers wherever applicable.

## 4. Fuzzing
- Fuzz tests (e.g., `go test -fuzz`) are **mandatory** for all parsers, network codecs, decoders, and cryptography implementations.

## 5. Go Idiomatic Code and Standards
- The code must be strictly idiomatic Go (no C++ or other foreign artifacts/patterns).
- Use standard Go library interfaces wherever applicable: `net.Conn`, `net.Listener`, `io.Reader`, `io.Writer`, `io.Closer`, `iter.Seq`, `iter.Seq2`.
- Always use standard error wrapping and checking: `errors.Is`, `errors.As`, `fmt.Errorf("%w", err)`.
- The code must pass `golangci-lint run` cleanly without any deceptive workarounds (no `//nolint`).

## 6. No Hacks or Workarounds
- It is strictly forbidden to use stubs, dummy implementations, or mocks in production code to artificially inflate metrics.
- Solutions must be architecturally clean, robust, and sustainable.
