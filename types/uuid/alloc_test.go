// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !race

package uuid_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/types/uuid"
)

func TestUUID_Allocations(t *testing.T) {
	var buf [uuid.StringLength]byte
	parsed := uuid.MustParse("018e3a2b-7c4d-7000-8000-000000000000")

	allocs := testing.AllocsPerRun(100, func() {
		_ = parsed.Append(buf[:0])
	})
	if allocs > 0 {
		t.Errorf("expected 0 allocations for Append, got %v", allocs)
	}

	allocsParse := testing.AllocsPerRun(100, func() {
		_, _ = uuid.Parse("018e3a2b-7c4d-7000-8000-000000000000")
	})
	if allocsParse > 0 {
		t.Errorf("expected 0 allocations for Parse, got %v", allocsParse)
	}

	textBytes := []byte("018e3a2b-7c4d-7000-8000-000000000000")
	var uText uuid.UUID
	allocsUnmarshal := testing.AllocsPerRun(100, func() {
		_ = uText.UnmarshalText(textBytes)
	})
	if allocsUnmarshal > 0 {
		t.Errorf("expected 0 allocations for UnmarshalText, got %v", allocsUnmarshal)
	}

	allocsNilString := testing.AllocsPerRun(100, func() {
		_ = uuid.Nil.String()
	})
	if allocsNilString > 0 {
		t.Errorf("expected 0 allocations for Nil.String(), got %v", allocsNilString)
	}
}
