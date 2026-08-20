// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package limiter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
)

// AdaptiveLimiter controls concurrency dynamically using a Vegas-style congestion algorithm.
// It measures round-trip times (RTT) to dynamically adjust the allowed concurrent request limit.
//
// An AdaptiveLimiter is safe for concurrent use by multiple goroutines.
type AdaptiveLimiter struct {
	mu          sync.Mutex
	limit       float64
	minLimit    float64
	maxLimit    float64
	alpha, beta float64
	active      int
	waitChs     []chan struct{}
	minRTT      time.Duration
	smoothedRTT time.Duration
	lastReset   time.Time
}

// NewAdaptiveLimiter initializes an [AdaptiveLimiter] instance with default settings.
func NewAdaptiveLimiter(initialLimit float64) *AdaptiveLimiter {
	return &AdaptiveLimiter{
		limit:     initialLimit,
		minLimit:  1.0,
		maxLimit:  1000.0,
		alpha:     2.0,
		beta:      5.0,
		lastReset: time.Now(),
	}
}

// Acquire blocks until a concurrent execution slot becomes available or context is cancelled.
//
// It returns nil on successful acquisition. It returns [context.Canceled] or
// [context.DeadlineExceeded] if the context expires before a slot becomes available.
func (l *AdaptiveLimiter) Acquire(ctx context.Context) error {
	l.mu.Lock()
	if l.active < int(l.limit) {
		l.active++
		l.mu.Unlock()
		return nil
	}

	ch := make(chan struct{})
	l.waitChs = append(l.waitChs, ch)
	l.mu.Unlock()

	select {
	case <-ctx.Done():
		l.mu.Lock()

		found := false
		for i, w := range l.waitChs {
			if w == ch {
				l.waitChs = append(l.waitChs[:i], l.waitChs[i+1:]...)
				found = true
				break
			}
		}

		l.mu.Unlock()

		if !found {
			// The slot was already allocated to us by Release before we processed
			// the cancellation. We must return nil to ensure this slot is eventually released.
			return nil
		}

		return ctx.Err()

	case <-ch:
		return nil
	}
}

// Release registers request completion, updates RTT metrics, and recalculates limits.
//
// It adjusts the concurrency limit based on Vegas queuing limits (alpha and beta thresholds)
// and unblocks waiting goroutines if slots become available.
func (l *AdaptiveLimiter) Release(rtt time.Duration) {
	// Defensive check: sanitise input RTT to prevent division-by-zero or NaN propagation.
	if rtt <= 0 {
		rtt = time.Nanosecond
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.active--

	if time.Since(l.lastReset) > 30*time.Second {
		l.minRTT = 0
		l.lastReset = time.Now()
	}

	if l.minRTT == 0 || rtt < l.minRTT {
		l.minRTT = rtt
	}

	if l.smoothedRTT <= 0 {
		l.smoothedRTT = rtt
	} else {
		l.smoothedRTT = time.Duration(0.9*float64(l.smoothedRTT) + 0.1*float64(rtt))
	}

	queue := l.limit * (1.0 - float64(l.minRTT)/float64(l.smoothedRTT))

	if queue > l.beta {
		l.limit = max(l.minLimit, l.limit-1.0)
	} else if queue < l.alpha {
		l.limit = min(l.maxLimit, l.limit+1.0)
	}

	slots := int(l.limit) - l.active
	for slots > 0 && len(l.waitChs) > 0 {
		ch := l.waitChs[0]
		l.waitChs = l.waitChs[1:]

		close(ch)

		l.active++
		slots--
	}
}

// Limit returns the active dynamic concurrency limit.
func (l *AdaptiveLimiter) Limit() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

// ErrClosed is returned when operations are performed on a closed [KeyedLimiter].
var ErrClosed = errors.New("limiter is closed")

type limiterEntry struct {
	limiter   *rate.Limiter
	lastTouch time.Time
}

// KeyedLimiter manages dynamic rate limiters per key, automatically cleaning up
// inactive limiters from memory after a configured TTL duration.
//
// A KeyedLimiter is safe for concurrent use by multiple goroutines.
type KeyedLimiter[K comparable] struct {
	mu       sync.Mutex
	r        rate.Limit
	b        int
	ttl      time.Duration
	limiters map[K]*limiterEntry
	closeCh  chan struct{}
	wg       sync.WaitGroup
	closed   bool
}

// NewKeyedLimiter creates a new [KeyedLimiter] with the specified rate limit, burst size,
// and TTL for inactive limiters. It starts a background sweeper goroutine to clean up
// expired entries.
func NewKeyedLimiter[K comparable](r rate.Limit, b int, ttl time.Duration) *KeyedLimiter[K] {
	kl := &KeyedLimiter[K]{
		r:        r,
		b:        b,
		ttl:      ttl,
		limiters: make(map[K]*limiterEntry),
		closeCh:  make(chan struct{}),
	}

	kl.wg.Go(kl.sweepLoop)

	return kl
}

func (kl *KeyedLimiter[K]) sweepLoop() {
	sweepInterval := max(kl.ttl/2, time.Millisecond)

	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-kl.closeCh:
			return
		case <-ticker.C:
			kl.sweep()
		}
	}
}

func (kl *KeyedLimiter[K]) sweep() {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	now := time.Now()

	for k, entry := range kl.limiters {
		if now.Sub(entry.lastTouch) > kl.ttl {
			delete(kl.limiters, k)
		}
	}
}

// getLimiter retrieves or creates a [rate.Limiter] for the given key.
func (kl *KeyedLimiter[K]) getLimiter(key K) (*rate.Limiter, error) {
	kl.mu.Lock()
	defer kl.mu.Unlock()

	if kl.closed {
		return nil, ErrClosed
	}

	entry, ok := kl.limiters[key]
	if !ok {
		entry = &limiterEntry{
			limiter: rate.NewLimiter(kl.r, kl.b),
		}
		kl.limiters[key] = entry
	}

	entry.lastTouch = time.Now()

	return entry.limiter, nil
}

// Allow reports whether an event may occur for the given key immediately.
func (kl *KeyedLimiter[K]) Allow(key K) (bool, error) {
	lim, err := kl.getLimiter(key)
	if err != nil {
		return false, err
	}

	return lim.Allow(), nil
}

// Wait blocks until the rate limiter allows an event to occur for the key,
// or until the context is cancelled.
func (kl *KeyedLimiter[K]) Wait(ctx context.Context, key K) error {
	lim, err := kl.getLimiter(key)
	if err != nil {
		return err
	}

	return lim.Wait(ctx)
}

// Close stops the background sweeper and marks the limiter as closed.
//
// Subsequent operations on [KeyedLimiter] will return [ErrClosed].
// Close is idempotent and safe to call concurrently.
func (kl *KeyedLimiter[K]) Close() error {
	kl.mu.Lock()
	if kl.closed {
		kl.mu.Unlock()
		return nil
	}

	kl.closed = true

	close(kl.closeCh)
	kl.mu.Unlock()

	kl.wg.Wait()

	return nil
}

// Len returns the number of active key limiters currently stored in memory.
func (kl *KeyedLimiter[K]) Len() int {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	return len(kl.limiters)
}
