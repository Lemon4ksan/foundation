// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit

import (
	"errors"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrInvalidFileURI indicates that a URI cannot be converted to a local OS file path.
	ErrInvalidFileURI = errors.New("pathkit: invalid file:// URI")
)

// PathToURI converts a local OS file path into an RFC 8089 compliant file:// URI.
// On Windows, drive paths like "C:\foo\bar" become "file:///C:/foo/bar".
func PathToURI(osPath string) string {
	if osPath == "" {
		return ""
	}

	if strings.HasPrefix(osPath, "file://") ||
		strings.HasPrefix(osPath, "http://") ||
		strings.HasPrefix(osPath, "https://") {
		return osPath
	}

	normalized := filepath.ToSlash(osPath)

	// Windows UNC path: //server/share/file
	if strings.HasPrefix(normalized, "//") {
		return "file:" + normalized
	}

	// Windows drive letter: C:/foo/bar
	if len(normalized) >= 2 && normalized[1] == ':' {
		return "file:///" + normalized
	}

	// POSIX absolute path: /etc/hosts
	if strings.HasPrefix(normalized, "/") {
		return "file://" + normalized
	}

	// Relative path
	return "file:" + normalized
}

// URIToPath converts an RFC 8089 file:// URI into a local OS file path.
// On Windows, "file:///C:/foo/bar" becomes "C:\foo\bar".
// On Unix, "file:///etc/hosts" becomes "/etc/hosts".
func URIToPath(uriStr string) (string, error) {
	if uriStr == "" {
		return "", nil
	}

	if !strings.HasPrefix(uriStr, "file://") {
		return "", ErrInvalidFileURI
	}

	u, err := url.Parse(uriStr)
	if err != nil {
		return "", fmtURIError(err)
	}

	rawPath, err := url.PathUnescape(u.Path)
	if err != nil {
		return "", err
	}

	if runtime.GOOS == "windows" {
		// Windows drive letter: /C:/foo -> C:\foo
		if len(rawPath) >= 3 && rawPath[0] == '/' && rawPath[2] == ':' {
			rawPath = rawPath[1:]
		}

		// Windows UNC share
		if u.Host != "" && u.Host != "localhost" {
			return filepath.FromSlash("//" + u.Host + rawPath), nil
		}

		return filepath.FromSlash(rawPath), nil
	}

	if u.Host != "" && u.Host != "localhost" {
		return "", errors.New("pathkit: remote file host not supported on non-windows platform")
	}

	return rawPath, nil
}

func fmtURIError(err error) error {
	return errors.New("pathkit: parse URI: " + err.Error())
}
