//go:build !race

package gzip_test

import (
	"bytes"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/gzip"
)

func TestWriterAllocs(t *testing.T) {
	data := bytes.Repeat([]byte("hello gzip compression! "), 100)
	var buf bytes.Buffer
	buf.Grow(4096)
	w := gzip.NewWriter(&buf)
	
	// Warm up
	_, _ = w.Write(data)
	w.Close()
	
	allocs := testing.AllocsPerRun(10, func() {
		buf.Reset()
		w.Reset(&buf)
		_, _ = w.Write(data)
		w.Close()
	})
	
	if allocs > 0 {
		t.Errorf("Writer steady state allocated %v times", allocs)
	}
}

func TestReaderAllocs(t *testing.T) {
	data := bytes.Repeat([]byte("hello gzip compression! "), 100)
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(data)
	w.Close()
	compressed := buf.Bytes()
	
	rdr := bytes.NewReader(compressed)
	r, _ := gzip.NewReader(rdr)
	outBuf := make([]byte, 4096)
	for {
		_, err := r.Read(outBuf)
		if err != nil {
			break
		}
	}
	r.Close()
	
	allocs := testing.AllocsPerRun(10, func() {
		rdr.Reset(compressed)
		_ = r.Reset(rdr)
		for {
			_, err := r.Read(outBuf)
			if err != nil {
				break
			}
		}
		r.Close()
	})
	
	if allocs > 0 {
		t.Errorf("Reader steady state allocated %v times", allocs)
	}
}
