# High-Performance Memory Buffers (`bufkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/bufkit)

`bufkit` provides zero-allocation, cacheline-aligned memory buffer primitives, chunked scatter-gather chains, and lock-free ring buffers designed for high-throughput networking, streaming, and I/O pipelines.

## Motivation & Architecture

Traditional memory buffers in Go rely on continuous reallocation when growing dynamic payloads (HTTP/2 frames, WebSocket payloads, RPC packets). Resizing triggers heap copies and GC churn.

`bufkit` eliminates this overhead through three foundational primitives:

1. **`AlignedBuffer`**: Memory buffers aligned to CPU cache lines (64 bytes) or pages (4096 bytes) for vector/SIMD operations and DMA controllers.
2. **`BufferChain`**: Scatter-gather chunked buffer chaining fixed-size pooled memory blocks, supporting zero-copy append, slicing, and multi-chunk streaming reads.
3. **`RingBuffer`**: Lock-free circular buffer optimized for high-throughput single-producer single-consumer (SPSC) pipelines.

## Key APIs & Usage

### 1. Cacheline-Aligned Buffers (`bufkit.AlignedBuffer`)

```go
package main

import (
    "github.com/lemon4ksan/foundation/bufkit"
)

func main() {
    // Allocate 4KB buffer aligned to 64-byte CPU cache line boundary
    buf := bufkit.NewAlignedBuffer(4096, 64)
    defer buf.Free()

    slice := buf.Bytes()
    // Perform SIMD / AVX2 vector operations directly on aligned memory
    _ = slice
}
```

### 2. Scatter-Gather Buffer Chain (`bufkit.BufferChain`)

```go
package main

import (
    "github.com/lemon4ksan/foundation/bufkit"
)

func main() {
    chain := bufkit.NewBufferChain(4096)
    defer chain.Release()

    chain.WriteString("HTTP/1.1 200 OK\r\n")
    chain.WriteString("Content-Length: 1024\r\n\r\n")

    // Read directly into destination without continuous reallocations
    var dest [512]byte
    n, _ := chain.Read(dest[:])
    _ = n
}
```

### 3. SPSC Ring Buffer (`bufkit.RingBuffer`)

```go
package main

import (
    "github.com/lemon4ksan/foundation/bufkit"
)

func main() {
    ring := bufkit.NewRingBuffer(64 * 1024)

    // Producer writes bytes
    ring.Write([]byte("streaming frame payload"))

    // Consumer reads bytes
    var out [100]byte
    n, _ := ring.Read(out[:])
    _ = n
}
```
