// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package matchfinder_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/brotli/matchfinder"
)

func TestM0MatchFinder(t *testing.T) {
	m := matchfinder.M0{}
	m.Reset()

	// Repeated pattern with high match probability
	pattern := []byte("The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog.")
	var matches []matchfinder.Match
	matches = m.FindMatches(matches, pattern)

	if len(matches) == 0 {
		t.Fatalf("expected matches for repeated pattern, got none")
	}

	for _, match := range matches {
		if match.Distance <= 0 && match.Length > 0 {
			t.Fatalf("invalid match distance: %d", match.Distance)
		}
	}
}

func TestM4MatchFinder(t *testing.T) {
	m := &matchfinder.M4{}
	m.Reset()

	data := bytes.Repeat([]byte("ABCDEFGHIJKLMN"), 10)
	var matches []matchfinder.Match
	matches = m.FindMatches(matches, data)

	if len(matches) == 0 {
		t.Fatalf("expected matches in repeated data, got none")
	}
}
