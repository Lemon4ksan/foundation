# ADR-0006: Domain boundaries, ecosystem isolation, and Layer 7 protocol delegation

Status: Accepted  
Date: 2026-10-10  
Deciders: Lemon4ksan  

## Context

In early iterations of this ecosystem, `foundation` suffered from scope creep. As new capabilities were prototyped, the repository accumulated heterogeneous systems components:
1. **Layer 7 Application Protocols:** Full HTTP/1.1 and HTTP/2 engines, HPACK state machines, WebSocket frame parsers, and gRPC scaffolding.
2. **Heavy Codecs & Compression:** C-ported decompressors and compressors (Zstandard, LZ4, Brotli, Snappy).
3. **Application Transports & Proxies:** SOCKS5 proxies, Shadowsocks framing, and QUIC connection multiplexers.
4. **CLI Frameworks & User Interfaces:** Terminal UI widgets, ANSI box renderers, progress bars, and command-line argument parsers.

This monolithic accumulation created severe architectural failure modes:
- **Dependency & Compilation Bloat:** Simple utility packages imported sprawling protocol state machines, causing slow build cycles and large binary footprints.
- **Specification Churn Entanglement:** Application protocols (e.g. HTTP RFC revisions, WebSocket extensions) require frequent API updates and behavioral adjustments. When housed in `foundation`, this churn threatened the stability of low-level memory, cryptographic, and concurrency primitives that demand long-term API immutability.
- **Blurred Architectural Focus:** Engineers could not easily determine whether a new network utility belonged in low-level transport code or application-layer handlers.

## Decision

We establish strict sovereign repository boundaries across the ecosystem. `foundation` strictly delegates Layer 7 protocols, heavy compression algorithms, and CLI/TUI interfaces to specialized sovereign repositories.

### Ecosystem Repository Taxonomy

```text
┌────────────────────────────────────────────────────────────────────────┐
│ High-Level Ecosystem Repositories                                      │
├───────────────────┬───────────────────┬────────────────────────────────┤
│       mach        │       seal        │             ctrl               │
│ Layer 7 Protocols │ Heavy Compression │ CLI Application Framework      │
│ & Proxy Engines   │ & Archival Codecs │ & Hardware-Efficient TUI       │
│ (HTTP, WS, SOCKS) │ (Zstd, LZ4, Bro)  │ (argkit, tuikit)               │
└─────────┬─────────┴─────────┬─────────┴───────────────┬────────────────┘
          │                   │                         │
          ▼                   ▼                         ▼
┌────────────────────────────────────────────────────────────────────────┐
│                              foundation                                │
│ Core Systems Substrate & Standard Library v2.0 (Layers 0 – 5)          │
│ [silicon, borrow, bufkit, sync, async, crypto, encoding, net/dns, ...] │
└────────────────────────────────────────────────────────────────────────┘
```

1. **`foundation` (Systems Substrate & Standard Library v2.0):**
   - Strictly limited to low-level computational mechanics, memory safety, concurrency, cryptographic primitives, wire-format codecs, and core I/O ([ADR-0001](0001-stdlib-v2.md), [ADR-0002](0002-layering-and-isolation.md)).
   - Retains transport-level wire codecs: DNS wire messages and SVCB/HTTPS resource records (`net/dns/*`), vectored I/O queues (`iokit.BufferQueue`), and base TCP/UDP socket helpers ([ADR-0010](0010-vectored-io-and-dns-wire-codecs.md)).
   - Retains foundational serialization: JSON (`encoding/json`), Protocol Buffers (`encoding/proto`), binary encoders (`encoding/bin`), Base64 (`encoding/base64`), and UUIDs (`types/uuid`).
   - Zero external dependencies. Zero high-level protocol state machines.

2. **`mach` (Layer 7 Protocol Engine & High-Throughput Proxy Substrate):**
   - Implements full Layer 7 protocol engines: HTTP/1.1 pipelining, HTTP/2 multiplexing and HPACK compression, HTTP/3 connection abstractions and QPACK, WebSocket frame negotiation, and SOCKS5 proxy tunnels.
   - Built on top of `foundation` primitives (`iokit.BufferQueue`, `silicon/simd`, `borrow.Slot[T]`, `async/task`).

3. **`seal` (Heavy Data Compression & Archival Engines):**
   - Houses heavy, compute-intensive compression and archival codecs: Zstandard (RFC 8878), LZ4, Brotli, DEFLATE, and tar/zip container formats.
   - Depends on `foundation` for SIMD acceleration (`silicon/simd`) and allocation-free streaming buffers (`bufkit`).

4. **`ctrl` (CLI Framework & Terminal UI Engine):**
   - Houses zero-dependency CLI application routing, POSIX flag parsing (`argkit`), and terminal UI widgets (`tuikit`: boxes, progress bars, tables, sparklines, ANSI styling).
   - Depends on `foundation` for base string utilities and memory efficiency.

### Dependency Direction Invariant

The dependency flow between repositories is strictly unidirectional:
- `mach`, `seal`, and `ctrl` may import `foundation`.
- `foundation` **never** imports `mach`, `seal`, or `ctrl`.

## Consequences

### What gets easier:
- **Forever-Frozen Core Substrate:** `foundation` achieves maximum architectural stability; its core APIs remain unchanged for years without pressure from evolving L7 application specs.
- **Fast Build Times & Minimal Binaries:** Projects importing `foundation` incur zero build overhead from unused HTTP engines, decompression tables, or terminal graphics.
- **Targeted Evolution:** Protocol optimizations in `mach` and compression improvements in `seal` can release on their own cadence without requiring version bumps in the underlying substrate.

### What gets harder / what we pay:
- Features spanning transport and application layers require disciplined separation (e.g. raw DNS wire codecs live in `foundation/net/dns`, but HTTP/3 Alt-Svc / SVCB discovery orchestration lives in `mach`).
- Cross-repository API adjustments require coordinated release cycles across GitHub repositories.

## Alternatives Considered

- **Monolithic Mega-Repository:** Consolidating all projects into a single monorepo. Rejected: inflates CI execution times, entangles decoupled semantic version tags, and obscures package dependency boundaries.
- **Embedding Minimal HTTP/1.1 in `foundation`:** Rejected. Even a "minimal" HTTP parser requires header normalizers, chunked transfer-encoding state machines, trailer handling, and URI validation that belong squarely in application protocol layers.
