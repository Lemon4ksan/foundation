# QUIC Varint Encoding (`encoding/varint`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/encoding/varint)

`encoding/varint` implements the variable-length integer encoding described in RFC 9000 (QUIC). It utilizes assembly-optimized SIMD intrinsics where available to process varints with zero allocations and minimal CPU cycles.

## 1. Features

- **Standard Compliance**: Fully conforms to RFC 9000 variable-length integer encoding.
- **Hardware Acceleration**: Employs `.s` assembly generated from C/LLVM via `c2plan9` for ultra-fast, zero-allocation integer decoding.
- **Bounds Checking**: Fails fast and securely on malformed or truncated varint streams.

## 2. Examples

```go
import "github.com/lemon4ksan/foundation/encoding/varint"

func readLen(b []byte) (uint64, int, error) {
    // Read QUIC varint from byte slice
    val, bytesRead, err := varint.Read(b)
    if err != nil {
        return 0, 0, err
    }
    return val, bytesRead, nil
}
```
