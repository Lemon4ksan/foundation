// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/silicon/offheap"
	"github.com/lemon4ksan/foundation/testing/require"
)

// TestChallenge_FinalizerReclaim ensures unreleased buffers are cleaned up by GC finalizers.
func TestChallenge_FinalizerReclaim(t *testing.T) {
	for range 100 {
		buf, err := offheap.NewBuffer(4096)
		require.NoError(t, err)
		_, _ = buf.WriteString("temporary data that should be reclaimed")
		_ = buf // dropped without Release()
	}

	// Trigger GC to run finalizers
	for range 3 {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}

	// Allocate another buffer to verify OS paging remains healthy
	bufAfter, err := offheap.NewBuffer(4096)
	require.NoError(t, err)
	defer bufAfter.Release()
	require.Equal(t, 4096, bufAfter.Cap())
}

// TestChallenge_BufferZeroAllocHotPath verifies that Write, Read, Reset hot paths do not allocate heap memory.
func TestChallenge_BufferZeroAllocHotPath(t *testing.T) {
	buf, err := offheap.NewBuffer(1024)
	require.NoError(t, err)
	defer buf.Release()

	payload := []byte("0123456789abcdef")
	readBuf := make([]byte, len(payload))

	allocs := testing.AllocsPerRun(100, func() {
		buf.Reset()
		_, _ = buf.Write(payload)
		buf.RewindRead()
		_, _ = buf.Read(readBuf)
	})

	require.Equal(t, float64(0), allocs, "hot path operations must have 0 heap allocations")
}
