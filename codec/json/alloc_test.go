// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package json_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/json"
)

func TestJSONIter_Allocs(t *testing.T) {
	data := []byte(`{"id": 1, "name": "foo", "active": true}`)

	allocs := testing.AllocsPerRun(10, func() {
		for key, val := range json.ObjectEntries(data) {
			_ = key
			_ = val
		}
	})

	if allocs > 0 {
		t.Errorf("ObjectEntries allocated %v times, want 0", allocs)
	}
}
