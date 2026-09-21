// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pipeline

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"sync"

	"golang.org/x/time/rate"
)

// Config defines the operational scaling, rate limiting,
// and error-propagation parameters for a [Pipeline].
type Config struct {
	// Workers specifies the maximum number of concurrent goroutines processing tasks.
	Workers int
	// RPS defines the rate limit in requests per second. Set to 0.0 for unlimited rate.
	RPS float64
	// Burst defines the token bucket size for the rate limiter. Defaults to 1 if not specified.
	Burst int
	// FailFast enables immediate cancellation of all other active workers on the first encountered error.
	FailFast bool
}

// resolveDefaults applies safe fallback defaults for unconfigured scaling parameters.
func (c *Config) resolveDefaults() {
	if c.Workers <= 0 {
		c.Workers = 1
	}

	if c.Burst <= 0 {
		c.Burst = 1
	}
}

// task holds metadata, payload, and the processing outcome of an execution item.
type task[In, Out any] struct {
	idx int
	val In
	res Out
	err error
}

// Map is a high-level helper utility that transforms a slice of inputs concurrently
// using the provided mapper function under a temporary Pipeline.
//
// It preserves the original index order of the inputs in the returned slice.
func Map[In, Out any](
	ctx context.Context,
	cfg Config,
	inputs []In,
	mapper func(context.Context, In) (Out, error),
) ([]Out, error) {
	if mapper == nil {
		return nil, errors.New("pipeline: mapper function is nil")
	}

	p := New[In, Out](cfg)

	return p.Process(ctx, inputs, mapper)
}

// ForEach is a high-level helper utility that processes a slice of inputs concurrently
// for side-effects, ignoring individual successful return values.
func ForEach[In any](
	ctx context.Context,
	cfg Config,
	inputs []In,
	fn func(context.Context, In) error,
) error {
	if fn == nil {
		return errors.New("pipeline: function is nil")
	}

	p := New[In, struct{}](cfg)
	_, err := p.Process(ctx, inputs, func(c context.Context, in In) (struct{}, error) {
		return struct{}{}, fn(c, in)
	})

	return err
}

// ProcessSeq concurrently transforms elements from the input iterator using mapper
// under bounded concurrency governed by cfg.Workers, yielding (Out, error) pairs.
// If yield returns false or ctx is cancelled, workers are cancelled immediately
// and iteration halts without leaking background goroutines.
//
// Concurrency Guarantees:
// ProcessSeq is safe for concurrent invocation from multiple goroutines.
func ProcessSeq[In, Out any](
	ctx context.Context,
	cfg Config,
	in iter.Seq[In],
	mapper func(context.Context, In) (Out, error),
) iter.Seq2[Out, error] {
	return func(yield func(Out, error) bool) {
		if mapper == nil {
			var zero Out
			yield(zero, errors.New("pipeline: mapper function is nil"))
			return
		}
		cfg.resolveDefaults()
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		type resItem struct {
			out Out
			err error
		}

		inCh := make(chan In, cfg.Workers*2)
		outCh := make(chan resItem, cfg.Workers*2)

		var inWg sync.WaitGroup
		inWg.Add(1)
		go func() {
			defer inWg.Done()
			defer close(inCh)
			for item := range in {
				select {
				case <-runCtx.Done():
					return
				case inCh <- item:
				}
			}
		}()

		var workerWg sync.WaitGroup
		for range cfg.Workers {
			workerWg.Add(1)
			go func() {
				defer workerWg.Done()
				for item := range inCh {
					select {
					case <-runCtx.Done():
						return
					default:
					}
					out, err := mapper(runCtx, item)
					select {
					case <-runCtx.Done():
						return
					case outCh <- resItem{out: out, err: err}:
					}
				}
			}()
		}

		go func() {
			workerWg.Wait()
			close(outCh)
		}()

		for res := range outCh {
			if !yield(res.out, res.err) {
				cancel()
				break
			}
		}
	}
}

// ProcessSeq concurrently transforms the input iterator using mapper under the pipeline's configuration.
func (p *Pipeline[In, Out]) ProcessSeq(
	ctx context.Context,
	in iter.Seq[In],
	mapper func(context.Context, In) (Out, error),
) iter.Seq2[Out, error] {
	return ProcessSeq(ctx, p.config, in, mapper)
}

// Pipeline coordinates concurrent data processing, rate limiting, and order preservation
// across slice-based or channel-based data streams.
//
// A Pipeline is safe for concurrent use by multiple goroutines.
type Pipeline[In, Out any] struct {
	config  Config
	limiter *rate.Limiter
	pool    sync.Pool
}

// New instantiates and configures a new [Pipeline] with the given [Config].
func New[In, Out any](cfg Config) *Pipeline[In, Out] {
	cfg.resolveDefaults()

	p := &Pipeline[In, Out]{
		config: cfg,
	}
	if cfg.RPS > 0 {
		p.limiter = rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst)
	}

	p.pool.New = func() any {
		return &task[In, Out]{}
	}

	return p
}

