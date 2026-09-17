# Text Processing (`text/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/text)

`text/` provides high-performance text parsing, encoding, and string transformation tools. 

## 1. Sub-Packages

* **`text/casing`**: Zero-allocation string case converters. Convert between CamelCase, snake_case, PascalCase, kebab-case, and screaming variations efficiently.
* **`text/diff`**: Line and character-level differential algorithms to compute edits, patches, and unified diffs.
* **`text/encoding`**: Multi-language charset encoders and decoders (e.g. charmap, Japanese, Korean, Unicode).
* **`text/extract`**: Fast text extraction primitives.
* **`text/htmlkit`**: Fast, safe HTML entity decoding, sanitization, and tokenization.
* **`text/transform`**: Extensible transformer pipeline for bulk streaming text mutations.

## 2. Zero-Allocation Focus

Like the rest of `foundation`, the `text` utilities rely on heavily optimized loops and buffer recycling. Conversions like `casing.ToSnake` can be executed strictly into pre-allocated `bufkit` byte arrays.
