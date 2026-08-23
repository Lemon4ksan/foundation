# Low-Level Network Protocol Primitives (`net/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/net)

`net/` provides high-performance, RFC-compliant protocol encoders, low-level network framing primitives, proxy connectors, and DNS resolvers optimized for zero memory allocations.

## 1. Package Structure

```text
foundation/net/
├── cachestatus/  // RFC 9211 HTTP Cache-Status header parser
├── cookie/       // RFC 6265 cookie engine, path sorting, and proxy isolation
├── dns/          // DoH, DoQ, and DoT secure DNS resolvers
├── grpcweb/      // 5-byte framed gRPC-Web payload stream decoder & trailer validator
├── headkit/      // High-speed header canonicalization & case-preserving maps
├── hpack/        // RFC 7541 HTTP/2 HPACK table encoder & decoder
├── ip/           // Subnet calculators, CIDR matchers, and IPv6 address pool rotators
├── pkce/         // RFC 7636 OAuth2 Proof Key for Code Exchange generators
├── proxy/        // SOCKS4, SOCKS5, HTTP CONNECT, and PROXY protocol v1/v2 parsers
├── psl/          // Compiled Public Suffix List domain boundary resolver
├── sse/          // Real-time Server-Sent Events frame scanner
├── tls/          // SPKI public key hash pinning and TLS certificate verifiers
├── url/          // Fast zero-copy URL parser and parameter encoder
└── weblink/      // RFC 8288 Web Link relation parser (pagination, canonical)
```

## 2. Core Protocol Engines

### A. HPACK Header Compression (`net/hpack`)
RFC 7541 compliant HTTP/2 header table encoder and decoder optimized with SIMD Huffman decoding.

```go
decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
    fmt.Printf("Header: %s: %s (sensitive=%v)\n", f.Name, f.Value, f.Sensitive)
})
_, err := decoder.Write(hpackBlock)
```

### B. gRPC-Web 5-Byte Stream Framing (`net/grpcweb`)
Parses gRPC-Web binary streams, extracting payloads and validating trailers (`grpc-status`, `grpc-message`).

```go
parser := grpcweb.NewStreamParser(rawReader)
frame, err := parser.NextFrame()
if frame.IsTrailer {
    status := grpcweb.ParseTrailerStatus(frame.Payload)
}
```

### C. RFC 9211 Cache-Status Header Parser (`net/cachestatus`)
Decodes standardized `Cache-Status` headers emitted by modern CDNs (Cloudflare, Fastly, Akamai):

```go
status, err := cachestatus.Parse(`"origin-cache"; hit; ttl=3600; key="xyz"`)
if status.Hit {
    fmt.Println("Cache hit from:", status.TargetURI)
}
```

### D. Secure DNS Resolvers (`net/dns`)
Ultra-low-latency DNS-over-HTTPS (DoH), DNS-over-QUIC (DoQ), and DNS-over-TLS (DoT) client engines.

```go
resolver := dns.NewDoHResolver("https://1.1.1.1/dns-query")
ips, err := resolver.LookupIP(ctx, "api.example.com")
```

## 3. Performance & Standards Compliance

| Protocol / Component | Relevant Standard | Zero-Alloc Optimization |
| :--- | :--- | :---: |
| **HPACK Framing** | RFC 7541 | Static table zero-allocation lookup |
| **Cache-Status** | RFC 9211 | Direct byte tokenizer without regex |
| **Public Suffix List** | W3C / Mozilla PSL | Compressed perfect hash table lookup |
| **Web Link Relations** | RFC 8288 | In-place delimiter scan |
