// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package values provides lenient scalar primitives for JSON, text, and SQL serialization,
// facilitating seamless type coercion across loosely typed APIs, microservices, and databases.
//
// # Architecture
//
// Modern distributed systems frequently encounter inconsistent JSON payload schemas where numeric,
// boolean, or temporal fields arrive alternatively as quoted strings (e.g., `"123"`, `"true"`, `"null"`)
// or native JSON literals (e.g., `123`, `true`). This package defines specialized scalar wrappers
// that implement [json.Marshaler], [json.Unmarshaler], and [fmt.Stringer]:
//   - [BoolString]: Lenient boolean accepting booleans (`true`/`false`), numeric strings (`"1"`/`"0"`),
//     textual representations (`"t"`, `"f"`, `"yes"`, `"no"`), and empty strings/nulls as false.
//   - [Int64String], [Uint64String], [Float64String]: Lenient numbers accepting raw numbers, quoted numbers,
//     and empty strings/nulls as zero values.
//   - [TimeString]: Flexible timestamp parser accepting RFC 3339 strings, ISO 8601 strings,
//     and Unix epoch millisecond/second numeric values.
//
// # Design Rationale
//
// Standard library unmarshaling strictly enforces JSON schema types. When third-party webhooks,
// legacy RPC systems, or frontend JavaScript clients serialize integers or booleans as strings,
// standard unmarshaling fails with unmarshal type errors. [values] solves this at the data structure level,
// eliminating manual string-to-number transformation boilerplate and minimizing intermediate allocations.
//
// # Concurrency Guarantees
//
// All scalar types in this package are value types. They are immutable once copied and fully safe
// for concurrent reads across multiple goroutines. Pointers to these types must not be modified
// concurrently without external synchronization.
//
// # Example
//
//	package main
//
//	import (
//		"encoding/json"
//		"fmt"
//
//		"github.com/lemon4ksan/foundation/types/values"
//	)
//
//	type Payload struct {
//		ID     values.Uint64String `json:"id"`
//		Active values.BoolString   `json:"active"`
//	}
//
//	func main() {
//		// Payload with string-encoded numeric and boolean values
//		data := []byte(`{"id": "92837192847192", "active": "1"}`)
//
//		var p Payload
//		if err := json.Unmarshal(data, &p); err != nil {
//			panic(err)
//		}
//		fmt.Printf("Parsed ID: %d, Active: %v\n", uint64(p.ID), bool(p.Active))
//	}
package values
