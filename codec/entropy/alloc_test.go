// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package entropy_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/entropy"
)

func TestAllocsPerRun_ShannonEntropy(t *testing.T) {
	sample := make([]byte, 1024)
	for i := range sample {
		sample[i] = byte(i)
	}

	allocs := testing.AllocsPerRun(100, func() {
		_ = entropy.ShannonEntropy(sample)
	})

	if allocs > 0 {
		t.Errorf("ShannonEntropy allocated %f times per run, want 0", allocs)
	}
}
