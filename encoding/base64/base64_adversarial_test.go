// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base64_test

import (
	stdbase64 "encoding/base64"
	"fmt"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/encoding/base64"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestBase64EncodeURL_Adversarial_PanicContract(t *testing.T) {
	t.Parallel()

	expectedPanicMsg := "base64: dst buffer too small for Base64EncodeURL"

	lengths := []int{1, 2, 3, 4, 10, 64, 128, 1024, 8192}

	for _, srcLen := range lengths {
		t.Run(fmt.Sprintf("src_len_%d", srcLen), func(t *testing.T) {
			src := make([]byte, srcLen)
			for i := range src {
				src[i] = byte(i % 256)
			}

			reqLen := base64.Base64URLEncodedLen(srcLen)

			// Exact 1 byte less than required must panic with exact message
			t.Run("one_byte_short_panics", func(t *testing.T) {
				dst := make([]byte, reqLen-1)
				defer func() {
					r := recover()
					require.NotNil(t, r, "must panic when dst is 1 byte too short")
					assert.Equal(t, expectedPanicMsg, r)
				}()
				_ = base64.Base64EncodeURL(src, dst)
			})

			// 0-length dst must panic
			t.Run("zero_length_dst_panics", func(t *testing.T) {
				var dst []byte
				defer func() {
					r := recover()
					require.NotNil(t, r, "must panic when dst is 0 length")
					assert.Equal(t, expectedPanicMsg, r)
				}()
				_ = base64.Base64EncodeURL(src, dst)
			})

			// Exactly required length must NOT panic
			t.Run("exact_length_succeeds", func(t *testing.T) {
				dst := make([]byte, reqLen)
				n := base64.Base64EncodeURL(src, dst)
				assert.Equal(t, reqLen, n)

				expected := stdbase64.RawURLEncoding.EncodeToString(src)
				assert.Equal(t, expected, string(dst))
			})

			// Larger length must NOT panic
			t.Run("larger_length_succeeds", func(t *testing.T) {
				dst := make([]byte, reqLen+10)
				n := base64.Base64EncodeURL(src, dst)
				assert.Equal(t, reqLen, n)

				expected := stdbase64.RawURLEncoding.EncodeToString(src)
				assert.Equal(t, expected, string(dst[:n]))
			})
		})
	}

	t.Run("zero_length_src", func(t *testing.T) {
		// When src is empty, reqLen is 0, so dst of length 0 must succeed without panic
		var src []byte
		var dst []byte
		n := base64.Base64EncodeURL(src, dst)
		assert.Equal(t, 0, n)
	})
}

func TestBase64EncodeURL_Adversarial_Concurrency(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	goroutines := 30

	for g := range goroutines {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			src := []byte(fmt.Sprintf("worker data payload for thread id %d with arbitrary values", workerID))
			reqLen := base64.Base64URLEncodedLen(len(src))
			dst := make([]byte, reqLen)

			n := base64.Base64EncodeURL(src, dst)
			assert.Equal(t, reqLen, n)

			expected := stdbase64.RawURLEncoding.EncodeToString(src)
			assert.Equal(t, expected, string(dst))
		}(g)
	}

	wg.Wait()
}

func TestBase64EncodeURL_Adversarial_ZeroAlloc(t *testing.T) {
	src := []byte("hello world zero allocation base64 test buffer")
	reqLen := base64.Base64URLEncodedLen(len(src))
	dst := make([]byte, reqLen)

	allocs := testing.AllocsPerRun(100, func() {
		n := base64.Base64EncodeURL(src, dst)
		if n != reqLen {
			t.Fatal("length mismatch")
		}
	})
	assert.Equal(t, float64(0), allocs, "Base64EncodeURL must be 0 allocs/op")
}
