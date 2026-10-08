// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/tuikit"
)

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
