// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lifecycle

import (
	"context"
	"errors"
	"sync"
)

// ErrAlreadyRunning is returned by [BehaviorRunner.Start] when attempting to start
// a runner that has already been initiated and is currently running.
var ErrAlreadyRunning = errors.New("behavior runner is already running")

// Behavior represents a managed, modular execution loop that can be run
// concurrently by the [BehaviorRunner] or adapted to a [Service].
type Behavior interface {
	// Name returns the unique, static identifier for the behavior.
	// The runner uses this name to prevent duplicate registrations.
	Name() string

	// Run starts the main execution loop of the behavior. It must block and
	// remain active until the provided context is cancelled, or until an
	// unrecoverable error is encountered.
	//
	// Upon context cancellation, Run should release its resources and return
	// the context cancellation error (e.g., [context.Canceled]).
	Run(ctx context.Context) error
}

// Option defines a configuration function for customizing a [BehaviorRunner].
type Option func(*BehaviorRunner)

// Logger defines the structured logging interface used by [BehaviorRunner].
type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type discardLogger struct{}

func (discardLogger) Info(string, ...any)  {}
func (discardLogger) Warn(string, ...any)  {}
func (discardLogger) Error(string, ...any) {}

// WithLogger returns an [Option] that configures the runner to log
// lifecycle transitions, warnings, and task failures using the provided [Logger].
// If the logger is nil, the default no-op logger is retained.
func WithLogger(l Logger) Option {
	return func(o *BehaviorRunner) {
		if l != nil {
			o.logger = l
		}
	}
}

// WithFailFast returns an [Option] that configures the runner to abort
// all running behaviors immediately if any single behavior returns an error.
//
// When enabled, a non-nil error returned by any behavior's Run method causes
// the runner to cancel the run context shared by all other behaviors.
func WithFailFast() Option {
	return func(o *BehaviorRunner) {
		o.failFast = true
	}
}

// BehaviorRunner coordinates the lifecycle, startup, and graceful teardown of
// multiple [Behavior] tasks.
//
// It ensures that behaviors are launched in isolated goroutines, monitors
// their failures, handles optional fail-fast cascades, and blocks during
// shutdown until all background goroutines have fully exited.
//
// A BehaviorRunner is safe for concurrent use by multiple goroutines.
type BehaviorRunner struct {
	logger    Logger
	behaviors []Behavior
	mu        sync.RWMutex
	running   bool
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	failFast  bool
}

// NewBehaviorRunner instantiates and configures a new [BehaviorRunner] with the
// provided [Option] modifiers.
//
// By default, it initializes with a silent, no-op logger and is in non-fail-fast mode.
func NewBehaviorRunner(opts ...Option) *BehaviorRunner {
	o := &BehaviorRunner{
		logger:    discardLogger{},
		behaviors: make([]Behavior, 0),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}

	return o
}

// Register appends a [Behavior] to the runner's managed set.
//
// Register must be called before [BehaviorRunner.Start]. If the runner
// is already running, or if a behavior with the same [Behavior.Name] has
// already been registered, Register ignores the behavior and logs a warning.
func (o *BehaviorRunner) Register(b Behavior) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		o.logger.Warn("Cannot register behavior while runner is running", "name", b.Name())
		return
	}

	for _, existing := range o.behaviors {
		if existing.Name() == b.Name() {
			o.logger.Warn("Behavior with this name is already registered, skipping", "name", b.Name())
			return
		}
	}

	o.behaviors = append(o.behaviors, b)
}

// Start launches all registered behaviors in separate background goroutines,
// passing them a shared context derived from the provided ctx.
//
// If the runner is already running, Start returns [ErrAlreadyRunning].
// If any behavior returns an error before the run context is cancelled, it is logged.
// If fail-fast mode is enabled, the first behavior failure triggers the cancellation
// of the shared context, causing all other behaviors to shut down.
func (o *BehaviorRunner) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return ErrAlreadyRunning
	}

	o.running = true
	runCtx, cancel := context.WithCancel(ctx)
	o.cancel = cancel

	for _, b := range o.behaviors {
		o.wg.Add(1)

		go func(beh Behavior) {
			defer o.wg.Done()

			o.logger.Info("Starting behavior", "name", beh.Name())

			if err := beh.Run(runCtx); err != nil {
				// Only log the error if the context hasn't been cancelled yet,
				// which distinguishes unexpected crashes from planned shutdowns.
				if runCtx.Err() == nil {
					o.logger.Error("Behavior failed", "name", beh.Name(), "error", err)

					if o.failFast {
						cancel()
					}
				}
			} else {
				o.logger.Info("Behavior stopped gracefully", "name", beh.Name())
			}
		}(b)
	}

	return nil
}

// Stop initiates a graceful shutdown of all active behaviors.
//
// Stop cancels the shared run context, signaling all background goroutines
// to exit, and blocks until all behaviors have fully terminated.
//
// Calling Stop on an inactive runner is a safe no-op. Stop is safe for
// concurrent invocation; multiple concurrent calls to Stop will safely wait for
// the same teardown process to complete.
func (o *BehaviorRunner) Stop() {
	o.mu.Lock()
	if !o.running {
		o.mu.Unlock()
		return
	}

	cancel := o.cancel
	o.mu.Unlock()

	// Cancel the context outside the mutex lock to prevent blocking
	// concurrent read operations (like Count) during the teardown phase.
	cancel()
	o.wg.Wait()

	o.mu.Lock()
	o.running = false
	o.mu.Unlock()
}

// Count returns the number of currently registered behaviors.
//
// It is safe to call concurrently while the runner is running.
func (o *BehaviorRunner) Count() int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	return len(o.behaviors)
}

// behaviorService wraps a [Behavior] to satisfy the [Service] interface.
type behaviorService struct {
	behavior Behavior
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	err      error
}

// AsService wraps a [Behavior] to be used as a [Service] in [Orchestrator].
func AsService(b Behavior) Service {
	return &behaviorService{behavior: b}
}

func (bs *behaviorService) Name() string {
	return bs.behavior.Name()
}

func (bs *behaviorService) Init(ctx context.Context) error {
	return nil
}

func (bs *behaviorService) Start(ctx context.Context) error {
	runCtx, cancel := context.WithCancel(ctx)
	bs.cancel = cancel
	bs.wg.Go(func() {
		bs.err = bs.behavior.Run(runCtx)
	})

	return nil
}

func (bs *behaviorService) Stop(ctx context.Context) error {
	if bs.cancel != nil {
		bs.cancel()
	}

	bs.wg.Wait()

	return bs.err
}
