// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cpukit provides unified hardware feature detection and CPU capability
// probes for x86 and ARM microarchitectures.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: exported CPU feature detection.
//
// Rejected compromise: The standard library hides CPU feature detection in an internal package, forcing developers to rely on third-party libraries for hardware-optimized fallbacks.
//
// Accepted cost: This package must be manually updated to support new CPU features and architectures.
//
// Allocations: zero
package cpukit
