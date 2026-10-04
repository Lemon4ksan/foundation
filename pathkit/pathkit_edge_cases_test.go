// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit_test

import (
	"runtime"
	"testing"

	"github.com/lemon4ksan/foundation/pathkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestPath_EdgeCasesAndUncoveredBranches(t *testing.T) {
	t.Parallel()

	// 1. FromFilePath and FromURI
	p1 := pathkit.FromFilePath("/var/log/syslog")
	assert.Equal(t, "/var/log/syslog", p1.String())

	p2, err := pathkit.FromURI("https://api.github.com/repos")
	require.NoError(t, err)
	assert.Equal(t, "https://api.github.com/repos", p2.String())

	// FromURL nil
	pNil := pathkit.FromURL(nil)
	assert.True(t, pNil.IsEmpty())

	// 2. Scheme normalization (uppercase and mixed-case)
	assert.Equal(t, "http", pathkit.New("HTTP://example.com").Scheme())
	assert.Equal(t, "https", pathkit.New("HTTPS://example.com").Scheme())
	assert.Equal(t, "file", pathkit.New("FILE:///var/log").Scheme())
	assert.Equal(t, "s3", pathkit.New("S3://my-bucket/key").Scheme())
	assert.Equal(t, "ws", pathkit.New("WS://socket.io").Scheme())
	assert.Equal(t, "wss", pathkit.New("WSS://socket.io").Scheme())
	assert.Equal(t, "custom", pathkit.New("CUSTOM://my/path").Scheme())

	// 3. Base edge cases
	assert.Equal(t, "/", pathkit.New("/").Base())
	assert.Equal(t, "/", pathkit.New(`\`).Base())
	assert.Equal(t, ".", pathkit.New("").Base())
	assert.Equal(t, "C:", pathkit.New("C:/").Base())
	assert.Equal(t, "file", pathkit.New("https://domain.com/path/to/file/").Base())

	// 4. Dir edge cases
	assert.Equal(t, ".", pathkit.New("").Dir().String())
	assert.Equal(t, "/", pathkit.New("/").Dir().String())
	assert.Equal(t, "C:/", pathkit.New("C:/file.txt").Dir().String())
	assert.Equal(t, ".", pathkit.New("file.txt").Dir().String())
	assert.Equal(t, "/", pathkit.New("/file.txt").Dir().String())
	assert.Equal(t, "https://api.com", pathkit.New("https://api.com/v1").Dir().String())
	assert.Equal(t, "https://api.com", pathkit.New("https://api.com").Dir().String())

	// 5. WithExt edge cases
	assert.True(t, pathkit.New("").WithExt(".txt").IsEmpty())

	// 6. Join edge cases
	pEmpty := pathkit.New("")
	assert.Equal(t, "a/b", pEmpty.Join("a", "b").String())
	assert.Equal(t, "/var/log", pathkit.New("/var/log").Join().String())

	// 7. AppendFilePath & AppendURI on file://
	fileURI := pathkit.New("file:///C:/Users/senya/app.json")
	buf := make([]byte, 0, 64)
	buf = fileURI.AppendFilePath(buf)
	assert.NotEmpty(t, buf)

	buf = buf[:0]
	buf = fileURI.AppendURI(buf)
	assert.Equal(t, "file:///C:/Users/senya/app.json", string(buf))

	// URL URI
	urlP := pathkit.New("https://example.com/api")
	buf = buf[:0]
	buf = urlP.AppendURI(buf)
	assert.Equal(t, "https://example.com/api", string(buf))
}

func TestFileURI_EdgeCases(t *testing.T) {
	t.Parallel()

	// 1. Invalid URI prefix
	_, err := pathkit.URIToPath("https://example.com/file")
	assert.ErrorIs(t, err, pathkit.ErrInvalidFileURI)

	// Empty URI
	pathEmpty, err := pathkit.URIToPath("")
	require.NoError(t, err)
	assert.Equal(t, "", pathEmpty)

	// 2. Percent-encoded URI
	pathWithSpaces, err := pathkit.URIToPath("file:///var/log/my%20app%20dir/file.log")
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, `\var\log\my app dir\file.log`, pathWithSpaces)
	} else {
		assert.Equal(t, "/var/log/my app dir/file.log", pathWithSpaces)
	}

	// 3. Invalid percent encoding
	_, err = pathkit.URIToPath("file:///var/log/bad%2")
	assert.Error(t, err)

	_, err = pathkit.URIToPath("file:///var/log/bad%ZZ")
	assert.Error(t, err)

	// 4. Localhost host in URI
	localHostPath, err := pathkit.URIToPath("file://localhost/etc/hosts")
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, `\etc\hosts`, localHostPath)
	} else {
		assert.Equal(t, "/etc/hosts", localHostPath)
	}

	// 5. UNC Windows PathToURI
	uncPath := `\\server\share\folder\file.txt`
	uncURI := pathkit.PathToURI(uncPath)
	assert.Equal(t, "file://server/share/folder/file.txt", uncURI)

	// Relative PathToURI
	relPath := `relative/sub/file.txt`
	relURI := pathkit.PathToURI(relPath)
	assert.Equal(t, "file:relative/sub/file.txt", relURI)

	// Pass-through existing URIs
	assert.Equal(t, "file:///etc/hosts", pathkit.PathToURI("file:///etc/hosts"))
	assert.Equal(t, "https://example.com", pathkit.PathToURI("https://example.com"))
	assert.Equal(t, "http://example.com", pathkit.PathToURI("http://example.com"))

	// 6. Windows UNC URIToPath
	if runtime.GOOS == "windows" {
		uncBack, err := pathkit.URIToPath("file://server/share/folder/file.txt")
		require.NoError(t, err)
		assert.Equal(t, `\\server\share\folder\file.txt`, uncBack)
	} else {
		_, err := pathkit.URIToPath("file://server/share/folder/file.txt")
		assert.Error(t, err)
	}
}

func TestClean_EdgeCases(t *testing.T) {
	t.Parallel()

	// Lexical root popping
	assert.Equal(t, "/", pathkit.Clean("/.."))
	assert.Equal(t, "/", pathkit.Clean("/a/b/../../.."))
	assert.Equal(t, "../../a", pathkit.Clean("../../a"))
	assert.Equal(t, ".", pathkit.Clean("a/.."))
	assert.Equal(t, "a", pathkit.Clean("./a"))
	assert.Equal(t, "/", pathkit.Clean("///"))

	// URL clean with no path after authority
	assert.Equal(t, "https://domain.com", pathkit.Clean("https://domain.com"))
}

func TestEarlyTermination(t *testing.T) {
	t.Parallel()

	// Segments2 early termination
	p := pathkit.New("/a/b/c/d/e")
	count := 0
	for range p.Segments2() {
		count++
		if count == 2 {
			break
		}
	}
	assert.Equal(t, 2, count)

	// Ancestors early termination
	ancCount := 0
	for range p.Ancestors() {
		ancCount++
		if ancCount == 2 {
			break
		}
	}
	assert.Equal(t, 2, ancCount)
}
