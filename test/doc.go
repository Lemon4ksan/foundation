// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package test contains integration tests and comparative performance benchmarks.
//
// As a pure test package containing no production code, it exists solely to verify
// cross-package interactions and to benchmark foundation's performance against
// the Go standard library across various subsystems. It ensures that the overall
// performance objectives and integration invariants of the repository are met.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: repository-wide integration testing
//
// Rejected compromise: keeping all tests isolated within packages, missing cross-boundary issues
//
// Accepted cost: maintaining a dedicated package solely for testing purposes
//
// Allocations: unconstrained
package test
