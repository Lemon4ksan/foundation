// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/tuikit"
)

func TestProgress_TTY_Deterministic(t *testing.T) {
	var buf bytes.Buffer
	p := tuikit.NewProgress(
		tuikit.WithWriter(&buf),
		tuikit.WithPrefix("Extracting"),
		tuikit.WithTotal(100),
		tuikit.WithUnit(tuikit.UnitCount),
		tuikit.WithInteractive(true),
	)

	p.SetCurrent(50)
	p.Render()

	out := buf.String()
	if !strings.Contains(out, "Extracting") {
		t.Fatalf("expected prefix 'Extracting' in output: %q", out)
	}
	if !strings.Contains(out, "50%") {
		t.Fatalf("expected '50%%' in output: %q", out)
	}
	if !strings.Contains(out, "(50 / 100)") {
		t.Fatalf("expected count '(50 / 100)' in output: %q", out)
	}

	p.Finish("✓ Successfully extracted 100 items")
	finalOut := buf.String()
	if !strings.Contains(finalOut, "✓ Successfully extracted 100 items") {
		t.Fatalf("expected finish message in output: %q", finalOut)
	}
}

func TestProgress_TTY_BytesUnit(t *testing.T) {
	var buf bytes.Buffer
	p := tuikit.NewProgress(
		tuikit.WithWriter(&buf),
		tuikit.WithPrefix("Uploading"),
		tuikit.WithTotal(10*1024*1024), // 10 MB
		tuikit.WithUnit(tuikit.UnitBytes),
		tuikit.WithInteractive(true),
	)

	p.SetCurrent(5 * 1024 * 1024) // 5 MB
	p.Render()

	out := buf.String()
	if !strings.Contains(out, "5.0 MB") {
		t.Fatalf("expected '5.0 MB' in output: %q", out)
	}
	if !strings.Contains(out, "10.0 MB") {
		t.Fatalf("expected '10.0 MB' in output: %q", out)
	}
	if !strings.Contains(out, "50%") {
		t.Fatalf("expected 50%% in output: %q", out)
	}

	p.Done()
}

func TestProgress_TTY_Indeterminate(t *testing.T) {
	var buf bytes.Buffer
	p := tuikit.NewProgress(
		tuikit.WithWriter(&buf),
		tuikit.WithPrefix("Packing"),
		tuikit.WithTotal(0),
		tuikit.WithUnit(tuikit.UnitCount),
		tuikit.WithInteractive(true),
	)

	p.Add(1500)
	p.SetExtraBytes(45 * 1024 * 1024)
	p.Render()

	out := buf.String()
	if !strings.Contains(out, "Packing") {
		t.Fatalf("expected 'Packing' in output: %q", out)
	}
	if !strings.Contains(out, "1,500 items") {
		t.Fatalf("expected '1,500 items' in output: %q", out)
	}
	if !strings.Contains(out, "45.0 MB") {
		t.Fatalf("expected '45.0 MB' in output: %q", out)
	}

	p.Done()
}

func TestProgress_NonTTY(t *testing.T) {
	var buf bytes.Buffer
	p := tuikit.NewProgress(
		tuikit.WithWriter(&buf),
		tuikit.WithPrefix("Processing"),
		tuikit.WithTotal(200),
		tuikit.WithInteractive(false),
	)

	p.SetCurrent(100)
	p.Render()

	out := buf.String()
	if strings.Contains(out, "\r") {
		t.Fatalf("non-TTY output should not contain carriage returns: %q", out)
	}
	if !strings.Contains(out, "Processing: 50% (100 / 200)") {
		t.Fatalf("expected clean non-TTY log line: %q", out)
	}
}

func TestProgressReader(t *testing.T) {
	var buf bytes.Buffer
	p := tuikit.NewProgress(
		tuikit.WithWriter(&buf),
		tuikit.WithTotal(1000),
		tuikit.WithUnit(tuikit.UnitBytes),
		tuikit.WithInteractive(true),
	)

	srcData := make([]byte, 500)
	reader := p.NewReader(bytes.NewReader(srcData))

	dest := make([]byte, 250)
	n, err := io.ReadFull(reader, dest)
	if err != nil || n != 250 {
		t.Fatalf("read failed: n=%d, err=%v", n, err)
	}

	p.Render()
	out := buf.String()
	if !strings.Contains(out, "25%") {
		t.Fatalf("expected 25%% after reading 250 bytes: %q", out)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes uint64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{10 * 1024 * 1024, "10.0 MB"},
		{1500 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		got := tuikit.FormatBytes(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
