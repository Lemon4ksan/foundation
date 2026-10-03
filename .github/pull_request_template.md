## What and why

<!-- What does this change do, and why? Link the issue: Closes #... -->

Packages touched:

## Checklist

- [ ] The PR title is a Conventional Commit (`feat(pool): ...`, `fix(urlkit): ...`, `docs: ...`). It becomes the squash commit message.
- [ ] `make check` passes (lint, race tests, architecture, coverage ratchet).
- [ ] Layers and import rules still hold. No foundation package imports `generic`.
- [ ] The package's allocation tier still holds; hot paths are covered by `testing.AllocsPerRun` or a `-benchmem` run.
- [ ] `doc.go` and examples reflect the change, including the comparison block (`Stdlib counterpart`, `Rejected compromise`, `Accepted cost`, `Allocations`).
- [ ] New package, new layer edge or new substrate? Then an ADR is in `docs/adr/`.
