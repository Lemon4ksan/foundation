// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
	"testing"
)

func TestFuzzTargetsConfiguration(t *testing.T) {
	if len(targets) == 0 {
		t.Fatal("expected targets list to be populated")
	}

	seen := make(map[string]bool, len(targets))
	for _, tgt := range targets {
		if tgt.pkg == "" {
			t.Errorf("empty package in fuzz target: %+v", tgt)
		}
		if !strings.HasPrefix(tgt.pkg, "./") {
			t.Errorf("fuzz target package %q should start with './'", tgt.pkg)
		}
		if tgt.name == "" {
			t.Errorf("empty name in fuzz target: %+v", tgt)
		}
		if !strings.HasPrefix(tgt.name, "Fuzz") {
			t.Errorf("fuzz target name %q should start with 'Fuzz'", tgt.name)
		}
		key := tgt.pkg + "::" + tgt.name
		if seen[key] {
			t.Errorf("duplicate fuzz target configured: %s", key)
		}
		seen[key] = true
	}
}
