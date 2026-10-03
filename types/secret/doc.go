// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package secret provides protected in-memory containers for sensitive authentication material.
//
// The [Secret] container wraps sensitive data (passwords, bearer tokens, API keys) to prevent
// accidental exposure across logging pipelines, stack traces, and serialized outputs.
// It leverages the standard library's encoding/json to ensure secrets remain masked
// during JSON serialization. The raw sensitive value can only be retrieved via explicit calls
// to [Secret.Value] or [Secret.Expose] when strictly necessary for authorized operations.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: protected in-memory containers for sensitive material
//
// Rejected compromise: exposing raw strings in standard library structures that accidentally serialize to logs or JSON endpoints
//
// Accepted cost: manual explicit extraction of the raw material when it is actually needed
//
// Allocations: zero
package secret
