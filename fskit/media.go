// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fskit

import (
	"slices"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

func toLowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func findExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		b := path[i]
		if b == '/' || b == '\\' {
			return ""
		}
		if b == '.' {
			return path[i:]
		}
	}
	return ""
}

func findExtBytes(path []byte) []byte {
	for i, b := range slices.Backward(path) {

		if b == '/' || b == '\\' {
			return nil
		}
		if b == '.' {
			return path[i:]
		}
	}
	return nil
}

// IsIncompressibleExt reports whether a file extension corresponds to already compressed or encrypted media.
// It accepts extensions with or without a leading dot (e.g. ".zip" or "zip") in a case-insensitive manner.
func IsIncompressibleExt(ext string) bool {
	if len(ext) > 0 && ext[0] == '.' {
		ext = ext[1:]
	}
	switch len(ext) {
	case 2:
		c0, c1 := toLowerByte(ext[0]), toLowerByte(ext[1])
		return (c0 == '7' && c1 == 'z') ||
			(c0 == 'g' && c1 == 'z') ||
			(c0 == 'x' && c1 == 'z')
	case 3:
		c0, c1, c2 := toLowerByte(ext[0]), toLowerByte(ext[1]), toLowerByte(ext[2])
		switch c0 {
		case 'a':
			return (c1 == 'v' && c2 == 'i') || (c1 == 'a' && c2 == 'c') || (c1 == 'p' && c2 == 'k') // avi, aac, apk
		case 'b':
			return c1 == 'z' && c2 == '2' // bz2
		case 'e':
			return c1 == 'o' && c2 == 'n' // eon
		case 'g':
			return c1 == 'i' && c2 == 'f' // gif
		case 'i':
			return c1 == 's' && c2 == 'o' // iso
		case 'j':
			return (c1 == 'a' && c2 == 'r') || (c1 == 'p' && c2 == 'g') // jar, jpg
		case 'm':
			return (c1 == 'p' && (c2 == '3' || c2 == '4')) ||
				(c1 == 'k' && c2 == 'v') ||
				(c1 == 'o' && c2 == 'v') // mp3, mp4, mkv, mov
		case 'o':
			return c1 == 'g' && c2 == 'g' // ogg
		case 'p':
			return (c1 == 'n' && c2 == 'g') || (c1 == 'd' && c2 == 'f') // png, pdf
		case 'r':
			return c1 == 'a' && c2 == 'r' // rar
		case 't':
			return c1 == 'g' && c2 == 'z' // tgz
		case 'z':
			return (c1 == 'i' && c2 == 'p') || (c1 == 's' && c2 == 't') // zip, zst
		}
		return false
	case 4:
		c0, c1, c2, c3 := toLowerByte(ext[0]), toLowerByte(ext[1]), toLowerByte(ext[2]), toLowerByte(ext[3])
		switch c0 {
		case 'w':
			return (c1 == 'e' && c2 == 'b' && (c3 == 'm' || c3 == 'p')) // webm, webp
		case 'f':
			return c1 == 'l' && c2 == 'a' && c3 == 'c' // flac
		case 'j':
			return c1 == 'p' && c2 == 'e' && c3 == 'g' // jpeg
		case 'd':
			return c1 == 'o' && c2 == 'c' && c3 == 'x' // docx
		case 'x':
			return c1 == 'l' && c2 == 's' && c3 == 'x' // xlsx
		case 'p':
			return c1 == 'p' && c2 == 't' && c3 == 'x' // pptx
		}
		return false
	default:
		return false
	}
}

// IsIncompressibleExtBytes reports whether a byte slice file extension corresponds to already compressed or encrypted media.
// It accepts extensions with or without a leading dot in a case-insensitive manner.
func IsIncompressibleExtBytes(ext []byte) bool {
	return IsIncompressibleExt(bytesconv.B2S(ext))
}

// IsIncompressibleExtension checks if a filename extension corresponds to already compressed or encrypted media.
func IsIncompressibleExtension(name string) bool {
	ext := findExt(name)
	if ext == "" {
		return false
	}
	return IsIncompressibleExt(ext)
}

// IsIncompressibleExtensionBytes checks if a byte slice filename extension corresponds to already compressed or encrypted media.
func IsIncompressibleExtensionBytes(name []byte) bool {
	ext := findExtBytes(name)
	if len(ext) == 0 {
		return false
	}
	return IsIncompressibleExtBytes(ext)
}
