// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
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

func TestTUI_App(t *testing.T) {
	var executed bool
	testCmd := &tuikit.FuncCommand{
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

	stdout.Reset()
	app.PrintUsage(&stdout)
	usage := stdout.String()
	if !strings.Contains(usage, "testapp 1.0.0 — Test Description") {
		t.Errorf("expected clean header in usage: %q", usage)
	}
	if !strings.Contains(usage, "ping") || !strings.Contains(usage, "Test ping command") {
		t.Errorf("expected command listing in usage: %q", usage)
	}
	if strings.Contains(usage, "COMMAND") && strings.Contains(usage, "SYNOPSIS") {
		t.Errorf("usage should not contain ASCII table headers: %q", usage)
	}
	if strings.Contains(usage, "────") {
		t.Errorf("usage should not contain ASCII table divider lines: %q", usage)
	}
}
type safeBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (s *safeBuffer) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestAdversarial_FuncCommand_ContextCancellation(t *testing.T) {
	t.Parallel()

	// 1. Pre-canceled context
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	cmd := &tuikit.FuncCommand{
		CmdName: "cancellable",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
	}

	err := cmd.Run(canceledCtx, []string{"arg1"}, io.Discard, io.Discard)
	assert.ErrorIs(t, err, context.Canceled)

	// 2. Cancellation during execution
	blockingCmd := &tuikit.FuncCommand{
		CmdName: "blocker",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}

	liveCtx, liveCancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- blockingCmd.Run(liveCtx, nil, io.Discard, io.Discard)
	}()

	time.Sleep(10 * time.Millisecond)
	liveCancel()

	select {
	case err := <-errCh:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(1 * time.Second):
		t.Fatal("command did not terminate on context cancellation")
	}

	// 3. Context timeout / deadline exceeded
	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer timeoutCancel()

	timeoutErr := blockingCmd.Run(timeoutCtx, nil, io.Discard, io.Discard)
	assert.ErrorIs(t, timeoutErr, context.DeadlineExceeded)
}

func TestAdversarial_FuncCommand_ConcurrentContextCancellations(t *testing.T) {
	t.Parallel()

	var completedCount atomic.Int64
	var canceledCount atomic.Int64

	stressCmd := &tuikit.FuncCommand{
		CmdName: "stress",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			select {
			case <-ctx.Done():
				canceledCount.Add(1)
				return ctx.Err()
			case <-time.After(15 * time.Millisecond):
				completedCount.Add(1)
				return nil
			}
		},
	}

	const goroutines = 40
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(idx%30)*time.Millisecond)
			defer cancel()

			err := stressCmd.Run(ctx, []string{"--opt", fmt.Sprintf("val-%d", idx)}, io.Discard, io.Discard)
			if err != nil {
				if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
					t.Errorf("unexpected error: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, int64(goroutines), completedCount.Load()+canceledCount.Load())
}

func TestAdversarial_FuncCommand_MalformedArguments(t *testing.T) {
	t.Parallel()

	var lastArgs []string
	captureCmd := &tuikit.FuncCommand{
		CmdName:     "capture",
		CmdAliases:  []string{"c", "cap"},
		CmdSynopsis: "Capture args",
		CmdUsage:    "capture [args...]",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			lastArgs = args
			return nil
		},
	}

	// 1. Nil args
	err := captureCmd.Run(context.Background(), nil, io.Discard, io.Discard)
	assert.NoError(t, err)
	assert.Nil(t, lastArgs)

	// 2. Empty strings in args
	emptyStrings := []string{"", "", ""}
	err = captureCmd.Run(context.Background(), emptyStrings, io.Discard, io.Discard)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(lastArgs))

	// 3. Huge argument slice
	const hugeCount = 20_000
	hugeArgs := make([]string, hugeCount)
	for i := range hugeCount {
		hugeArgs[i] = fmt.Sprintf("arg-%d", i)
	}
	err = captureCmd.Run(context.Background(), hugeArgs, io.Discard, io.Discard)
	assert.NoError(t, err)
	assert.Equal(t, hugeCount, len(lastArgs))

	// 4. Arguments containing null bytes, unicode, ANSI escapes
	adversarialArgs := []string{
		"\x00\x00\x00",
		"\u001b[31mRED\u001b[0m",
		"日本語 / 한국어 / 中文 / Русский",
		strings.Repeat("Z", 10_000),
	}
	err = captureCmd.Run(context.Background(), adversarialArgs, io.Discard, io.Discard)
	assert.NoError(t, err)
	assert.Equal(t, len(adversarialArgs), len(lastArgs))

	// 5. Nil CmdRun
	nilRunCmd := &tuikit.FuncCommand{CmdName: "noop"}
	err = nilRunCmd.Run(context.Background(), adversarialArgs, io.Discard, io.Discard)
	assert.NoError(t, err)

	// Verify getters
	assert.Equal(t, "capture", captureCmd.Name())
	assert.Equal(t, []string{"c", "cap"}, captureCmd.Aliases())
	assert.Equal(t, "Capture args", captureCmd.Synopsis())
	assert.Equal(t, "capture [args...]", captureCmd.Usage())
}

