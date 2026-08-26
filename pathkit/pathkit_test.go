// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit_test

import (
	"net/url"
	"runtime"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"

	"github.com/lemon4ksan/foundation/pathkit"
)

func TestPath_ConstructorsAndClassification(t *testing.T) {
	t.Parallel()

	// 1. Web URL
	p1 := pathkit.New("https://api.github.com/v1/users")
	assert.True(t, p1.IsURL())
	assert.False(t, p1.IsFile())
	assert.True(t, p1.IsAbs())
	assert.Equal(t, "https", p1.Scheme())
	assert.Equal(t, "https://api.github.com/v1/users", p1.String())

	// 2. Local Unix path
	p2 := pathkit.New("/var/log/app.log")
	assert.False(t, p2.IsURL())
	assert.True(t, p2.IsFile())
	assert.True(t, p2.IsAbs())
	assert.Equal(t, "", p2.Scheme())

	// 3. Local Windows path
	p3 := pathkit.New(`C:\Users\senya\app.json`)
	assert.False(t, p3.IsURL())
	assert.True(t, p3.IsFile())
	assert.True(t, p3.IsAbs())
	assert.Equal(t, "C:/Users/senya/app.json", p3.String())

	// 4. File URI
	p4 := pathkit.New("file:///C:/Users/senya/app.json")
	assert.False(t, p4.IsURL())
	assert.True(t, p4.IsFile())
	assert.True(t, p4.IsAbs())
	assert.Equal(t, "file", p4.Scheme())

	// 5. FromURL
	u, _ := url.Parse("https://example.com/test")
	p5 := pathkit.FromURL(u)
	assert.Equal(t, "https://example.com/test", p5.String())
	assert.True(t, p5.IsURL())

	// 6. Empty
	empty := pathkit.New("")
	assert.True(t, empty.IsEmpty())
	assert.False(t, empty.IsAbs())
}

func TestPath_Navigation(t *testing.T) {
	t.Parallel()

	// 1. Base, Ext, Stem, Dir on file path
	p := pathkit.New("/var/log/app.prod.json")
	assert.Equal(t, "app.prod.json", p.Base())
	assert.Equal(t, ".json", p.Ext())
	assert.Equal(t, "app.prod", p.Stem())
	assert.Equal(t, "/var/log", p.Dir().String())

	// 2. WithExt
	yamlP := p.WithExt(".yaml")
	assert.Equal(t, "/var/log/app.prod.yaml", yamlP.String())
	assert.Equal(t, ".yaml", yamlP.Ext())

	withExtNoDot := p.WithExt("toml")
	assert.Equal(t, "/var/log/app.prod.toml", withExtNoDot.String())

	// 3. Base & Dir on URL
	urlP := pathkit.New("https://api.example.com/v1/users/42")
	assert.Equal(t, "42", urlP.Base())
	assert.Equal(t, "https://api.example.com/v1/users", urlP.Dir().String())
	assert.Equal(t, "", urlP.Ext())
}

func TestPath_JoinAndClean(t *testing.T) {
	t.Parallel()

	// 1. URL Join (preserves https:// without collapsing)
	api := pathkit.New("https://api.example.com")
	joined := api.Join("v1", "users", "123")
	assert.Equal(t, "https://api.example.com/v1/users/123", joined.String())

	// Trailing and leading slashes
	apiSlash := pathkit.New("https://api.example.com/")
	joined2 := apiSlash.Join("/v1/", "/users/", "123")
	assert.Equal(t, "https://api.example.com/v1/users/123", joined2.String())

	// 2. Clean URL
	dirtyURL := pathkit.New("https://api.com/v1/../v2/users/./42")
	assert.Equal(t, "https://api.com/v2/users/42", dirtyURL.Clean().String())

	// 3. Clean File
	dirtyFile := pathkit.New("/a/b/../c/./d")
	assert.Equal(t, "/a/c/d", dirtyFile.Clean().String())
}

func TestPath_CrossPlatformFileURI(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		winPath := `C:\CodingProjects\aoni\go.mod`
		uri := pathkit.PathToURI(winPath)
		assert.Equal(t, "file:///C:/CodingProjects/aoni/go.mod", uri)

		backPath, err := pathkit.URIToPath(uri)
		require.NoError(t, err)
		assert.Equal(t, winPath, backPath)

		p := pathkit.New(uri)
		assert.Equal(t, winPath, p.FilePath())
	} else {
		unixPath := "/etc/hosts"
		uri := pathkit.PathToURI(unixPath)
		assert.Equal(t, "file:///etc/hosts", uri)

		backPath, err := pathkit.URIToPath(uri)
		require.NoError(t, err)
		assert.Equal(t, unixPath, backPath)

		p := pathkit.New(uri)
		assert.Equal(t, unixPath, p.FilePath())
	}
}

func TestSanitize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "etc/passwd", pathkit.Sanitize("../../../etc/passwd"))
	assert.Equal(t, "var/log", pathkit.Sanitize(`..\..\var\log`))
	assert.Equal(t, "safe/path", pathkit.Sanitize("./safe/path"))
	assert.Equal(t, "", pathkit.Sanitize(""))
}

func BenchmarkPath_JoinURL(b *testing.B) {
	base := "https://api.github.com/v1"

	b.ReportAllocs()
	for b.Loop() {
		_ = pathkit.JoinURL(base, "repos", "lemon4ksan", "aoni", "issues", "100")
	}
}

func BenchmarkPath_New_And_Methods(b *testing.B) {
	s := "https://api.example.com/v1/service/app.json"

	b.ReportAllocs()
	for b.Loop() {
		p := pathkit.New(s)
		_ = p.Base()
		_ = p.Ext()
		_ = p.Dir()
	}
}
