# ADR-0001: Foundation as Go standard library v2.0

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
Go's standard library is full of compromises made so that it stays simple for newcomers and portable everywhere. Algorithms target lowest-common-denominator hardware, and APIs favor convenience over zero-allocation guarantees. High-throughput systems hit CPU and memory bottlenecks when relying on standard packages.

## Decision
We build `foundation` as what the standard library would look like without those compromises (a "stdlib v2.0" for the author's own systems work). It provides drop-in, high-performance replacements for standard packages.

We maintain API autonomy and do not mimic flawed stdlib signatures.

## Consequences
We get predictable zero-allocation, hardware-accelerated performance on Tier-1 hardware (amd64 / arm64). We pay by requiring newer Go compilers and forcing callers to handle breaking API improvements before v1.0.

## Alternatives considered
- Ad-hoc utility repositories: scattered packages inside each codebase. Hard to share.
- Fork the Go compiler and standard library: too much maintenance overhead.
