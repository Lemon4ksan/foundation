# Generational Borrow Checker & Arenas (`borrow/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/borrow)

`borrow/` brings Rust-like memory safety principles to Go, providing generational arenas (`Scope`) and strictly verified immutable (`Ref[T]`) and mutable (`Mut[T]`) references. It allows you to safely recycle memory without risking Use-After-Free (UAF) or concurrent mutation data races.

## 1. Core Concepts

* **`Scope`**: A lexical arena. Memory allocated from a scope is contiguous and released automatically when the scope exits. Zero heap allocations.
* **`Slot`**: Tracks the generation and active borrows of a resource.
* **`Ref[T]`**: A shared, read-only reference. Multiple readers can hold `Ref[T]` concurrently.
* **`Mut[T]`**: An exclusive, mutable reference. Only one writer can hold a `Mut[T]`.
* **`Box[T]`**: An owned resource that can be lent out as `Ref` or `Mut`.

## 2. Safety Invariants

If a generation expires (i.e. the memory was recycled or the scope exited) or a borrow conflict occurs (e.g. concurrent mutation), the system will either return a specific error (`ErrAlreadyBorrowedMut`, `ErrExpiredGeneration`) or panic immediately to prevent memory corruption.

```go
borrow.Scoped(func(s *borrow.Scope) (string, error) {
    // Allocates an owned box inside the scope
    box := borrow.Alloc[int](s)
    
    mut := box.BorrowMut()
    mut.Write(42)
    mut.Release()
    
    return "", nil
})
```
