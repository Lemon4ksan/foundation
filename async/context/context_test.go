// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package context_test

import (
	stdctx "context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lemon4ksan/foundation/async/context"
)

type keyType string

const (
	keyTraceID keyType = "trace_id"
	keyUserID  keyType = "user_id"
	keyRole    keyType = "role"
)

func TestFastContext_BasicValues(t *testing.T) {
	t.Parallel()

	bg := context.Background()
	require.NotNil(t, bg)

	ctx1 := context.WithValue(bg, keyTraceID, "trace-123")
	ctx2 := context.WithValue(ctx1, keyUserID, 42)
	ctx3 := context.WithValue(ctx2, keyRole, "admin")

	// Verify standard Value lookup
	assert.Equal(t, "trace-123", ctx3.Value(keyTraceID))
	assert.Equal(t, 42, ctx3.Value(keyUserID))
	assert.Equal(t, "admin", ctx3.Value(keyRole))
	assert.Nil(t, ctx3.Value("missing_key"))

	// Verify typed generic.Get
	traceOpt := context.Get[string](ctx3, keyTraceID)
	assert.True(t, traceOpt.IsPresent())
	val, ok := traceOpt.Value()
	assert.True(t, ok)
	assert.Equal(t, "trace-123", val)

	userOpt := context.Get[int](ctx3, keyUserID)
	assert.True(t, userOpt.IsPresent())
	assert.Equal(t, 42, context.GetOr[int](ctx3, keyUserID, 0))

	// Wrong type returns None
	wrongTypeOpt := context.Get[int](ctx3, keyTraceID)
	assert.False(t, wrongTypeOpt.IsPresent())

	// Missing key returns default
	assert.Equal(t, "guest", context.GetOr[string](ctx3, "missing", "guest"))
}

func TestFastContext_OverflowCapacity(t *testing.T) {
	t.Parallel()

	ctx := stdctx.Context(context.Background())

	// Add 12 values (exceeding inlineCapacity of 8)
	for i := 0; i < 12; i++ {
		ctx = context.WithValue(ctx, i, i*10)
	}

	for i := 0; i < 12; i++ {
		assert.Equal(t, i*10, context.GetOr[int](ctx, i, -1))
	}
}

func TestFastContext_WithCancelAndTimeout(t *testing.T) {
	t.Parallel()

	t.Run("WithCancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		assert.Nil(t, ctx.Err())

		cancel()
		assert.ErrorIs(t, ctx.Err(), stdctx.Canceled)
		select {
		case <-ctx.Done():
		default:
			t.Fatal("expected Done channel to be closed")
		}
	})

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		time.Sleep(50 * time.Millisecond)
		assert.ErrorIs(t, ctx.Err(), stdctx.DeadlineExceeded)
	})
}

func TestFastContext_Pool(t *testing.T) {
	t.Parallel()

	pool := context.NewPool()
	fc := pool.Acquire(stdctx.Background())
	require.NotNil(t, fc)

	ctx := context.WithValue(fc, "key", "val")
	assert.Equal(t, "val", context.GetOr[string](ctx, "key", ""))

	pool.Release(fc)
}

func BenchmarkFastContext_WithValue_Lookup(b *testing.B) {
	ctx := stdctx.Context(context.Background())
	for i := 0; i < 5; i++ {
		ctx = context.WithValue(ctx, i, i*100)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = context.GetOr[int](ctx, 2, 0)
	}
}

func BenchmarkStdlibContext_WithValue_Lookup(b *testing.B) {
	ctx := stdctx.Background()
	for i := 0; i < 5; i++ {
		ctx = stdctx.WithValue(ctx, i, i*100)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if val, ok := ctx.Value(2).(int); ok {
			_ = val
		}
	}
}