// Process concurrently transforms the input slice using the provided mapper function.
//
// It preserves the original index order of the inputs in the returned output slice.
// If FailFast is enabled, the first encountered error cancels all running workers and
// is returned immediately. Otherwise, all encountered errors are aggregated using [errors.Join].
func (p *Pipeline[In, Out]) Process(
	ctx context.Context,
	inputs []In,
	mapper func(context.Context, In) (Out, error),
) ([]Out, error) {
	if p == nil {
		return nil, errors.New("pipeline: pipeline is nil")
	}

	if mapper == nil {
		return nil, errors.New("pipeline: mapper function is nil")
	}

	if len(inputs) == 0 {
		return nil, nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	tasksCh := make(chan *task[In, Out], len(inputs))
	resultsCh := make(chan *task[In, Out], len(inputs))

	tasksChClosed := false
	// Recycle any unprocessed tasks back into the sync.Pool on early exit.
	defer func() {
		if !tasksChClosed {
			close(tasksCh)
		}

		var (
			zeroIn  In
			zeroOut Out
		)

		for t := range tasksCh {
			t.val = zeroIn
			t.res = zeroOut
			t.err = nil
			p.pool.Put(t)
		}
	}()

	if err := runCtx.Err(); err != nil {
		return nil, err
	}

	var zeroOut Out
	for i, in := range inputs {
		t := p.pool.Get().(*task[In, Out])
		t.idx = i
		t.val = in
		t.res = zeroOut

		t.err = nil
		tasksCh <- t
	}

	close(tasksCh)

	tasksChClosed = true

	var wg sync.WaitGroup

	workers := min(p.config.Workers, len(inputs))

	for range workers {
		wg.Go(func() {
			var currentTask *task[In, Out]
			defer func() {
				// Prevent user-provided mapper panics from crashing the worker runtime
				// and causing deadlocks in the results collector.
				if r := recover(); r != nil {
					if currentTask != nil {
						currentTask.err = fmt.Errorf("pipeline: mapper panicked: %v", r)
						resultsCh <- currentTask
					}
				}
			}()

			for {
				select {
				case <-runCtx.Done():
					return
				case t, ok := <-tasksCh:
					if !ok {
						return
					}

					currentTask = t

					if err := runCtx.Err(); err != nil {
						t.err = err
						resultsCh <- t

						currentTask = nil

						return
					}

					if p.limiter != nil {
						if err := p.limiter.Wait(runCtx); err != nil {
							t.err = err
							resultsCh <- t

							currentTask = nil

							continue
						}
					}

					res, err := mapper(runCtx, t.val)
					t.res = res

					t.err = err
					resultsCh <- t

					currentTask = nil
				}
			}
		})
	}

	// Gracefully close results channel once all background workers complete.
	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	outputs := make([]Out, len(inputs))

	var (
		collectedErrors []error
		firstErr        error
		zeroIn          In
	)

	for t := range resultsCh {
		if t.err != nil {
			if p.config.FailFast {
				if firstErr == nil {
					firstErr = t.err

					cancel() // Cancel all other active workers immediately
				}
			} else {
				collectedErrors = append(collectedErrors, t.err)
			}
		} else {
			outputs[t.idx] = t.res
		}

		// Recycle task utilizing stack-allocated zero values to prevent allocations.
		t.val = zeroIn
		t.res = zeroOut
		t.err = nil
		p.pool.Put(t)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if firstErr != nil {
		return nil, firstErr
	}

	if len(collectedErrors) > 0 {
		return nil, errors.Join(collectedErrors...)
	}

	return outputs, nil
}

// Stream processes items from the input channel concurrently and yields results to the output channel.
//
// It closes both returned channels upon completion or early cancellation.
// All error dispatches are non-blocking to prevent worker goroutine leakage.
func (p *Pipeline[In, Out]) Stream(
	ctx context.Context,
	in <-chan In,
	mapper func(context.Context, In) (Out, error),
) (<-chan Out, <-chan error) {
	out := make(chan Out, p.config.Workers)
	errs := make(chan error, p.config.Workers)

	if mapper == nil {
		errs <- errors.New("pipeline: mapper function is nil")
		close(out)
		close(errs)
		return out, errs
	}

	go func() {
		defer close(out)
		defer close(errs)

		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		var (
			wg      sync.WaitGroup
			errOnce sync.Once
		)

		workers := p.config.Workers
		for range workers {
			wg.Go(func() {
				defer func() {
					// Safe panic recovery for streaming worker goroutines.
					if r := recover(); r != nil {
						err := fmt.Errorf("pipeline: stream mapper panicked: %v", r)
						select {
						case errs <- err:
						default:
						}

						errOnce.Do(func() {
							if p.config.FailFast {
								cancel()
							}
						})
					}
				}()

				for {
					select {
					case <-runCtx.Done():
						return
					case val, ok := <-in:
						if !ok {
							return
						}

						if p.limiter != nil {
							if err := p.limiter.Wait(runCtx); err != nil {
								select {
								case errs <- err:
								default:
								}

								errOnce.Do(func() {
									if p.config.FailFast {
										cancel()
									}
								})

								return
							}
						}

						res, err := mapper(runCtx, val)
						if err != nil {
							select {
							case errs <- err:
							default:
							}

							errOnce.Do(func() {
								if p.config.FailFast {
									cancel()
								}
							})

							if p.config.FailFast {
								return
							}
						} else {
							select {
							case out <- res:
							case <-runCtx.Done():
								return
							}
						}
					}
				}
			})
		}

		wg.Wait()
	}()

	return out, errs
}
