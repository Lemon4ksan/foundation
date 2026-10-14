# ADR-0008: Canonical package documentation via doc.go and automated AST verification

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

In long-lived software systems and high-performance libraries, architectural documentation frequently degrades into technical debt due to three classic failure modes:

1. **Documentation Rot in External Markdown Files:** Maintaining architectural rationale, benchmark numbers, or API guides in separate Markdown files (`docs/packages/*.md` or external wikis) guarantees drift. As function signatures, allocation profiles, and concurrency contracts evolve, external documentation is rarely updated because it sits outside the compiler's view.
2. **Missing Architectural Rationale:** Standard Go doc comments often describe *what* a package does without explaining *why* it exists. In a systems substrate designed to replace or complement standard library components ([ADR-0001](0001-stdlib-v2.md)), developers need to understand what specific standard library trade-off was rejected, what operational cost was accepted, and what memory budget applies.
3. **Inconsistent Documentation Quality:** Without automated enforcement, documentation quality varies wildly: some packages contain multi-page essays while others contain only a single terse comment, leaving API consumers guessing about safety invariants.

## Decision

We establish that **`doc.go`** and companion **`example_test.go`** files are the **sole and canonical source of truth** for package-level documentation across `foundation`.

### The Canonical doc.go Contract

Every public package in `foundation` must include a `doc.go` file containing a comprehensive package doc comment structured with exactly the following four formal sections:

```go
// Stdlib counterpart: <Name of Go standard library package or "none - fills gap: ...">
//
// Rejected compromise: <The exact portability, allocation, or abstraction trade-off in stdlib that foundation rejects>
//
// Accepted cost: <The engineering cost accepted in foundation, e.g. caller destination buffers, explicit release>
//
// Allocations: <Formal allocation tier: zero | amortized | bounded(N/op) | unconstrained>
package mypackage
```

1. **`Stdlib counterpart`**:
   Explicitly declares the Go standard library package (e.g. `encoding/json`, `sync.Pool`, `net`, `time`) that this package replaces, or documents the specific gap filled if no direct standard counterpart exists.
2. **`Rejected compromise`**:
   Pinpoints the precise architectural compromise present in standard Go that forced the creation of this package (e.g. interface boxing, un-vectorized scalar loops, runtime reflection, absence of generational borrow safety, or non-deterministic GC allocation).
3. **`Accepted cost`**:
   Clearly states the trade-off accepted in exchange for performance or safety (e.g. developer discipline of explicit `defer ref.Release()`, lack of support for 32-bit legacy CPU architectures, caller-provided preallocated destination buffers, or rigid layer isolation).
4. **`Allocations`**:
   Formally states the package's memory allocation tier conforming strictly to [ADR-0003](0003-allocation-budgets.md) (`zero`, `amortized`, `bounded(N/op)`, or `unconstrained`).

### Executable Examples in `example_test.go`

High-level usage guides and tutorials must not be written as static text in Markdown files. Instead, they must be implemented as runnable Go test examples in `example_test.go` (`func Example...()`).
- Examples are compiled and executed by standard `go test ./...`.
- If an API changes, out-of-date examples fail the build immediately, preventing documentation rot at compile time.
- Godoc automatically renders these examples in the official package documentation.

### Automated CI and AST Enforcement

We enforce this standard programmatically:
- `TestArchitecture_PublicPackagesDocGo` in [`root_test.go`](../../root_test.go) walks every public package in the module. If any public package lacks a `doc.go` file, the test fails immediately.
- The `.github/pull_request_template.md` includes a mandatory checklist item requiring reviewers to verify the four canonical sections before approval.
- Stale exemptions in test whitelists are ratcheted down automatically.

## Consequences

### What gets easier:
- **Zero Documentation Drift:** Because documentation lives in `doc.go` next to the implementation and examples are compiled Go code, documentation remains synchronized with code refactors.
- **Unified Architectural Consistency:** Every package across the entire substrate presents the same professional, structured narrative to consumers and contributors.
- **Godoc Integration:** Standard Go documentation tools (`pkgsite`, `godoc`) automatically render the canonical trade-offs and runnable examples without requiring custom documentation static site generators.

### What gets harder / what we pay:
- Every new package requires authoring a rigorous `doc.go` satisfying all four sections before CI will allow merging.
- Free-form Markdown guides for individual packages are forbidden; developers who want to write guides must author runnable `example_test.go` files instead.

## Alternatives Considered

- **External Markdown Documentation (`docs/packages/*.md`):** Rejected. Guaranteed to suffer documentation rot within months as engineers modify Go APIs without updating peripheral Markdown directories.
- **Unstructured Free-Form Doc Comments:** Rejected. Without rigid heading requirements, doc comments quickly degrade into superficial one-sentence summaries that omit memory allocation tiers and architectural trade-offs.
- **Inline Struct/Func Comments Only:** Rejected. Field-level comments are essential for API details, but fail to convey high-level architectural context, stdlib trade-offs, and memory budgets.
