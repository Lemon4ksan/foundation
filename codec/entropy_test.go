// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package codec_test

import (
	"crypto/rand"
	"testing"

	"github.com/lemon4ksan/foundation/codec"
)

func TestShannonEntropy(t *testing.T) {
	// Zero entropy
	zero := make([]byte, 1000)
	if e := codec.ShannonEntropy(zero); e != 0.0 {
		t.Errorf("expected 0 entropy for uniform zero, got %f", e)
	}

	// High entropy (random data)
	random := make([]byte, 4096)
	_, _ = rand.Read(random)
	if e := codec.ShannonEntropy(random); e < 7.8 {
		t.Errorf("expected random entropy > 7.8, got %f", e)
	}
	if !codec.IsIncompressibleSample(random) {
		t.Errorf("expected random data to be detected as incompressible")
	}

	// Low entropy (text sample)
	text := []byte("hello world this is a repetitive text sequence with low entropy")
	if codec.IsIncompressibleSample(text) {
		t.Errorf("expected text sample to not be incompressible")
	}
}
