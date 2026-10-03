// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kdf provides standard Key Derivation Functions including HKDF, PBKDF2, and Argon2id.
//
// kdf wraps robust cryptographic derivation algorithms, offering predefined security profiles tailored to different memory and timing budgets. It facilitates deterministic domain-separated subkey and nonce derivation for extended cryptographic hierarchies.
//
// See [Argon2id], [PBKDF2], and [DeriveKey] for primary usage.
//
// # Compared to the standard library
//
// Stdlib counterpart: golang.org/x/crypto (argon2, pbkdf2, hkdf)
//
// Rejected compromise: golang.org/x/crypto requires callers to manually choose tuning parameters, leading to weak or mismatched parameter configurations.
//
// Accepted cost: Constrains users to predefined profiles, potentially limiting extreme or non-standard customization needs.
//
// Allocations: amortized
package kdf
