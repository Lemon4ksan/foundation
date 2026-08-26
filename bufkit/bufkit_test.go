// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bufkit

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

func TestChain_WriteAndRead(t *testing.T) {
	c := NewChain()
	defer c.Release()

	// Write smaller than a chunk
	msg1 := []byte("hello, world!")
	n, err := c.Write(msg1)
	if err != nil || n != len(msg1) {
		t.Fatalf("Write failed: n=%d, err=%v", n, err)
	}

	if c.Len() != len(msg1) {
		t.Fatalf("expected len=%d, got %d", len(msg1), c.Len())
	}

	// Read exact
	buf := make([]byte, len(msg1))
	rn, rerr := c.Read(buf)
	if rerr != nil || rn != len(msg1) {
		t.Fatalf("Read failed: rn=%d, err=%v", rn, rerr)
	}
	if !bytes.Equal(buf, msg1) {
		t.Fatalf("expected %s, got %s", msg1, buf)
	}

	if c.Len() != 0 {
		t.Fatalf("expected empty chain, got len=%d", c.Len())
	}
}

func TestChain_MultiChunkWriteTo(t *testing.T) {
	c := NewChain()
	defer c.Release()

	// Write large payload that spans across multiple 4KB chunks
	largePayload := make([]byte, 10000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}

	n, err := c.Write(largePayload)
	if err != nil || n != len(largePayload) {
		t.Fatalf("Write failed: %v", err)
	}

	if c.Len() != 10000 {
		t.Fatalf("expected 10000, got %d", c.Len())
	}

	// Direct WriteTo
	var out bytes.Buffer
	wn, werr := c.WriteTo(&out)
	if werr != nil || wn != 10000 {
		t.Fatalf("WriteTo failed: wn=%d, err=%v", wn, werr)
	}

	if !bytes.Equal(out.Bytes(), largePayload) {
		t.Fatalf("payload mismatch after WriteTo")
	}

	if c.Len() != 0 {
		t.Fatalf("expected empty chain after WriteTo, got %d", c.Len())
	}
}

func TestChain_ChunksVectored(t *testing.T) {
	c := NewChain()
	defer c.Release()

	c.WriteString("chunk-data-1")
	chunks := c.Chunks()
	if len(chunks) != 1 || string(chunks[0]) != "chunk-data-1" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestRing_SPSC_Concurrency(t *testing.T) {
	ring := NewRing[int](128)
	const count = 100000

	var wg sync.WaitGroup
	wg.Add(2)

	// Producer
	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			for !ring.Push(i) {
				// Yield / spin
			}
		}
	}()

	// Consumer
	go func() {
		defer wg.Done()
		for i := 0; i < count; i++ {
			for {
				val, ok := ring.Pop()
				if ok {
					if val != i {
						t.Errorf("expected %d, got %d", i, val)
					}
					break
				}
			}
		}
	}()

	wg.Wait()
}

func TestAlignedBytes(t *testing.T) {
	b := AlignedBytes(1024, CacheLineSize)
	if len(b) != 1024 {
		t.Fatalf("expected len 1024, got %d", len(b))
	}
	if !IsAligned(b, CacheLineSize) {
		t.Fatalf("expected buffer to be 64-byte aligned")
	}

	bPage := AlignedBytes(8192, PageSize)
	if !IsAligned(bPage, PageSize) {
		t.Fatalf("expected buffer to be page aligned")
	}
}

func BenchmarkChain_WriteAndWriteTo(b *testing.B) {
	c := NewChain()
	defer c.Release()
	payload := make([]byte, 8192)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = c.Write(payload)
		_, _ = c.WriteTo(io.Discard)
	}
}
