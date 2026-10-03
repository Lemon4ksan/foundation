// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package foundation_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type pkgInfo struct {
	Dir        string   `json:"Dir"`
	ImportPath string   `json:"ImportPath"`
	Imports    []string `json:"Imports"`
}

func getPackageLayer(pkgPath string) (int, string) {
	const prefix = "github.com/lemon4ksan/foundation"
	rel := strings.TrimPrefix(pkgPath, prefix)
	rel = strings.TrimPrefix(rel, "/")
	if rel == "" || rel == "generic" {
		return -1, "external"
	}
	parts := strings.Split(rel, "/")
	root := parts[0]
	switch root {
	case "silicon":
		return 0, "Layer 0: Substrate"
	case "borrow", "structures", "refkit", "bufkit":
		return 1, "Layer 1: Memory & Structures"
	case "sync", "async":
		return 2, "Layer 2: Concurrency"
	case "codec", "crypto", "encoding", "text", "iokit":
		return 3, "Layer 3: Codecs & Cryptography"
	case "net":
		return 4, "Layer 4: Wire & Transport"
	case "types", "timekit", "argkit", "fskit", "pathkit", "tuikit", "testing", "cmd", "test", "scripts":
		return 5, "Layer 5: Tooling & Utilities"
	default:
		return 99, "Unknown"
	}
}

// Allowed temporary violations that must only decrease over time.
var (
	// Whitelist of packages permitted to import generic during transition.
	allowedGenericImports = map[string]bool{}

	// Whitelist of cross-imports within silicon/*.
	allowedSiliconCrossImports = map[string]string{}

	// Whitelist of public packages temporarily permitted without doc.go.
	allowedMissingDocGo = map[string]bool{
		"github.com/lemon4ksan/foundation/cmd/c2plan9":                       true,
		"github.com/lemon4ksan/foundation/codec/compress":                    true,
		"github.com/lemon4ksan/foundation/codec/compress/brotli/matchfinder": true,
		"github.com/lemon4ksan/foundation/codec/compress/flate":              true,
		"github.com/lemon4ksan/foundation/codec/compress/fse":                true,
		"github.com/lemon4ksan/foundation/codec/compress/gzip":               true,
		"github.com/lemon4ksan/foundation/codec/compress/huff0":              true,
		"github.com/lemon4ksan/foundation/codec/compress/lz4":                true,
		"github.com/lemon4ksan/foundation/codec/compress/zstd":               true,
		"github.com/lemon4ksan/foundation/codec/compress/zstd/xxhash":        true,
		"github.com/lemon4ksan/foundation/codec/entropy":                     true,
		"github.com/lemon4ksan/foundation/codec/filter/bcj":                  true,
		"github.com/lemon4ksan/foundation/codec/filter/delta":                true,
		"github.com/lemon4ksan/foundation/codec/filter/shuffle":              true,
		"github.com/lemon4ksan/foundation/codec/json":                        true,
		"github.com/lemon4ksan/foundation/crypto/aead":                       true,
		"github.com/lemon4ksan/foundation/crypto/kdf":                        true,
		"github.com/lemon4ksan/foundation/crypto/sign":                       true,
		"github.com/lemon4ksan/foundation/encoding/base64":                   true,
		"github.com/lemon4ksan/foundation/encoding/varint":                   true,
		"github.com/lemon4ksan/foundation/net/ipc":                           true,
		"github.com/lemon4ksan/foundation/net/netutil":                       true,
		"github.com/lemon4ksan/foundation/net/proxy":                         true,
		"github.com/lemon4ksan/foundation/net/proxy/socks":                   true,
		"github.com/lemon4ksan/foundation/net/proxy/socks/sockstest":         true,
		"github.com/lemon4ksan/foundation/net/quic":                          true,
		"github.com/lemon4ksan/foundation/net/quic/testutils":                true,
		"github.com/lemon4ksan/foundation/net/quic/varint":                   true,
		"github.com/lemon4ksan/foundation/net/tls/cert":                      true,
		"github.com/lemon4ksan/foundation/net/urlkit":                        true,
		"github.com/lemon4ksan/foundation/silicon/bytesconv":                 true,
		"github.com/lemon4ksan/foundation/silicon/clock":                     true,
		"github.com/lemon4ksan/foundation/silicon/hexkit":                    true,
		"github.com/lemon4ksan/foundation/silicon/randkit":                   true,
		"github.com/lemon4ksan/foundation/silicon/ringbuf":                   true,
		"github.com/lemon4ksan/foundation/silicon/simd":                      true,
		"github.com/lemon4ksan/foundation/silicon/simd/gfni":                 true,
		"github.com/lemon4ksan/foundation/silicon/sysnet":                    true,
		"github.com/lemon4ksan/foundation/silicon/trie":                      true,
		"github.com/lemon4ksan/foundation/structures/bitset":                 true,
		"github.com/lemon4ksan/foundation/structures/deque":                  true,
		"github.com/lemon4ksan/foundation/structures/disjointset":            true,
		"github.com/lemon4ksan/foundation/structures/linkedlist":             true,
		"github.com/lemon4ksan/foundation/structures/minheap":                true,
		"github.com/lemon4ksan/foundation/structures/ringbuffer":             true,
		"github.com/lemon4ksan/foundation/testing/assert":                    true,
		"github.com/lemon4ksan/foundation/testing/gomock":                    true,
		"github.com/lemon4ksan/foundation/testing/require":                   true,
		"github.com/lemon4ksan/foundation/text/casing":                       true,
		"github.com/lemon4ksan/foundation/text/diff":                         true,
		"github.com/lemon4ksan/foundation/text/encoding":                     true,
		"github.com/lemon4ksan/foundation/text/encoding/charmap":             true,
		"github.com/lemon4ksan/foundation/text/encoding/htmlindex":           true,
		"github.com/lemon4ksan/foundation/text/encoding/japanese":            true,
		"github.com/lemon4ksan/foundation/text/encoding/korean":              true,
		"github.com/lemon4ksan/foundation/text/encoding/simplifiedchinese":   true,
		"github.com/lemon4ksan/foundation/text/encoding/traditionalchinese":  true,
		"github.com/lemon4ksan/foundation/text/encoding/unicode":             true,
		"github.com/lemon4ksan/foundation/text/transform":                    true,
	}
)

