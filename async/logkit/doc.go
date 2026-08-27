// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logkit provides a high-performance, asynchronous, structured logger
// designed for both human readability and machine efficiency.
//
// The logger uses a non-blocking architecture where logkit messages are formatted
// in the calling goroutine and then sent to a background worker via a fixed-size
// buffer (channel). This ensures that logging operations have minimal impact
// on the latency of the main application logic.
//
// # Key Components
//
//   - [Logger]: The primary interface that defines all structured logging methods.
//   - [Field]: A key-value pair used to represent structured context.
//   - [Config]: Controls the output destination, severity level, and visual style of the logger.
//   - [AsyncLogger]: The default thread-safe, non-blocking implementation of the [Logger] interface.
//
// # Basic Usage (Asynchronous Text Logger)
//
//	package main
//
//	import (
//		"github.com/lemon4ksan/foundation/async/logkit"
//	)
//
//	func main() {
//		// Initialize default configuration
//		cfg := logkit.DefaultConfig(logkit.LevelInfo)
//		logger := logkit.New(cfg)
//		defer logger.Close()
//
//		// logkit messages with structured context
//		logger.Info("user logged in",
//			logkit.String("username", "john_doe"),
//			logkit.Int("attempts", 3),
//		)
//	}
//
// # Context-Aware Tracing (Correlation ID)
//
// The logging package provides built-in support for context-aware request tracing
// and transaction correlation. By generating a unique correlation ID and embedding it
// into a [context.Context], any downstream logging method called with that context
// (e.g., [Logger.InfoContext], [Logger.ErrorContext]) will automatically extract
// and append the ID as a structured "correlation_id" field in the output.
//
//	package main
//
//	import (
//		"context"
//		"github.com/lemon4ksan/foundation/async/logkit"
//	)
//
//	func handleRequest(ctx context.Context, logger logkit.Logger) {
//		// Generate a new secure 16-byte hex correlation ID and inject it into the context
//		reqCtx := logkit.WithCorrelationID(ctx, logkit.GenerateCorrelationID())
//
//		// Logging context-aware message: the "correlation_id" is appended automatically
//		logger.InfoContext(reqCtx, "processing user transaction",
//			logkit.String("action", "transfer_funds"),
//		)
//	}
package logkit
