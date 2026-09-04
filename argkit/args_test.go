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

func TestArgKit_EdgeCases(t *testing.T) {
	fs := flag.NewFlagSet("test_edge", flag.ContinueOnError)

	var (
		v   bool
		out string
	)
	argkit.BoolVar(fs, &v, "verbose", "v", false, "verbose")
	argkit.StringVar(fs, &out, "output", "o", "", "output")

	// 1. Single dash should be treated as a positional argument (e.g. reading from stdin)
	args := []string{"-", "-v", "-"}
	pos, err := argkit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !v {
		t.Errorf("expected v=true")
	}
	if len(pos) != 2 || pos[0] != "-" || pos[1] != "-" {
		t.Errorf("expected positionals [\"-\", \"-\"], got %v", pos)
	}

	// 2. Double dash terminator stops flag parsing
	fs2 := flag.NewFlagSet("test_dd", flag.ContinueOnError)
	var v2 bool
	argkit.BoolVar(fs2, &v2, "verbose", "v", false, "verbose")

	args2 := []string{"--", "-v", "--other"}
	pos2, err := argkit.ParseInterspersedFlags(fs2, args2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v2 {
		t.Errorf("expected v2=false after --")
	}
	if len(pos2) != 2 || pos2[0] != "-v" || pos2[1] != "--other" {
		t.Errorf("expected positionals [\"-v\", \"--other\"], got %v", pos2)
	}
}

func TestArgKit_StringSliceFlag(t *testing.T) {
	var s argkit.StringSliceFlag
	if s.String() != "" {
		t.Errorf("expected empty string, got %q", s.String())
	}

	_ = s.Set("a")
	_ = s.Set("b")
	if s.String() != "a,b" {
		t.Errorf("expected \"a,b\", got %q", s.String())
	}
}
