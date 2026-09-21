// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
)

// TestParseTag_AdversarialInputs verifies that ParseTag handles arbitrary
// malicious, malformed, and pathological strings without panicking.
func TestParseTag_AdversarialInputs(t *testing.T) {
	t.Parallel()

	maliciousCases := []string{
		// Unclosed and lone quotes
		`"`,
		`'`,
		`"""`,
		`'''`,
		`"unclosed quote`,
		`'unclosed single quote`,
		`field,"unclosed quote with, commas inside, and no closing`,
		`field,'unclosed single with, commas inside`,
		`field,"double","single'`,
		`field,"nested 'single' inside unclosed double`,
		`field,'nested "double" inside unclosed single`,

		// Unmatched and malformed brackets
		`field,[unclosed bracket`,
		`field,{unclosed brace`,
		`field,(unclosed paren`,
		`field,[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[[nested`,
		`field,]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]]closed_without_open`,
		`field,[{(()}]mismatched`,
		`field,[(a, b), {c, d}]`,
		`field,([)]`,

		// Delimiter storms
		`,`,
		`,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,`,
		` , , , , , , , , `,
		`   `,
		"\t\r\n",
		`="",='',=[],={}`,
		`=`,
		`==`,
		`===`,
		`key=`,
		`=val`,
		`key==val`,
		`key=val=more`,

		// Escape attempts
		`field,"escaped\"quote"`,
		`field,'escaped\'single'`,
		`field,\\\\`,
	}

	for i, input := range maliciousCases {
		t.Run(fmt.Sprintf("Case_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("ParseTag panicked on input %q: %v", input, r)
				}
			}()

			tag := refkit.ParseTag(input)
			// Tag methods must never panic on parsed results
			_ = tag.IsEmpty()
			_ = tag.IsIgnored()
			_ = tag.Has("anything")
			_ = tag.Get("key")
			_, _ = tag.GetInt("key")
			_, _ = tag.GetFloat("key")
			_ = tag.SplitOption("key", ",")
		})
	}
}

// TestParseTag_BoundaryAllocations tests options count around stack buffer limits (8).
func TestParseTag_BoundaryAllocations(t *testing.T) {
	t.Parallel()

	counts := []int{0, 1, 6, 7, 8, 9, 15, 16, 17, 64, 128, 500}

	for _, count := range counts {
		var parts []string
		parts = append(parts, "mainField")
		for i := 1; i <= count; i++ {
			parts = append(parts, fmt.Sprintf("opt%d", i))
		}
		tagStr := strings.Join(parts, ",")

		tag := refkit.ParseTag(tagStr)
		if tag.Name != "mainField" {
			t.Fatalf("count %d: expected name 'mainField', got %q", count, tag.Name)
		}
		if len(tag.Options) != count {
			t.Fatalf("count %d: expected %d options, got %d", count, count, len(tag.Options))
		}
		for i := 1; i <= count; i++ {
			expectedOpt := fmt.Sprintf("opt%d", i)
			if !tag.Has(expectedOpt) {
				t.Errorf("count %d: missing option %q", count, expectedOpt)
			}
		}
	}
}

// TestParseTag_ExtremeLength tests pathological large strings.
func TestParseTag_ExtremeLength(t *testing.T) {
	t.Parallel()

	// 50,000 char ASCII tag
	longName := strings.Repeat("a", 10000)
	longOpt := strings.Repeat("b", 40000)
	tagStr := longName + "," + longOpt

	tag := refkit.ParseTag(tagStr)
	if tag.Name != longName {
		t.Fatalf("long tag name mismatch: len=%d vs len=%d", len(tag.Name), len(longName))
	}
	if len(tag.Options) != 1 || tag.Options[0] != longOpt {
		t.Fatalf("long tag option mismatch")
	}

	// 10,000 comma separated tokens
	manyParts := strings.Repeat("x,", 10000) + "end"
	tagMany := refkit.ParseTag(manyParts)
	if tagMany.Name != "x" {
		t.Fatalf("many parts: expected Name 'x', got %q", tagMany.Name)
	}
	if len(tagMany.Options) != 10000 {
		t.Fatalf("many parts: expected 10000 options, got %d", len(tagMany.Options))
	}
}

