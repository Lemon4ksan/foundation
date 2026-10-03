// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package aead provides authenticated encryption with associated data (AEAD) primitives.
//
// aead streamlines symmetric encryption algorithms such as AES-GCM and ChaCha20-Poly1305. It supports chunked streaming encryption, ensuring data authenticity and confidentiality over large or unbounded streams without buffering the entire payload.
//
// Use [New] to instantiate a cipher, and [EncryptChunks] for streaming.
//
// # Compared to the standard library
//
// Stdlib counterpart: crypto/cipher
//
// Rejected compromise: crypto/cipher lacks built-in streaming AEAD chunking, forcing consumers to implement their own complex and error-prone chunk framing.
//
// Accepted cost: Forces adherence to a specific chunked framing format which may not be compatible with non-conforming external receivers.
//
// Allocations: zero
package aead
