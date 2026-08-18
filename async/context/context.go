// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package context

import (
	stdctx "context"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/generic"
)

const inlineCapacity = 8

type kvEntry struct {
	key   any
	value any
}

// Context is a high-performance, flat-array implementation of [stdctx.Context].
//
// Unlike standard [stdctx.WithValue], which forms deep O(N) linked lists in the heap,
// [Context] aggregates key-value pairs into a single, contiguous array buffer.
//
// For up to 8 key-value pairs, [Context] requires zero slice allocations and performs
// value resolution with L1/L2 cache locality.
type Context struct {
	parent stdctx.Context
	inline [inlineCapacity]kvEntry
	extra  []kvEntry
	count  int
}

// Background returns a non-nil, empty [Context] wrapping [stdctx.Background].
func Background() *Context {
	return &Context{parent: stdctx.Background()}
}

// TODO returns a non-nil, empty [Context] wrapping [stdctx.TODO].
func TODO() *Context {
	return &Context{parent: stdctx.TODO()}
}

// Wrap converts any standard [stdctx.Context] into a [Context].
// If parent is already a [*Context], it is returned as is.
func Wrap(parent stdctx.Context) *Context {
	if parent == nil {
		parent = stdctx.Background()
	}

	if fc, ok := parent.(*Context); ok {
		return fc
	}

	return &Context{parent: parent}
}

// Deadline returns the time when work done on behalf of this context should be canceled.
func (c *Context) Deadline() (deadline time.Time, ok bool) {
	if c == nil || c.parent == nil {
		return time.Time{}, false
	}

	return c.parent.Deadline()
}

// Done returns a channel that's closed when work done on behalf of this context should be canceled.
func (c *Context) Done() <-chan struct{} {
	if c == nil || c.parent == nil {
		return nil
	}

	return c.parent.Done()
}

// Err returns a non-nil error value after [Done] is closed.
func (c *Context) Err() error {
	if c == nil || c.parent == nil {
		return nil
	}

	return c.parent.Err()
}

// Value returns the value associated with this context for key, or nil if no value is associated.
func (c *Context) Value(key any) any {
	if c == nil || key == nil {
		return nil
	}

	for i := len(c.extra) - 1; i >= 0; i-- {
		if c.extra[i].key == key {
			return c.extra[i].value
		}
	}

	limit := min(c.count, inlineCapacity)

	for i := limit - 1; i >= 0; i-- {
		if c.inline[i].key == key {
			return c.inline[i].value
		}
	}

	if c.parent != nil {
		return c.parent.Value(key)
	}

	return nil
}

// WithValue returns a copy of parent carrying key and val in a flat structure.
func WithValue(parent stdctx.Context, key any, val any) *Context {
	if parent == nil {
		parent = stdctx.Background()
	}

	fc, ok := parent.(*Context)
	if !ok {
		fc = &Context{parent: parent}
	}

	// Clone flat context to guarantee immutability across branches
	clone := &Context{
		parent: fc.parent,
		count:  fc.count + 1,
	}
	copy(clone.inline[:], fc.inline[:])

	if len(fc.extra) > 0 {
		clone.extra = append(clone.extra, fc.extra...)
	}

	if fc.count < inlineCapacity {
		clone.inline[fc.count] = kvEntry{key: key, value: val}
	} else {
		clone.extra = append(clone.extra, kvEntry{key: key, value: val})
	}

	return clone
}

// WithCancel returns a copy of parent with a new Done channel.
func WithCancel(parent stdctx.Context) (*Context, stdctx.CancelFunc) {
	if parent == nil {
		parent = stdctx.Background()
	}

	cancelCtx, cancel := stdctx.WithCancel(parent)
	return Wrap(cancelCtx), cancel
}

// WithTimeout returns a copy of parent with a deadline set to timeout from now.
func WithTimeout(parent stdctx.Context, timeout time.Duration) (*Context, stdctx.CancelFunc) {
	if parent == nil {
		parent = stdctx.Background()
	}

	timeoutCtx, cancel := stdctx.WithTimeout(parent, timeout)
	return Wrap(timeoutCtx), cancel
}

// WithDeadline returns a copy of parent with the deadline adjusted to be no later than d.
func WithDeadline(parent stdctx.Context, d time.Time) (*Context, stdctx.CancelFunc) {
	if parent == nil {
		parent = stdctx.Background()
	}

	deadlineCtx, cancel := stdctx.WithDeadline(parent, d)
	return Wrap(deadlineCtx), cancel
}

// Get extracts a typed value from ctx wrapped in a [generic.Optional].
//
// If the key is not found or cannot be safely cast to T, it returns a [generic.None].
func Get[T any](ctx stdctx.Context, key any) generic.Optional[T] {
	if ctx == nil || key == nil {
		return generic.None[T]()
	}

	raw := ctx.Value(key)
	if raw == nil {
		return generic.None[T]()
	}

	typed, ok := raw.(T)
	if !ok {
		return generic.None[T]()
	}

	return generic.Some(typed)
}

// GetOr extracts a typed value from ctx, or returns defaultVal if missing or of mismatching type.
func GetOr[T any](ctx stdctx.Context, key any, defaultVal T) T {
	return Get[T](ctx, key).ValueOr(defaultVal)
}

// Pool provides object pooling for [Context] instances in extreme high-throughput pipelines.
type Pool struct {
	pool sync.Pool
}

// NewPool constructs a reusable [Pool] of [Context] instances.
func NewPool() *Pool {
	return &Pool{
		pool: sync.Pool{
			New: func() any {
				return &Context{}
			},
		},
	}
}

// Acquire gets a clean [Context] from the pool with parent set.
func (p *Pool) Acquire(parent stdctx.Context) *Context {
	if parent == nil {
		parent = stdctx.Background()
	}

	fc := p.pool.Get().(*Context)
	fc.parent = parent
	fc.count = 0
	fc.extra = fc.extra[:0]
	clear(fc.inline[:])

	return fc
}

// Release resets and returns fc back to the pool.
func (p *Pool) Release(fc *Context) {
	if fc == nil {
		return
	}

	fc.parent = nil
	fc.count = 0
	fc.extra = fc.extra[:0]
	clear(fc.inline[:])

	p.pool.Put(fc)
}
