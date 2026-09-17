# Zero-Allocation Binary Encoding & Serialization (`encoding/bin`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/encoding/bin)

`encoding/bin` provides zero-allocation sequential binary encoding, decoding, fluent cursor I/O with sticky error handling, and JIT-cached struct serialization.

## Core Capabilities

1. **Fluent Sequential Reader (`encoding/bin.Reader`)**: Decodes primitive types (`U8`, `U16BE/LE`, `U32BE/LE`, `U64BE/LE`, `I8`, `I16BE/LE`, `I32BE/LE`, `I64BE/LE`, `Varint`, `Uvarint`, `Bytes`, `FixedBytes`, `String`) with inline bounds checking and sticky error propagation (`r.Err()`).
2. **Buffer Writer (`encoding/bin.Writer`)**: Sequential binary byte packing into pre-allocated slices without heap reallocation.
3. **JIT Struct Marshalling (`encoding/bin.MarshalStruct` / `encoding/bin.UnmarshalStruct`)**: Analyzes struct layouts once into bytecode offsets, performing direct memory writes via standard ABI-safe pointers without runtime reflection overhead on subsequent calls.
4. **Endianness Tags**: Supports struct field tags `encoding/bin:"be"`, `encoding/bin:"le"`, and `encoding/bin:"-"` (skip).

## Key APIs & Usage

### 1. Sequential Cursor Reader with Sticky Errors

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/encoding/bin"
)

func decodePacket(payload []byte) error {
    r := encoding/bin.NewReader(payload)

    magic := r.U16BE()
    version := r.U8()
    length := r.U32BE()
    body := r.Bytes(int(length))

    if err := r.Err(); err != nil {
        return fmt.Errorf("malformed packet: %w", err)
    }

    fmt.Printf("Magic: %x, Version: %d, Length: %d, Data: %d bytes\n", magic, version, length, len(body))
    return nil
}
```

### 2. JIT-Cached Struct Serialization

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/encoding/bin"
)

type FrameHeader struct {
    Magic    uint16 `encoding/bin:"be"`
    StreamID uint32 `encoding/bin:"be"`
    Length   uint32 `encoding/bin:"be"`
    Flags    uint8  `encoding/bin:"be"`
}

func main() {
    hdr := FrameHeader{
        Magic:    0xABCD,
        StreamID: 101,
        Length:   512,
        Flags:    0x01,
    }

    // Zero-alloc struct encoding
    buf, _ := encoding/bin.MarshalStruct(&hdr)

    // Decode directly into target struct
    var decoded FrameHeader
    _ = encoding/bin.UnmarshalStruct(buf, &decoded)

    fmt.Printf("Decoded: %+v\n", decoded)
}
```
