// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ctxkit_test

import (
	"context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/ctxkit"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

type keyType string

const (
	keyTraceID keyType = "trace_id"
	keyUserID  keyType = "user_id"
	keyRole    keyType = "role"
)

func TestFastContext_BasicValues(t *testing.T) {
	t.Parallel()

	bg := ctxkit.Background()
	require.NotNil(t, bg)

	todo := ctxkit.TODO()
	require.NotNil(t, todo)

	ctx1 := ctxkit.WithValue(bg, keyTraceID, "trace-123")
	ctx2 := ctxkit.WithValue(ctx1, keyUserID, 42)
	ctx3 := ctxkit.WithValue(ctx2, keyRole, "admin")

	// Verify standard Value lookup
	assert.Equal(t, "trace-123", ctx3.Value(keyTraceID))
	assert.Equal(t, 42, ctx3.Value(keyUserID))
	assert.Equal(t, "admin", ctx3.Value(keyRole))
	assert.Nil(t, ctx3.Value("missing_key"))

	// Verify typed generic.Get
	traceOpt := ctxkit.Get[string](ctx3, keyTraceID)
	assert.True(t, traceOpt.IsPresent())
	val, ok := traceOpt.Value()
	assert.True(t, ok)
	assert.Equal(t, "trace-123", val)

	userOpt := ctxkit.Get[int](ctx3, keyUserID)
	assert.True(t, userOpt.IsPresent())
	assert.Equal(t, 42, ctxkit.GetOr[int](ctx3, keyUserID, 0))

	// Wrong type returns None
	wrongTypeOpt := ctxkit.Get[int](ctx3, keyTraceID)
	assert.False(t, wrongTypeOpt.IsPresent())

	// Missing key returns default
	assert.Equal(t, "guest", ctxkit.GetOr[string](ctx3, "missing", "guest"))

	// Nil context or nil key
	assert.False(t, ctxkit.Get[string](nil, "key").IsPresent())
	assert.False(t, ctxkit.Get[string](ctx3, nil).IsPresent())
}

func TestFastContext_Wrap_And_NilBranches(t *testing.T) {
	t.Parallel()

	// 1. Wrap nil -> Background
	wNil := ctxkit.Wrap(nil)
	assert.NotNil(t, wNil)

	// 2. Wrap already FastContext -> returns same
	wSame := ctxkit.Wrap(wNil)
	assert.Same(t, wNil, wSame)

	// 3. Wrap standard context
	stdCtx := context.WithValue(context.Background(), "k", "v")
	wStd := ctxkit.Wrap(stdCtx)
	assert.Equal(t, "v", wStd.Value("k"))

	// 4. Methods on nil *Context
	var nilCtx *ctxkit.Context
	_, ok := nilCtx.Deadline()
	assert.False(t, ok)
	assert.Nil(t, nilCtx.Done())
	assert.Nil(t, nilCtx.Err())
	assert.Nil(t, nilCtx.Value("k"))
	assert.Nil(t, nilCtx.Set("k", "v"))
}

func TestFastContext_Set_InPlace(t *testing.T) {
	t.Parallel()

	ctx := ctxkit.Background()

	// Set nil key
	ctx.Set(nil, "val")

	// Set within inline capacity (8 items)
	for i := 0; i < 8; i++ {
		ctx.Set(i, i*10)
	}

	// Update existing inline key in-place
	ctx.Set(3, 999)
	assert.Equal(t, 999, ctx.Value(3))

	// Set beyond inline capacity -> into extra slice
	for i := 8; i < 12; i++ {
		ctx.Set(i, i*10)
	}
	assert.Equal(t, 100, ctx.Value(10))

	// Update existing key in extra slice
	ctx.Set(10, 888)
	assert.Equal(t, 888, ctx.Value(10))
}

func TestFastContext_OverflowCapacity(t *testing.T) {
	t.Parallel()

	ctx := context.Context(ctxkit.Background())

	// Add 12 values (exceeding inlineCapacity of 8)
	for i := 0; i < 12; i++ {
		ctx = ctxkit.WithValue(ctx, i, i*10)
	}

	for i := 0; i < 12; i++ {
		assert.Equal(t, i*10, ctxkit.GetOr[int](ctx, i, -1))
	}

	// WithValue from nil parent
	ctxNilParent := ctxkit.WithValue(nil, "key", "val")
	assert.Equal(t, "val", ctxNilParent.Value("key"))
}

func TestFastContext_WithCancelAndTimeout(t *testing.T) {
	t.Parallel()

	t.Run("WithCancel", func(t *testing.T) {
		ctx, cancel := ctxkit.WithCancel(ctxkit.Background())
		assert.Nil(t, ctx.Err())

		cancel()
		assert.ErrorIs(t, ctx.Err(), context.Canceled)
		select {
		case <-ctx.Done():
		default:
			t.Fatal("expected Done channel to be closed")
		}

		// WithCancel nil parent
		ctxNil, cancelNil := ctxkit.WithCancel(nil)
		defer cancelNil()
		assert.NotNil(t, ctxNil)
	})

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := ctxkit.WithTimeout(ctxkit.Background(), 20*time.Millisecond)
		defer cancel()

		time.Sleep(50 * time.Millisecond)
		assert.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)

		// WithTimeout nil parent
		ctxNil, cancelNil := ctxkit.WithTimeout(nil, time.Hour)
		defer cancelNil()
		assert.NotNil(t, ctxNil)
	})

	t.Run("WithDeadline", func(t *testing.T) {
		dl := time.Now().Add(50 * time.Millisecond)
		ctx, cancel := ctxkit.WithDeadline(ctxkit.Background(), dl)
		defer cancel()

		d, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.Equal(t, dl, d)

		// WithDeadline nil parent
		ctxNil, cancelNil := ctxkit.WithDeadline(nil, dl)
		defer cancelNil()
		assert.NotNil(t, ctxNil)
	})
}

func TestFastContext_Pool(t *testing.T) {
	t.Parallel()

	pool := ctxkit.NewPool()

	// Acquire nil parent -> defaults to Background
	cNil := pool.Acquire(nil)
	assert.NotNil(t, cNil)
	pool.Release(cNil)

	// Release nil
	pool.Release(nil)

	// Acquire and populate
	c1 := pool.Acquire(context.Background())
	c1.Set("k1", "v1")
	assert.Equal(t, "v1", c1.Value("k1"))
	pool.Release(c1)

	// Re-acquire must be clean
	c2 := pool.Acquire(context.Background())
	assert.Nil(t, c2.Value("k1"))
	pool.Release(c2)
}

func BenchmarkStdContext_Lookup_Hit(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		ctx = context.WithValue(ctx, i, i*10)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ctx.Value(0)
	}
}

func BenchmarkFastContext_Lookup_Hit(b *testing.B) {
	ctx := context.Context(ctxkit.Background())
	for i := 0; i < 5; i++ {
		ctx = ctxkit.WithValue(ctx, i, i*10)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ctx.Value(0)
	}
}

func BenchmarkStdContext_Lookup_Miss(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		ctx = context.WithValue(ctx, i, i*10)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ctx.Value(999)
	}
}

func BenchmarkFastContext_Lookup_Miss_FastReject(b *testing.B) {
	ctx := context.Context(ctxkit.Background())
	for i := 0; i < 5; i++ {
		ctx = ctxkit.WithValue(ctx, i, i*10)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = ctx.Value(999)
	}
}
