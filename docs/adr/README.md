# Architecture Decision Records (ADRs)

This directory documents all significant architectural decisions made in `foundation`. Each record captures the context, options evaluated, chosen strategy, tradeoffs accepted, and verification methods.

## Index of Records

| ADR | Title | Status | Date | Target Packages |
| :--- | :--- | :--- | :--- | :--- |
| [0000](0000-template.md) | Architectural Decision Record Template | Template | 2026-10-03 | All |
| [0001](0001-stdlib-v2.md) | Foundation as Go Standard Library v2.0 | Accepted | 2026-10-03 | Foundation-wide |
| [0002](0002-layering-and-isolation.md) | Layering Hierarchy and Dependency Isolation | Accepted | 2026-10-03 | Root & All Packages |
| [0003](0003-allocation-budgets.md) | Four-Tier Memory Allocation Contracts | Accepted | 2026-10-03 | Foundation-wide |
| [0004](0004-c2plan9-simd-substrate.md) | c2plan9 and Silicon SIMD Substrate | Accepted | 2026-10-03 | `silicon/*`, `csrc/*`, `cmd/c2plan9` |
| [0005](0005-affine-ownership-borrow.md) | Runtime Generational Affine Ownership (`borrow`) | Accepted | 2026-10-03 | `borrow`, `codec/compress/*` |
| [0006](0006-l7-boundary-and-mach.md) | L7 Domain Protocol Boundary and Extraction to Mach | Accepted | 2026-10-03 | `net/*`, Protocol Codecs |
| [0007](0007-coverage-ratchet.md) | Baseline Statement Coverage Ratchet | Accepted | 2026-10-03 | Testing & CI |
| [0008](0008-doc-go-canonical-truth.md) | Canonical Package Documentation via `doc.go` | Accepted | 2026-10-03 | Docs & Packages |
| [0009](0009-pr-release-lifecycle.md) | Pull-Request Only Workflow and Automated Releases | Accepted | 2026-10-03 | Workflow & Tooling |

## Guidelines for Authors and AI Agents

1. Whenever proposing a new structural package, changing dependency layers, or modifying allocation guarantees, submit a new ADR following [0000-template.md](0000-template.md).
2. Numbering is sequential (`0001`, `0002`, ...).
3. The ADR must be referenced in the corresponding Pull Request.
