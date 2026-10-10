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

func TestReproduce_ConcurrentUnmarshalRace(t *testing.T) {
	var s secret.Secret[string]
	
	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			_ = s.UnmarshalJSON([]byte(`"concurrent1"`))
		}
		done <- struct{}{}
	}()
	
	for i := 0; i < 1000; i++ {
		_ = s.UnmarshalJSON([]byte(`"concurrent2"`))
	}
	<-done
}
