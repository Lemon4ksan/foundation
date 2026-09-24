// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offheap_test

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/silicon/offheap"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestBuffer_Adversarial_ZeroAndNegativeCapacity(t *testing.T) {
	// Zero capacity should default to 64KB
	bufZero, err := offheap.NewBuffer(0)
	require.NoError(t, err)
	require.NotNil(t, bufZero)
	assert.Equal(t, 64*1024, bufZero.Cap())
	assert.Equal(t, 0, bufZero.Len())
	bufZero.Release()

	// Negative capacity should default to 64KB
	bufNeg, err := offheap.NewBuffer(-1024)
	require.NoError(t, err)
	require.NotNil(t, bufNeg)
	assert.Equal(t, 64*1024, bufNeg.Cap())
	bufNeg.Release()

	// Exact 1-byte buffer boundary
	buf1, err := offheap.NewBuffer(1)
	require.NoError(t, err)
	defer buf1.Release()

	assert.Equal(t, 1, buf1.Cap())
	n, err := buf1.Write([]byte{0x42})
	assert.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, buf1.Len())

	// Overflowing by 1 byte
	n2, err2 := buf1.Write([]byte{0x43})
	assert.True(t, errors.Is(err2, offheap.ErrBufferFull))
	assert.Equal(t, 0, n2)
	assert.Equal(t, 1, buf1.Len())
}

func TestBuffer_Adversarial_HugeAllocations(t *testing.T) {
	// 16MB allocation directly from kernel
	const hugeSize = 16 * 1024 * 1024
	buf, err := offheap.NewBuffer(hugeSize)
	require.NoError(t, err)
	require.NotNil(t, buf)
	defer buf.Release()

	assert.Equal(t, hugeSize, buf.Cap())

	// Fill first and last chunk
	payload := bytes.Repeat([]byte{0xAB}, 64*1024)
	n, err := buf.Write(payload)
	assert.NoError(t, err)
	assert.Equal(t, len(payload), n)

	// Reslice to full 16MB via RawBytes
	raw := buf.RawBytes(hugeSize)
	require.NotNil(t, raw)
	assert.Equal(t, hugeSize, len(raw))

	// Touch beginning, middle, and end
	raw[0] = 0x11
	raw[hugeSize/2] = 0x22
	raw[hugeSize-1] = 0x33

	assert.Equal(t, byte(0x11), raw[0])
	assert.Equal(t, byte(0x22), raw[hugeSize/2])
	assert.Equal(t, byte(0x33), raw[hugeSize-1])
}

func TestBuffer_Adversarial_OperationsAfterReleaseAndNilReceiver(t *testing.T) {
	buf, err := offheap.NewBuffer(1024)
	require.NoError(t, err)
	_, _ = buf.WriteString("temporary data")

	// Release buffer
	buf.Release()

	// Double Release should be an idempotent no-op
	buf.Release()

	// All mutating and reading methods must safely return ErrBufferClosed or nil
	_, wErr := buf.Write([]byte("more"))
	assert.True(t, errors.Is(wErr, offheap.ErrBufferClosed))

	_, sErr := buf.WriteString("more string")
	assert.True(t, errors.Is(sErr, offheap.ErrBufferClosed))

	dst := make([]byte, 10)
	_, rErr := buf.Read(dst)
	assert.True(t, errors.Is(rErr, offheap.ErrBufferClosed))

	assert.Nil(t, buf.Bytes())
	assert.Nil(t, buf.RawBytes(100))
	assert.Equal(t, 0, buf.Len())
	assert.Equal(t, 0, buf.Cap())

	// Reset and RewindRead on released buffer should not panic
	buf.Reset()
	buf.RewindRead()

	// Nil receiver safety
	var nilBuf *offheap.OffHeapBuffer
	_, nilWErr := nilBuf.Write([]byte("abc"))
	assert.True(t, errors.Is(nilWErr, offheap.ErrBufferClosed))

	_, nilSErr := nilBuf.WriteString("abc")
	assert.True(t, errors.Is(nilSErr, offheap.ErrBufferClosed))

	_, nilRErr := nilBuf.Read(dst)
	assert.True(t, errors.Is(nilRErr, offheap.ErrBufferClosed))

	assert.Nil(t, nilBuf.Bytes())
	assert.Nil(t, nilBuf.RawBytes(10))
	assert.Equal(t, 0, nilBuf.Len())
	assert.Equal(t, 0, nilBuf.Cap())
	nilBuf.Reset()
	nilBuf.RewindRead()
	nilBuf.Release()
}

func TestBuffer_Adversarial_BoundaryRawReslicing(t *testing.T) {
	const capSize = 4096
	buf, err := offheap.NewBuffer(capSize)
	require.NoError(t, err)
	defer buf.Release()

	// Zero or negative length
	assert.Nil(t, buf.RawBytes(0))
	assert.Nil(t, buf.RawBytes(-1))
	assert.Nil(t, buf.RawBytes(-100))

	// Exactly capSize
	full := buf.RawBytes(capSize)
	require.NotNil(t, full)
	assert.Equal(t, capSize, len(full))
	assert.Equal(t, capSize, buf.Len())

	// Exceeding capacity by 1 byte
	assert.Nil(t, buf.RawBytes(capSize+1))
	// Len() should remain unchanged after rejected reslice
	assert.Equal(t, capSize, buf.Len())

	// Extreme length
	assert.Nil(t, buf.RawBytes(1<<30))

	// Reslice to smaller
	half := buf.RawBytes(capSize / 2)
	require.NotNil(t, half)
	assert.Equal(t, capSize/2, len(half))
	assert.Equal(t, capSize/2, buf.Len())
}

func TestBuffer_Adversarial_SynchronizedConcurrentWrites(t *testing.T) {
	// OffHeapBuffer documents that external synchronization is required.
	// We stress-test concurrent writers coordinating through a sync.Mutex.
	const totalWriters = 32
	const writesPerWorker = 50
	const chunkSize = 64

	buf, err := offheap.NewBuffer(totalWriters * writesPerWorker * chunkSize)
	require.NoError(t, err)
	defer buf.Release()

	var mu sync.Mutex
	var wg sync.WaitGroup

	for w := 0; w < totalWriters; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			payload := bytes.Repeat([]byte{byte(workerID)}, chunkSize)
			for i := 0; i < writesPerWorker; i++ {
				mu.Lock()
				n, wErr := buf.Write(payload)
				mu.Unlock()
				assert.NoError(t, wErr)
				assert.Equal(t, chunkSize, n)
			}
		}(w)
	}

	wg.Wait()

	expectedTotal := totalWriters * writesPerWorker * chunkSize
	assert.Equal(t, expectedTotal, buf.Len())

	// Validate sequential reading through io.Reader
	readDst := make([]byte, 256)
	totalRead := 0
	for {
		n, rErr := buf.Read(readDst)
		totalRead += n
		if errors.Is(rErr, io.EOF) {
			break
		}
		assert.NoError(t, rErr)
	}
	assert.Equal(t, expectedTotal, totalRead)
}
