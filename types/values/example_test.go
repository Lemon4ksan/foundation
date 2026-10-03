package values_test

import (
	"encoding/json"
	"fmt"

	"github.com/lemon4ksan/foundation/types/values"
)

type Payload struct {
	ID     values.Uint64String `json:"id"`
	Active values.BoolInt      `json:"active"`
}

func Example() {
	// Payload with string-encoded numeric and boolean values
	data := []byte(`{"id": "92837192847192", "active": "1"}`)

	var p Payload
	if err := json.Unmarshal(data, &p); err != nil {
		panic(err)
	}
	fmt.Printf("Parsed ID: %d, Active: %v\n", uint64(p.ID), bool(p.Active))

	// Output:
	// Parsed ID: 92837192847192, Active: true
}
