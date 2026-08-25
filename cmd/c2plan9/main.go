// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	var (
		cFile      = flag.String("c", "", "Path to input C / C++ / LLVM IR source file")
		oFile      = flag.String("o", "", "Path to output Plan 9 assembly (.s) file")
		stubFile   = flag.String("stub", "", "Path to output Go companion stub (.go) file (optional)")
		pkgName    = flag.String("pkg", "", "Target Go package name (defaults to output dir name)")
		clangPath  = flag.String("clang", "clang", "Path to Clang executable")
		extraFlags = flag.String("flags", "", "Extra flags to pass to Clang (comma or space separated)")
		targetArch = flag.String("arch", "", "Target architecture: amd64 (default) or arm64")
		useWSL     = flag.Bool("wsl", false, "Compile via Clang inside WSL")
	)

	flag.Parse()

	if *cFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: c2plan9 -c <source.c> -o <output_amd64.s> [-arch <amd64|arm64>] [-stub <output_amd64.go>] [-pkg <pkg>] [-wsl]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *targetArch == "" {
		if strings.Contains(*oFile, "_arm64") || strings.Contains(*cFile, "_arm64") {
			*targetArch = "arm64"
		} else {
			*targetArch = "amd64"
		}
	}

	if *oFile == "" {
		base := strings.TrimSuffix(filepath.Base(*cFile), filepath.Ext(*cFile))
		*oFile = filepath.Join(filepath.Dir(*cFile), base+"_"+*targetArch+".s")
	}

	if *pkgName == "" {
		*pkgName = filepath.Base(filepath.Dir(*oFile))
		if *pkgName == "." || *pkgName == "/" || *pkgName == "\\" {
			*pkgName = "main"
		}
	}

	var flags []string
	if *extraFlags != "" {
		flags = strings.Fields(*extraFlags)
	}

	fmt.Printf("[c2plan9] Compiling %s for %s via Clang/LLVM (WSL=%v)...\n", *cFile, *targetArch, *useWSL)
	objPath, err := Compile(CompileOptions{
		ClangPath:  *clangPath,
		SourceFile: *cFile,
		TargetArch: *targetArch,
		ExtraFlags: flags,
		TargetSysV: true,
		UseWSL:     *useWSL,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[c2plan9] Compilation error: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(objPath)

	fmt.Printf("[c2plan9] Parsing object file %s...\n", objPath)
	obj, err := ParseObject(objPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[c2plan9] Object parsing error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[c2plan9] Disassembling %d symbol(s) into Plan 9 %s assembly...\n", len(obj.Symbols), *targetArch)
	asmBytes, err := EmitPlan9Assembly(*pkgName, obj.Symbols, nil, obj.ROData, obj.Relocations, *targetArch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[c2plan9] Assembly generation error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*oFile, asmBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "[c2plan9] Failed to write %s: %v\n", *oFile, err)
		os.Exit(1)
	}
	fmt.Printf("[c2plan9] Successfully emitted %s (%d bytes)\n", *oFile, len(asmBytes))

	if *stubFile != "" {
		paramCount := 6
		if *targetArch == "arm64" {
			paramCount = 8
		}
		var sigs []FuncSignature
		for _, sym := range obj.Symbols {
			if !sym.IsLocal {
				sigs = append(sigs, FuncSignature{
					Name:       sym.Name,
					ParamCount: paramCount,
					HasReturn:  true,
				})
			}
		}
		stubBytes := EmitGoStub(*pkgName, sigs, *targetArch)
		if err := os.WriteFile(*stubFile, stubBytes, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "[c2plan9] Failed to write stub %s: %v\n", *stubFile, err)
			os.Exit(1)
		}
		fmt.Printf("[c2plan9] Successfully emitted Go stub %s\n", *stubFile)
	}
}
