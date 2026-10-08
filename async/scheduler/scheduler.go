// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scheduler

import (
	"container/heap"
	"context"
	"sync"
	"time"
)

// Priority defines the execution precedence level of a [Task].
type Priority int

const (
	// PriorityLow represents the lowest task execution priority.
	PriorityLow Priority = iota
	// PriorityNormal represents the standard task execution priority.
	PriorityNormal
	// PriorityHigh represents the highest task execution priority.
	PriorityHigh
)

// Task represents a single schedulable unit of work.
// Acquire instances using [Scheduler.AcquireTask] and release them with [Scheduler.ReleaseTask].
type Task struct {
	// ID is the unique identifier of the task. Required for cancellation.
	ID string
	// Priority determines execution precedence when tasks are scheduled for the exact same time.
	Priority Priority
	// NextRun is the absolute time when the task is scheduled to execute next.
	NextRun time.Time
	// Interval is the duration between execution cycles for periodic tasks.
	// Set to <= 0 for one-off tasks.
	Interval time.Duration
	// FixedRate determines how the next run is calculated for periodic tasks.
	// If true, NextRun is strictly calculated from the previous scheduled time (cron-like).
	// If false, NextRun is calculated from the time of completion (fixed-delay).
	FixedRate bool
	// Execute is the context-aware function executed by the scheduler.
	Execute func(ctx context.Context) error

	index    int
	canceled bool // internal marker for O(1) lazy cancellation
}

// Debounce returns a thread-safe wrapped function that delays invoking fn until interval has elapsed.
// Each invocation of the returned function restarts the delay timer.
func Debounce(interval time.Duration, fn func()) func() {
	var (
		mu    sync.Mutex
		timer *time.Timer
	)

	return func() {
		mu.Lock()
		defer mu.Unlock()

		if timer == nil {
			timer = time.AfterFunc(interval, fn)
		} else {
			timer.Reset(interval) // Zero-allocation reset
		}
	}
}

// Throttle returns a thread-safe wrapped function that invokes fn at most once per interval.
// The first invocation runs asynchronously in a new goroutine, and subsequent calls are ignored during the cooldown.
func Throttle(interval time.Duration, fn func()) func() {
	var (
		mu      sync.Mutex
		lastRun time.Time
	)

	return func() {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if now.Sub(lastRun) >= interval {
			lastRun = now

			go fn()
		}
	}
}

// DebounceFn returns a thread-safe wrapped function that delays invoking fn with parameter arg
// until interval has elapsed since the most recent call. Each call restarts the timer with the latest argument.
//
// Concurrency Guarantees:
// The returned function is safe for concurrent use across multiple goroutines.
func DebounceFn[T any](interval time.Duration, fn func(T)) func(T) {
	var (
		mu    sync.Mutex
		timer *time.Timer
		last  T
	)

	return func(arg T) {
		mu.Lock()
		defer mu.Unlock()

		last = arg
		if timer == nil {
			timer = time.AfterFunc(interval, func() {
				mu.Lock()
				val := last
				mu.Unlock()
				fn(val)
			})
		} else {
			timer.Reset(interval) // Zero-allocation reset
		}
	}
}

// ThrottleFn returns a thread-safe wrapped function that invokes fn with parameter arg at most once per interval.
// The first invocation runs asynchronously in a new goroutine, and subsequent calls within the interval window are ignored.
//
// Concurrency Guarantees:
// The returned function is safe for concurrent use across multiple goroutines.
func ThrottleFn[T any](interval time.Duration, fn func(T)) func(T) {
	var (
		mu      sync.Mutex
		lastRun time.Time
	)

	return func(arg T) {
		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		if now.Sub(lastRun) >= interval {
			lastRun = now
			val := arg
			go fn(val)
		}
	}
}

// Scheduler coordinates prioritized, time-exact task execution.
//
// Create new instances of Scheduler using the [New] constructor function.
// All scheduler operations are safe for concurrent use by multiple goroutines.
type Scheduler struct {
	mu        sync.Mutex
	tasks     taskHeap
	tasksByID map[string]*Task
	wakeChan  chan struct{}
	taskPool  sync.Pool

	wg        sync.WaitGroup
	workerSem chan struct{} // bounded concurrency semaphore
}

// New initializes and returns a new [Scheduler] instance.
func New() *Scheduler {
	s := &Scheduler{
		wakeChan:  make(chan struct{}, 1),
		tasksByID: make(map[string]*Task),
	}
	s.taskPool.New = func() any {
		return &Task{index: -1}
	}

	return s
}

// SetMaxConcurrency limits the maximum number of concurrent tasks executing at the same time.
// By default (or if limit <= 0), the scheduler uses unbounded concurrency.
func (s *Scheduler) SetMaxConcurrency(limit int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit > 0 {
		s.workerSem = make(chan struct{}, limit)
	} else {
		s.workerSem = nil
	}
}

// AcquireTask retrieves a clean, recycled [Task] instance from the internal pool.
//
// Always configure and schedule tasks returned by this method using [Scheduler.Schedule].
func (s *Scheduler) AcquireTask() *Task {
	return s.taskPool.Get().(*Task)
}

