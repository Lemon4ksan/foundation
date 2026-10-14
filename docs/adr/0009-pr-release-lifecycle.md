# ADR-0009: Trunk-based PR review lifecycle, Conventional Commits, and automated releases

Status: Accepted  
Date: 2026-10-03  
Deciders: Lemon4ksan  

## Context

In high-velocity software engineering—particularly when utilizing advanced agentic coding assistants—unconstrained development workflows introduce severe quality and maintenance risks:

1. **Direct Pushes to Main Branch:** Pushing directly to `main` bypasses continuous integration checks, allowing untested code, broken platform builds, or subtle race conditions to enter production branches unnoticed.
2. **AI "Vibe Coding" Hazards:** Autonomous AI assistants can rapidly write large volumes of code that appear plausible on the surface but violate delicate low-level architectural invariants (e.g. introducing hidden heap allocations in `zero` tier packages, violating layer boundaries, or introducing cyclic dependencies). Without mandatory branch isolation and human code review, AI-generated subtle bugs easily slip through.
3. **Manual Release Fatigue and Git History Rot:** Manually tagging git releases, updating `CHANGELOG.md`, and calculating Semantic Versioning (SemVer) increments are tedious and error-prone. Inconsistent tag formats and missing release notes undermine trust for downstream library consumers (`mach`, `seal`, `ctrl`).
4. **Merge Commit Sprawl:** Uncontrolled branch merges produce non-linear, tangled git history graphs that make git bisecting, cherry-picking, and auditing regressions painful.

## Decision

We establish a disciplined **trunk-based development workflow** governed by mandatory branch protection, human-in-the-loop review, Conventional Commits squash merges, and fully automated semantic releases via Google's `release-please`.

### The Development and Release Lifecycle

```text
 ┌──────────────────────┐
 │  Feature / Fix Task  │
 └──────────┬───────────┘
            │
            ▼
 ┌────────────────────────────────────────────────────────┐
 │ Branch off main:                                       │
 │ feat/*, fix/*, docs/*, refactor/*, test/*              │
 └──────────┬─────────────────────────────────────────────┘
            │
            ▼
 ┌────────────────────────────────────────────────────────┐
 │ Pull Request with Conventional Commit Title:           │
 │ e.g. feat(iokit): implement vectored buffer queue      │
 └──────────┬─────────────────────────────────────────────┘
            │
            ├─────────────────────────────────────────────┐
            ▼                                             ▼
 ┌──────────────────────────────────────┐    ┌───────────────────────────┐
 │ GitHub Actions CI:                   │    │ Human Maintainer Review:  │
 │ - Semantic PR Title validation       │    │ - Architectural vetting   │
 │ - Multi-OS Matrix (Linux, Mac, Win)  │    │ - Memory budget audit     │
 │ - Race detector & benchmark checks   │    │ - Layer isolation audit   │
 │ - Layer & doc.go AST verification    │    │ - AI generation oversight │
 │ - Coverage ratchet check             │    │                           │
 └──────────────────┬───────────────────┘    └─────────────┬─────────────┘
                    │                                      │
                    └──────────────────┬───────────────────┘
                                       ▼
                     ┌───────────────────────────────────┐
                     │ Squash and Merge to main:         │
                     │ Produces single Conventional      │
                     │ Commit on linear main branch      │
                     └─────────────────┬─────────────────┘
                                       │
                                       ▼
                     ┌───────────────────────────────────┐
                     │ Google Release Please Action:     │
                     │ - Parses Conventional Commits     │
                     │ - Calculates SemVer bump          │
                     │ - Updates CHANGELOG.md            │
                     │ - Publishes GitHub Release & Tag  │
                     └───────────────────────────────────┘
```

### Core Invariants and Rules

1. **Trunk-Based Branch Isolation:**
   - The `main` branch is permanently protected. Direct pushes to `main` are strictly forbidden for both humans and AI agents.
   - All work occurs in dedicated, short-lived feature or bugfix branches (`feat/*`, `fix/*`, `docs/*`, `refactor/*`, `test/*`).
2. **Conventional Commits PR Title Validation:**
   - Every pull request title must strictly follow the [Conventional Commits specification](https://www.conventionalcommits.org/) (`feat(...)`, `fix(...)`, `docs(...)`, `refactor(...)`, `perf(...)`, `test(...)`, `ci(...)`, `chore(...)`).
   - Validated automatically in CI via `amannn/action-semantic-pull-request` ([`pr-title.yml`](../../.github/workflows/pr-title.yml)).
3. **Mandatory Human-in-the-Loop Review:**
   - Every PR requires review and explicit approval by the repository decider (`Lemon4ksan`).
   - Human review specifically audits non-obvious design trade-offs, validates that AI-assisted code adheres to the zero-dependency standard, and ensures memory allocation tiers ([ADR-0003](0003-allocation-budgets.md)) are upheld.
4. **Strict CI Gate:**
   - A pull request cannot be merged unless all automated checks in [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) pass cleanly:
     - Cross-platform build on Linux, macOS, and Windows.
     - Race detection (`go test -race`).
     - Benchmark allocation regression tests (`go test -bench=. -benchmem`).
     - Architectural layer and `doc.go` AST verification (`root_test.go`, [ADR-0002](0002-layering-and-isolation.md), [ADR-0008](0008-doc-go-canonical-truth.md)).
     - Statement coverage ratchet check ([ADR-0007](0007-coverage-ratchet.md)).
5. **Linear History via Squash and Merge:**
   - Pull requests are merged exclusively via **Squash and Merge**.
   - The squash commit message defaults to the validated Conventional Commit PR title, ensuring that `main` maintains a 100% linear, clean, and legible git history.
6. **Automated Semantic Releases via Release Please:**
   - The [`.github/workflows/release.yml`](../../.github/workflows/release.yml) workflow runs Google's `release-please-action` on every push to `main`.
   - Release Please scans conventional commit messages since the last tag, calculates the next SemVer version (`MAJOR.MINOR.PATCH`), automatically generates `CHANGELOG.md`, and opens a Release PR.
   - Merging the Release PR automatically tags the repository and publishes a GitHub Release.

## Consequences

### What gets easier:
- **Clean Linear History:** `git log --oneline main` displays an immaculate, chronological list of features and bug fixes without noisy intermediate WIP commits or merge bubbles.
- **Zero-Touch Releases:** Releasing a new version requires zero manual tag manipulation or changelog writing; the entire release engineering lifecycle is deterministic and automated.
- **Safe Agentic Pair Programming:** Developers and AI assistants can rapidly iterate and commit on feature branches with absolute confidence that unvetted code cannot reach production without passing rigorous CI and human review.

### What gets harder / what we pay:
- Even minor documentation typo fixes must follow the branch -> PR -> CI -> squash merge workflow.
- PR titles must be formatted carefully; invalid conventional commit titles fail CI immediately.

## Alternatives Considered

- **GitFlow (develop / release / main branches):** Rejected. Creates unnecessary branching ceremony, slow release cadences, and complex merge conflicts for a modern Go systems substrate.
- **Manual Release Tagging & Changelog Curation:** Rejected. Inevitably leads to forgotten release tags, incomplete changelogs, and accidental breaking changes released under minor SemVer tags.
- **Merge Commits (Non-Squash):** Rejected. Pollutes `main` git history with dozens of intermediate "fix typo" or "wip" commits, making `git bisect` frustrating during regression debugging.
