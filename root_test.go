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
		"github.com/lemon4ksan/foundation/silicon/simd/gfni": true,
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
