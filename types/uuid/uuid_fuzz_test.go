// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uuid_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/types/uuid"
)

func FuzzUUIDParse(f *testing.F) {
	f.Add("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Add("ffffffff-ffff-ffff-ffff-ffffffffffff")
	f.Add("6BA7B810-9DAD-11D1-80B4-00C04FD430C8")
	f.Add("invalid-uuid-string-length-not-36")
	f.Add("6ba7b8109dad11d180b400c04fd430c8")
	f.Add("")

	f.Fuzz(func(t *testing.T, s string) {
		u, err := uuid.Parse(s)
		isValid := uuid.IsValid(s)

		if (err == nil) != isValid {
			t.Fatalf("IsValid and Parse discrepancy on %q: IsValid=%v, ParseErr=%v", s, isValid, err)
		}

		if err == nil {
			str := u.String()
			reparsed, err2 := uuid.Parse(str)
			if err2 != nil {
				t.Fatalf("failed to re-parse formatted UUID %q: %v", str, err2)
			}
			if u != reparsed {
				t.Fatalf("roundtrip mismatch: orig=%v, reparsed=%v", u, reparsed)
			}
		}
	})
}
