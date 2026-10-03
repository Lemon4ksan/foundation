# ADR-0006: L7 Domain Protocol Boundary and Extraction to Mach

- **Status**: Accepted
- **Date**: 2026-10-03
- **Author / Deciders**: Lemon4ksan
- **Target Packages**: `net/*`, Protocol Codecs, External mach repository

## 1. Context & Problem Statement

Early versions of `foundation` accumulated Layer 7 protocol implementations (HTTP/2 framing, HPACK header compression, QPACK for HTTP/3, and domain-specific framing). Over time, this caused several architectural issues:
1. **Scope Bloat**: `foundation` was conflating low-level transport/runtime mechanics with high-level wire semantics.
2. **Coupling**: Changes to HTTP protocol specs forced version bumps and churn in low-level memory and concurrency packages.
3. **Loss of Focus**: The mission of `foundation` as a universal Go standard library replacement was obscured by domain-specific application protocol machinery.

## 2. Decision Drivers

- Maintain strict domain-neutrality for `foundation`.
- Isolate protocol-specific RFC details and complex framing machines in specialized repositories.
- Keep `foundation` bounded strictly to L4 transport and QUIC stream primitives.

## 3. Considered Options

1. **Option 1: Retain L7 Codecs in `foundation/net/http*`**: Keep everything in one monolithic library.
2. **Option 2: Extract L7 Codecs to a Dedicated Transport Repo (`mach`)**: Move HTTP/1-3, HPACK, QPACK, and high-level wire framing into `mach`, keeping only L4 transport (`net/quic`, `net/proxy`, `net/ip`, `net/tls`) in `foundation`.

## 4. Decision Outcome

**Chosen Option**: **Option 2: Extract L7 Codecs to `mach`**.

### 4.1. Division of Responsibilities

- **In `foundation`**:
  - `net/quic`: Low-level QUIC transport (RFC 9000), congestion control, packet framing, connection management.
  - `net/proxy`: SOCKS5 and low-level proxying.
  - `net/ip`, `net/tls`: Low-level IP parsing and TLS certificate handling.
  - `encoding/varint`, `silicon/simd`: Bit-level and vector framing primitives.
- **In `mach` (and consumer repositories)**:
  - HTTP/1.1, HTTP/2, HTTP/3 engines.
  - HPACK (RFC 7541) and QPACK (RFC 9204) dynamic table compressors.
  - Application RPC and game state protocol frames.

## 5. Verification & Testing

- Git history confirms commits `04d0a83` ("refactor: excise domain-specific L7 protocols and subsystems from foundation") and `315ed5e` ("refactor: extract hpack and qpack to mach repo").
- Architectural tests in `root_test.go` reject any introduction of L7 application protocol packages.
