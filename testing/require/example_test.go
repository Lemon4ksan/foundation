package require_test

import (
	"errors"
	"testing"

	"github.com/lemon4ksan/foundation/testing/require"
)

func Example() {
	// In a real test, this would be a *testing.T passed by the test runner.
	var t *testing.T

	// Requirements terminate the test immediately on failure.
	err := errors.New("example error")

	// require.NoError(t, err) // Would halt execution
	_ = err
	require.NotNil(t, []string{"data"})
}
