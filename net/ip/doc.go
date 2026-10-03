// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ip provides lock-free IP address rotation and IPv6 subnet generation.
//
// The package supports pooling local IP addresses for source selection and generating
// randomized IPv6 addresses within a specified prefix. [SourceIPRotator] uses atomic
// pointers and copy-on-write semantics to maintain a pool of IPs, ensuring that
// address selection via [SourceIPRotator.Next] and [SourceIPRotator.NextOK] is completely
// lock-free and contention-free under parallel execution. It also offers discovery of
// host network interfaces.
//
// # Compared to the standard library
//
// Stdlib counterpart: net
//
// Rejected compromise: standard library dialers force external synchronization or duplicated states to rotate source IPs.
//
// Accepted cost: callers must explicitly manage the rotator lifecycle and handle potential address exhaustion.
//
// Allocations: bounded(1/op)
package ip
