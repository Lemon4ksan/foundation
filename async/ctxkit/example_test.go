package ctxkit_test

import (
	"context"
	"fmt"

	"github.com/lemon4ksan/foundation/async/ctxkit"
)

type TraceInfo struct {
	TraceID string
	SpanID  string
}

func ExampleContext_Set() {
	// Wrap standard context once at pipeline ingress
	ctx := ctxkit.Wrap(context.Background())

	// Set mutates the context in-place with zero allocations
	ctx.Set("auth_token", "jwt-header-xyz")
	ctx.Set("retry_count", 3)
	ctx.Set("is_premium", true)

	fmt.Println(ctxkit.ValueOr(ctx, "retry_count", 0))
	// Output: 3
}

func ExampleValue() {
	ctx := context.Background()
	ctx = ctxkit.WithValue(ctx, "trace_key", TraceInfo{TraceID: "abc-123", SpanID: "span-456"})

	// Read typed value using comma-ok idiom
	if trace, ok := ctxkit.Value[TraceInfo](ctx, "trace_key"); ok {
		fmt.Println("Trace ID:", trace.TraceID)
	}

	// Read with fallback default
	tenantID := ctxkit.ValueOr[string](ctx, "tenant_key", "default-tenant")
	fmt.Println("Tenant ID:", tenantID)

	// Output:
	// Trace ID: abc-123
	// Tenant ID: default-tenant
}

func ExamplePool() {
	pool := ctxkit.NewPool()

	// Acquire a clean context wrapped around a parent
	ctx := pool.Acquire(context.Background())
	defer pool.Release(ctx)

	ctx.Set("conn_id", 9876)
	fmt.Println(ctxkit.ValueOr(ctx, "conn_id", 0))

	// Output: 9876
}
