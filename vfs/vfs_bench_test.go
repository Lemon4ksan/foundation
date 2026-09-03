// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs_test

import (
	"path"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/pathkit"
	"github.com/lemon4ksan/foundation/vfs"
)

var samplePaths = []string{
	"docs/api/v1/users.json",
	`C:\Users\senya\projects\seal\internal\pool\pool.go`,
	"nested/../deep/./folder/file.txt",
	"archive/entries/data.tar.gz",
	"/var/log/application/prod.log",
}

// Legacy path cleaning pattern used in standard Go libraries
func legacyCleanPath(name string) string {
	clean := strings.TrimPrefix(path.Clean(strings.ReplaceAll(name, "\\", "/")), "/")
	if clean == "." {
		return ""
	}
	return clean
}

func BenchmarkCleanPath_Legacy(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, p := range samplePaths {
			_ = legacyCleanPath(p)
		}
	}
}

func BenchmarkCleanPath_Pathkit(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, p := range samplePaths {
			_ = vfs.CleanPath(p)
		}
	}
}

func BenchmarkSafePath_Pathkit(b *testing.B) {
	destDir := `C:\Users\senya\AppData\Local\Temp\extract_target`
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, p := range samplePaths {
			_, _ = vfs.SafePath(destDir, p)
		}
	}
}

func BenchmarkPathkit_Navigation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for _, p := range samplePaths {
			pk := pathkit.New(p)
			_ = pk.Base()
			_ = pk.Ext()
			_ = pk.Stem()
			_ = pk.Dir()
		}
	}
}
