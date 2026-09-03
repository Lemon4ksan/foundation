// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/lemon4ksan/foundation/tuikit"
)

func TestProbeTerminal_NilAndBuffer(t *testing.T) {
	if tuikit.ProbeTerminal(nil) {
		t.Errorf("expected ProbeTerminal(nil) to be false")
	}

	var buf bytes.Buffer
	if tuikit.ProbeTerminal(&buf) {
		t.Errorf("expected ProbeTerminal(bytes.Buffer) to be false")
	}

	if tuikit.IsInteractive(&buf) {
		t.Errorf("expected IsInteractive(bytes.Buffer) to be false")
	}
}

func TestProbeTerminal_Pipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if tuikit.ProbeTerminal(r) {
		t.Errorf("expected pipe read end to NOT be a terminal")
	}

	if tuikit.ProbeTerminal(w) {
		t.Errorf("expected pipe write end to NOT be a terminal")
	}

	if tuikit.IsInteractive(w) {
		t.Errorf("expected pipe write end to NOT be interactive")
	}
}
