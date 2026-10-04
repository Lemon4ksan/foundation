# ADR-0010: Vectored I/O queues and DNS wire transport codecs

Status: Accepted
Date: 2026-10-04
Deciders: Lemon4ksan

## Context

`foundation` serves as the systems substrate and standard library v2.0 for high-throughput network engines ([ADR-0001](0001-stdlib-v2.md)). While Layer 7 application protocols (HTTP, HPACK/QPACK, WebSocket) are strictly excluded and delegated to `mach` ([ADR-0006](0006-l7-boundary-and-mach.md)), two essential transport-level capabilities were previously missing from `foundation`:

1. **Vectored Scatter-Gather I/O**: High-performance network and disk writers frequently assemble fragmented buffers (e.g. precomputed framing headers, payload chunks, and trailers). Standard `io.Writer` abstractions force either intermediate memory flattening (allocating contiguous slices) or multiple syscalls. Go's runtime supports OS-level vector writes (`writev` on POSIX, `WSASend` on Windows) via `net.Buffers`, but lacks bounded, pooled queue primitives with Go 1.23+ push iterators.
2. **Layer 4 DNS Wire Codecs & Service Discovery**: Modern transport bootstrapping (especially QUIC / HTTP/3 discovery via SVCB and HTTPS resource records per RFC 9460, as well as Encrypted ClientHello / ECH) requires wire-level DNS parsing and encoding. Go's standard library `net.Resolver` allocates heavily, lacks RFC 9460 SVCB/HTTPS support, and cannot apply EDNS0 block padding (RFC 7830) or Client Subnet options (RFC 7871) directly on raw network buffers.

Prototypes of these components were temporarily drafted inside `mach`, violating `mach`'s L7 protocol engine boundary.

## Decision

We incorporate these transport and I/O primitives directly into `foundation`'s layered hierarchy ([ADR-0002](0002-layering-and-isolation.md)):

1. **Vectored Scatter-Gather Queue in Layer 3 (`iokit`)**:
   - Introduce `BufferQueue` into `iokit` ([ADR-0002](0002-layering-and-isolation.md)). `BufferQueue` manages a zero-copy list of non-contiguous byte slices backed by `net.Buffers`, offering configurable capacity limits (`WithMaxSlices`), pooled reuse (`AcquireBufferQueue`, `ReleaseBufferQueue`), and zero-allocation push-iterators (`Buffers() iter.Seq[[]byte]`, `All() iter.Seq2[int, []byte]`).
   - Flushes directly execute OS-level `writev` / `WSASend` when writing to a `net.Conn`.
   - Allocation tier: `amortized` ([ADR-0003](0003-allocation-budgets.md)).

2. **DNS Wire Message and SVCB/HTTPS Codecs in Layer 4 (`net/dns/*`)**:
   - Introduce `net/dns/wire`: low-level, zero-allocation binary encoders and decoders for RFC 1035, RFC 2181, RFC 3596, and RFC 6891 wire messages. Includes query packing with EDNS0 128-byte block padding and Client Subnet (ECS) options, DNS name compression traversal (`SkipDomainName`), and ECH extraction. Allocation tier: `zero` ([ADR-0003](0003-allocation-budgets.md)).
   - Introduce `net/dns/svcb`: wire-format parser and serializer for Service Binding (SVCB RR 64) and HTTPS (RR 65) records conforming strictly to RFC 9460. Supports ALPN negotiation, IP hints, port prefix naming, and client compatibility verification. Allocation tier: `bounded(1/op)` ([ADR-0003](0003-allocation-budgets.md)).
   - In accordance with ADR-0002, neither package imports `generic` or Layer 5 packages.

3. **Exclusion of Raw L3 Networking**:
   - Prototypes for raw IPv4/IPv6 packet headers, ICMP framing, and TCP MSS options are excluded from `foundation`. `foundation` targets socket-based Layer 4 transports (`net.Conn`, QUIC), not userspace IP routing or raw packet filtering.

## Consequences

- What gets easier:
  - Downstream network engines (`mach`, `sein`, `aoni`) can queue non-contiguous frame fragments into `iokit.BufferQueue` and transmit them in a single syscall without memory copies.
  - Bootstrapping HTTP/3 and negotiating ECH can be performed natively with zero external dependencies and zero heap allocations on query packing.
  - Strict adherence to ADR-0002 layer boundaries is preserved: imports point strictly down the layer stack.
- What gets harder / what we pay:
  - Callers of `BufferQueue` must uphold buffer lifetime guarantees (slices passed to `Push` must not be modified until `WriteTo` or `Reset` completes).
  - DNS wire parsers operate directly on binary slice offsets; callers are responsible for managing message buffers.

## Alternatives considered

- **Retaining in `mach`**: Rejected. Violates `mach-guidelines` and ADR-0006 by entangling transport wire mechanics with application protocol logic.
- **Importing external DNS libraries (e.g. `miekg/dns`)**: Rejected. External libraries allocate heavily on every message, do not conform to `foundation`'s allocation budgets ([ADR-0003](0003-allocation-budgets.md)), and breach the zero-dependency substrate principle.
