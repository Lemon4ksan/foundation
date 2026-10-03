# ADR-0008: Canonical package documentation in doc.go

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
Maintaining documentation outside Go source code leads to rot. External Markdown files are not checked by the Go compiler or godoc, causing examples to diverge when APIs change.

## Decision
`doc.go` and `example_test.go` are the only package documentation. We do not add Markdown guides for individual packages.

Every package documents itself in `doc.go` with exactly this format, one paragraph each:
- Stdlib counterpart
- Rejected compromise
- Accepted cost
- Allocations

This is enforced by `TestArchitecture_PublicPackagesDocGo` in `root_test.go`.

## Consequences
API documentation stays accurate and examples compile. We pay by reading documentation in godoc format rather than stylized Markdown pages.
