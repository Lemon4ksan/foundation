// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit

import (
	"errors"
	"runtime"
	"strings"

	"github.com/lemon4ksan/foundation/silicon/bytesconv"
)

// ErrInvalidFileURI indicates that a URI cannot be converted to a local OS file path.
var ErrInvalidFileURI = errors.New("pathkit: invalid file:// URI")

// AppendPathToURI converts a local OS file path into an RFC 8089 compliant file:// URI,
// appending the result into dst. When dst has sufficient capacity, it appends without heap allocations.
func AppendPathToURI(dst []byte, osPath string) []byte {
	if osPath == "" {
		return dst
	}

	if strings.HasPrefix(osPath, "file://") ||
		strings.HasPrefix(osPath, "http://") ||
		strings.HasPrefix(osPath, "https://") {
		return append(dst, osPath...)
	}

	// Check for UNC path (starts with // or \\)
	isUNC := len(osPath) >= 2 && (osPath[0] == '/' || osPath[0] == '\\') && (osPath[1] == '/' || osPath[1] == '\\')
	if isUNC {
		dst = append(dst, "file:"...)
		for i := 0; i < len(osPath); i++ {
			b := osPath[i]
			if b == '\\' {
				dst = append(dst, '/')
			} else {
				dst = append(dst, b)
			}
		}
		return dst
	}

	// Check for Windows drive letter: C:\ or C:/
	isDrive := len(osPath) >= 2 && osPath[1] == ':' &&
		((osPath[0] >= 'a' && osPath[0] <= 'z') || (osPath[0] >= 'A' && osPath[0] <= 'Z'))

	if isDrive {
		dst = append(dst, "file:///"...)
		for i := 0; i < len(osPath); i++ {
			b := osPath[i]
			if b == '\\' {
				dst = append(dst, '/')
			} else {
				dst = append(dst, b)
			}
		}
		return dst
	}

	// POSIX absolute path: /etc/hosts or \etc\hosts
	if len(osPath) > 0 && (osPath[0] == '/' || osPath[0] == '\\') {
		dst = append(dst, "file://"...)
		for i := 0; i < len(osPath); i++ {
			b := osPath[i]
			if b == '\\' {
				dst = append(dst, '/')
			} else {
				dst = append(dst, b)
			}
		}
		return dst
	}

	// Relative path
	dst = append(dst, "file:"...)
	for i := 0; i < len(osPath); i++ {
		b := osPath[i]
		if b == '\\' {
			dst = append(dst, '/')
		} else {
			dst = append(dst, b)
		}
	}

	return dst
}

// PathToURI converts a local OS file path into an RFC 8089 compliant file:// URI.
// On Windows, drive paths like "C:\foo\bar" become "file:///C:/foo/bar".
func PathToURI(osPath string) string {
	if osPath == "" {
		return ""
	}

	buf := make([]byte, 0, len(osPath)+8)
	buf = AppendPathToURI(buf, osPath)
	return bytesconv.B2S(buf)
}

// AppendURIToPath converts an RFC 8089 file:// URI into a local OS file path,
// appending the result into dst. When dst has sufficient capacity, it appends without heap allocations.
// On Windows, "file:///C:/foo/bar" becomes "C:\foo\bar".
// On Unix, "file:///etc/hosts" becomes "/etc/hosts".
func AppendURIToPath(dst []byte, uriStr string) ([]byte, error) {
	if uriStr == "" {
		return dst, nil
	}

	if !strings.HasPrefix(uriStr, "file:") {
		return dst, ErrInvalidFileURI
	}

	rest := uriStr[5:]

	var host string
	var rawPath string

	if strings.HasPrefix(rest, "//") {
		afterSlashes := rest[2:]
		slashIdx := strings.IndexByte(afterSlashes, '/')
		if slashIdx == -1 {
			host = afterSlashes
			rawPath = "/"
		} else {
			host = afterSlashes[:slashIdx]
			rawPath = afterSlashes[slashIdx:]
		}
	} else {
		rawPath = rest
	}

	if host == "localhost" {
		host = ""
	}

	if runtime.GOOS != "windows" && host != "" {
		return dst, errors.New("pathkit: remote file host not supported on non-windows platform")
	}

	unescapedStart := len(dst)
	var err error
	dst, err = unescapePercent(dst, rawPath)
	if err != nil {
		return dst, err
	}

	if runtime.GOOS == "windows" {
		unescaped := dst[unescapedStart:]

		// Windows drive letter: /C:/foo -> C:\foo
		if len(unescaped) >= 3 && unescaped[0] == '/' && unescaped[2] == ':' {
			copy(dst[unescapedStart:], unescaped[1:])
			dst = dst[:len(dst)-1]
			unescaped = dst[unescapedStart:]
		}

		if host != "" {
			// Windows UNC share: \\host\path
			temp := make([]byte, len(unescaped))
			copy(temp, unescaped)
			dst = dst[:unescapedStart]
			dst = append(dst, '\\', '\\')
			dst = append(dst, host...)
			for i := range temp {
				if temp[i] == '/' {
					dst = append(dst, '\\')
				} else {
					dst = append(dst, temp[i])
				}
			}
			return dst, nil
		}

		// Convert forward slashes to Windows backslashes
		for i := unescapedStart; i < len(dst); i++ {
			if dst[i] == '/' {
				dst[i] = '\\'
			}
		}
		return dst, nil
	}

	// On Unix, ensure forward slashes
	for i := unescapedStart; i < len(dst); i++ {
		if dst[i] == '\\' {
			dst[i] = '/'
		}
	}

	return dst, nil
}

// URIToPath converts an RFC 8089 file:// URI into a local OS file path.
// On Windows, "file:///C:/foo/bar" becomes "C:\foo\bar".
// On Unix, "file:///etc/hosts" becomes "/etc/hosts".
func URIToPath(uriStr string) (string, error) {
	if uriStr == "" {
		return "", nil
	}

	buf := make([]byte, 0, len(uriStr))
	var err error
	buf, err = AppendURIToPath(buf, uriStr)
	if err != nil {
		return "", err
	}

	return bytesconv.B2S(buf), nil
}

func unescapePercent(dst []byte, s string) ([]byte, error) {
	hasPercent := false
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			hasPercent = true
			break
		}
	}
	if !hasPercent {
		return append(dst, s...), nil
	}

	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == '%' {
			if i+2 >= len(s) {
				return dst, errors.New("pathkit: invalid percent encoding in URI")
			}
			h1 := fromHex(s[i+1])
			h2 := fromHex(s[i+2])
			if h1 < 0 || h2 < 0 {
				return dst, errors.New("pathkit: invalid percent encoding in URI")
			}
			dst = append(dst, byte((h1<<4)|h2))
			i += 2
		} else {
			dst = append(dst, b)
		}
	}
	return dst, nil
}

func fromHex(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c - 'a' + 10)
	case 'A' <= c && c <= 'F':
		return int(c - 'A' + 10)
	}
	return -1
}
