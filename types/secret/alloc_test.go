//go:build !race

package secret_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/types/secret"
)

func TestSecret_Allocations(t *testing.T) {
	s := secret.New("my-super-secret-key")
	allocs := testing.AllocsPerRun(100, func() {
		_ = s.Value()
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}
