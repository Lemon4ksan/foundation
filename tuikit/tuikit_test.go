// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/tuikit"
)

func TestTUI_VisibleWidthAndStripANSI(t *testing.T) {
	styled := tuikit.Cyan("Hello") + " " + tuikit.Bold(tuikit.Green("World!"))
	if tuikit.StripANSI(styled) != "Hello World!" {
		t.Errorf("expected 'Hello World!', got %q", tuikit.StripANSI(styled))
	}
	if tuikit.VisibleWidth(styled) != 12 {
		t.Errorf("expected visible width 12, got %d", tuikit.VisibleWidth(styled))
	}

	unicodeStr := "⚡ Microsecond ✔ PASS"
	if tuikit.VisibleWidth(unicodeStr) != 20 {
		t.Errorf("expected visible width 20, got %d", tuikit.VisibleWidth(unicodeStr))
	}
}

func TestTUI_Padding(t *testing.T) {
	s := "test"
	if tuikit.PadRight(s, 10) != "test      " {
		t.Errorf("pad right failed: %q", tuikit.PadRight(s, 10))
	}
	if tuikit.PadLeft(s, 10) != "      test" {
		t.Errorf("pad left failed: %q", tuikit.PadLeft(s, 10))
	}
	if tuikit.PadCenter(s, 10) != "   test   " {
		t.Errorf("pad center failed: %q", tuikit.PadCenter(s, 10))
	}
}

func TestTUI_Table(t *testing.T) {
	tbl := tuikit.NewTable("SERVICE", "THROUGHPUT", "STATUS")
	tbl.SetAlignment(1, tuikit.AlignRight)
	tbl.SetAlignment(2, tuikit.AlignCenter)
	tbl.AddRow("Powned", "8.35B ops/s", tuikit.BadgePass())

	out := tbl.String()
	if !strings.Contains(out, "SERVICE") || !strings.Contains(out, "Powned") || !strings.Contains(out, "PASS") {
		t.Errorf("unexpected table rendering:\n%s", out)
	}
}

func TestTUI_Box(t *testing.T) {
	box := tuikit.NewBox("Card Title", 50)
	box.AddLine("Line 1 content")
	box.AddDivider()
	box.AddRow("Left Side", "Right Side")

	out := box.String()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d:\n%s", len(lines), out)
	}
	for i, l := range lines {
		w := tuikit.VisibleWidth(l)
		if w != 54 { // 50 inner + 2 borders + 2 margin
			t.Errorf("line %d width %d != 54: %q", i, w, l)
		}
	}
}

func TestTUI_InterspersedFlags(t *testing.T) {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	var (
		long    bool
		all     bool
		sortStr string
	)
	tuikit.BoolVar(fs, &long, "long", "l", false, "long mode")
	tuikit.BoolVar(fs, &all, "all", "a", false, "all files")
	tuikit.StringVar(fs, &sortStr, "sort", "s", "name", "sort key")

	// Mixed args: positional, flags, short flags, long flags
	args := []string{"target_dir", "-l", "--sort=time", "-a", "extra_pos"}
	pos, err := tuikit.ParseInterspersedFlags(fs, args)
	if err != nil {
		t.Fatalf("ParseInterspersedFlags failed: %v", err)
	}

	if !long {
		t.Errorf("expected long=true")
	}
	if !all {
		t.Errorf("expected all=true")
	}
	if sortStr != "time" {
		t.Errorf("expected sortStr='time', got %q", sortStr)
	}
	if len(pos) != 2 || pos[0] != "target_dir" || pos[1] != "extra_pos" {
		t.Errorf("unexpected pos args: %v", pos)
	}
}

func TestTUI_App(t *testing.T) {
	var executed bool
	testCmd := &tuikit.SimpleCommand{
		CmdName:     "ping",
		CmdSynopsis: "Test ping command",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			executed = true
			fmt.Fprintln(stdout, "PONG")
			return nil
		},
	}

	var stdout, stderr bytes.Buffer
	app := tuikit.NewApp("testapp", "1.0.0", "Test Description", testCmd)
	app.Stdout = &stdout
	app.Stderr = &stderr

	err := app.Run(context.Background(), []string{"ping"})
	if err != nil {
		t.Fatalf("app run failed: %v", err)
	}
	if !executed {
		t.Errorf("command was not executed")
	}
	if !strings.Contains(stdout.String(), "PONG") {
		t.Errorf("expected PONG output, got %q", stdout.String())
	}
}
