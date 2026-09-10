// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package borrow

import (
	"testing"
)

func TestRepro_Slot_UnderflowAllowsConcurrentReadWrite(t *testing.T) {
	slot := NewSlot()

	// Stray or unbalanced ReleaseShared must NOT underflow the readers count
	slot.ReleaseShared()

	// An active reader acquires a shared borrow
	if err := slot.TryAcquireShared(slot.Generation()); err != nil {
		t.Fatalf("unexpected error acquiring shared borrow: %v", err)
	}

	// While the reader is active, exclusive mutable borrow MUST fail with ErrAlreadyBorrowedShared
	err := slot.TryAcquireExclusive(slot.Generation())
	if err == nil {
		t.Fatal(
			"CRITICAL INVARIANT VIOLATION: Exclusive borrow acquired while reader is active due to readers underflow!",
		)
	}
	if err != ErrAlreadyBorrowedShared {
		t.Fatalf("expected ErrAlreadyBorrowedShared, got %v", err)
	}
}
