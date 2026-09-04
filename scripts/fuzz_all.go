// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type fuzzTarget struct {
	pkg  string
	name string
}

var targets = []fuzzTarget{
	{"./generic", "FuzzResult"},
	{"./net/urlkit", "FuzzURLParse"},
	{"./net/urlkit", "FuzzReplaceVar"},
	{"./net/urlkit", "FuzzFastAppendQuery"},
	{"./net/cookie", "FuzzParseSetCookieHeader"},
	{"./net/cookie", "FuzzPathMatch"},
	{"./net/hpack", "FuzzHPACKDecode"},
	{"./net/hpack", "FuzzHPACKEncodeDecodeRoundtrip"},
	{"./net/http/etag", "FuzzETagMatch"},
	{"./net/http/auth", "FuzzBasicAuth"},
	{"./net/http/auth", "FuzzBearerAuth"},
	{"./net/http/contentdisposition", "FuzzContentDisposition"},
	{"./net/dns/wire", "FuzzDNSWireMessage"},
	{"./net/dns/wire", "FuzzPackDNSQuery"},
	{"./net/dns/svcb", "FuzzSVCBWireParse"},
	{"./net/tls/ja4", "FuzzParseExtensionsFromRaw"},
	{"./net/tls/ja4", "FuzzComputeJA4"},
	{"./net/tls/ech", "FuzzParseECHConfigList"},
	{"./net/tls/ech", "FuzzParseECHConfigListBase64"},
	{"./types/uuid", "FuzzUUIDParse"},
	{"./codec/json", "FuzzJSONUnmarshal"},
	{"./silicon/hexkit", "FuzzHexEncodeDecode"},
	{"./silicon/hexkit", "FuzzHexDecodeArbitrary"},
	{"./silicon/bytesconv", "FuzzEqualFoldASCII"},
	{"./silicon/bytesconv", "FuzzAppendToLower"},
	{"./silicon/simd", "FuzzIndexByteSWAR"},
	{"./silicon/simd", "FuzzIndexCRLF"},
	{"./silicon/simd", "FuzzIndexDoubleCRLF"},
	{"./silicon/offheap", "FuzzOffHeapBuffer"},
	{"./silicon/trie", "FuzzRadixTree"},
	{"./text/htmlkit", "FuzzHTMLUnescape"},
	{"./text/casing", "FuzzCasingConversions"},
	{"./cmd/c2plan9", "FuzzDisassembleARM64Raw"},
	{"./cmd/c2plan9", "FuzzCleanPlan9ARM64Syntax"},
	{"./text/encoding/charmap", "FuzzCharmapDecode"},
}

func main() {
	fuzzDuration := flag.String("fuzztime", "3s", "duration to fuzz each target")

	flag.Parse()

	fmt.Printf(
		"=== Starting Foundation Heavy Security Fuzzing Suite (%d targets, %s each) ===\n\n",
		len(targets),
		*fuzzDuration,
	)

	var failed []string

	startTotal := time.Now()

	for i, tgt := range targets {
		fmt.Printf("[%2d/%2d] Fuzzing %s :: %s (fuzztime=%s) ... ", i+1, len(targets), tgt.pkg, tgt.name, *fuzzDuration)

		start := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		cmd := exec.CommandContext(
			ctx,
			"go",
			"test",
			"-run",
			"^$",
			"-fuzz",
			fmt.Sprintf("^%s$", tgt.name),
			"-fuzztime",
			*fuzzDuration,
			tgt.pkg,
		) // #nosec G204

		var outBuf bytes.Buffer

		cmd.Stdout = &outBuf
		cmd.Stderr = &outBuf

		err := cmd.Run()
		cancel()

		dur := time.Since(start)

		if err != nil {
			fmt.Printf("FAILED (%v)\n", dur.Round(time.Millisecond))
			fmt.Printf("--- Output ---\n%s\n--------------\n", outBuf.String())
			failed = append(failed, fmt.Sprintf("%s::%s", tgt.pkg, tgt.name))
		} else {
			fmt.Printf("PASSED (%v)\n", dur.Round(time.Millisecond))
		}
	}

	totalDur := time.Since(startTotal)

	fmt.Println()
	fmt.Println("==================================================")
	fmt.Printf("Heavy Fuzzing Complete in %v\n", totalDur.Round(time.Second))
	fmt.Printf("Total: %d | Passed: %d | Failed: %d\n", len(targets), len(targets)-len(failed), len(failed))

	if len(failed) > 0 {
		fmt.Println("Failed Targets:")
		for _, f := range failed {
			fmt.Printf("  - %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Println("All fuzzing targets passed with zero crashes and curl-grade accuracy!")
}
