# Multi-Algorithm Compression & Codec Substrate (`codec`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/codec)

`codec` provides a unified, zero-allocation multi-algorithm compression engine, stream filters, entropy estimation, and high-performance JSON serialization.

## Architectural Components

```text
foundation/codec/
├── compress/            # Core compression and decompression engine
│   ├── brotli/          # RFC 7932 Brotli codec
│   ├── zstd/            # RFC 8878 Zstandard codec & xxhash
│   ├── gzip/            # RFC 1952 Gzip codec
│   ├── flate/           # RFC 1951 Deflate codec
│   ├── lz4/             # Frame and block LZ4 codec
│   ├── lzma/            # LZMA / XZ codec
│   ├── fse/             # Finite State Entropy (FSE) codec
│   ├── huff0/           # Huff0 prefix entropy codec
│   ├── dcb.go           # Shared Dictionary Compression for Brotli (RFC 7932)
│   └── dcz.go           # Shared Dictionary Compression for Zstandard (RFC 8878)
│
├── filter/              # Pre-compression transformation filters
│   ├── bcj/             # Branch Call Jump filter (x86, ARM, ARM64, PowerPC, SPARC)
│   ├── delta/           # Numerical delta sequence filter
│   └── shuffle/         # Byte-transposition shuffle filter
│
├── json/                # SIMD-accelerated zero-allocation pure-Go JSON encoder/decoder
└── entropy.go           # Shannon entropy calculation & compressibility heuristics
```

## Core Capabilities

1. **Unified Decompression (`compress.Decompress`)**: Automatically transparently decodes `gzip`, `br`, `zstd`, `deflate`, `lz4`, `lzma` streams into pre-allocated destination slices.
2. **Decompression Bomb Protection**: Enforces maximum uncompressed payload thresholds and amplification limits (up to 250x) to prevent resource exhaustion attacks (`ErrDecompressionBomb`).
3. **Per-P Storage Allocation**: Uses thread-local CPU storage (`pool.PerPStorage`) for zstd, gzip, and deflate decoders, avoiding lock contention and GC overhead.
4. **Dictionary Compression**: Supports shared dictionary compression for Brotli (`dcb`) and Zstd (`dcz`).
5. **Shannon Entropy Estimation (`codec.ShannonEntropy`)**: Detects dense encrypted/compressed payloads (> 7.92 bits/byte) to skip redundant compression attempts.

## Key APIs & Usage

### 1. Transparent Decompression with Safety Limits

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/codec/compress"
)

func main() {
    compressedData := []byte{...}
    
    // Decompress payload (auto-detects gzip, brotli, zstd, deflate)
    decompressed, err := compress.Decompress(compressedData, compress.EncodingZstd, 10*1024*1024)
    if err != nil {
        fmt.Printf("Decompression error: %v\n", err)
        return
    }

    fmt.Printf("Decompressed %d bytes\n", len(decompressed))
}
```

### 2. Shannon Entropy Estimation

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/codec"
)

func main() {
    payload := []byte("plain text sample repeating patterns")
    
    entropy := codec.ShannonEntropy(payload)
    incompressible := codec.IsIncompressibleSample(payload)

    fmt.Printf("Entropy: %.2f bits/byte (Incompressible: %v)\n", entropy, incompressible)
}
```
