## Summary & Rationale

<!-- Concisely describe the motivation and context for this change. Reference issue: Closes #... -->

### Target Packages
- `...`

## Architectural & Quality Verification

- [ ] **Conventional Commits**: PR title adheres strictly to the conventional format (`feat(...)`, `fix(...)`, `perf(...)`, `refactor(...)`, `docs(...)`).
- [ ] **Layering & Isolation**:
  - Layers L0–L5 respected (passes architectural tests in `root_test.go`).
  - Internal packages DO NOT import `generic`.
  - `silicon/*` packages are autonomous and do not cross-import each other.
- [ ] **Allocation Contract**:
  - Declared allocation tier (`zero` / `amortized` / `bounded` / `unconstrained`) upheld.
  - Hot paths verified via `testing.AllocsPerRun` or `-benchmem` checks.
- [ ] **Canonical Documentation (`doc.go`)**:
  - Metadata block verified or updated:
    ```go
    // Stdlib counterpart: ...
    // Rejected compromise: ...
    // Accepted cost: ...
    // Allocations: ...
    ```
- [ ] **Quality & Verification**:
  - `make check` passes cleanly (lint, test, race, arch, coverage-ratchet).
  - Zero data race warnings (`go test -race`).
  - Statement test coverage meets or exceeds baseline ratchet.
- [ ] **Architecture Decision Records**:
  - If introducing a new package, altering boundaries, or introducing an algorithmic substrate, an ADR is included in `docs/adr/`.
