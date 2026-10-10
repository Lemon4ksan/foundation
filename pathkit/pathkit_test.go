// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit_test

import (
	"net/url"
	"runtime"
	"slices"
	"testing"

	"github.com/lemon4ksan/foundation/pathkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
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

	pNil := pathkit.FromURL(nil)
	assert.True(t, pNil.IsEmpty())

	// 6. FromFilePath & FromURI
	pFromPath := pathkit.FromFilePath("/var/log/syslog")
	assert.Equal(t, "/var/log/syslog", pFromPath.String())

	pFromURI, err := pathkit.FromURI("https://api.github.com/repos")
	require.NoError(t, err)
	assert.Equal(t, "https://api.github.com/repos", pFromURI.String())

	// 7. Scheme normalization (uppercase & mixed case)
	assert.Equal(t, "http", pathkit.New("HTTP://example.com").Scheme())
	assert.Equal(t, "https", pathkit.New("HTTPS://example.com").Scheme())
	assert.Equal(t, "file", pathkit.New("FILE:///var/log").Scheme())
	assert.Equal(t, "s3", pathkit.New("S3://my-bucket/key").Scheme())
	assert.Equal(t, "ws", pathkit.New("WS://socket.io").Scheme())
	assert.Equal(t, "wss", pathkit.New("WSS://socket.io").Scheme())
	assert.Equal(t, "custom", pathkit.New("CUSTOM://my/path").Scheme())

	// 8. Empty
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

	assert.True(t, pathkit.New("").WithExt(".txt").IsEmpty())

	// 3. Base & Dir on URL
	urlP := pathkit.New("https://api.example.com/v1/users/42")
	assert.Equal(t, "42", urlP.Base())
	assert.Equal(t, "https://api.example.com/v1/users", urlP.Dir().String())
	assert.Equal(t, "", urlP.Ext())

	// 4. Base edge cases
	assert.Equal(t, "/", pathkit.New("/").Base())
	assert.Equal(t, "/", pathkit.New(`\`).Base())
	assert.Equal(t, ".", pathkit.New("").Base())
	assert.Equal(t, "C:", pathkit.New("C:/").Base())
	assert.Equal(t, "file", pathkit.New("https://domain.com/path/to/file/").Base())

	// 5. Dir edge cases
	assert.Equal(t, ".", pathkit.New("").Dir().String())
	assert.Equal(t, "/", pathkit.New("/").Dir().String())
	assert.Equal(t, "C:/", pathkit.New("C:/file.txt").Dir().String())
	assert.Equal(t, ".", pathkit.New("file.txt").Dir().String())
	assert.Equal(t, "/", pathkit.New("/file.txt").Dir().String())
	assert.Equal(t, "https://api.com", pathkit.New("https://api.com/v1").Dir().String())
	assert.Equal(t, "https://api.com", pathkit.New("https://api.com").Dir().String())
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

	// Join with empty Path
	assert.Equal(t, "a/b", pathkit.New("").Join("a", "b").String())
	assert.Equal(t, "/var/log", pathkit.New("/var/log").Join().String())

	// 2. Clean URL
	dirtyURL := pathkit.New("https://api.com/v1/../v2/users/./42")
	assert.Equal(t, "https://api.com/v2/users/42", dirtyURL.Clean().String())

	// 3. Clean File
	dirtyFile := pathkit.New("/a/b/../c/./d")
	assert.Equal(t, "/a/c/d", dirtyFile.Clean().String())

	// Lexical root popping in Clean
	assert.Equal(t, "/", pathkit.Clean("/.."))
	assert.Equal(t, "/", pathkit.Clean("/a/b/../../.."))
	assert.Equal(t, "../../a", pathkit.Clean("../../a"))
	assert.Equal(t, ".", pathkit.Clean("a/.."))
	assert.Equal(t, "a", pathkit.Clean("./a"))
	assert.Equal(t, "/", pathkit.Clean("///"))
	assert.Equal(t, "https://domain.com", pathkit.Clean("https://domain.com"))
}

func TestPath_AppendMethods(t *testing.T) {
	t.Parallel()

	p := pathkit.New("/var/log/app.json")

	// 1. Append
	buf := make([]byte, 0, 64)
	buf = p.Append(buf)
	assert.Equal(t, "/var/log/app.json", string(buf))

	// 2. AppendFilePath
	buf = buf[:0]
	buf = p.AppendFilePath(buf)
	assert.Equal(t, p.FilePath(), string(buf))

	// 3. AppendURI
	buf = buf[:0]
	buf = p.AppendURI(buf)
	assert.Equal(t, p.URI(), string(buf))

	// 4. Append on file:// and URL
	fileURI := pathkit.New("file:///C:/Users/senya/app.json")
	buf = buf[:0]
	buf = fileURI.AppendFilePath(buf)
	assert.NotEmpty(t, buf)

	buf = buf[:0]
	buf = fileURI.AppendURI(buf)
	assert.Equal(t, "file:///C:/Users/senya/app.json", string(buf))

	urlP := pathkit.New("https://example.com/api")
	buf = buf[:0]
	buf = urlP.AppendURI(buf)
	assert.Equal(t, "https://example.com/api", string(buf))

	// 5. AppendJoinURL
	buf = buf[:0]
	buf = pathkit.AppendJoinURL(buf, "https://api.example.com", "v1", "users", "100")
	assert.Equal(t, "https://api.example.com/v1/users/100", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendJoinURL(buf, "", "base", "part")
	assert.Equal(t, "base/part", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendJoinURL(buf, "https://api.example.com/", "/v1/", "/users/")
	assert.Equal(t, "https://api.example.com/v1/users/", string(buf))

	// 6. AppendClean
	buf = buf[:0]
	buf = pathkit.AppendClean(buf, "/a/b/../c/./d")
	assert.Equal(t, "/a/c/d", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendClean(buf, "https://domain.com/v1/../v2/app")
	assert.Equal(t, "https://domain.com/v2/app", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendClean(buf, "")
	assert.Equal(t, ".", string(buf))

	// 7. AppendSanitize
	buf = buf[:0]
	buf = pathkit.AppendSanitize(buf, "../../../etc/passwd")
	assert.Equal(t, "etc/passwd", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendSanitize(buf, `..\..\windows\system32`)
	assert.Equal(t, "windows/system32", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendSanitize(buf, "")
	assert.Equal(t, "", string(buf))
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

		// Windows UNC
		uncPath := `\\server\share\folder\file.txt`
		uncURI := pathkit.PathToURI(uncPath)
		assert.Equal(t, "file://server/share/folder/file.txt", uncURI)

		uncBack, err := pathkit.URIToPath(uncURI)
		require.NoError(t, err)
		assert.Equal(t, uncPath, uncBack)
	} else {
		unixPath := "/etc/hosts"
		uri := pathkit.PathToURI(unixPath)
		assert.Equal(t, "file:///etc/hosts", uri)

		backPath, err := pathkit.URIToPath(uri)
		require.NoError(t, err)
		assert.Equal(t, unixPath, backPath)

		p := pathkit.New(uri)
		assert.Equal(t, unixPath, p.FilePath())

		_, err = pathkit.URIToPath("file://server/share/folder/file.txt")
		assert.Error(t, err)
	}

	// General URI edge cases
	_, err := pathkit.URIToPath("https://example.com/file")
	assert.ErrorIs(t, err, pathkit.ErrInvalidFileURI)

	pathEmpty, err := pathkit.URIToPath("")
	require.NoError(t, err)
	assert.Equal(t, "", pathEmpty)

	// Percent-encoded URI
	pathWithSpaces, err := pathkit.URIToPath("file:///var/log/my%20app%20dir/file.log")
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, `\var\log\my app dir\file.log`, pathWithSpaces)
	} else {
		assert.Equal(t, "/var/log/my app dir/file.log", pathWithSpaces)
	}

	// Invalid percent encoding
	_, err = pathkit.URIToPath("file:///var/log/bad%2")
	assert.Error(t, err)
	_, err = pathkit.URIToPath("file:///var/log/bad%ZZ")
	assert.Error(t, err)

	// Localhost
	localHostPath, err := pathkit.URIToPath("file://localhost/etc/hosts")
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		assert.Equal(t, `\etc\hosts`, localHostPath)
	} else {
		assert.Equal(t, "/etc/hosts", localHostPath)
	}

	// Relative PathToURI
	assert.Equal(t, "file:relative/sub/file.txt", pathkit.PathToURI(`relative/sub/file.txt`))
	assert.Equal(t, "file:///etc/hosts", pathkit.PathToURI("file:///etc/hosts"))
	assert.Equal(t, "https://example.com", pathkit.PathToURI("https://example.com"))
}

func TestPath_Sanitize(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "etc/passwd", pathkit.Sanitize("../../../etc/passwd"))
	assert.Equal(t, "var/log", pathkit.Sanitize(`..\..\var\log`))
	assert.Equal(t, "safe/path", pathkit.Sanitize("./safe/path"))
	assert.Equal(t, "", pathkit.Sanitize(""))
	assert.Equal(t, "Windows/System32", pathkit.Sanitize(`C:\Windows\System32`))
	assert.Equal(t, "Windows/System32", pathkit.Sanitize(`C:/Windows/System32`))
}

func TestPath_Dir_WindowsDriveRoot(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "C:/", pathkit.New("C:/").Dir().String())
	assert.Equal(t, "C:/", pathkit.New(`C:\`).Dir().String())
}

func TestPath_Iterators(t *testing.T) {
	t.Parallel()

	pathStr := "/usr/local/bin/foundation/exec"

	// 1. SegmentsSeq early termination
	count := 0
	for range pathkit.Segments(pathStr) {
		count++
		break
	}
	assert.Equal(t, 1, count)

	count = 0
	for range pathkit.Segments(pathStr) {
		count++
		if count == 2 {
			break
		}
	}
	assert.Equal(t, 2, count)

	// 2. Full SegmentsSeq traversal
	var segments []string
	for seg := range pathkit.Segments(pathStr) {
		segments = append(segments, seg)
	}
	expected := []string{"usr", "local", "bin", "foundation", "exec"}
	assert.True(t, slices.Equal(segments, expected))

	// 3. SegmentsSeq edge cases (empty & multiple slashes)
	for range pathkit.Segments("") {
		t.Fatal("expected no segments for empty string")
	}
	for range pathkit.Segments("////") {
		t.Fatal("expected no segments for slashes only")
	}

	// 4. URL format segments
	urlPath := "https://api.github.com/v1/repos/foundation"
	var urlSegs []string
	for seg := range pathkit.Segments(urlPath) {
		urlSegs = append(urlSegs, seg)
	}
	assert.Equal(t, []string{"v1", "repos", "foundation"}, urlSegs)

	// 5. Segments2
	p := pathkit.New("/usr/local/bin")
	var idxs []int
	var segs []string
	for i, seg := range p.Segments2() {
		idxs = append(idxs, i)
		segs = append(segs, seg)
	}
	assert.Equal(t, []int{0, 1, 2}, idxs)
	assert.Equal(t, []string{"usr", "local", "bin"}, segs)

	count2 := 0
	for range p.Segments2() {
		count2++
		if count2 == 2 {
			break
		}
	}
	assert.Equal(t, 2, count2)

	// 6. Ancestors
	ancP := pathkit.New("/a/b/c")
	var anc []string
	for a := range ancP.Ancestors() {
		anc = append(anc, a.String())
	}
	assert.Equal(t, []string{"/a/b", "/a", "/"}, anc)

	urlP := pathkit.New("https://example.com/v1/api/users")
	var urlAnc []string
	for a := range urlP.Ancestors() {
		urlAnc = append(urlAnc, a.String())
	}
	assert.Equal(t, []string{"https://example.com/v1/api", "https://example.com/v1", "https://example.com"}, urlAnc)

	rootP := pathkit.New("/")
	var rootAnc []string
	for a := range rootP.Ancestors() {
		rootAnc = append(rootAnc, a.String())
	}
	assert.Empty(t, rootAnc)

	ancCount := 0
	for range ancP.Ancestors() {
		ancCount++
		if ancCount == 2 {
			break
		}
	}
	assert.Equal(t, 2, ancCount)
}

func TestPath_LexicalRelations(t *testing.T) {
	t.Parallel()

	// 1. HasPrefix
	p1 := pathkit.New("/var/log/nginx/access.log")
	assert.True(t, p1.HasPrefix(pathkit.New("/var/log")))
	assert.True(t, p1.HasPrefix(pathkit.New("/var/log/nginx")))
	assert.True(t, p1.HasPrefix(pathkit.New("/var/log/nginx/access.log")))
	assert.False(t, p1.HasPrefix(pathkit.New("/var/lib")))
	assert.False(t, p1.HasPrefix(pathkit.New("/var/log/ngin")))
	assert.False(t, p1.HasPrefix(pathkit.New("https://var/log")))

	// 2. Rel
	base := pathkit.New("/a/b")
	rel, ok := pathkit.New("/a/b/c/d").Rel(base)
	require.True(t, ok)
	assert.Equal(t, "c/d", rel.String())

	rel, ok = pathkit.New("/a/b").Rel(base)
	require.True(t, ok)
	assert.Equal(t, ".", rel.String())

	rel, ok = pathkit.New("/a/c").Rel(base)
	require.True(t, ok)
	assert.Equal(t, "../c", rel.String())

	_, ok = pathkit.New("https://api.com/a").Rel(base)
	assert.False(t, ok)

	// 3. Match
	pMatch := pathkit.New("/var/log/app.prod.json")
	assert.True(t, pMatch.Match("*.json"))
	assert.True(t, pMatch.Match("app.*.json"))
	assert.False(t, pMatch.Match("*.log"))
}

func TestPath_ZeroAllocGuarantees(t *testing.T) {
	// 1. SegmentsSeq on Unix path
	unixPath := "/usr/local/bin/exec"
	allocs := testing.AllocsPerRun(1000, func() {
		for seg := range pathkit.Segments(unixPath) {
			_ = seg
		}
	})
	assert.Equal(t, 0.0, allocs)

	// 2. SegmentsSeq on Windows path (contains backslashes)
	winPath := `C:\Windows\System32\drivers\etc\hosts`
	allocsWin := testing.AllocsPerRun(1000, func() {
		for seg := range pathkit.Segments(winPath) {
			_ = seg
		}
	})
	assert.Equal(t, 0.0, allocsWin)

	// 3. Base, Ext, Stem, IsAbs, Scheme on Path
	p := pathkit.New("https://api.github.com/v1/repos/foundation/app.json")
	allocsMethods := testing.AllocsPerRun(1000, func() {
		_ = p.Base()
		_ = p.Ext()
		_ = p.Stem()
		_ = p.IsAbs()
		_ = p.Scheme()
	})
	assert.Equal(t, 0.0, allocsMethods)

	// 4. AppendJoinURL with pre-allocated buffer
	scratch := make([]byte, 0, 128)
	allocsJoin := testing.AllocsPerRun(1000, func() {
		buf := scratch[:0]
		_ = pathkit.AppendJoinURL(buf, "https://api.github.com/v1", "repos", "lemon4ksan", "aoni")
	})
	assert.Equal(t, 0.0, allocsJoin)

	// 5. AppendClean with pre-allocated buffer
	allocsClean := testing.AllocsPerRun(1000, func() {
		buf := scratch[:0]
		_ = pathkit.AppendClean(buf, "/a/b/../c/./d")
	})
	assert.Equal(t, 0.0, allocsClean)

	// 6. Ancestors iteration
	ancPath := pathkit.New("/var/log/nginx/access.log")
	allocsAnc := testing.AllocsPerRun(1000, func() {
		for a := range ancPath.Ancestors() {
			_ = a
		}
	})
	assert.Equal(t, 0.0, allocsAnc)

	// 7. Rel zero-alloc on identical paths (returns ".")
	pIdentical := pathkit.New("/var/log/nginx/access.log")
	allocsRelIdentical := testing.AllocsPerRun(1000, func() {
		_, _ = pIdentical.Rel(pIdentical)
	})
	assert.Equal(t, 0.0, allocsRelIdentical)

	// 8. Rel zero-alloc on incompatible paths
	pIncompat := pathkit.New("https://api.github.com")
	allocsRelIncompat := testing.AllocsPerRun(1000, func() {
		_, _ = pIdentical.Rel(pIncompat)
	})
	assert.Equal(t, 0.0, allocsRelIncompat)
}

func BenchmarkSegmentsSeq_ZeroAlloc(b *testing.B) {
	path := "/usr/local/bin/foundation/exec"

	b.ReportAllocs()

	for b.Loop() {
		for seg := range pathkit.Segments(path) {
			_ = seg
		}
	}
}

func BenchmarkSegmentsSeq_Windows_ZeroAlloc(b *testing.B) {
	path := `C:\Windows\System32\drivers\etc\hosts`

	b.ReportAllocs()

	for b.Loop() {
		for seg := range pathkit.Segments(path) {
			_ = seg
		}
	}
}

func BenchmarkAppendJoinURL_ZeroAlloc(b *testing.B) {
	scratch := make([]byte, 0, 128)
	base := "https://api.github.com/v1"

	b.ReportAllocs()

	for b.Loop() {
		buf := scratch[:0]
		_ = pathkit.AppendJoinURL(buf, base, "repos", "lemon4ksan", "aoni", "issues", "100")
	}
}

func BenchmarkAncestors_ZeroAlloc(b *testing.B) {
	p := pathkit.New("/var/log/nginx/app/access.log")

	b.ReportAllocs()

	for b.Loop() {
		for a := range p.Ancestors() {
			_ = a
		}
	}
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

func TestPath_AuthorityComparison(t *testing.T) {
	t.Parallel()

	// 1. Authority extraction
	assert.Equal(t, "google.com", pathkit.New("https://google.com/api").Authority())
	assert.Equal(t, "google.com", pathkit.New("https://google.com").Authority())
	assert.Equal(t, "google.com", pathkit.New("https://google.com/").Authority())
	assert.Equal(t, "example.com:8080", pathkit.New("http://example.com:8080/v1").Authority())
	assert.Equal(t, "user:pass@example.com:9000", pathkit.New("https://user:pass@example.com:9000/path").Authority())
	assert.Equal(t, "my-bucket", pathkit.New("s3://my-bucket/key").Authority())
	assert.Equal(t, "", pathkit.New("file:///C:/path").Authority())
	assert.Equal(t, "server", pathkit.New("file://server/share").Authority())
	assert.Equal(t, "", pathkit.New("/var/log").Authority())
	assert.Equal(t, "", pathkit.New(`C:\Users`).Authority())

	// 2. HasPrefix with authorities
	pGoogle := pathkit.New("https://google.com/api/v1")
	pGooglePrefix := pathkit.New("https://google.com/api")
	pApple := pathkit.New("https://apple.com/api")
	pGooglePort := pathkit.New("https://google.com:8080/api")

	assert.True(t, pGoogle.HasPrefix(pGooglePrefix))
	assert.False(t, pGoogle.HasPrefix(pApple))
	assert.False(t, pGoogle.HasPrefix(pGooglePort))

	// s3 buckets
	assert.True(t, pathkit.New("s3://bucket-a/data/file.csv").HasPrefix(pathkit.New("s3://bucket-a/data")))
	assert.False(t, pathkit.New("s3://bucket-a/data/file.csv").HasPrefix(pathkit.New("s3://bucket-b/data")))

	// 3. Rel with authorities
	rel, ok := pathkit.New("https://google.com/api/v1/users").Rel(pGooglePrefix)
	require.True(t, ok)
	assert.Equal(t, "v1/users", rel.String())

	// Different authorities must fail
	_, ok = pathkit.New("https://google.com/api").Rel(pApple)
	assert.False(t, ok)

	_, ok = pathkit.New("https://google.com:8080/api").Rel(pGooglePrefix)
	assert.False(t, ok)

	// Incompatible schemes
	_, ok = pathkit.New("http://google.com/api").Rel(pathkit.New("https://google.com/api"))
	assert.False(t, ok)
}

func TestPath_WindowsIsAbs_Validation(t *testing.T) {
	t.Parallel()

	// 1. Bare drive letter or drive-relative paths (MUST NOT be absolute)
	assert.False(t, pathkit.New("C:").IsAbs())
	assert.False(t, pathkit.New("c:").IsAbs())
	assert.False(t, pathkit.New("D:").IsAbs())
	assert.False(t, pathkit.New("C:foo").IsAbs())
	assert.False(t, pathkit.New(`C:foo\bar`).IsAbs())
	assert.False(t, pathkit.New("c:relative/path").IsAbs())
	assert.False(t, pathkit.New("1:/foo").IsAbs())
	assert.False(t, pathkit.New("?:/foo").IsAbs())

	// 2. Windows drive paths with root separator (MUST be absolute)
	assert.True(t, pathkit.New("C:/").IsAbs())
	assert.True(t, pathkit.New(`C:\`).IsAbs())
	assert.True(t, pathkit.New("C:/Windows").IsAbs())
	assert.True(t, pathkit.New(`C:\Windows\System32`).IsAbs())
	assert.True(t, pathkit.New("c:/users/app").IsAbs())
	assert.True(t, pathkit.New(`d:\data`).IsAbs())

	// 3. Windows UNC paths starting with \\ or // (MUST be absolute)
	assert.True(t, pathkit.New(`\\server\share`).IsAbs())
	assert.True(t, pathkit.New(`\\server\share\folder\file.txt`).IsAbs())
	assert.True(t, pathkit.New("//server/share").IsAbs())
	assert.True(t, pathkit.New("//server/share/folder/file.txt").IsAbs())

	// 4. Unix paths
	assert.True(t, pathkit.New("/").IsAbs())
	assert.True(t, pathkit.New("/var/log").IsAbs())
	assert.False(t, pathkit.New("var/log").IsAbs())
	assert.False(t, pathkit.New("./var/log").IsAbs())
	assert.False(t, pathkit.New("../var/log").IsAbs())
	assert.False(t, pathkit.New("").IsAbs())

	// 5. URLs and file URIs
	assert.True(t, pathkit.New("https://google.com").IsAbs())
	assert.True(t, pathkit.New("file:///C:/path").IsAbs())
	assert.True(t, pathkit.New("s3://bucket/key").IsAbs())
}

func TestPath_Rel_ParallelTraversal(t *testing.T) {
	t.Parallel()

	// 1. Common prefix cases
	base := pathkit.New("/a/b")
	rel, ok := pathkit.New("/a/b/c/d").Rel(base)
	require.True(t, ok)
	assert.Equal(t, "c/d", rel.String())

	// 2. Target is ancestor of base (needs ../)
	rel, ok = pathkit.New("/a/b").Rel(pathkit.New("/a/b/c/d"))
	require.True(t, ok)
	assert.Equal(t, "../..", rel.String())

	// 3. Divergence after common prefix
	rel, ok = pathkit.New("/a/b/x/y").Rel(pathkit.New("/a/b/c/d"))
	require.True(t, ok)
	assert.Equal(t, "../../x/y", rel.String())

	// 4. Divergence at root
	rel, ok = pathkit.New("/x/y").Rel(pathkit.New("/a/b"))
	require.True(t, ok)
	assert.Equal(t, "../../x/y", rel.String())

	// 5. Identical paths
	rel, ok = pathkit.New("/a/b/c").Rel(pathkit.New("/a/b/c"))
	require.True(t, ok)
	assert.Equal(t, ".", rel.String())

	// 6. Base is root
	rel, ok = pathkit.New("/a/b").Rel(pathkit.New("/"))
	require.True(t, ok)
	assert.Equal(t, "a/b", rel.String())

	// Target is root
	rel, ok = pathkit.New("/").Rel(pathkit.New("/a/b"))
	require.True(t, ok)
	assert.Equal(t, "../..", rel.String())

	// 7. Relative paths
	rel, ok = pathkit.New("a/b/c").Rel(pathkit.New("a/b"))
	require.True(t, ok)
	assert.Equal(t, "c", rel.String())

	rel, ok = pathkit.New("a/b").Rel(pathkit.New("a/b/c"))
	require.True(t, ok)
	assert.Equal(t, "..", rel.String())

	rel, ok = pathkit.New("a/x").Rel(pathkit.New("a/y"))
	require.True(t, ok)
	assert.Equal(t, "../x", rel.String())

	// 8. Incompatible: absolute vs relative
	_, ok = pathkit.New("/a/b").Rel(pathkit.New("a/b"))
	assert.False(t, ok)

	_, ok = pathkit.New("a/b").Rel(pathkit.New("/a/b"))
	assert.False(t, ok)

	// 9. Windows drive letters
	rel, ok = pathkit.New("C:/foo/bar/baz").Rel(pathkit.New("C:/foo"))
	require.True(t, ok)
	assert.Equal(t, "bar/baz", rel.String())

	rel, ok = pathkit.New("C:/foo/bar").Rel(pathkit.New("C:/foo/baz"))
	require.True(t, ok)
	assert.Equal(t, "../bar", rel.String())

	// Case-insensitive drive letter
	rel, ok = pathkit.New("C:/foo/bar").Rel(pathkit.New("c:/foo/baz"))
	require.True(t, ok)
	assert.Equal(t, "../bar", rel.String())

	// Different drive letters (MUST fail)
	_, ok = pathkit.New("C:/foo").Rel(pathkit.New("D:/foo"))
	assert.False(t, ok)

	// 10. Windows UNC paths
	rel, ok = pathkit.New("//server/share/sub/file.txt").Rel(pathkit.New("//server/share/sub"))
	require.True(t, ok)
	assert.Equal(t, "file.txt", rel.String())

	rel, ok = pathkit.New("//server/share/dirA").Rel(pathkit.New("//server/share/dirB"))
	require.True(t, ok)
	assert.Equal(t, "../dirA", rel.String())

	// Different UNC shares (MUST fail)
	_, ok = pathkit.New("//server1/share/dir").Rel(pathkit.New("//server2/share/dir"))
	assert.False(t, ok)

	// 11. URLs
	rel, ok = pathkit.New("https://example.com/v1/api/endpoint").Rel(pathkit.New("https://example.com/v1"))
	require.True(t, ok)
	assert.Equal(t, "api/endpoint", rel.String())

	rel, ok = pathkit.New("https://example.com/v1").Rel(pathkit.New("https://example.com/v1/api/endpoint"))
	require.True(t, ok)
	assert.Equal(t, "../..", rel.String())
}

func BenchmarkPath_Rel(b *testing.B) {
	target := pathkit.New("/var/log/nginx/access.log")
	base := pathkit.New("/var/log")

	b.ReportAllocs()
	for b.Loop() {
		_, _ = target.Rel(base)
	}
}

func BenchmarkPath_Rel_ZeroAlloc(b *testing.B) {
	p := pathkit.New("/var/log/nginx/access.log")

	b.ReportAllocs()
	for b.Loop() {
		_, _ = p.Rel(p)
	}
}
