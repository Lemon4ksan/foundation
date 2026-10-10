// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secret_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
	"github.com/lemon4ksan/foundation/types/secret"
)

func TestSecret_Masking(t *testing.T) {
	t.Parallel()

	sec := secret.New("super-secret-api-token")

	// Raw value
	assert.Equal(t, "super-secret-api-token", sec.Value())
	assert.Equal(t, "super-secret-api-token", sec.Expose())

	// String & Printf masking
	assert.Equal(t, "******", sec.String())
	assert.Equal(t, "token: ******", fmt.Sprintf("token: %s", sec))
	assert.Equal(t, "token: ******", fmt.Sprintf("token: %v", sec))
	assert.Equal(t, "secret.Secret(******)", fmt.Sprintf("%#v", sec))

	// JSON masking
	marshaled, err := json.Marshal(sec)
	require.NoError(t, err)
	assert.Equal(t, `"******"`, string(marshaled))

	// JSON Unmarshal
	var unmarshaled secret.Secret[string]

	err = json.Unmarshal([]byte(`"new-secret"`), &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, "new-secret", unmarshaled.Value())
}

func TestSecret_Allocations(t *testing.T) {
	s := secret.New("my-super-secret-key")
	allocs := testing.AllocsPerRun(100, func() {
		_ = s.Value()
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations, got %v", allocs)
	}
}

func TestRepro_SecretMemoryWipeVerification(t *testing.T) {
	buf := []byte("highly-sensitive-signing-key")
	sec := secret.New(buf)

	assert.Equal(t, "highly-sensitive-signing-key", string(sec.Value()))

	sec.Wipe()

	// Verify the wrapped value was zeroed
	assert.Equal(t, 0, len(sec.Value()))

	// Verify the underlying memory was actively wiped with zeroes
	for i, b := range buf {
		if b != 0 {
			t.Fatalf("byte at index %d was not wiped: %v", i, b)
		}
	}

	// String wipe
	strSec := secret.New("secret-string")
	strSec.Wipe()
	assert.Equal(t, "", strSec.Value())

	// Zero allocations test
	s := secret.New([]byte("my-alloc-test-secret"))
	allocs := testing.AllocsPerRun(100, func() {
		s.Wipe()
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations for Wipe, got %v", allocs)
	}
}


