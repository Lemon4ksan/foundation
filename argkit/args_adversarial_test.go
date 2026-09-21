// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package argkit_test

import (
	"flag"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/argkit"
)

// referenceLevenshtein implements the standard 2D Wagner-Fischer DP algorithm
// as an independent verification oracle.
func referenceLevenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)
	if n1 == 0 {
		return n2
	}
	if n2 == 0 {
		return n1
	}

	dp := make([][]int, n1+1)
	for i := range dp {
		dp[i] = make([]int, n2+1)
		dp[i][0] = i
	}
	for j := 0; j <= n2; j++ {
		dp[0][j] = j
	}

	for i := 1; i <= n1; i++ {
		for j := 1; j <= n2; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			del := dp[i-1][j] + 1
			ins := dp[i][j-1] + 1
			sub := dp[i-1][j-1] + cost
			dp[i][j] = min(del, min(ins, sub))
		}
	}
	return dp[n1][n2]
}

// TestLevenshtein_EdgeCases verifies boundary conditions and edge cases.
func TestLevenshtein_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"", "abcdef", 6},
		{"abcdef", "", 6},
		{"a", "a", 0},
		{"a", "b", 1},
		{"ab", "ba", 2},
		{"abc", "def", 3},
		{"prefix", "prefix_suffix", 7},
		{"prefix_suffix", "prefix", 7},
		{"middle_sub", "middle_X_sub", 2},
	}

	for _, tc := range tests {
		got := argkit.Levenshtein(tc.s1, tc.s2)
		if got != tc.expected {
			t.Errorf("Levenshtein(%q, %q): expected %d, got %d", tc.s1, tc.s2, tc.expected, got)
		}
		// Symmetry check
		rev := argkit.Levenshtein(tc.s2, tc.s1)
		if rev != got {
			t.Errorf("Symmetry violated for (%q, %q): %d vs %d", tc.s1, tc.s2, got, rev)
		}
	}
}

// TestLevenshtein_BoundaryLengths verifies behavior at the 64-character stack limit.
func TestLevenshtein_BoundaryLengths(t *testing.T) {
	t.Parallel()

	lengths := []int{63, 64, 65, 66, 128, 256}

	for _, n := range lengths {
		base := strings.Repeat("x", n)
		mutated := base[:n-1] + "y"

		got := argkit.Levenshtein(base, mutated)
		if got != 1 {
			t.Errorf("len %d: expected distance 1, got %d", n, got)
		}

		// Asymmetric lengths: one <= 64, one > 64
		shortStr := "x"
		longStr := strings.Repeat("x", n)
		gotAsym := argkit.Levenshtein(shortStr, longStr)
		if gotAsym != n-1 {
			t.Errorf("asymmetric (1, %d): expected %d, got %d", n, n-1, gotAsym)
		}
		gotAsymRev := argkit.Levenshtein(longStr, shortStr)
		if gotAsymRev != n-1 {
			t.Errorf("asymmetric (%d, 1): expected %d, got %d", n, n-1, gotAsymRev)
		}
	}
}

// TestLevenshtein_UnicodeAndMultibyte verifies Unicode rune handling across scripts.
func TestLevenshtein_UnicodeAndMultibyte(t *testing.T) {
	t.Parallel()

	cases := []struct {
		s1       string
		s2       string
		expected int
	}{
		// Cyrillic
		{"тестирование", "тестирований", 1},
		{"проверка", "проверка", 0},
		// Chinese
		{"测试代码", "测试工具", 2},
		{"你好世界", "世界你好", 4},
		// Emojis (4-byte runes)
		{"🔥🚀✨", "🔥🛸✨", 1},
		{"🎉", "🎊", 1},
		// Mixed ASCII and Unicode (homoglyph attack: Latin 'a' vs Cyrillic 'а')
		{"flag", "flаg", 1}, // second string has Cyrillic 'а' (0xD0 0xB0)
		{"hello", "héllo", 1},
		// Unicode exceeding 64 runes
		{
			strings.Repeat("ж", 65),
			strings.Repeat("ж", 64) + "з",
			1,
		},
		{
			strings.Repeat("中", 70),
			strings.Repeat("中", 70),
			0,
		},
	}

	for _, tc := range cases {
		got := argkit.Levenshtein(tc.s1, tc.s2)
		if got != tc.expected {
			t.Errorf("Levenshtein(%q, %q): expected %d, got %d", tc.s1, tc.s2, tc.expected, got)
		}
		oracle := referenceLevenshtein(tc.s1, tc.s2)
		if got != oracle {
			t.Errorf("Levenshtein oracle mismatch for (%q, %q): got %d, oracle %d", tc.s1, tc.s2, got, oracle)
		}
	}
}

// TestLevenshtein_SurrogateAndPathologicalBytes verifies resilience against invalid UTF-8.
func TestLevenshtein_SurrogateAndPathologicalBytes(t *testing.T) {
	t.Parallel()

	pathological := []string{
		"\xed\xa0\x80",             // UTF-8 encoded high surrogate
		"\xed\xbf\xbf",             // UTF-8 encoded low surrogate
		"\xff\xfe\xfd",             // Invalid UTF-8 bytes
		string([]byte{0x80, 0x81}), // Lone continuation bytes
		"",
	}

	for i, s1 := range pathological {
		for j, s2 := range pathological {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Levenshtein panicked on pathological inputs [%d, %d]: %v", i, j, r)
				}
			}()

			got := argkit.Levenshtein(s1, s2)
			oracle := referenceLevenshtein(s1, s2)
			if got != oracle {
				t.Errorf("pathological [%d, %d]: got %d, oracle %d", i, j, got, oracle)
			}
		}
	}
}

