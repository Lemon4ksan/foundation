# ADR-0006: Domain boundaries and strict isolation

Status: Accepted
Date: 2026-10-10
Deciders: Lemon4ksan

## Context
Early versions of this repository implemented Layer 7 protocols, heavy C-ported codecs (zstd/lz4), TUI frameworks, and complex transport engines (QUIC, SOCKS proxies). This conflated low-level computational mechanics with high-level application logic, causing massive repository bloat.

## Decision
We strictly exclude all high-level application protocols, heavy data-transformation codecs, and user-interface libraries from `foundation`.
- Complex L4 transports and proxy routing (`net/quic`, `net/proxy`) belong in `mach`.
- Heavy data sealing and compression (`codec/compress`, `text/encoding`) belong in `seal`.
- Terminal UI and command argument routing (`tuikit`, `argkit`) belong in `ctrl`.

`foundation` provides strictly the bare systems substrate: memory (`silicon`), data structures, concurrency (`async`, `sync`), raw I/O, base TCP/UDP/IP, and essential serialization (`codec/json`).

## Consequences
The repository stays mathematically focused, lean, and avoids churn from application-level changes. We pay by needing to coordinate dependency trees across `foundation`, `mach`, `seal`, and `ctrl`.
