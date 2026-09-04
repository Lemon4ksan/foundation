# Zero-Allocation Binary Encoding & Serialization (`binkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/binkit)

`binkit` provides zero-allocation sequential binary encoding, decoding, fluent cursor I/O with sticky error handling, and JIT-cached struct serialization.

## Core Capabilities

1. **Fluent Sequential Reader (`binkit.Reader`)**: Decodes primitive types (`U8`, `U16BE/LE`, `U32BE/LE`, `U64BE/LE`, `I8`, `I16BE/LE`, `I32BE/LE`, `I64BE/LE`, `Varint`, `Uvarint`, `Bytes`, `FixedBytes`, `String`) with inline bounds checking and sticky error propagation (`r.Err()`).
2. **Buffer Writer (`binkit.Writer`)**: Sequential binary byte packing into pre-allocated slices without heap reallocation.
3. **JIT Struct Marshalling (`binkit.MarshalStruct` / `binkit.UnmarshalStruct`)**: Analyzes struct layouts once into bytecode offsets, performing direct memory writes via standard ABI-safe pointers without runtime reflection overhead on subsequent calls.
4. **Endianness Tags**: Supports struct field tags `binkit:"be"`, `binkit:"le"`, and `binkit:"-"` (skip).

## Key APIs & Usage

### 1. Sequential Cursor Reader with Sticky Errors

```go
package main

import (
    "fmt"

    "github.com/lemon4ksan/foundation/binkit"
)

func decodePacket(payload []byte) error {
    r := binkit.NewReader(payload)

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

    "github.com/lemon4ksan/foundation/binkit"
)

type FrameHeader struct {
    Magic    uint16 `binkit:"be"`
    StreamID uint32 `binkit:"be"`
    Length   uint32 `binkit:"be"`
    Flags    uint8  `binkit:"be"`
}

func main() {
    hdr := FrameHeader{
        Magic:    0xABCD,
        StreamID: 101,
        Length:   512,
        Flags:    0x01,
    }

    // Zero-alloc struct encoding
    buf, _ := binkit.MarshalStruct(&hdr)

    // Decode directly into target struct
    var decoded FrameHeader
    _ = binkit.UnmarshalStruct(buf, &decoded)

    fmt.Printf("Decoded: %+v\n", decoded)
}
```
