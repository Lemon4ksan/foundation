// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package argkit_test

import (
	"flag"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/argkit"
)

func TestArgKit_Clumping(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var (
		l bool
		a bool
		A bool
		F bool
	)

	argkit.BoolVar(fs, &l, "long", "l", false, "long format")
	argkit.BoolVar(fs, &a, "all", "a", false, "all files")
	argkit.BoolVar(fs, &A, "almost-all", "A", false, "almost all files")
	argkit.BoolVar(fs, &F, "classify", "F", false, "classify")

	// Test -la
	args := []string{"-la", "target"}
	pos, err := argkit.ParseInterspersedFlags(fs, args)
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
	_, err = argkit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("ParseInterspersedFlags -lAF failed: %v", err)
	}
	if !l || !A || !F {
		t.Errorf("expected l=true, A=true, F=true; got l=%v, A=%v, F=%v", l, A, F)
	}
}

func TestArgKit_AttachedValue(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var (
		ignore string
		sort   string
	)

	argkit.StringVar(fs, &ignore, "ignore", "I", "", "ignore pattern")
	argkit.StringVar(fs, &sort, "sort", "s", "", "sort key")

	args := []string{"-I*.tmp", "-sname", "file.txt"}
	pos, err := argkit.ParseInterspersedFlags(fs, args)
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

func TestArgKit_ClumpWithAttachedValue(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var (
		l      bool
		a      bool
		ignore string
	)

	argkit.BoolVar(fs, &l, "long", "l", false, "long format")
	argkit.BoolVar(fs, &a, "all", "a", false, "all files")
	argkit.StringVar(fs, &ignore, "ignore", "I", "", "ignore pattern")

	args := []string{"-laI*.log", "targetDir"}
	pos, err := argkit.ParseInterspersedFlags(fs, args)
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

func TestArgKit_TypoSuggestion(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)

	var where string
	argkit.StringVar(fs, &where, "where", "w", "", "where expression")

	args := []string{"--whre", "foo"}
	_, err := argkit.ParseInterspersedFlags(fs, args)
	if err == nil {
		t.Fatalf("expected error on mistyped flag, got nil")
	}

	if !strings.Contains(err.Error(), "did you mean: --where?") {
		t.Errorf("expected typo suggestion for --where in error, got: %v", err)
	}
}
