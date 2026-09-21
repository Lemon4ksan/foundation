// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package secret provides protected in-memory containers for sensitive authentication material,
// preventing accidental exposure across logging pipelines, stack traces, and serialized outputs.
//
// # Example
//
//	package main
//
//	import (
//		"fmt"
//		"log/slog"
//
//		"github.com/lemon4ksan/foundation/types/secret"
//	)
//
//	func main() {
//		apiKey := secret.New("sk_live_983274982374982374")
//
//		// Formatted printing masks the value
//		fmt.Println("Key:", apiKey) // Output: Key: [REDACTED]
//
//		// Structured logging masks the value
//		slog.Info("Authenticating client", "api_key", apiKey)
//
//		// Explicit extraction for authorized network requests
//		rawKey := apiKey.Value()
//		_ = rawKey
//	}
package secret
