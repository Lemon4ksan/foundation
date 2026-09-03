// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package delta implements the byte distance prediction filter (Delta) used in 7-Zip archives.
// The Delta filter converts sequence differences (e.g. in uncompressed audio PCM, table columns,
// RGB bitmap planes) into small delta values, dramatically boosting subsequent entropy compression.
package delta

import (
	"errors"
	"io"
	"sync"
)

var buf64KPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024)
		return &b
	},
}

const (
	// MinDistance defines the minimum delta distance (1 byte).
	MinDistance = 1
	// MaxDistance defines the maximum delta distance (256 bytes).
	MaxDistance = 256
	// StateSize defines the size of the internal historical state buffer.
	StateSize = 256
)

// ErrInvalidDistance indicates an out-of-range delta distance.
var ErrInvalidDistance = errors.New("delta: distance must be between 1 and 256")

// Filter encapsulates the state for Delta byte prediction encoding and decoding.
type Filter struct {
	distance int
	state    [StateSize]byte
}

// NewFilter constructs a Delta filter for the specified distance (1..256).
func NewFilter(distance int) (*Filter, error) {
	if distance < MinDistance || distance > MaxDistance {
		return nil, ErrInvalidDistance
	}
	return &Filter{distance: distance}, nil
}

// Reset clears the filter's historical state buffer.
func (f *Filter) Reset() {
	for i := range f.state {
		f.state[i] = 0
	}
}

// Encode applies in-place Delta encoding to buf.
func (f *Filter) Encode(buf []byte) {
	if len(buf) == 0 {
		return
	}

	delta := f.distance
	size := len(buf)

	var temp [StateSize]byte
	for i := 0; i < delta; i++ {
		temp[i] = f.state[i]
	}

	if size <= delta {
		for i := 0; i < size; i++ {
			b := buf[i]
			buf[i] = b - temp[i]
			temp[i] = b
		}
		for k, i := 0, size; k < delta; k++ {
			if i == delta {
				i = 0
			}
			f.state[k] = temp[i]
			i++
		}
		return
	}

	// For data larger than delta:
	p := size - delta
	for i := 0; i < delta; i++ {
		f.state[i] = buf[p+i]
	}

	if delta == 1 {
		for i := size - 1; i >= 1; i-- {
			buf[i] -= buf[i-1]
		}
		buf[0] -= temp[0]
		return
	}

	for i := size - 1; i >= delta; i-- {
		buf[i] -= buf[i-delta]
	}
	for i := delta - 1; i >= 0; i-- {
		buf[i] -= temp[i]
	}
}

// Decode applies in-place Delta decoding to buf.
func (f *Filter) Decode(buf []byte) {
	if len(buf) == 0 {
		return
	}

	delta := f.distance
	size := len(buf)

	if size <= delta {
		for i := 0; i < size; i++ {
			buf[i] += f.state[i]
		}
		for i := 0; i+size < delta; i++ {
			f.state[i] = f.state[i+size]
		}
		for i := 0; i < size; i++ {
			f.state[delta-size+i] = buf[i]
		}
		return
	}

	for i := 0; i < delta; i++ {
		buf[i] += f.state[i]
	}

	if delta == 1 {
		prev := buf[0]
		for i := 1; i < size; i++ {
			prev += buf[i]
			buf[i] = prev
		}
		f.state[0] = buf[size-1]
		return
	}

	for i := delta; i < size; i++ {
		buf[i] += buf[i-delta]
	}

	for i := 0; i < delta; i++ {
		f.state[i] = buf[size-delta+i]
	}
}

// Reader wraps an io.Reader and decodes Delta-filtered data on the fly.
type Reader struct {
	r      io.Reader
	filter *Filter
}

// NewReader constructs a new Delta streaming decompressor filter.
func NewReader(r io.Reader, distance int) (*Reader, error) {
	f, err := NewFilter(distance)
	if err != nil {
		return nil, err
	}
	return &Reader{r: r, filter: f}, nil
}

// Read implements io.Reader.
func (r *Reader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	if n > 0 {
		r.filter.Decode(p[:n])
	}
	return n, err
}

// Writer wraps an io.Writer and encodes data through Delta filter before writing.
type Writer struct {
	w      io.Writer
	filter *Filter
}

// NewWriter constructs a new Delta streaming encoder filter.
func NewWriter(w io.Writer, distance int) (*Writer, error) {
	f, err := NewFilter(distance)
	if err != nil {
		return nil, err
	}
	return &Writer{w: w, filter: f}, nil
}

// Write implements io.Writer with zero heap allocations.
func (w *Writer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	bufPtr := buf64KPool.Get().(*[]byte)
	defer buf64KPool.Put(bufPtr)

	totalWritten := 0
	for totalWritten < len(p) {
		chunkSize := min(len(p)-totalWritten, len(*bufPtr))
		chunk := (*bufPtr)[:chunkSize]
		copy(chunk, p[totalWritten:totalWritten+chunkSize])
		w.filter.Encode(chunk)
		n, err := w.w.Write(chunk)
		if n > 0 {
			totalWritten += n
		}
		if err != nil {
			return totalWritten, err
		}
	}
	return totalWritten, nil
}
