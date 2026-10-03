package uuid_test

import (
	"fmt"

	"github.com/lemon4ksan/foundation/types/uuid"
)

func ExampleUUID() {
	// Parse existing UUID string with zero allocations
	parsed := uuid.MustParse("018e3a2b-7c4d-7000-8000-000000000000")
	fmt.Println("Parsed Version:", parsed.Version())

	// Format directly into existing buffer
	var buf [uuid.StringLength]byte
	parsed.Format(&buf)
	fmt.Println("Formatted:", string(buf[:]))

	// Output:
	// Parsed Version: 7
	// Formatted: 018e3a2b-7c4d-7000-8000-000000000000
}
