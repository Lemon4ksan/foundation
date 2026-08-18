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
	// ID is the unique identifier of the task.
	ID string
	// Priority determines execution precedence when tasks are scheduled for the exact same time.
	Priority Priority
	// NextRun is the absolute time when the task is scheduled to execute next.
	NextRun time.Time
	// Interval is the duration between execution cycles for periodic tasks.
	// Set to <= 0 for one-off tasks.
	Interval time.Duration
	// Execute is the context-aware function executed by the scheduler.
	Execute func(ctx context.Context) error

	index int
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

		if timer != nil {
			timer.Stop()
		}

		timer = time.AfterFunc(interval, fn)
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

// Scheduler coordinates prioritized, time-exact task execution.
//
// Create new instances of Scheduler using the [New] constructor function.
// All scheduler operations are safe for concurrent use by multiple goroutines.
type Scheduler struct {
	mu       sync.Mutex
	tasks    taskHeap
	wakeChan chan struct{}
	taskPool sync.Pool
}

// New initializes and returns a new [Scheduler] instance.
func New() *Scheduler {
	s := &Scheduler{
		wakeChan: make(chan struct{}, 1),
	}
	s.taskPool.New = func() any {
		return &Task{index: -1}
	}

	return s
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
	t.ID = ""
	t.Execute = nil
	t.Interval = 0
	t.NextRun = time.Time{}
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
	s.mu.Lock()
	heap.Push(&s.tasks, t)
	s.mu.Unlock()

	s.triggerWake()
}

// Start runs the main scheduler coordination loop.
//
// It blocks the calling goroutine until the provided context is cancelled.
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

				return

			case <-timerChan:
				continue

			case <-s.wakeChan:
				// If woken up by a newly scheduled high-priority task,
				// stop and drain the active timer to prevent premature triggering.
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
		s.mu.Unlock()

		go s.runTask(ctx, task)
	}
}

// triggerWake non-blockingly signals the active Start loop to wake up and reassess.
func (s *Scheduler) triggerWake() {
	select {
	case s.wakeChan <- struct{}{}:
	default:
	}
}

// runTask executes the work payload of a task, safely recovering from panics.
func (s *Scheduler) runTask(ctx context.Context, t *Task) {
	defer func() {
		// Prevent a single panicking task from crashing the scheduler runtime.
		if r := recover(); r != nil {
			if t.Interval > 0 {
				s.mu.Lock()
				t.NextRun = time.Now().Add(t.Interval)
				heap.Push(&s.tasks, t)
				s.mu.Unlock()
				s.triggerWake()
			} else {
				s.ReleaseTask(t)
			}
		}
	}()

	if t.Execute != nil {
		_ = t.Execute(ctx)
	}

	if t.Interval > 0 {
		s.mu.Lock()
		t.NextRun = time.Now().Add(t.Interval)
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
