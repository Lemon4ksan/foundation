// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package context_test

import (
	stdctx "context"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"

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

// -----------------------------------------------------------------------------
// BENCHMARKS: FastContext vs Standard stdctx.Context
// -----------------------------------------------------------------------------

// 1. Chained Writes: adding 5 values (Trace, Proxy, Tenant, Timeout, Retry)
func BenchmarkFastContext_ChainWrites_5(b *testing.B) {
	bg := context.Background()

	b.ReportAllocs()

	for b.Loop() {
		ctx := context.WithValue(bg, 1, "trace-123")
		ctx = context.WithValue(ctx, 2, "proxy-us-east")
		ctx = context.WithValue(ctx, 3, "tenant-premium")
		ctx = context.WithValue(ctx, 4, 3000)
		ctx = context.WithValue(ctx, 5, true)
		_ = ctx
	}
}

func BenchmarkStdlibContext_ChainWrites_5(b *testing.B) {
	bg := stdctx.Background()

	b.ReportAllocs()

	for b.Loop() {
		ctx := stdctx.WithValue(bg, 1, "trace-123")
		ctx = stdctx.WithValue(ctx, 2, "proxy-us-east")
		ctx = stdctx.WithValue(ctx, 3, "tenant-premium")
		ctx = stdctx.WithValue(ctx, 4, 3000)
		ctx = stdctx.WithValue(ctx, 5, true)
		_ = ctx
	}
}

// 2. Deep Lookup: reading the FIRST key added (bottom of linked-list for stdctx)
func BenchmarkFastContext_DeepLookup_FirstKey(b *testing.B) {
	ctx := stdctx.Context(context.Background())
	for i := range 6 {
		ctx = context.WithValue(ctx, i, i*100)
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = context.GetOr[int](ctx, 0, 0)
	}
}

func BenchmarkStdlibContext_DeepLookup_FirstKey(b *testing.B) {
	ctx := stdctx.Background()
	for i := range 6 {
		ctx = stdctx.WithValue(ctx, i, i*100)
	}

	b.ReportAllocs()

	for b.Loop() {
		if val, ok := ctx.Value(0).(int); ok {
			_ = val
		}
	}
}

// 3. Full Pipeline Lifecycle: Create -> 5 Middleware Writes -> 5 Pipeline Reads
func BenchmarkFastContext_PipelineLifecycle(b *testing.B) {
	bg := context.Background()
	b.ReportAllocs()

	for b.Loop() {
		// 5 Middleware stages enrich context
		ctx := context.WithValue(bg, 1, "trace-id-abc")
		ctx = context.WithValue(ctx, 2, "proxy-exit-node")
		ctx = context.WithValue(ctx, 3, 42)
		ctx = context.WithValue(ctx, 4, 5000)
		ctx = context.WithValue(ctx, 5, true)

		// 5 Downstream layers read their typed values
		_ = context.GetOr[string](ctx, 1, "")
		_ = context.GetOr[string](ctx, 2, "")
		_ = context.GetOr[int](ctx, 3, 0)
		_ = context.GetOr[int](ctx, 4, 0)
		_ = context.GetOr[bool](ctx, 5, false)
	}
}

func BenchmarkStdlibContext_PipelineLifecycle(b *testing.B) {
	bg := stdctx.Background()
	b.ReportAllocs()

	for b.Loop() {
		// 5 Middleware stages enrich context
		ctx := stdctx.WithValue(bg, 1, "trace-id-abc")
		ctx = stdctx.WithValue(ctx, 2, "proxy-exit-node")
		ctx = stdctx.WithValue(ctx, 3, 42)
		ctx = stdctx.WithValue(ctx, 4, 5000)
		ctx = stdctx.WithValue(ctx, 5, true)

		// 5 Downstream layers read their typed values
		if v, ok := ctx.Value(1).(string); ok {
			_ = v
		}
		if v, ok := ctx.Value(2).(string); ok {
			_ = v
		}
		if v, ok := ctx.Value(3).(int); ok {
			_ = v
		}
		if v, ok := ctx.Value(4).(int); ok {
			_ = v
		}
		if v, ok := ctx.Value(5).(bool); ok {
			_ = v
		}
	}
}

// 4. In-Place Pipeline Lifecycle: 0 ALLOCS inside single-request pipeline
func BenchmarkFastContext_InPlace_PipelineLifecycle(b *testing.B) {
	bg := context.Background()
	b.ReportAllocs()

	for b.Loop() {
		// Single allocation context wrapper at ingress
		ctx := context.Wrap(bg)

		// 5 Middleware stages enrich context in-place (0 ALLOCATIONS)
		ctx.Set(1, "trace-id-abc")
		ctx.Set(2, "proxy-exit-node")
		ctx.Set(3, 42)
		ctx.Set(4, 5000)
		ctx.Set(5, true)

		// 5 Downstream layers read their typed values
		_ = context.GetOr[string](ctx, 1, "")
		_ = context.GetOr[string](ctx, 2, "")
		_ = context.GetOr[int](ctx, 3, 0)
		_ = context.GetOr[int](ctx, 4, 0)
		_ = context.GetOr[bool](ctx, 5, false)
	}
}

// 5. Pooled Pipeline Lifecycle: 0 ALLOCS / 0 B/op (Absolute Zero-Alloc)
func BenchmarkFastContext_Pool_PipelineLifecycle(b *testing.B) {
	pool := context.NewPool()
	bg := stdctx.Background()
	b.ReportAllocs()

	for b.Loop() {
		ctx := pool.Acquire(bg)

		// 5 Middleware stages enrich context in-place
		ctx.Set(1, "trace-id-abc")
		ctx.Set(2, "proxy-exit-node")
		ctx.Set(3, 42)
		ctx.Set(4, 5000)
		ctx.Set(5, true)

		// 5 Downstream layers read their typed values
		_ = context.GetOr[string](ctx, 1, "")
		_ = context.GetOr[string](ctx, 2, "")
		_ = context.GetOr[int](ctx, 3, 0)
		_ = context.GetOr[int](ctx, 4, 0)
		_ = context.GetOr[bool](ctx, 5, false)

		pool.Release(ctx)
	}
}
