# Tactical Synchronization (`sync`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/sync)

`sync` provides hardened synchronization, locking, rate limiting, circuit breaking, and backoff primitives designed for high-concurrency environments.

## Motivation & Problem Context

High-concurrency synchronization frequently exposes limitations in standard primitives. Locking resources per unique key via `map[string]*sync.Mutex` leaks memory over time as keys accumulate without automated cleanup. Static concurrency limits fail to adapt to downstream latency degradation, and unhandled network failures risk cascading outages across microservice boundaries.

## 1. Striped Key Locking (`keylock`)

### Mechanics
`keylock.KeyMutex` maintains reference counters for active keys. When the reference count drops to 0 upon unlock, the entry is automatically deleted from memory.

```go
package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/sync/keylock"
)

func main() {
	lock := keylock.New[string]()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Serializes modifications for user "usr_100"
			lock.Lock("usr_100")
			defer lock.Unlock("usr_100") // Auto-reclaims memory when last goroutine unlocks!

			fmt.Printf("Worker %d in critical section for usr_100\n", workerID)
			time.Sleep(10 * time.Millisecond)
		}(i)
	}
	wg.Wait()
}
```

## 2. Rate Limiting (`limiter`)

### Vegas Congestion Adaptive Limiter (RTT-based)
Static rate limiters either underutilize bandwidth or overwhelm downstream services during degradation. Vegas measures latency (RTT) and automatically contracts limits during latency spikes.

```go
import "github.com/lemon4ksan/foundation/sync/limiter"

adLim := limiter.NewAdaptiveLimiter(10.0) // Start with initial limit of 10

func HandleRequest(ctx context.Context) error {
	if err := adLim.Acquire(ctx); err != nil {
		return err
	}

	start := time.Now()
	err := callExternalAPI(ctx)
	rtt := time.Since(start)

	// Release slot and adjust capacity based on latency
	adLim.Release(rtt)
	return err
}
```

### Keyed Rate Limiter with Auto-TTL Cleanup
Rate limiting users by IP address without leaking memory over millions of unique IPs.

```go
import (
	"golang.org/x/time/rate"
	"github.com/lemon4ksan/foundation/sync/limiter"
)

// 10 req/sec, burst 20, TTL 5 minutes of inactivity
kl := limiter.NewKeyedLimiter[string](rate.Limit(10), 20, 5*time.Minute)
defer kl.Close()

func ProcessUserAPI(ctx context.Context, ip string) error {
	return kl.Wait(ctx, ip)
}
```

## 3. Generics Circuit Breaker (`breaker`)

Compile-time type-safe Circuit Breaker preventing cascading microservice outages with Closed, Open, and Half-Open trial transitions.

```go
import (
	"errors"
	"time"

	"github.com/lemon4ksan/foundation/sync/breaker"
)

cb := breaker.New[UserProfile](breaker.Config{
	FailureThreshold: 0.5,              // Trip if 50% of calls fail
	Cooldown:         10 * time.Second, // Test Half-Open after 10s
	MinRequests:      5,                // Minimum sample size
})

profile, err := cb.Do(ctx, func(ctx context.Context) (UserProfile, error) {
	return fetchRemoteProfile(ctx)
})
if err != nil {
	if errors.Is(err, breaker.ErrCircuitOpen) {
		// Fast-path fallback
		return getCachedProfile(), nil
	}
	return UserProfile{}, err
}
```

## 4. Exponential Backoff with Jitter (`backoff`)

Mathematical backoff calculators with Full, Equal, and Decorrelated Jitter distributions to prevent thundering herd spikes.

```go
import (
	"time"

	"github.com/lemon4ksan/foundation/sync/backoff"
)

b := backoff.New(
	backoff.WithBaseDelay(100*time.Millisecond),
	backoff.WithMaxDelay(10*time.Second),
	backoff.WithJitter(backoff.JitterFull),
)

for attempt := 0; attempt < 5; attempt++ {
	err := tryConnect()
	if err == nil {
		break
	}
	delay := b.Delay(attempt)
	time.Sleep(delay)
}
```

## 5. Primitives (`semaphore`, `lazy`, `spinlock`)

* **`semaphore.Semaphore`**: Cancellable semaphore with runtime `.Resize(n)` capability.
* **`lazy.Lazy`**: Thread-safe lazy initializer with `.Reset()` support allowing re-initialization after network failures.
* **`spinlock.SpinLock`**: Low-overhead busy-waiting lock for nanosecond-critical sections.
