// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package limiter

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
)

// Adaptive dynamically regulates concurrency utilizing a Vegas-style congestion avoidance algorithm.
//
// Its zero value is not valid; use [NewAdaptive] to initialize.
type Adaptive struct {
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

// NewAdaptive returns an [Adaptive] concurrency limiter initialized with initialLimit.
func NewAdaptive(initialLimit float64) *Adaptive {
	return &Adaptive{
		limit:     initialLimit,
		minLimit:  1.0,
		maxLimit:  1000.0,
		alpha:     2.0,
		beta:      5.0,
		lastReset: time.Now(),
	}
}

// Acquire blocks until a concurrency slot becomes available or ctx expires.
// It returns nil on success, or ctx.Err() if the context expires before a slot is available.
// Acquire panics if ctx is nil.
func (l *Adaptive) Acquire(ctx context.Context) error {
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
				copy(l.waitChs[i:], l.waitChs[i+1:])
				l.waitChs[len(l.waitChs)-1] = nil
				l.waitChs = l.waitChs[:len(l.waitChs)-1]
				found = true
				break
			}
		}

		l.mu.Unlock()

		if !found {
			// The slot was already allocated to us by Release before we processed
			// the cancellation. Release the allocated slot back so it's not leaked.
			l.Release(0)

			return ctx.Err()
		}

		return ctx.Err()

	case <-ch:
		return nil
	}
}

// Release returns a concurrency slot and updates the Vegas congestion control window with the observed rtt.
func (l *Adaptive) Release(rtt time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.active > 0 {
		l.active--
	}

	if rtt > 0 {
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
	}

	slots := int(l.limit) - l.active
	for slots > 0 && len(l.waitChs) > 0 {
		ch := l.waitChs[0]
		copy(l.waitChs, l.waitChs[1:])
		l.waitChs[len(l.waitChs)-1] = nil
		l.waitChs = l.waitChs[:len(l.waitChs)-1]

		close(ch)

		l.active++
		slots--
	}
}

// Limit returns the currently active dynamic concurrency limit.
func (l *Adaptive) Limit() float64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.limit
}

// ErrClosed is returned when operations are attempted on a closed [Keyed] rate limiter.
var ErrClosed = errors.New("limiter is closed")

type limiterEntry struct {
	limiter   *rate.Limiter
	lastTouch time.Time
}

// Keyed manages dynamic rate limiters per key, automatically evicting inactive limiters from memory after a configured TTL duration.
//
// Its zero value is not valid; use [NewKeyed] to initialize.
// The background sweeper may prematurely evict a limiter if a request blocks in [Keyed.Wait] longer than the configured TTL, causing subsequent requests for the same key to allocate a new limiter and temporarily violate specified rate limits.
type Keyed[K comparable] struct {
	mu       sync.Mutex
	r        rate.Limit
	b        int
	ttl      time.Duration
	limiters map[K]*limiterEntry
	closeCh  chan struct{}
	wg       sync.WaitGroup
	closed   bool
}

// NewKeyed returns a [Keyed] limiter maintaining rate limit r and burst b per key,
// evicting inactive limiters after ttl.
func NewKeyed[K comparable](r rate.Limit, b int, ttl time.Duration) *Keyed[K] {
	kl := &Keyed[K]{
		r:        r,
		b:        b,
		ttl:      ttl,
		limiters: make(map[K]*limiterEntry),
		closeCh:  make(chan struct{}),
	}

	kl.wg.Go(kl.sweepLoop)

	return kl
}

func (kl *Keyed[K]) sweepLoop() {
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

func (kl *Keyed[K]) sweep() {
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
func (kl *Keyed[K]) getLimiter(key K) (*rate.Limiter, error) {
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

// Allow reports whether an event for key is permitted immediately, returning [ErrClosed] if closed.
func (kl *Keyed[K]) Allow(key K) (bool, error) {
	lim, err := kl.getLimiter(key)
	if err != nil {
		return false, err
	}

	return lim.Allow(), nil
}

// Wait blocks until an event for key is permitted or ctx expires.
// It returns [ErrClosed] if closed. Wait panics if ctx is nil.
func (kl *Keyed[K]) Wait(ctx context.Context, key K) error {
	lim, err := kl.getLimiter(key)
	if err != nil {
		return err
	}

	return lim.Wait(ctx)
}

// Close stops the background sweeper and marks the limiter as permanently closed.
func (kl *Keyed[K]) Close() error {
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

// Len returns the count of active, unexpired key limiters currently in memory.
func (kl *Keyed[K]) Len() int {
	kl.mu.Lock()
	defer kl.mu.Unlock()
	return len(kl.limiters)
}

// SlidingWindow implements a sliding window log rate limiter.
//
// Its zero value is not valid; use [NewSlidingWindow] to initialize.
type SlidingWindow struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}

// NewSlidingWindow returns a [SlidingWindow] rate limiter for limit events per window duration.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
	if limit < 0 {
		limit = 0
	}

	return &SlidingWindow{
		limit:      limit,
		window:     window,
		timestamps: make([]time.Time, 0, limit),
	}
}

// Allow reports whether an event at time now is permitted within the sliding window,
// returning (true, 0) if allowed or (false, retryAfter) if rate-limited.
func (l *SlidingWindow) Allow(now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.limit <= 0 {
		return false, l.window
	}

	cutoff := now.Add(-l.window)

	validIdx := 0
	for i, ts := range l.timestamps {
		if ts.After(cutoff) {
			validIdx = i
			break
		}

		if i == len(l.timestamps)-1 {
			validIdx = len(l.timestamps)
		}
	}

	if validIdx > 0 {
		l.timestamps = slices.Delete(l.timestamps, 0, validIdx)
	}

	if len(l.timestamps) < l.limit {
		l.timestamps = append(l.timestamps, now)
		return true, 0
	}

	oldest := l.timestamps[0]
	waitTime := max(oldest.Add(l.window).Sub(now), 0)

	return false, waitTime
}
