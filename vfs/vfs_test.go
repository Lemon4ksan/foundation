// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs_test

import (
	"path/filepath"
	"testing"

	"github.com/lemon4ksan/foundation/vfs"
)

func TestSafePath(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "target")

	// Valid path
	safe, err := vfs.SafePath(dest, "sub/file.txt")
	if err != nil {
		t.Fatalf("SafePath failed: %v", err)
	}
	if !filepath.IsAbs(safe) {
		t.Errorf("expected absolute path: %s", safe)
	}

	// Zip Slip attempt with ../
	_, err = vfs.SafePath(dest, "../../etc/passwd")
	if err == nil {
		t.Errorf("expected error on path traversal escape")
	}

	// Control character attempt
	_, err = vfs.SafePath(dest, "evil\x00file.txt")
	if err == nil {
		t.Errorf("expected error on control character filename")
	}
}

func TestTree_Rendering(t *testing.T) {
	entries := []struct {
		Name  string
		IsDir bool
	}{
		{"docs/readme.md", false},
		{"docs/guide.txt", false},
		{"src/main.go", false},
	}

	root := vfs.BuildTree(entries)
	rendered := vfs.RenderTree(root)

	if rendered == "" {
		t.Fatal("rendered tree is empty")
	}
}
