# High-Performance Streaming I/O (`io/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/io)

`io/` provides low-level streaming I/O wrappers, replayable body decorators, multi-read buffers, and pooled zero-allocation copy helpers designed for line-rate data processing.

## 1. Architectural Components

```text
foundation/io/
├── io.go            // Replayable body buffers, copy pools, stream limits, tee readers
├── bytes_reader.go  // High-speed, allocation-free slice reader
└── io_test.go       // Unit & stress benchmark suite
```

## 2. Core Capabilities

### A. Replayable Multi-Read Body (`MultiReadBody`)
Allows consuming an `io.ReadCloser` stream multiple times (e.g. for inspection, hashing, and retry) with automatic spillover to off-heap memory or disk when payload exceeds memory thresholds.

```go
// Wrap incoming request/response body:
replayable := fio.NewReplayableBody(resp.Body, fio.ReplayConfig{
    MaxRAMBytes: 2 * 1024 * 1024, // 2 MB in RAM
    AllowDisk:   true,             // Spill over to temporary disk file if > 2 MB
})

// Read 1st time (e.g. calculating checksum):
hash := sha256.Sum256(fio.ReadAll(replayable))

// Rewind and read 2nd time (e.g. unmarshaling JSON):
replayable.Rewind()
var payload DataPayload
json.NewDecoder(replayable).Decode(&payload)
```

### B. High-Speed Zero-Allocation `BytesReader`
Optimized byte stream reader that avoids pointer conversions and allocations in high-frequency decoding loops.

```go
reader := fio.NewBytesReader(rawBytes)
buf := make([]byte, 1024)
n, err := reader.Read(buf)
```

### C. Pooled Stream Copy with Limit (`LimitToContentLength`)
Guarantees that socket readers do not read past the declared `Content-Length`, protecting Keep-Alive connections from trailing garbage without allocating `io.LimitReader` structs.

```go
safeReader := fio.LimitToContentLength(resp.Body, resp.ContentLength)
```

## 3. Performance Characteristics

| Operation | Standard `io` / `bytes` | `foundation/io` | Performance Delta |
| :--- | :---: | :---: | :---: |
| **Stream Copy (32 KB buffer)** | `io.Copy` (1 alloc) | `fio.Copy` (Pooled buffer) | **0 allocs / 0 B/op** |
| **Multi-Read Rewind** | Re-allocation (`bytes.Clone`) | `fio.ReplayableBody` | **Zero-Copy memory reuse** |
| **Length-Capped Read** | `io.LimitReader` (1 alloc) | `fio.LimitToContentLength` | **Inline value reuse** |
