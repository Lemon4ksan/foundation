// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/lemon4ksan/foundation/fskit"
)

func TestFastWalk(t *testing.T) {
	tempDir := t.TempDir()

	// Create test structure
	os.MkdirAll(filepath.Join(tempDir, "a", "b"), 0o755)
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(tempDir, "a", "file2.txt"), []byte("world"), 0o644)
	os.WriteFile(filepath.Join(tempDir, "a", "b", "file3.txt"), []byte("!"), 0o644)

	found := make(map[string]bool)
	err := fskit.FastWalk(tempDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			rel, _ := filepath.Rel(tempDir, path)
			found[filepath.ToSlash(rel)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FastWalk failed: %v", err)
	}

	expected := []string{"file1.txt", "a/file2.txt", "a/b/file3.txt"}
	for _, exp := range expected {
		if !found[exp] {
			t.Errorf("expected to find %s, but missing", exp)
		}
	}
}
