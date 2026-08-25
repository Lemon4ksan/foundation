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
		useWSL     = flag.Bool("wsl", false, "Compile via Clang inside WSL")
	)

	flag.Parse()

	if *cFile == "" {
		fmt.Fprintf(os.Stderr, "Usage: c2plan9 -c <source.c> -o <output_amd64.s> [-stub <output_amd64.go>] [-pkg <pkg>] [-wsl]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *oFile == "" {
		base := strings.TrimSuffix(filepath.Base(*cFile), filepath.Ext(*cFile))
		*oFile = filepath.Join(filepath.Dir(*cFile), base+"_amd64.s")
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

	fmt.Printf("[c2plan9] Compiling %s via Clang/LLVM (WSL=%v)...\n", *cFile, *useWSL)
	objPath, err := Compile(CompileOptions{
		ClangPath:  *clangPath,
		SourceFile: *cFile,
		TargetArch: "amd64",
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

	fmt.Printf("[c2plan9] Disassembling %d symbol(s) into Plan 9 assembly...\n", len(obj.Symbols))
	asmBytes, err := EmitPlan9Assembly(*pkgName, obj.Symbols, nil)
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
		var sigs []FuncSignature
		for _, sym := range obj.Symbols {
			if !sym.IsLocal {
				sigs = append(sigs, FuncSignature{
					Name:       sym.Name,
					ParamCount: 6,
					HasReturn:  true,
				})
			}
		}
		stubBytes := EmitGoStub(*pkgName, sigs)
		if err := os.WriteFile(*stubFile, stubBytes, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "[c2plan9] Failed to write stub %s: %v\n", *stubFile, err)
			os.Exit(1)
		}
		fmt.Printf("[c2plan9] Successfully emitted Go stub %s\n", *stubFile)
	}
}
