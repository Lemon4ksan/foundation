// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit

import (
	"path/filepath"
	"strings"
)

var incompressibleExtensions = map[string]struct{}{
	".zip": {}, ".7z": {}, ".rar": {}, ".gz": {}, ".tgz": {},
	".bz2": {}, ".xz": {}, ".zst": {}, ".eon": {},
	".mp4": {}, ".mkv": {}, ".avi": {}, ".mov": {}, ".webm": {},
	".mp3": {}, ".aac": {}, ".ogg": {}, ".flac": {},
	".jpg": {}, ".jpeg": {}, ".png": {}, ".webp": {}, ".gif": {},
	".pdf": {}, ".docx": {}, ".xlsx": {}, ".pptx": {},
	".jar": {}, ".apk": {}, ".iso": {},
}

// IsIncompressibleExtension checks if a filename extension corresponds to already compressed or encrypted media.
func IsIncompressibleExtension(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := incompressibleExtensions[ext]
	return ok
}
