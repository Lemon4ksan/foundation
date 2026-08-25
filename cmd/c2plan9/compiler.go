// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CompileOptions represents compilation parameters for Clang.
type CompileOptions struct {
	ClangPath  string
	SourceFile string
	OutputFile string
	TargetArch string // amd64, arm64
	TargetOS   string // linux, windows, darwin (defaults to sysv-compatible)
	ExtraFlags []string
	TargetSysV bool // Force System V ABI (standard for Go AMD64 ASM)
	UseWSL     bool // Execute Clang inside WSL
}

func toWSLPath(p string) string {
	abs, err := filepath.Abs(p)
	if err == nil {
		p = abs
	}
	p = filepath.ToSlash(p)
	if len(p) >= 2 && p[1] == ':' {
		drive := strings.ToLower(string(p[0]))
		return "/mnt/" + drive + p[2:]
	}
	return p
}

// Compile compiles the given source file using Clang into an object file.
func Compile(opts CompileOptions) (string, error) {
	clang := opts.ClangPath
	if clang == "" {
		clang = "clang"
	}

	out := opts.OutputFile
	if out == "" {
		tmpDir, err := os.MkdirTemp("", "c2plan9-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp dir: %w", err)
		}
		out = filepath.Join(tmpDir, "kernel.o")
	}

	useWSL := opts.UseWSL && runtime.GOOS == "windows"
	if !useWSL && runtime.GOOS == "windows" {
		// On Windows, if native clang is not found, automatically fallback to WSL
		if _, err := exec.LookPath(clang); err != nil {
			if _, err := exec.LookPath("wsl"); err == nil {
				useWSL = true
			}
		}
	}

	if useWSL {
		wslSrc := toWSLPath(opts.SourceFile)
		wslOut := toWSLPath(out)
		wslArgs := []string{
			"clang", "-c", wslSrc, "-o", wslOut,
			"-O3",
			"-ffreestanding",
			"-fno-asynchronous-unwind-tables",
			"-fno-exceptions",
			"-fno-rtti",
			"-fno-stack-protector",
			"-fomit-frame-pointer",
			"-fno-jump-tables",
			"-mno-red-zone",
			"-mavx2",
			"-mpclmul",
			"-mfma",
		}
		wslArgs = append(wslArgs, opts.ExtraFlags...)
		cmd := exec.Command("wsl", wslArgs...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("wsl clang compilation failed (%w): %s", err, stderr.String())
		}
		return out, nil
	}

	args := []string{
		"-c",
		opts.SourceFile,
		"-o", out,
		"-O3",
		"-ffreestanding",
		"-fno-asynchronous-unwind-tables",
		"-fno-exceptions",
		"-fno-rtti",
		"-fno-stack-protector",
		"-fomit-frame-pointer",
		"-fno-jump-tables",
	}

	if opts.TargetArch == "" || opts.TargetArch == "amd64" {
		args = append(args,
			"-mno-red-zone",
			"-mavx2",
			"-mpclmul",
			"-mfma",
		)

		// On Windows, compile with SysV ABI target to maintain standard DI/SI/DX/CX argument register mapping
		if opts.TargetSysV || runtime.GOOS == "windows" {
			args = append(args, "-target", "x86_64-unknown-linux-gnu")
		}
	}

	args = append(args, opts.ExtraFlags...)

	cmd := exec.Command(clang, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("clang compilation failed (%w): %s\ncommand: %s %s",
			err, stderr.String(), clang, strings.Join(args, " "))
	}

	return out, nil
}
