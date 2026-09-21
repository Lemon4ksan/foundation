// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lazy

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

type stressPayload struct {
	id       int64
	data     [256]byte
	checksum [32]byte
}

func makePayload(id int64) stressPayload {
	var p stressPayload
	p.id = id
	for i := range p.data {
		p.data[i] = byte(id + int64(i))
	}
	h := sha256.New()
	var idBytes [8]byte
	binary.LittleEndian.PutUint64(idBytes[:], uint64(id))
	h.Write(idBytes[:])
	h.Write(p.data[:])
	copy(p.checksum[:], h.Sum(nil))
	return p
}

func verifyPayload(p stressPayload) bool {
	h := sha256.New()
	var idBytes [8]byte
	binary.LittleEndian.PutUint64(idBytes[:], uint64(p.id))
	h.Write(idBytes[:])
	h.Write(p.data[:])
	expected := h.Sum(nil)
	for i := range 32 {
		if p.checksum[i] != expected[i] {
			return false
		}
	}
	return true
}

// TestLazy_HighContentionRace verifies that concurrent Get and Reset cycles
// under extreme contention never exhibit word tearing, partial struct publication,
// or data races.
func TestLazy_HighContentionRace(t *testing.T) {
	var seq atomic.Int64

	l := New(func() (stressPayload, error) {
		id := seq.Add(1)
		return makePayload(id), nil
	})

	const (
		numReaders   = 64
		numResetters = 16
		durationOps  = 500
	)

	stop := make(chan struct{})
	var wg sync.WaitGroup

	// Readers
	for r := range numReaders {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					p, err := l.Get()
					if err != nil {
						t.Errorf("reader %d: unexpected error: %v", readerID, err)
						return
					}
					if !verifyPayload(p) {
						t.Errorf("reader %d: corrupted payload observed! word tearing detected: id=%d", readerID, p.id)
						return
					}
				}
			}
		}(r)
	}

	// Resetters
	for range numResetters {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					l.Reset()
				}
			}
		}()
	}

	// Main controller drives progress
	for range durationOps {
		p, err := l.Get()
		if err != nil {
			t.Fatalf("main: unexpected error: %v", err)
		}
		if !verifyPayload(p) {
			t.Fatalf("main: corrupted payload observed! id=%d", p.id)
		}
	}

	close(stop)
	wg.Wait()
}

// TestLazy_ErrorCachingAndReset verifies error caching semantics under concurrent access.
func TestLazy_ErrorCachingAndReset(t *testing.T) {
	expectedErr := errors.New("transient failure")
	var shouldFail atomic.Bool
	shouldFail.Store(true)
	var callCount atomic.Int32

	l := New(func() (string, error) {
		callCount.Add(1)
		if shouldFail.Load() {
			return "", expectedErr
		}
		return "success", nil
	})

	// Concurrent reads while failing
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := l.Get()
			if !errors.Is(err, expectedErr) {
				t.Errorf("expected %v, got %v", expectedErr, err)
			}
			if val != "" {
				t.Errorf("expected empty string, got %q", val)
			}
		}()
	}
	wg.Wait()

	if callCount.Load() != 1 {
		t.Fatalf("expected init to be called once while cached, got %d", callCount.Load())
	}

	// Reset and transition to success
	shouldFail.Store(false)
	l.Reset()

	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			val, err := l.Get()
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if val != "success" {
				t.Errorf("expected 'success', got %q", val)
			}
		}()
	}
	wg.Wait()

	if callCount.Load() != 2 {
		t.Fatalf("expected init to be called exactly twice across resets, got %d", callCount.Load())
	}
}

// TestLazy_PanicSafety verifies that if init panics, the lock is properly
// released and does not deadlock future callers.
func TestLazy_PanicSafety(t *testing.T) {
	var shouldPanic atomic.Bool
	shouldPanic.Store(true)

	l := New(func() (int, error) {
		if shouldPanic.Load() {
			panic("init panicked")
		}
		return 42, nil
	})

	// First call panics
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic, got nil")
			}
		}()
		_, _ = l.Get()
	}()

	// Second call with shouldPanic = false should acquire lock and succeed without deadlocking
	shouldPanic.Store(false)
	val, err := l.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}
}

// TestLazy_FastPathZeroAlloc verifies that initialized Get produces 0 allocations.
func TestLazy_FastPathZeroAlloc(t *testing.T) {
	l := New(func() (int, error) {
		return 12345, nil
	})

	if _, err := l.Get(); err != nil {
		t.Fatalf("failed initial Get: %v", err)
	}

	allocs := testing.AllocsPerRun(1000, func() {
		val, err := l.Get()
		if err != nil || val != 12345 {
			t.Fatalf("unexpected: val=%d err=%v", val, err)
		}
	})

	if allocs != 0 {
		t.Fatalf("expected 0 allocs on initialized Get(), got %f", allocs)
	}
}
