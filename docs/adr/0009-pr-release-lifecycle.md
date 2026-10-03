# ADR-0009: Pull-request only workflow and automated releases

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
Direct pushes to `main` without branch reviews or release tags led to unpredictable quality and made dependency rollbacks hazardous. We needed human review for AI-generated code.

## Decision
We follow a trunk-based development workflow. All work happens in feature branches. The PR title is a Conventional Commit.

Every change requires human review. The process is: issue -> branch -> PR -> green CI -> review -> squash merge. 

Branch protection on `main` is a GitHub setting the owner turns on. The `release-please` GitHub Action cuts releases automatically based on squash commit messages.

## Consequences
We get a clean linear git history and predictable semantic versioning. We pay by losing the ability to push trivial fixes directly to `main`.