func loadPackageList(t *testing.T) []pkgInfo {
	cmd := exec.Command("go", "list", "-json", "./...")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to run 'go list -json ./...': %v", err)
	}

	dec := json.NewDecoder(strings.NewReader(string(out)))
	var pkgs []pkgInfo
	for dec.More() {
		var p pkgInfo
		if err := dec.Decode(&p); err != nil {
			t.Fatalf("failed to decode go list output: %v", err)
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func TestArchitecture_LayersAndIsolation(t *testing.T) {
	pkgs := loadPackageList(t)
	const modulePrefix = "github.com/lemon4ksan/foundation"

	seenGeneric := make(map[string]bool)
	seenSiliconCross := make(map[string]string)

	for _, p := range pkgs {
		src := p.ImportPath
		if src == modulePrefix || strings.HasPrefix(src, modulePrefix+"/test") {
			continue
		}
		srcLayer, srcLayerName := getPackageLayer(src)

		for _, imp := range p.Imports {
			if !strings.HasPrefix(imp, modulePrefix) {
				continue // external standard library or third-party dependency
			}
			dst := imp
			if dst == modulePrefix {
				continue
			}
			dstLayer, dstLayerName := getPackageLayer(dst)

			// Law 1: No internal package may import generic
			if strings.HasPrefix(dst, modulePrefix+"/generic") && !strings.HasPrefix(src, modulePrefix+"/generic") {
				if !allowedGenericImports[src] {
					t.Errorf(
						"[VIOLATION] Package %q imports external-only package %q. Generic is reserved for external consumers.",
						src,
						dst,
					)
				} else {
					seenGeneric[src] = true
				}
			}

			// Law 2: Silicon packages must not import non-silicon foundation packages
			if srcLayer == 0 {
				if dstLayer != 0 {
					t.Errorf(
						"[VIOLATION] Silicon substrate package %q imports higher-layer package %q (%s). Silicon must be standalone.",
						src,
						dst,
						dstLayerName,
					)
				} else {
					// Law 3: Silicon packages must not cross-import each other
					srcParts := strings.Split(strings.TrimPrefix(src, modulePrefix+"/silicon/"), "/")
					dstParts := strings.Split(strings.TrimPrefix(dst, modulePrefix+"/silicon/"), "/")
					if srcParts[0] != dstParts[0] {
						expectedDst, allowed := allowedSiliconCrossImports[src]
						if !allowed || expectedDst != dst {
							t.Errorf(
								"[VIOLATION] Silicon package %q cross-imports sibling silicon package %q. Silicon packages must be autonomous.",
								src,
								dst,
							)
						} else {
							seenSiliconCross[src] = dst
						}
					}
				}
			}

			// Law 4: Layer monotonicity (L_B <= L_A)
			if srcLayer >= 0 && dstLayer >= 0 && dstLayer > srcLayer {
				t.Errorf(
					"[VIOLATION] Layer inversion: package %q (%s) imports %q (%s). Inversions are forbidden.",
					src,
					srcLayerName,
					dst,
					dstLayerName,
				)
			}
		}
	}

	// Verify ratchet: whitelists must not contain stale exceptions
	for allowed := range allowedGenericImports {
		if !seenGeneric[allowed] {
			t.Errorf(
				"[RATCHET] Whitelisted generic import for %q is no longer needed. Remove it from allowedGenericImports in root_test.go!",
				allowed,
			)
		}
	}
	for src, dst := range allowedSiliconCrossImports {
		if seenSiliconCross[src] != dst {
			t.Errorf(
				"[RATCHET] Whitelisted silicon cross-import from %q to %q is no longer needed. Remove it from allowedSiliconCrossImports in root_test.go!",
				src,
				dst,
			)
		}
	}
}

func TestArchitecture_PublicPackagesDocGo(t *testing.T) {
	pkgs := loadPackageList(t)
	const modulePrefix = "github.com/lemon4ksan/foundation"

	for _, p := range pkgs {
		pkg := p.ImportPath
		if pkg == modulePrefix || strings.Contains(pkg, "/internal") || strings.Contains(pkg, "/scripts") ||
			strings.HasSuffix(pkg, "/test") {
			continue
		}

		docPath := filepath.Join(p.Dir, "doc.go")
		_, err := os.Stat(docPath)
		hasDoc := err == nil

		if !hasDoc {
			if !allowedMissingDocGo[pkg] {
				t.Errorf("[VIOLATION] Public package %q is missing canonical doc.go file.", pkg)
			}
		} else {
			if allowedMissingDocGo[pkg] {
				t.Errorf(
					"[RATCHET] Package %q now has doc.go! Remove it from allowedMissingDocGo in root_test.go.",
					pkg,
				)
			}
		}
	}
}
