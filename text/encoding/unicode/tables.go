// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package unicode

import (
	"unicode/utf8"

	"github.com/lemon4ksan/foundation/text/transform"
)

const (
	loCB = 0x80
	hiCB = 0xBF
)

const (
	as = 0xF0
	xx = 0xF1
	s1 = 0x02
	s2 = 0x13
	s3 = 0x03
	s4 = 0x23
	s5 = 0x34
	s6 = 0x04
	s7 = 0x44

	firstInvalid = xx
	sizeMask     = 7
	acceptShift  = 4
)

var first = [256]uint8{
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	as, as, as, as, as, as, as, as, as, as, as, as, as, as, as, as,
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx,
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx,
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx,
	xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx,
	xx, xx, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1,
	s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1, s1,
	s2, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s3, s4, s3, s3,
	s5, s6, s6, s6, s7, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx, xx,
}

type acceptRange struct {
	Lo uint8
	Hi uint8
}

var acceptRanges = [...]acceptRange{
	0: {loCB, hiCB},
	1: {0xA0, hiCB},
	2: {loCB, 0x9F},
	3: {0x90, hiCB},
	4: {loCB, 0x8F},
}

type replaceIllFormed struct{ transform.NopResetter }

func (replaceIllFormed) Span(src []byte, atEOF bool) (n int, err error) {
	for r, size := rune(0), 0; n < len(src); {
		if r = rune(src[n]); r < utf8.RuneSelf {
			size = 1
		} else if r, size = utf8.DecodeRune(src[n:]); size == 1 {
			if !atEOF && !utf8.FullRune(src[n:]) {
				err = transform.ErrShortSrc
			} else {
				err = transform.ErrEndOfSpan
			}
			break
		}
		n += size
	}
	return
}

func (replaceIllFormed) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for r, size := rune(0), 0; nSrc < len(src); {
		if r = rune(src[nSrc]); r < utf8.RuneSelf {
			if nDst >= len(dst) {
				err = transform.ErrShortDst
				break
			}
			dst[nDst] = src[nSrc]
			nDst++
			nSrc++
			continue
		} else if r, size = utf8.DecodeRune(src[nSrc:]); size == 1 {
			if !atEOF && !utf8.FullRune(src[nSrc:]) {
				err = transform.ErrShortSrc
				break
			}
			if nDst+3 > len(dst) {
				err = transform.ErrShortDst
				break
			}
			nDst += utf8.EncodeRune(dst[nDst:], utf8.RuneError)
			nSrc++
			continue
		}
		if nDst+size > len(dst) {
			err = transform.ErrShortDst
			break
		}
		nDst += copy(dst[nDst:], src[nSrc:nSrc+size])
		nSrc += size
	}
	return
}
