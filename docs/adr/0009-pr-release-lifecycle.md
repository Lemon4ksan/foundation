# ADR-0009: Pull-Request Only Workflow and Automated Releases

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: GitHub workflows, release automation, repository process

## 1. Context & Problem Statement

All 160+ prior commits in `foundation` were pushed directly to `main` without branch reviews, PR templates, or formal release tags. Consequently:
- Codebases like `aoni`, `g-man`, and `seal` relied on volatile pseudo-versions (`v0.0.0-2026...`), making dependency rollbacks hazardous.
- AI agents could commit sweeping changes without human review of the diff, resulting in loss of mental ownership by the project author.
- Conventional commit conventions were adhered to informally without programmatic enforcement.

## 2. Decision Drivers

- Enforce human-in-the-loop review of every change: Intent is defined by the human; execution is carried out by the agent; diff is reviewed and approved by the human.
- Zero direct pushes to `main` via GitHub branch protection.
- Fully automated SemVer releases, Git tags, and changelog generation.

## 3. Considered Options

1. **Option 1: Direct Pushes to `main` with Manual Git Tagging**: Unpredictable quality control.
2. **Option 2: Git Flow with Develop and Release Branches**: Heavy ceremony ill-suited for rapid systems iteration.
3. **Option 3: Trunk-Based Development with Squash PRs and Release-Please**:
   - Feature branches for all work.
   - PR title must satisfy Conventional Commits.
   - Squash-and-merge to `main` ensures a clean, linear git history.
   - `release-please` automatically parses commit messages and creates release PRs with CHANGELOG updates and SemVer tags (`v0.x.y`).

## 4. Decision Outcome

**Chosen Option**: **Option 3: Trunk-Based Development with Squash PRs and Release-Please**.

### 4.1. Lifecycle Mechanics

1. **Issue Definition**: Created using `.github/ISSUE_TEMPLATE/task.yml` defining target packages, allocation budget, and consumer requirements.
2. **Dedicated Branch**: Work occurs in a dedicated branch (`feat/...`, `fix/...`, `perf/...`).
3. **Pull Request & CI**: `.github/pull_request_template.md` checklist completed; CI runs `make check`.
4. **Author Review & Squash-Merge**: Author reviews the concise diff and merges via squash-and-merge.
5. **Automated Release**: GitHub Action (`release-please`) creates or updates a release PR with generated CHANGELOG; merging the release PR automatically tags `v0.x.y` and publishes release notes.

## 5. Verification & Testing

- `.github/workflows/pr-title.yml` validates PR titles against Conventional Commit standards.
- `.github/workflows/ci.yml` validates full test, race, lint, and coverage ratchet status before allowing merges.
