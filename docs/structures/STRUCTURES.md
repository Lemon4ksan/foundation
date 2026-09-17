# Data Structures (`structures/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/structures)

`structures/` contains specialized, high-performance, and zero-allocation data structures leveraging Go 1.18+ generics.

## 1. Packages

### `structures/minheap`

A generic, contiguous array-backed min-heap implementation. Useful for priority queues, timers, and nearest-neighbor search.

### `structures/ringbuffer`

A fast, bounded ring buffer for generic types. Does not perform locking internally, making it suitable for single-threaded usage or integration with custom lock-free synchronization.

### `structures/linkedlist`

A doubly linked list implemented using generics, providing `O(1)` insertions, deletions, and front/back operations.

## 2. Best Practices

Most structures in this namespace are designed to be instantiated directly or embedded in parent structs to minimize pointer indirection and GC tracking overhead.
