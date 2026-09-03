// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/tuikit"
)

func TestFlags_Clumping(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var (
		l bool
		a bool
		A bool
		F bool
	)

	tuikit.BoolVar(fs, &l, "long", "l", false, "long format")
	tuikit.BoolVar(fs, &a, "all", "a", false, "all files")
	tuikit.BoolVar(fs, &A, "almost-all", "A", false, "almost all files")
	tuikit.BoolVar(fs, &F, "classify", "F", false, "classify")

	// Test -la
	args := []string{"-la", "target"}
	pos, err := tuikit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("ParseInterspersedFlags failed: %v", err)
	}

	if !l || !a {
		t.Errorf("expected l=true, a=true; got l=%v, a=%v", l, a)
	}
	if len(pos) != 1 || pos[0] != "target" {
		t.Errorf("expected pos [target], got %v", pos)
	}

	// Test -lAF
	l, A, F = false, false, false
	args = []string{"-lAF"}
	_, err = tuikit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("ParseInterspersedFlags -lAF failed: %v", err)
	}
	if !l || !A || !F {
		t.Errorf("expected l=true, A=true, F=true; got l=%v, A=%v, F=%v", l, A, F)
	}
}

func TestFlags_AttachedValue(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var (
		ignore string
		sort   string
	)

	tuikit.StringVar(fs, &ignore, "ignore", "I", "", "ignore pattern")
	tuikit.StringVar(fs, &sort, "sort", "s", "", "sort key")

	args := []string{"-I*.tmp", "-sname", "file.txt"}
	pos, err := tuikit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("ParseInterspersedFlags attached value failed: %v", err)
	}

	if ignore != "*.tmp" {
		t.Errorf("expected ignore=*.tmp, got %q", ignore)
	}
	if sort != "name" {
		t.Errorf("expected sort=name, got %q", sort)
	}
	if len(pos) != 1 || pos[0] != "file.txt" {
		t.Errorf("expected pos [file.txt], got %v", pos)
	}
}

func TestFlags_ClumpWithAttachedValue(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var (
		l      bool
		a      bool
		ignore string
	)

	tuikit.BoolVar(fs, &l, "long", "l", false, "long format")
	tuikit.BoolVar(fs, &a, "all", "a", false, "all files")
	tuikit.StringVar(fs, &ignore, "ignore", "I", "", "ignore pattern")

	args := []string{"-laI*.log", "targetDir"}
	pos, err := tuikit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("ParseInterspersedFlags clump+attached failed: %v", err)
	}

	if !l || !a {
		t.Errorf("expected l=true, a=true; got l=%v, a=%v", l, a)
	}
	if ignore != "*.log" {
		t.Errorf("expected ignore=*.log, got %q", ignore)
	}
	if len(pos) != 1 || pos[0] != "targetDir" {
		t.Errorf("expected pos [targetDir], got %v", pos)
	}
}

func TestFlags_TypoSuggestion(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var where string
	tuikit.StringVar(fs, &where, "where", "w", "", "where expression")

	args := []string{"--whre", "foo"}
	_, err := tuikit.ParseInterspersedFlags(fs, args)
	if err == nil {
		t.Fatalf("expected error on mistyped flag, got nil")
	}

	if !strings.Contains(err.Error(), "did you mean: --where?") {
		t.Errorf("expected typo suggestion for --where in error, got: %v", err)
	}
}
