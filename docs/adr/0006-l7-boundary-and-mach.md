# ADR-0006: Layer 7 protocol boundary

Status: Accepted
Date: 2026-10-03
Deciders: Lemon4ksan

## Context
Early versions of this repository implemented Layer 7 protocols like HTTP/2, HPACK, and QPACK. This conflated low-level transport mechanics with high-level wire semantics, forcing version bumps for protocol spec changes and obscuring the repository's mission as a systems substrate.

## Decision
We strictly exclude all Layer 7 application protocols (HTTP, HPACK/QPACK, WebSocket, gRPC, Steam). `foundation` only provides Layer 4 transport and QUIC stream primitives (`net/*`, `net/quic`, `net/proxy`, `net/ip`, `net/tls`). 

Application protocol framing and HTTP engines live in external repositories or consumer codebases.

## Consequences
The repository stays focused and avoids churn from application protocol changes. We pay by needing to coordinate releases across repositories when lower-level APIs change.
