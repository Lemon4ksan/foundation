# High-Performance Cryptography (`crypto/`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/crypto)

`crypto/` provides a suite of advanced cryptographic primitives, key management, and threshold schemes, built with safety and zero-allocation processing in mind.

## 1. Sub-Packages

* **`crypto/aead`**: Authenticated Encryption with Associated Data wrappers, providing streamlined symmetric encryption algorithms (AES-GCM, ChaCha20Poly1305).
* **`crypto/envelope`**: Envelope encryption implementation, facilitating secure data encryption using external Data Encryption Keys (DEK) and Key Encryption Keys (KEK).
* **`crypto/kdf`**: Key Derivation Functions (e.g. HKDF, Argon2, PBKDF2) optimized for memory hardness and speed.
* **`crypto/kms`**: Interfaces and utilities for interacting with Key Management Systems, standardizing key rotation, generation, and retrieval operations.
* **`crypto/shamir`**: Shamir's Secret Sharing implementation, allowing a secret to be split into N shares, with a threshold M required to reconstruct it.
* **`crypto/sign`**: Digital signature utilities supporting high-throughput signing and verification schemas (e.g. Ed25519, ECDSA).

## 2. Usage Philosophy

Where possible, routines in this module are optimized to reuse buffers via `bufkit` or `silicon` pooling to prevent Go GC pressure during high-throughput cryptographic operations. All APIs prioritize misuse-resistance.
