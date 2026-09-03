// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vfs

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lemon4ksan/foundation/pathkit"
)

// ErrInsecurePath indicates that a file path attempts directory traversal (Zip Slip / Tar Slip) or contains forbidden control characters.
var ErrInsecurePath = errors.New("vfs: insecure file path")

var reservedWindowsDeviceNames = map[string]bool{
	"CON":  true,
	"PRN":  true,
	"AUX":  true,
	"NUL":  true,
	"COM1": true,
	"COM2": true,
	"COM3": true,
	"COM4": true,
	"COM5": true,
	"COM6": true,
	"COM7": true,
	"COM8": true,
	"COM9": true,
	"LPT1": true,
	"LPT2": true,
	"LPT3": true,
	"LPT4": true,
	"LPT5": true,
	"LPT6": true,
	"LPT7": true,
	"LPT8": true,
	"LPT9": true,
}

// CleanPath normalizes an archive entry path using pathkit.Sanitize (guaranteeing forward slashes and no leading slashes).
func CleanPath(name string) string {
	p := pathkit.Sanitize(name)
	if p == "." {
		return ""
	}
	return p
}

// SafePath resolves and validates a target filesystem destination path for an archive entry,
// guaranteeing that the resulting absolute path cannot escape destDir (Zip Slip / Tar Slip defense)
// and rejecting dangerous Windows device names (CON, AUX, NUL, COM1-9, LPT1-9).
func SafePath(destDir, fileName string) (string, error) {
	for i := 0; i < len(fileName); i++ {
		c := fileName[i]
		if c < 0x20 || c == 0x7F {
			return "", fmt.Errorf("%w: filename contains control characters (0x%02x)", ErrInsecurePath, c)
		}
	}

	p := pathkit.New(fileName)
	if p.IsAbs() || p.IsURL() || strings.HasPrefix(fileName, "/") || strings.HasPrefix(fileName, "\\") {
		return "", fmt.Errorf("%w: absolute or URL paths forbidden: %q", ErrInsecurePath, fileName)
	}

	// Validate depth traversal on raw un-sanitized input to detect active Zip Slip attempts
	rawNorm := strings.ReplaceAll(fileName, "\\", "/")
	parts := strings.Split(rawNorm, "/")
	depth := 0
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			depth--
			if depth < 0 {
				return "", fmt.Errorf("%w: path %q attempts directory traversal", ErrInsecurePath, fileName)
			}
		} else {
			depth++
			baseName := part
			if dotIdx := strings.IndexByte(baseName, '.'); dotIdx >= 0 {
				baseName = baseName[:dotIdx]
			}
			if reservedWindowsDeviceNames[strings.ToUpper(baseName)] {
				return "", fmt.Errorf("%w: reserved windows device name %q in %q", ErrInsecurePath, part, fileName)
			}
		}
	}

	sanitized := pathkit.Sanitize(fileName)
	destPath := pathkit.FromFilePath(destDir)
	cleanDest, err := filepath.Abs(destPath.FilePath())
	if err != nil {
		return "", err
	}

	targetPath := filepath.Clean(filepath.Join(cleanDest, filepath.FromSlash(sanitized)))

	rel, err := filepath.Rel(cleanDest, targetPath)
	if err != nil || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("%w: path %q escapes destination %q", ErrInsecurePath, fileName, destDir)
	}

	return targetPath, nil
}
