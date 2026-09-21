// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package internal_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding/internal"
	"github.com/lemon4ksan/foundation/text/encoding/internal/identifier"
	"github.com/lemon4ksan/foundation/text/transform"
)

func TestEncoding(t *testing.T) {
	enc := &internal.Encoding{
		Name: "test-enc",
		MIB:  identifier.MIB(1234),
	}

	if enc.String() != "test-enc" {
		t.Fatalf("enc.String() = %q, want test-enc", enc.String())
	}

	mib, other := enc.ID()
	if mib != 1234 || other != "" {
		t.Fatalf("enc.ID() = (%v, %q), want (1234, \"\")", mib, other)
	}
}

func TestSimpleEncoding(t *testing.T) {
	simple := &internal.SimpleEncoding{
		Decoder: transform.Nop,
		Encoder: transform.Nop,
	}

	dec := simple.NewDecoder()
	if dec.Transformer != transform.Nop {
		t.Fatalf("expected transform.Nop decoder")
	}

	enc := simple.NewEncoder()
	if enc.Transformer != transform.Nop {
		t.Fatalf("expected transform.Nop encoder")
	}
}