// TestParseTag_UnicodeAndInvalidBytes verifies Unicode handling and invalid UTF-8 byte resilience.
func TestParseTag_UnicodeAndInvalidBytes(t *testing.T) {
	t.Parallel()

	unicodeCases := []struct {
		input        string
		expectedName string
		expectedOpts []string
	}{
		{
			input:        "поле,обязательное,размер=100",
			expectedName: "поле",
			expectedOpts: []string{"обязательное", "размер=100"},
		},
		{
			input:        "用戶名,必填,長度=20",
			expectedName: "用戶名",
			expectedOpts: []string{"必填", "長度=20"},
		},
		{
			input:        "🚀rocket,🛸ufo,speed=⚡fast",
			expectedName: "🚀rocket",
			expectedOpts: []string{"🛸ufo", "speed=⚡fast"},
		},
		{
			input:        "mixed_tag, 'юникод, в кавычках', (параметры, скобки)",
			expectedName: "mixed_tag",
			expectedOpts: []string{"'юникод, в кавычках'", "(параметры, скобки)"},
		},
	}

	for _, tc := range unicodeCases {
		tag := refkit.ParseTag(tc.input)
		if tag.Name != tc.expectedName {
			t.Errorf("input %q: expected name %q, got %q", tc.input, tc.expectedName, tag.Name)
		}
		if len(tag.Options) != len(tc.expectedOpts) {
			t.Errorf("input %q: expected %d options, got %d", tc.input, len(tc.expectedOpts), len(tag.Options))
		} else {
			for i := range tag.Options {
				if tag.Options[i] != tc.expectedOpts[i] {
					t.Errorf("input %q: opt[%d] expected %q, got %q", tc.input, i, tc.expectedOpts[i], tag.Options[i])
				}
			}
		}
	}

	// Invalid UTF-8 byte sequences must not panic
	invalidBytes := []string{
		string([]byte{0xff, 0xfe, 0xfd}),
		string([]byte{'n', 'a', 'm', 'e', 0x80, ',', 0x81, 'o', 'p', 't'}),
		string([]byte{0xc0, 0xaf, ',', 0xe0, 0x80, 0xaf}),
	}

	for _, inv := range invalidBytes {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("ParseTag panicked on invalid UTF-8: %v", r)
			}
		}()
		tag := refkit.ParseTag(inv)
		_ = tag.IsEmpty()
		_ = tag.Get("key")
	}
}

// TestTag_MethodRobustness verifies GetInt, GetFloat, SplitOption with pathological values.
func TestTag_MethodRobustness(t *testing.T) {
	t.Parallel()

	tag := refkit.ParseTag(
		`field,min=5,max=99999999999999999999999999999999999999999999999999,rate=3.1415,bad_num=xyz,empty=,nan=NaN,inf=+Inf,delim=a|b||c| `,
	)

	// Valid Int
	if val, ok := tag.GetInt("min"); !ok || val != 5 {
		t.Errorf("expected min=5, got (%d, %v)", val, ok)
	}

	// Overflow Int
	if _, ok := tag.GetInt("max"); ok {
		t.Errorf("expected overflow int to fail ok=false")
	}

	// Invalid Int string
	if _, ok := tag.GetInt("bad_num"); ok {
		t.Errorf("expected non-numeric int to fail ok=false")
	}

	// Empty value
	if _, ok := tag.GetInt("empty"); ok {
		t.Errorf("expected empty value int to fail ok=false")
	}

	// Missing key
	if _, ok := tag.GetInt("nonexistent"); ok {
		t.Errorf("expected missing key int to fail ok=false")
	}

	// Valid Float
	if val, ok := tag.GetFloat("rate"); !ok || math.Abs(val-3.1415) > 1e-9 {
		t.Errorf("expected rate=3.1415, got (%f, %v)", val, ok)
	}

	// Float NaN & Inf
	if val, ok := tag.GetFloat("nan"); !ok || !math.IsNaN(val) {
		t.Errorf("expected NaN float")
	}
	if val, ok := tag.GetFloat("inf"); !ok || !math.IsInf(val, 1) {
		t.Errorf("expected +Inf float")
	}

	// SplitOption
	parts := tag.SplitOption("delim", "|")
	expected := []string{"a", "b", "c"}
	if len(parts) != len(expected) {
		t.Fatalf("SplitOption expected %v, got %v", expected, parts)
	}
	for i := range expected {
		if parts[i] != expected[i] {
			t.Errorf("parts[%d]: expected %q, got %q", i, expected[i], parts[i])
		}
	}

	// SplitOption on missing key
	if parts := tag.SplitOption("nonexistent", ","); parts != nil {
		t.Errorf("expected nil for missing SplitOption, got %v", parts)
	}
}

// TestParseTag_ZeroAllocGuarantees verifies 0 heap allocations for simple tags.
func TestParseTag_ZeroAllocGuarantees(t *testing.T) {
	simpleTag := "user_id"
	allocs := testing.AllocsPerRun(1000, func() {
		tag := refkit.ParseTag(simpleTag)
		if tag.Name != "user_id" {
			t.Fatalf("unexpected: %v", tag)
		}
	})
	if allocs != 0 {
		t.Errorf("expected 0 allocs for simple tag, got %f", allocs)
	}

	ignoredTag := "-"
	allocsIgnored := testing.AllocsPerRun(1000, func() {
		tag := refkit.ParseTag(ignoredTag)
		if tag.Name != "-" {
			t.Fatalf("unexpected: %v", tag)
		}
	})
	if allocsIgnored != 0 {
		t.Errorf("expected 0 allocs for ignored tag, got %f", allocsIgnored)
	}
}
