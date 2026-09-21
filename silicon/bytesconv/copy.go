package bytesconv

import (
	"errors"
	"io"
	"net"
	"os"
	"sync"
)

// NoCopy may be embedded into structs which must not be copied
// after the first use.
//
// See https://golang.org/issues/8005#issuecomment-190753527
// for details.
type NoCopy struct{}

// Lock is a no-op used by -copylocks checker from go vet.
func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}

// CopyZeroAlloc optimizes io.Copy by calling ReadFrom or WriteTo only when
// copying between os.File and net.TCPConn.
func CopyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	var readerIsFile, readerIsConn bool

	switch r := r.(type) {
	case *os.File:
		readerIsFile = true
	case *net.TCPConn:
		readerIsConn = true
	case io.WriterTo:
		return r.WriteTo(w)
	}

	switch w := w.(type) {
	case *os.File:
		if readerIsConn {
			return w.ReadFrom(r)
		}
	case *net.TCPConn:
		if readerIsFile {
			if rt, ok := r.(io.WriterTo); ok {
				return rt.WriteTo(w)
			}

			return w.ReadFrom(r)
		}

	case io.ReaderFrom:
		return w.ReadFrom(r)
	}

	vbuf := CopyBufPool.Get()
	buf := vbuf.([]byte)
	n, err := CopyBuffer(w, r, buf)

	CopyBufPool.Put(vbuf)

	return n, err
}

// CopyBuffer copies from src to dst using the provided buffer slice until either EOF is reached on src or an error occurs.
func CopyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0

				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}

			written += int64(nw)

			if ew != nil {
				err = ew
				break
			}

			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}

		if er != nil {
			if er != io.EOF {
				err = er
			}

			break
		}
	}

	return written, err
}

// CopyBufPool is a sync.Pool providing reusable 4KB byte slices for zero-allocation copy operations.
var CopyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 4096)
	},
}

// ErrBodyTooLarge is returned if either request or response body exceeds
// the given limit.
var ErrBodyTooLarge = errors.New("bytesconv: body size exceeds the given limit")

// CopyZeroAllocWithLimit copies up to maxBodySize bytes from r to w with zero heap allocations, returning ErrBodyTooLarge if the limit is exceeded.
func CopyZeroAllocWithLimit(w io.Writer, r io.Reader, maxBodySize int) (int64, error) {
	if maxBodySize <= 0 {
		return CopyZeroAlloc(w, r)
	}

	lr := &io.LimitedReader{
		R: r,
		N: int64(maxBodySize) + 1,
	}

	n, err := CopyZeroAlloc(w, lr)
	if err != nil {
		return n, err
	}

	if lr.N <= 0 {
		return n, ErrBodyTooLarge
	}

	return n, nil
}
