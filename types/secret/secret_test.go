// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secret_test

import (
	"encoding"
	"encoding/json"
	"fmt"
	"sync"
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

// Interface compliance assertions
var (
	_ encoding.TextMarshaler     = secret.Secret[string]{}
	_ encoding.TextUnmarshaler   = (*secret.Secret[string])(nil)
	_ encoding.BinaryMarshaler   = secret.Secret[string]{}
	_ encoding.BinaryUnmarshaler = (*secret.Secret[string])(nil)
)

func TestRepro_Secret_SerializationLeakPrevention(t *testing.T) {
	sec := secret.New("top-secret-api-key")

	// Text marshaler must mask sensitive value
	text, err := sec.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "******", string(text))

	// Binary marshaler must mask sensitive value
	bin, err := sec.MarshalBinary()
	require.NoError(t, err)
	assert.Equal(t, "******", string(bin))

	// Text unmarshaler
	var unmarshaledText secret.Secret[string]
	err = unmarshaledText.UnmarshalText([]byte("recovered-secret-text"))
	require.NoError(t, err)
	assert.Equal(t, "recovered-secret-text", unmarshaledText.Value())

	// Binary unmarshaler
	var unmarshaledBin secret.Secret[string]
	err = unmarshaledBin.UnmarshalBinary([]byte("recovered-secret-bin"))
	require.NoError(t, err)
	assert.Equal(t, "recovered-secret-bin", unmarshaledBin.Value())
}

func TestRepro_Secret_ConcurrentExposeAndWipeRace(t *testing.T) {
	for iter := 0; iter < 100; iter++ {
		sec := secret.New([]byte("concurrent-wipe-expose-secret"))
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				_ = sec.Expose()
			}
		}()

		go func() {
			defer wg.Done()
			sec.Wipe()
		}()

		wg.Wait()
	}
}
