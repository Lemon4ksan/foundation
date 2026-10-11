// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/codec/json"
)

type Config struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
}

func ExampleMarshal() {
	cfg := Config{Name: "gateway", Version: 2}

	data, _ := json.Marshal(cfg)
	fmt.Printf("%s\n", data)

	var decoded Config
	_ = json.Unmarshal(data, &decoded)
	fmt.Printf("Decoded Name: %s\n", decoded.Name)

	// Output:
	// {"name":"gateway","version":2}
	// Decoded Name: gateway
}

func ExampleObjectEntries() {
	data := []byte(`{"event":"connect", "id": 42, "metadata": {"region": "us-east"}}`)

	// ObjectEntries allows 0-allocation iteration over JSON keys and values.
	for key, val := range json.ObjectEntries(data) {
		fmt.Printf("%s: %s\n", key, val)
	}

	// Output:
	// event: "connect"
	// id: 42
	// metadata: {"region": "us-east"}
}
