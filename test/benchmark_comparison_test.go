// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	stdurl "net/url"
	"strings"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/async/rate"
	"github.com/lemon4ksan/foundation/net/hpack"
	"github.com/lemon4ksan/foundation/net/idna"
	"github.com/lemon4ksan/foundation/net/psl"
	foundationurl "github.com/lemon4ksan/foundation/net/url"
	"github.com/lemon4ksan/foundation/silicon/clock"
	"github.com/lemon4ksan/foundation/text/encoding/charmap"
	"github.com/lemon4ksan/foundation/text/encoding/htmlindex"
)

// -----------------------------------------------------------------------------
// 1. URL Engine: Standard net/url.Parse vs Foundation CRC32 Sharded URL Cache
// -----------------------------------------------------------------------------

const testRawURL = "https://api.gateway.internal:8443/v2/telemetry/events?session_id=9876543210&region=eu-central-1&trace=true#anchor"

func BenchmarkCompare_URL_Parse_Std(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u, err := stdurl.Parse(testRawURL)
		if err != nil || u.Host == "" {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_URL_Parse_Foundation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		u, err := foundationurl.Parse(testRawURL)
		if err != nil || u.Host == "" {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 2. Public Suffix List / eTLD+1: Standard String Scan vs Foundation Zero-Alloc Bytes
// -----------------------------------------------------------------------------

const testDomain = "subdomain.deep.service.co.uk"
var testDomainBytes = []byte(testDomain)

func BenchmarkCompare_PSL_eTLD_String_Std(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := psl.EffectiveTLDPlusOne(testDomain)
		if err != nil || res == "" {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_PSL_eTLD_Bytes_Foundation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := psl.EffectiveTLDPlusOneBytes(testDomainBytes)
		if err != nil || len(res) == 0 {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 3. Clock & Timestamping: Standard time.Now() vs Foundation CoarseClock
// -----------------------------------------------------------------------------

func BenchmarkCompare_Clock_Std_TimeNow(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := time.Now()
		if t.IsZero() {
			b.Fatal("zero time")
		}
	}
}

func BenchmarkCompare_Clock_Foundation_CoarseClock(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := clock.CoarseNowNano()
		if t == 0 {
			b.Fatal("zero time")
		}
	}
}

// -----------------------------------------------------------------------------
// 4. Rate Limiting: Foundation Rate Limiter (Token Bucket Allow vs Sometimes)
// -----------------------------------------------------------------------------

func BenchmarkCompare_Rate_TokenBucket_Allow(b *testing.B) {
	lim := rate.NewLimiter(rate.Inf, 1000000000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !lim.Allow() {
			b.Fatal("rate limit exceeded")
		}
	}
}

func BenchmarkCompare_Rate_Sometimes_FastPath(b *testing.B) {
	s := &rate.Sometimes{Interval: time.Hour}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Do(func() {})
	}
}

// -----------------------------------------------------------------------------
// 5. Web Encodings: WhatWG Charset Resolver & Windows-1251 Transcoding
// -----------------------------------------------------------------------------

func BenchmarkCompare_Encoding_WhatWG_Lookup(b *testing.B) {
	charsets := []string{"windows-1251", "utf-8", "shift_jis", "gbk", "iso-8859-1", "euc-kr"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		enc, err := htmlindex.Get(charsets[i%len(charsets)])
		if err != nil || enc == nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_Encoding_Windows1251_Transcode(b *testing.B) {
	sampleWin1251 := []byte("\xcf\xf0\xe8\xe2\xe5\xf2, \xcc\xe8\xf0! 1234567890 Тестирование производительности кремния")
	dec := charmap.Windows1251.NewDecoder()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := dec.Bytes(sampleWin1251)
		if err != nil || len(out) == 0 {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_Encoding_Windows1251_Transform_ZeroAlloc(b *testing.B) {
	sampleWin1251 := []byte("\xcf\xf0\xe8\xe2\xe5\xf2, \xcc\xe8\xf0! 1234567890 Тестирование производительности кремния")
	dec := charmap.Windows1251.NewDecoder()
	dst := make([]byte, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := dec.Transform(dst, sampleWin1251, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 6. IDNA & Punycode (RFC 3492 / 5891)
// -----------------------------------------------------------------------------

const internationalDomain = "президент.рф"

func BenchmarkCompare_IDNA_ToASCII_Foundation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err := idna.ToASCII(internationalDomain)
		if err != nil || !strings.HasPrefix(res, "xn--") {
			b.Fatal(err)
		}
	}
}

// -----------------------------------------------------------------------------
// 7. HPACK Huffman Decoding: Buffered vs Zero-Alloc Append
// -----------------------------------------------------------------------------

var sampleHuffman = hpack.AppendHuffmanString(nil, "https://api.gateway.internal:8443/v2/telemetry/events?session_id=9876543210")

func BenchmarkCompare_HPACK_Huffman_Decode_Buffered(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		str, err := hpack.HuffmanDecodeToString(sampleHuffman)
		if err != nil || len(str) == 0 {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompare_HPACK_Huffman_Decode_ZeroAlloc(b *testing.B) {
	buf := make([]byte, 0, 256)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		out, err := hpack.AppendHuffmanDecode(buf[:0], sampleHuffman)
		if err != nil || len(out) == 0 {
			b.Fatal(err)
		}
	}
}