// ReleaseTask resets task fields and returns the [Task] back to the internal pool.
//
// Do not read, write, or hold references to a task once it has been released,
// as doing so will cause serious data races and memory corruption.
func (s *Scheduler) ReleaseTask(t *Task) {
	if s == nil || t == nil {
		return
	}

	t.ID = ""
	t.Execute = nil
	t.Interval = 0
	t.NextRun = time.Time{}
	t.FixedRate = false
	t.canceled = false
	t.index = -1
	s.taskPool.Put(t)
}

// Schedule inserts a [Task] into the prioritized execution queue.
//
// It automatically signals the active scheduler loop to recalculate its sleep
// duration if the newly scheduled task has a higher priority (sooner run time)
// than current pending tasks.
//
// # Complexity
//
// Time Complexity: O(log N) where N is the number of scheduled tasks.
func (s *Scheduler) Schedule(t *Task) {
	if s == nil || t == nil {
		return
	}

	s.mu.Lock()
	if t.ID != "" {
		// If a task with the same ID exists, mark the old one as canceled (lazy deletion)
		if old, exists := s.tasksByID[t.ID]; exists {
			old.canceled = true
		}
		s.tasksByID[t.ID] = t
	}

	heap.Push(&s.tasks, t)
	s.mu.Unlock()

	s.triggerWake()
}

// CancelTask prevents a scheduled task with the given ID from executing.
// It uses O(1) lazy cancellation and is safe for concurrent use.
func (s *Scheduler) CancelTask(id string) {
	if id == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if t, exists := s.tasksByID[id]; exists {
		t.canceled = true
		delete(s.tasksByID, id)
	}
}

// Start runs the main scheduler coordination loop.
//
// It blocks the calling goroutine until the provided context is cancelled, at which point
// it waits for all active, currently executing tasks to complete before returning.
//
// Tasks are executed asynchronously in dedicated background goroutines as soon
// as their scheduled run times arrive. Start dynamically scales its sleep intervals
// using internal timers to match changing task deadlines.
func (s *Scheduler) Start(ctx context.Context) {
	var (
		timer     *time.Timer
		timerChan <-chan time.Time
	)

	for {
		s.mu.Lock()

		// Cooperative context cancellation check
		select {
		case <-ctx.Done():
			s.mu.Unlock()
			if timer != nil {
				timer.Stop()
			}
			s.wg.Wait() // Graceful Shutdown: wait for running tasks to finish
			return
		default:
		}

		if len(s.tasks) == 0 {
			s.mu.Unlock()
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				s.wg.Wait()
				return
			case <-s.wakeChan:
				continue
			}
		}

		now := time.Now()
		nextTask := s.tasks[0]

		if now.Before(nextTask.NextRun) {
			delay := nextTask.NextRun.Sub(now)
			if timer == nil {
				timer = time.NewTimer(delay)
			} else {
				timer.Reset(delay)
			}

			timerChan = timer.C
			s.mu.Unlock()

			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				s.wg.Wait()
				return
			case <-timerChan:
				continue
			case <-s.wakeChan:
				if timer != nil && !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				continue
			}
		}

		task := heap.Pop(&s.tasks).(*Task)
		if task.ID != "" {
			delete(s.tasksByID, task.ID)
		}

		// O(1) Lazy cancellation handling
		if task.canceled {
			s.mu.Unlock()
			s.ReleaseTask(task)
			continue
		}

		s.mu.Unlock()

		s.wg.Add(1)
		go s.runTask(ctx, task)
	}
}

func (s *Scheduler) triggerWake() {
	select {
	case s.wakeChan <- struct{}{}:
	default:
	}
}

// runTask executes the work payload of a task, safely recovering from panics.
func (s *Scheduler) runTask(ctx context.Context, t *Task) {
	defer s.wg.Done()

	// Enforce bounded concurrency if configured
	if s.workerSem != nil {
		s.workerSem <- struct{}{}
		defer func() { <-s.workerSem }()
	}

	defer func() {
		// Prevent a single panicking task from crashing the scheduler runtime.
		if r := recover(); r != nil {
			s.rescheduleOrRelease(t)
		}
	}()

	if t.Execute != nil {
		_ = t.Execute(ctx)
	}

	s.rescheduleOrRelease(t)
}

func (s *Scheduler) rescheduleOrRelease(t *Task) {
	if t.Interval > 0 && !t.canceled {
		s.mu.Lock()

		if t.FixedRate {
			// Drift-free: calculate strictly from the last scheduled time
			t.NextRun = t.NextRun.Add(t.Interval)
			// Catch up if falling significantly behind
			if now := time.Now(); t.NextRun.Before(now) {
				t.NextRun = now
			}
		} else {
			// Fixed-delay: calculate from the actual completion time
			t.NextRun = time.Now().Add(t.Interval)
		}

		if t.ID != "" {
			s.tasksByID[t.ID] = t
		}

		heap.Push(&s.tasks, t)
		s.mu.Unlock()

		s.triggerWake()
	} else {
		s.ReleaseTask(t)
	}
}

type taskHeap []*Task

func (h taskHeap) Len() int { return len(h) }

func (h taskHeap) Less(i, j int) bool {
	if h[i].NextRun.Equal(h[j].NextRun) {
		return h[i].Priority > h[j].Priority
	}

	return h[i].NextRun.Before(h[j].NextRun)
}

func (h taskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *taskHeap) Push(x any) {
	n := len(*h)
	item := x.(*Task)
	item.index = n
	*h = append(*h, item)
}

func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]

	return item
}
