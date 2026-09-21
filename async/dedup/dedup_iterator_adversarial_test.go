// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dedup_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/dedup"
)

func TestEmpirical_Dedup_KeysSeq_Empty(t *testing.T) {
	g := &dedup.Group[string, int]{}

	// Traversal on empty group yields zero keys
	for k := range g.KeysSeq() {
		t.Fatalf("expected no keys, got %s", k)
	}

	for k := range dedup.KeysSeq(g) {
		t.Fatalf("expected no keys from package function, got %s", k)
	}
}

func TestEmpirical_Dedup_KeysSeq_InFlightAndEarlyTermination(t *testing.T) {
	g := &dedup.Group[string, int]{}

	release := make(chan struct{})
	var started sync.WaitGroup

	const numKeys = 5
	for i := range numKeys {
		started.Add(1)
		key := fmt.Sprintf("key-%d", i)
		go func(k string) {
			_, _ = g.Do(context.Background(), k, func(ctx context.Context) (int, error) {
				started.Done()
				<-release
				return 42, nil
			})
		}(key)
	}

	// Wait until all 5 are in-flight
	started.Wait()

	// 1. Full traversal
	keys := make(map[string]bool)
	for k := range g.KeysSeq() {
		keys[k] = true
	}
	if len(keys) != numKeys {
		t.Fatalf("expected %d in-flight keys, got %d", numKeys, len(keys))
	}

	// 2. Early termination: break after 0 iterations (first item returns false)
	count := 0
	for range g.KeysSeq() {
		count++
		break
	}
	if count != 1 {
		t.Fatalf("expected count 1 on immediate break, got %d", count)
	}

	// 3. Early termination: break after 2 iterations
	count = 0
	for range g.KeysSeq() {
		count++
		if count == 2 {
			break
		}
	}
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// Release all workers
	close(release)
	// Allow tasks to finish
	time.Sleep(20 * time.Millisecond)

	// After completion, KeysSeq should be empty
	emptyCount := 0
	for range g.KeysSeq() {
		emptyCount++
	}
	if emptyCount != 0 {
		t.Fatalf("expected 0 in-flight keys after completion, got %d", emptyCount)
	}
}

func BenchmarkKeysSeq_Empty(b *testing.B) {
	g := &dedup.Group[string, int]{}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for k := range g.KeysSeq() {
			_ = k
		}
	}
}
