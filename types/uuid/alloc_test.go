//go:build !race

package uuid_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/types/uuid"
)

func TestUUID_Allocations(t *testing.T) {
	var buf [uuid.StringLength]byte
	parsed := uuid.MustParse("018e3a2b-7c4d-7000-8000-000000000000")

	allocs := testing.AllocsPerRun(100, func() {
		_ = parsed.Append(buf[:0])
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations for Append, got %v", allocs)
	}

	allocsParse := testing.AllocsPerRun(100, func() {
		_, _ = uuid.Parse("018e3a2b-7c4d-7000-8000-000000000000")
	})
	if allocsParse > 0 {
		t.Errorf("expected 0 allocations for Parse, got %v", allocsParse)
	}
}