// TestLevenshtein_GenerativeDifferentialOracle runs 500 randomized string pairs against
// the independent Wagner-Fischer oracle and verifies metric axioms.
func TestLevenshtein_GenerativeDifferentialOracle(t *testing.T) {
	t.Parallel()

	alphabet := []rune("abcdefghijklmnopqrstuvwxyz0123456789-_абвгдежзийклмноп🚀✨")
	rng := rand.New(rand.NewPCG(42, 100))

	randomString := func(maxLen int) string {
		n := rng.IntN(maxLen + 1)
		runes := make([]rune, n)
		for i := range runes {
			runes[i] = alphabet[rng.IntN(len(alphabet))]
		}
		return string(runes)
	}

	for i := range 500 {
		s1 := randomString(80)
		s2 := randomString(80)
		s3 := randomString(40)

		got := argkit.Levenshtein(s1, s2)
		oracle := referenceLevenshtein(s1, s2)

		if got != oracle {
			t.Fatalf("iteration %d: mismatch for (%q, %q): got %d, oracle %d", i, s1, s2, got, oracle)
		}

		// Metric Axiom: Symmetry
		if rev := argkit.Levenshtein(s2, s1); rev != got {
			t.Fatalf("iteration %d: symmetry failed for (%q, %q): %d vs %d", i, s1, s2, got, rev)
		}

		// Metric Axiom: Triangle inequality d(s1, s3) <= d(s1, s2) + d(s2, s3)
		d13 := argkit.Levenshtein(s1, s3)
		d23 := argkit.Levenshtein(s2, s3)
		if d13 > got+d23 {
			t.Fatalf("iteration %d: triangle inequality failed: d13=%d > d12=%d + d23=%d", i, d13, got, d23)
		}
	}
}

// TestSuggest_Adversarial verifies flag suggestion under varied conditions.
func TestSuggest_Adversarial(t *testing.T) {
	t.Parallel()

	fs := flag.NewFlagSet("test_suggest", flag.ContinueOnError)
	var v bool
	var o string
	var timeout string
	var u string
	argkit.BoolVar(fs, &v, "verbose", "v", false, "verbose")
	argkit.StringVar(fs, &o, "output", "o", "", "output")
	argkit.StringVar(fs, &timeout, "timeout", "t", "", "timeout")
	argkit.StringVar(fs, &u, "конфиг", "k", "", "unicode flag")

	cases := []struct {
		input    string
		expected string
	}{
		// Distance 1
		{"--verbos", "verbose"},
		{"-verbos", "verbose"},
		{"verbos", "verbose"},
		{"--outpu", "output"},
		{"--timeot", "timeout"},
		// Attached values
		{"--verbos=true", "verbose"},
		{"-outpu=file.txt", "output"},
		// Distance 2
		{"--verbo", "verbose"},
		// Unicode flag
		{"--конфи", "конфиг"},
		{"-конфиг", "конфиг"},
		// Distance >= 3 should return empty
		{"--verydifferent", ""},
		{"--verb", ""}, // distance between "verb" and "verbose" is 3 ('o', 's', 'e')
		// Edge cases: empty, delimiters
		{"", ""},
		{"-", ""},
		{"--", ""},
		{"---", ""},
		{"=", ""},
		{"===", ""},
		{strings.Repeat("a", 1000), ""},
	}

	for _, tc := range cases {
		got := argkit.Suggest(fs, tc.input)
		if got != tc.expected {
			t.Errorf("Suggest(%q): expected %q, got %q", tc.input, tc.expected, got)
		}
	}

	// Empty flag set
	emptyFS := flag.NewFlagSet("empty", flag.ContinueOnError)
	if match := argkit.Suggest(emptyFS, "--verbose"); match != "" {
		t.Errorf("expected empty string on empty FlagSet, got %q", match)
	}
}

// TestLevenshtein_ZeroAllocGuarantees verifies 0 heap allocations for strings up to 64 runes.
func TestLevenshtein_ZeroAllocGuarantees(t *testing.T) {
	// ASCII <= 64
	s1 := "this_is_a_flag_name"
	s2 := "this_is_a_flag_nme"
	allocsASCII := testing.AllocsPerRun(1000, func() {
		dist := argkit.Levenshtein(s1, s2)
		if dist != 1 {
			t.Fatalf("unexpected dist: %d", dist)
		}
	})
	if allocsASCII != 0 {
		t.Errorf("expected 0 allocs for ASCII Levenshtein, got %f", allocsASCII)
	}

	// Unicode <= 64 runes
	u1 := "конфигурационный_файл"
	u2 := "конфигурационый_файл"
	allocsUnicode := testing.AllocsPerRun(1000, func() {
		dist := argkit.Levenshtein(u1, u2)
		if dist != 1 {
			t.Fatalf("unexpected dist: %d", dist)
		}
	})
	if allocsUnicode != 0 {
		t.Errorf("expected 0 allocs for Unicode Levenshtein, got %f", allocsUnicode)
	}
}
