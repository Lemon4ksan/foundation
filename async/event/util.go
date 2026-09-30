// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package event

import (
	"context"
	"iter"
	"sync"
)

// SubscribeTo registers a type-safe listener callback running on a dedicated worker goroutine.
// The listener processes events of type T until ctx is cancelled, the bus is closed,
// or the returned cancel function is invoked.
//
// Parameters:
//   - ctx: Parent lifecycle context. If cancelled, the worker stops and subscription unregisters.
//   - bus: Event bus instance to subscribe to. If nil, returns a no-op cancel function.
//   - handler: Callback invoked for each received event of type T. If nil, returns no-op.
//   - bufSize: Optional channel buffer capacity (defaults to foundation's 128).
//
// Concurrency & Lifetime Guarantees:
//   - The returned cancel() function is thread-safe, idempotent, and guarantees immediate
//     unregistration from the event bus and closure of internal subscription channels.
//   - When ctx is cancelled externally, the worker goroutine terminates cleanly and releases
//     the bus subscription via deferred unregistration.
func SubscribeTo[E any](ctx context.Context, bus *Bus, handler func(E), bufSize ...int) (cancel func()) {
	if bus == nil || handler == nil {
		return func() {}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	subCtx, cancelFn := context.WithCancel(ctx)
	sub := SubscribeTyped[E](bus, bufSize...)

	var once sync.Once

	stopAfter := context.AfterFunc(ctx, sub.Unsubscribe)

	cancel = func() {
		once.Do(func() {
			stopAfter()
			cancelFn()
			sub.Unsubscribe()
		})
	}

	go func() {
		defer sub.Unsubscribe()

		for {
			select {
			case <-subCtx.Done():
				return
			case ev, ok := <-sub.C():
				if !ok {
					return
				}

				handler(ev)
			}
		}
	}()

	return cancel
}

// SubscribeSeq returns a Go 1.23+ streaming iterator (iter.Seq[T]) over typed events.
// It enables natural, idiomatic Go for-range iteration:
//
//	for ev := range eventutil.SubscribeSeq[*web.NewOfferEvent](ctx, bus) {
//	    log.Info("offer received", "id", ev.Offer.ID)
//	}
//
// Concurrency & Lifetime Guarantees:
//   - The underlying typed subscription is created immediately upon invocation of SubscribeSeq.
//   - When the loop terminates (via early break, context cancellation, or bus closure),
//     the subscription is guaranteed to be unsubscribed and all channels closed immediately
//     via deferred cleanup inside the iterator function.
//   - If ctx is cancelled before or during iteration, iteration yields zero further items and exits.
func SubscribeSeq[E any](ctx context.Context, bus *Bus, bufSize ...int) iter.Seq[E] {
	if bus == nil {
		return func(yield func(E) bool) {}
	}

	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		return func(yield func(E) bool) {}
	}

	sub := SubscribeTyped[E](bus, bufSize...)
	stop := context.AfterFunc(ctx, sub.Unsubscribe)

	return func(yield func(E) bool) {
		defer stop()
		defer sub.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.C():
				if !ok {
					return
				}

				if !yield(ev) {
					return
				}
			}
		}
	}
}