func TestAdversarial_App_ConcurrentRoutingAndFallbacks(t *testing.T) {
	t.Parallel()

	echoCmd := &tuikit.FuncCommand{
		CmdName:    "echo",
		CmdAliases: []string{"e", "print"},
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			fmt.Fprintln(stdout, strings.Join(args, " "))
			return nil
		},
	}

	failCmd := &tuikit.FuncCommand{
		CmdName: "fail",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			return errors.New("command failed deliberately")
		},
	}

	defaultCmd := &tuikit.FuncCommand{
		CmdName: "default",
		CmdRun: func(ctx context.Context, args []string, stdout, stderr io.Writer) error {
			fmt.Fprintln(stdout, "DEFAULT")
			return nil
		},
	}

	app := tuikit.NewApp("stresscli", "2.0.0", "Adversarial Stress CLI", echoCmd, failCmd)
	app.SetDefaultCommand(defaultCmd)
	app.Stdout = io.Discard
	app.Stderr = io.Discard

	const goroutines = 30
	const iterations = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				ctx := context.Background()

				switch (gid + i) % 6 {
				case 0:
					// Run valid command
					err := app.Run(ctx, []string{"echo", "hello", "world"})
					assert.NoError(t, err)
				case 1:
					// Run via alias
					err := app.Run(ctx, []string{"e", "aliased"})
					assert.NoError(t, err)
				case 2:
					// Run failing command
					err := app.Run(ctx, []string{"fail"})
					assert.Error(t, err)
				case 3:
					// Run default command fallback (empty args)
					err := app.Run(ctx, []string{})
					assert.NoError(t, err)
				case 4:
					// Run help flag
					err := app.Run(ctx, []string{"--help"})
					assert.NoError(t, err)
				case 5:
					// Run unknown command
					err := app.Run(ctx, []string{fmt.Sprintf("unknown-%d", i)})
					assert.Error(t, err)
				}
			}
		}(g)
	}

	wg.Wait()
}
func TestEmpirical_Tuikit_LinesSeq_EarlyTermination(t *testing.T) {
	text := "first line\r\nsecond line\nthird line\r\nfourth line\nfifth line"

	// 1. Break immediately after 0 iterations (first item returns false)
	count := 0
	for line := range tuikit.LinesSeq(text) {
		_ = line
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 2. Break after 2 lines
	count = 0
	for line := range tuikit.LinesSeq(text) {
		_ = line
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// 3. Full traversal
	var lines []string
	for line := range tuikit.LinesSeq(text) {
		lines = append(lines, line)
	}
	expected := []string{"first line", "second line", "third line", "fourth line", "fifth line"}
	if !slices.Equal(lines, expected) {
		t.Fatalf("expected lines %v, got %v", expected, lines)
	}

	// 4. Break mid-iteration
	mid := len(expected) / 2
	visited := 0
	for range tuikit.LinesSeq(text) {
		visited++
		if visited == mid {
			break
		}
	}
	if visited != mid {
		t.Fatalf("expected visited %d, got %d", mid, visited)
	}
}

func TestEmpirical_Tuikit_LinesSeq_EdgeCases(t *testing.T) {
	// Empty string
	for range tuikit.LinesSeq("") {
		t.Fatal("expected no lines for empty string")
	}

	// Single line without newline
	single := "single line without newline"
	var singleLines []string
	for l := range tuikit.LinesSeq(single) {
		singleLines = append(singleLines, l)
	}
	if len(singleLines) != 1 || singleLines[0] != single {
		t.Fatalf("expected [single line without newline], got %v", singleLines)
	}

	// Only CRLF
	crlf := "\r\n\r\n\r\n"
	var crlfLines []string
	for l := range tuikit.LinesSeq(crlf) {
		crlfLines = append(crlfLines, l)
	}
	// 3 newlines yield 3 empty lines
	if len(crlfLines) != 3 {
		t.Fatalf("expected 3 empty lines, got %d: %v", len(crlfLines), crlfLines)
	}
	for i, l := range crlfLines {
		if l != "" {
			t.Errorf("expected line %d to be empty, got %q", i, l)
		}
	}
}

func BenchmarkLinesSeq_ZeroAlloc(b *testing.B) {
	text := "header\r\nservice: worker\r\nthroughput: 100k\r\nstatus: ok\r\nfooter"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for line := range tuikit.LinesSeq(text) {
			_ = line
		}
	}
}
