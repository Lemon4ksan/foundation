// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pathkit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/pathkit"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

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
}

func TestPath_AppendJoinURL(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 0, 128)
	buf = pathkit.AppendJoinURL(buf, "https://api.example.com", "v1", "users", "100")
	assert.Equal(t, "https://api.example.com/v1/users/100", string(buf))

	// Empty base
	buf = buf[:0]
	buf = pathkit.AppendJoinURL(buf, "", "base", "part")
	assert.Equal(t, "base/part", string(buf))

	// Slashes edge cases
	buf = buf[:0]
	buf = pathkit.AppendJoinURL(buf, "https://api.example.com/", "/v1/", "/users/")
	assert.Equal(t, "https://api.example.com/v1/users/", string(buf))
}

func TestPath_AppendClean(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 0, 128)
	buf = pathkit.AppendClean(buf, "/a/b/../c/./d")
	assert.Equal(t, "/a/c/d", string(buf))

	// URL clean
	buf = buf[:0]
	buf = pathkit.AppendClean(buf, "https://domain.com/v1/../v2/app")
	assert.Equal(t, "https://domain.com/v2/app", string(buf))

	// Empty path
	buf = buf[:0]
	buf = pathkit.AppendClean(buf, "")
	assert.Equal(t, ".", string(buf))
}

func TestPath_AppendSanitize(t *testing.T) {
	t.Parallel()

	buf := make([]byte, 0, 128)
	buf = pathkit.AppendSanitize(buf, "../../../etc/passwd")
	assert.Equal(t, "etc/passwd", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendSanitize(buf, `..\..\windows\system32`)
	assert.Equal(t, "windows/system32", string(buf))

	buf = buf[:0]
	buf = pathkit.AppendSanitize(buf, "")
	assert.Equal(t, "", string(buf))
}

func TestPath_Segments2(t *testing.T) {
	t.Parallel()

	p := pathkit.New("/usr/local/bin")
	var idxs []int
	var segs []string

	for i, seg := range p.Segments2() {
		idxs = append(idxs, i)
		segs = append(segs, seg)
		if i == 1 {
			// Early termination test
		}
	}

	assert.Equal(t, []int{0, 1, 2}, idxs)
	assert.Equal(t, []string{"usr", "local", "bin"}, segs)
}

func TestPath_Ancestors(t *testing.T) {
	t.Parallel()

	// 1. File path
	p := pathkit.New("/a/b/c")
	var anc []string
	for a := range p.Ancestors() {
		anc = append(anc, a.String())
	}
	assert.Equal(t, []string{"/a/b", "/a", "/"}, anc)

	// 2. URL
	urlP := pathkit.New("https://example.com/v1/api/users")
	var urlAnc []string
	for a := range urlP.Ancestors() {
		urlAnc = append(urlAnc, a.String())
	}
	assert.Equal(t, []string{"https://example.com/v1/api", "https://example.com/v1", "https://example.com"}, urlAnc)

	// 3. Root path has no ancestors
	rootP := pathkit.New("/")
	var rootAnc []string
	for a := range rootP.Ancestors() {
		rootAnc = append(rootAnc, a.String())
	}
	assert.Empty(t, rootAnc)
}

func TestPath_HasPrefix(t *testing.T) {
	t.Parallel()

	p1 := pathkit.New("/var/log/nginx/access.log")
	assert.True(t, p1.HasPrefix(pathkit.New("/var/log")))
	assert.True(t, p1.HasPrefix(pathkit.New("/var/log/nginx")))
	assert.True(t, p1.HasPrefix(pathkit.New("/var/log/nginx/access.log")))
	assert.False(t, p1.HasPrefix(pathkit.New("/var/lib")))
	assert.False(t, p1.HasPrefix(pathkit.New("/var/log/ngin"))) // segment boundary check

	// Incompatible schemes
	assert.False(t, p1.HasPrefix(pathkit.New("https://var/log")))
}

func TestPath_Rel(t *testing.T) {
	t.Parallel()

	base := pathkit.New("/a/b")

	// 1. Child
	rel, ok := pathkit.New("/a/b/c/d").Rel(base)
	require.True(t, ok)
	assert.Equal(t, "c/d", rel.String())

	// 2. Same
	rel, ok = pathkit.New("/a/b").Rel(base)
	require.True(t, ok)
	assert.Equal(t, ".", rel.String())

	// 3. Sibling
	rel, ok = pathkit.New("/a/c").Rel(base)
	require.True(t, ok)
	assert.Equal(t, "../c", rel.String())

	// 4. Incompatible schemes
	_, ok = pathkit.New("https://api.com/a").Rel(base)
	assert.False(t, ok)
}

func TestPath_Match(t *testing.T) {
	t.Parallel()

	p := pathkit.New("/var/log/app.prod.json")
	assert.True(t, p.Match("*.json"))
	assert.True(t, p.Match("app.*.json"))
	assert.False(t, p.Match("*.log"))
}

func TestPath_ZeroAllocGuarantees(t *testing.T) {
	// 1. SegmentsSeq on Unix path
	unixPath := "/usr/local/bin/exec"
	allocs := testing.AllocsPerRun(1000, func() {
		for seg := range pathkit.SegmentsSeq(unixPath) {
			_ = seg
		}
	})
	assert.Equal(t, 0.0, allocs)

	// 2. SegmentsSeq on Windows path (contains backslashes)
	winPath := `C:\Windows\System32\drivers\etc\hosts`
	allocsWin := testing.AllocsPerRun(1000, func() {
		for seg := range pathkit.SegmentsSeq(winPath) {
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
}

func BenchmarkSegmentsSeq_Windows_ZeroAlloc(b *testing.B) {
	path := `C:\Windows\System32\drivers\etc\hosts`

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for seg := range pathkit.SegmentsSeq(path) {
			_ = seg
		}
	}
}

func BenchmarkAppendJoinURL_ZeroAlloc(b *testing.B) {
	scratch := make([]byte, 0, 128)
	base := "https://api.github.com/v1"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buf := scratch[:0]
		_ = pathkit.AppendJoinURL(buf, base, "repos", "lemon4ksan", "aoni", "issues", "100")
	}
}

func BenchmarkAncestors_ZeroAlloc(b *testing.B) {
	p := pathkit.New("/var/log/nginx/app/access.log")

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		for a := range p.Ancestors() {
			_ = a
		}
	}
}
