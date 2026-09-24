# Low-Level Network Protocol Primitives (`net/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/net)

`net/` provides high-performance, RFC-compliant protocol encoders, low-level network framing primitives, proxy connectors, and the QUIC transport engine optimized for zero memory allocations.

## 1. Package Structure

```text
foundation/net/
├── ip/       // Subnet calculators, CIDR matchers, and IPv6 address pool rotators
├── ipc/      // Inter-Process Communication (Unix domain sockets, Windows named pipes)
├── netutil/  // Host sanitization and address normalization (CleanHost, CleanHostPort)
├── proxy/    // SOCKS4, SOCKS5, HTTP CONNECT, and PROXY protocol v1/v2 parsers
├── quic/     // RFC 9000 pure-Go zero-allocation QUIC protocol engine
├── tls/      // SPKI public key hash pinning and TLS certificate verifiers
└── urlkit/   // Fast zero-copy URL parser and parameter encoder (CRC32 sharded cache)
```

## 2. Core Protocol Engines

### A. QUIC Transport Engine (`net/quic`)
RFC 9000 compliant pure-Go QUIC protocol implementation optimized for extreme line-rate performance with zero heap allocations on the packet processing path.

```go
// Push-iterator based Varint decoding directly from byte slices
for val := range varint.DecodeSeq(packetPayload) {
    processFrame(val)
}
```

### B. High-Speed URL Engine (`net/urlkit`)
Zero-allocation URL parsing, query string serialization, and path variable expansion backed by a CRC32 sharded cache.

```go
u, err := urlkit.Parse("https://api.internal/v1/users?id=123")
if err == nil {
    fmt.Println("Host:", u.Host, "Path:", u.Path)
}
```

### C. Proxy Protocols (`net/proxy`)
Comprehensive support for SOCKS4, SOCKS5 (with auth), HTTP `CONNECT` tunneling, and PROXY protocol v1/v2:

```go
dialer, err := proxy.FromURL("socks5://user:pass@127.0.0.1:1080")
conn, err := dialer.DialContext(ctx, "tcp", "api.example.com:443")
```

### D. IP Subnets & CIDR (`net/ip`)
Zero-allocation IP range parsing, subnet membership testing, and CIDR prefix tree matching:

```go
set := ip.NewCIDRSet()
set.AddCIDR("10.0.0.0/8")
if set.Contains(net.ParseIP("10.1.2.3")) {
    fmt.Println("Internal IP")
}
```

## 3. Performance & Standards Compliance

| Protocol / Component | Relevant Standard | Zero-Alloc Optimization |
| :--- | :--- | :---: |
| **QUIC Transport** | RFC 9000 | In-place packet decryption & push iterators |
| **URL Engine (`urlkit`)** | RFC 3986 | CRC32 sharded cache and zero-alloc string views |
| **SOCKS5 / HTTP Proxy** | RFC 1928 / RFC 7231 | State machine without buffer reallocation |
| **IP Routing & CIDR** | RFC 4632 / RFC 4291 | Fast bitwise masking & integer range comparison |
| **Host Normalization** | RFC 3492 / RFC 5891 | In-place bracket and zone stripping |
