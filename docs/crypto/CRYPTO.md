# High-Performance Cryptography (`crypto/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/crypto)

`crypto/` provides a suite of advanced cryptographic primitives, authenticated encryption, key derivation, and high-throughput signatures, built with safety and zero-allocation processing in mind.

## 1. Sub-Packages

* **`crypto/aead`**: Authenticated Encryption with Associated Data wrappers, providing streamlined symmetric encryption algorithms (AES-GCM, ChaCha20Poly1305).
* **`crypto/kdf`**: Key Derivation Functions (e.g. HKDF, Argon2, PBKDF2) optimized for memory hardness and speed.
* **`crypto/sign`**: Digital signature utilities supporting high-throughput signing and verification schemas (e.g. Ed25519, ECDSA).

## 2. Usage Philosophy

Where possible, routines in this module are optimized to reuse buffers via `bufkit` or `silicon` pooling to prevent Go GC pressure during high-throughput cryptographic operations. All APIs prioritize misuse-resistance.
