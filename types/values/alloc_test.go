//go:build !race

package values_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/types/values"
)

func TestValues_Allocations(t *testing.T) {
	data := []byte(`"12345"`)
	var v values.Uint64String
	allocs := testing.AllocsPerRun(100, func() {
		_ = v.UnmarshalJSON(data)
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
