---
name: foundation-guidelines
description: Rules and boundaries for AI agents working on foundation
trigger: always_on
---

# Working on foundation

`foundation` is Go's standard library without the compromises that keep it simple and portable: a stdlib v2.0 for the author's own systems work. Read [docs/VISION.md](../docs/VISION.md) first, and check [docs/adr/](../docs/adr/README.md) before you change structure. This file only lists what you must not get wrong.

## Scope

- Stay inside the packages the task names. If the fix needs another package, say so and ask.
- Every package must be needed by at least one consuming project (the list is in [docs/VISION.md](../docs/VISION.md)) and must know nothing about any of them.
- No L7 protocols (HTTP, HPACK/QPACK, WebSocket, gRPC, Steam). They belong in `mach` or the consuming project.
- No business logic, application models or database code.
- Do not add a helper that hides a heap allocation on a hot path (`any`, reflection, interface boxing), however convenient.

## Structure

- Import rules and layers are in [ADR-0002](../docs/adr/0002-layering-and-isolation.md) and are checked by `root_test.go`. In short: imports point down the stack, `silicon/*` imports nothing else from foundation, and nothing inside foundation imports `generic`.
- Low-level packages do not define their own `Optional` or `Result`. Return `(T, bool)` or `(T, error)`; callers who want a monad wrap it with `generic`.
- Each package declares an allocation tier ([ADR-0003](../docs/adr/0003-allocation-budgets.md)). Do not make a package allocate more than its tier allows.
- If you add a package or move a layer boundary, write an ADR.

## Documentation

`doc.go` and `example_test.go` are the only package documentation ([ADR-0008](../docs/adr/0008-doc-go-canonical-truth.md)). Do not add Markdown guides for a package. When you change behaviour, update `doc.go` in the same change.

## Quality

- `make check` must pass: lint, race tests, architecture tests, coverage ratchet.
- Coverage never goes down. Do not write trivial or fake tests to raise it.
- Everything is tested with `-race`; any race warning is a bug.
- Decoders, parsers and crypto code need fuzz tests.
- All files are in English.

## When to stop and ask

Stop and ask the user if the task is ambiguous, if it would cross a layer boundary, or if it needs a decision that an ADR should record.
