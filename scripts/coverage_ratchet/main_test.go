// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPkgStatsCoverage(t *testing.T) {
	stEmpty := &pkgStats{}
	if cov := stEmpty.Coverage(); cov != 100.0 {
		t.Fatalf("expected 100.0 for empty stats, got %f", cov)
	}

	st := &pkgStats{TotalStmts: 100, CoveredStmts: 85}
	if cov := st.Coverage(); cov != 85.0 {
		t.Fatalf("expected 85.0, got %f", cov)
	}
}

func TestParseBaselineAndWrite(t *testing.T) {
	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "baseline.txt")

	stats := map[string]*pkgStats{
		"github.com/lemon4ksan/foundation/sync/spinlock": {TotalStmts: 100, CoveredStmts: 30},
		"github.com/lemon4ksan/foundation/sync/keylock":  {TotalStmts: 50, CoveredStmts: 50},
	}

	if err := writeBaseline(baselinePath, stats); err != nil {
		t.Fatalf("failed to write baseline: %v", err)
	}

	baseMap, err := parseBaseline(baselinePath)
	if err != nil {
		t.Fatalf("failed to parse baseline: %v", err)
	}

	if len(baseMap) != 2 {
		t.Fatalf("expected 2 packages in baseline, got %d", len(baseMap))
	}

	if val := baseMap["github.com/lemon4ksan/foundation/sync/spinlock"]; val < 29.9 || val > 30.1 {
		t.Errorf("unexpected spinlock coverage: %f", val)
	}
	if val := baseMap["github.com/lemon4ksan/foundation/sync/keylock"]; val < 99.9 || val > 100.1 {
		t.Errorf("unexpected keylock coverage: %f", val)
	}
}

func TestParseCoverageProfile(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "coverage.out")

	content := `mode: set
github.com/lemon4ksan/foundation/sync/spinlock/spinlock.go:10.1,12.2 2 1
github.com/lemon4ksan/foundation/sync/spinlock/spinlock.go:14.1,16.2 3 0
`
	if err := os.WriteFile(profilePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write profile: %v", err)
	}

	stats, err := parseCoverageProfile(profilePath)
	if err != nil {
		t.Fatalf("failed to parse coverage profile: %v", err)
	}

	st, ok := stats["github.com/lemon4ksan/foundation/sync/spinlock"]
	if !ok {
		t.Fatalf("expected stats for sync/spinlock")
	}

	if st.TotalStmts != 5 {
		t.Errorf("expected 5 total statements, got %d", st.TotalStmts)
	}
	if st.CoveredStmts != 2 {
		t.Errorf("expected 2 covered statements, got %d", st.CoveredStmts)
	}
	if cov := st.Coverage(); cov != 40.0 {
		t.Errorf("expected 40.0%% coverage, got %f", cov)
	}
}
