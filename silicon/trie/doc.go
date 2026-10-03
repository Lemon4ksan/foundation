// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package trie provides zero-allocation radix and prefix search trees for high-speed
// URL routing, header matching, and domain wildcard lookups.
//
// Standard hash maps only support exact key lookups. Evaluating wildcard domain routes
// or longest URL path prefixes requires linear scans across all registered routes,
// taking O(N) time. Matching routes with regular expressions introduces heavy CPU
// overhead and dynamic memory allocations. Compressed radix search trees provide
// deterministic O(K) lookups where K is the path length, with zero heap allocations.
//
// # Compared to the standard library
//
// Stdlib counterpart: none - fills gap: prefix and wildcard string matching.
//
// Rejected compromise: Linear scanning of all routes or regex-based matching for longest prefix lookups.
//
// Accepted cost: Trees are much larger in memory compared to simple hash maps and take longer to build.
//
// Allocations: amortized
package trie
