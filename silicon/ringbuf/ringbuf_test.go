// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ringbuf_test

import (
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/ringbuf"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestRingBufferPushPop(t *testing.T) {
	rb := ringbuf.NewRingBuffer[string](16)

	val1 := "entry_1"
	val2 := "entry_2"

	assert.True(t, rb.Push(&val1))
	assert.True(t, rb.Push(&val2))

	assert.Equal(t, &val1, rb.Pop())
	assert.Equal(t, &val2, rb.Pop())
	assert.Nil(t, rb.Pop())
}

func TestRingBufferConcurrent(t *testing.T) {
	rb := ringbuf.NewRingBuffer[int](2048)

	var wg sync.WaitGroup

	numProducers := 16
	itemsPerProducer := 100

	for i := 0; i < numProducers; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := 0; j < itemsPerProducer; j++ {
				val := id*1000 + j
				for !rb.Push(&val) {
					// Retry if full
				}
			}
		}(i)
	}

	wg.Wait()
	assert.Equal(t, numProducers*itemsPerProducer, rb.Len())
}

func TestRingBufferConcurrentProduceConsume(t *testing.T) {
	rb := ringbuf.NewRingBuffer[int](128)

	var producerWg sync.WaitGroup

	numProducers := 8
	itemsPerProducer := 1000

	doneProducing := make(chan struct{})

	for i := 0; i < numProducers; i++ {
		producerWg.Add(1)

		go func(id int) {
			defer producerWg.Done()

			for j := 0; j < itemsPerProducer; j++ {
				val := id*10000 + j
				for !rb.Push(&val) {
					// Retry if full
				}
			}
		}(i)
	}

	go func() {
		producerWg.Wait()
		close(doneProducing)
	}()

	var consumeWg sync.WaitGroup

	for i := 0; i < 4; i++ {
		consumeWg.Add(1)

		go func() {
			defer consumeWg.Done()

			for {
				item := rb.Pop()
				if item != nil {
					_ = *item
				} else {
					select {
					case <-doneProducing:
						if rb.Len() == 0 {
							return
						}
					default:
					}
				}
			}
		}()
	}

	consumeWg.Wait()
}

func TestRingBuffer_EdgeCases(t *testing.T) {
	// Defaults for zero / negative capacity
	rbDef := ringbuf.NewRingBuffer[int](0)
	assert.NotNil(t, rbDef)

	spscDef := ringbuf.NewSPSCRingBuffer[int](-10)
	assert.NotNil(t, spscDef)
	assert.Equal(t, 2, spscDef.Cap()) // rounded to min power of 2

	// PacketBatchSoA tests
	batchDef := ringbuf.NewPacketBatchSoA(0)
	assert.NotNil(t, batchDef)

	batch := ringbuf.NewPacketBatchSoA(16)
	_ = batch.Append(6, 100, 1024, 0)
	_ = batch.Append(17, 200, 512, 0)
	_ = batch.Append(6, 300, 2048, 0)

	tcpIndices := batch.FilterByProtocol(6, nil)
	assert.Equal(t, []int{0, 2}, tcpIndices)

	batch.Reset()
	assert.Empty(t, batch.FilterByProtocol(6, nil))
}
