// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"reflect"
	"testing"

	"github.com/lemon4ksan/foundation/refkit"
)

func TestParseTag_ASCII_FastPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input       string
		expectedTag refkit.Tag
	}{
		{
			input:       "",
			expectedTag: refkit.Tag{},
		},
		{
			input:       "-",
			expectedTag: refkit.Tag{Name: "-"},
		},
		{
			input:       "user_id",
			expectedTag: refkit.Tag{Name: "user_id"},
		},
		{
			input:       "  name  ",
			expectedTag: refkit.Tag{Name: "name"},
		},
		{
			input:       "name,omitempty",
			expectedTag: refkit.Tag{Name: "name", Options: []string{"omitempty"}},
		},
		{
			input:       "name,omitempty,inline,default=10",
			expectedTag: refkit.Tag{Name: "name", Options: []string{"omitempty", "inline", "default=10"}},
		},
		{
			input:       "name, (1, 2), {x, y}, [a, b]",
			expectedTag: refkit.Tag{Name: "name", Options: []string{"(1, 2)", "{x, y}", "[a, b]"}},
		},
		{
			input:       "name, 'quoted,comma', \"double,quoted\"",
			expectedTag: refkit.Tag{Name: "name", Options: []string{"'quoted,comma'", "\"double,quoted\""}},
		},
		{
			input:       "name,   ",
			expectedTag: refkit.Tag{Name: "name"},
		},
		{
			input:       ",omitempty",
			expectedTag: refkit.Tag{Name: "", Options: []string{"omitempty"}},
		},
	}

	for _, tc := range tests {
		tag := refkit.ParseTag(tc.input)
		if tag.Name != tc.expectedTag.Name {
			t.Errorf("input %q: expected Name %q, got %q", tc.input, tc.expectedTag.Name, tag.Name)
		}
		if len(tag.Options) != len(tc.expectedTag.Options) {
			t.Errorf("input %q: expected Options %v, got %v", tc.input, tc.expectedTag.Options, tag.Options)
		} else {
			for i := range tag.Options {
				if tag.Options[i] != tc.expectedTag.Options[i] {
					t.Errorf(
						"input %q: opt[%d] expected %q, got %q",
						tc.input,
						i,
						tc.expectedTag.Options[i],
						tag.Options[i],
					)
				}
			}
		}
	}
}

func TestParseTag_Unicode(t *testing.T) {
	t.Parallel()

	input := "имя,omitempty,формат=строка"
	tag := refkit.ParseTag(input)

	if tag.Name != "имя" {
		t.Errorf("expected Name 'имя', got %q", tag.Name)
	}
	expectedOpts := []string{"omitempty", "формат=строка"}
	if !reflect.DeepEqual(tag.Options, expectedOpts) {
		t.Errorf("expected Options %v, got %v", expectedOpts, tag.Options)
	}
}

func BenchmarkParseTag_ASCII_Simple(b *testing.B) {
	tagStr := "user_id"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		tag := refkit.ParseTag(tagStr)
		if tag.Name != "user_id" {
			b.Fatalf("unexpected tag name: %s", tag.Name)
		}
	}
}

func BenchmarkParseTag_ASCII_Ignored(b *testing.B) {
	tagStr := "-"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		tag := refkit.ParseTag(tagStr)
		if tag.Name != "-" {
			b.Fatalf("unexpected tag name: %s", tag.Name)
		}
	}
}

func BenchmarkParseTag_ASCII_WithOptions(b *testing.B) {
	tagStr := "name,omitempty,inline"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		tag := refkit.ParseTag(tagStr)
		if tag.Name != "name" {
			b.Fatalf("unexpected tag name: %s", tag.Name)
		}
	}
}

func BenchmarkParseTag_Unicode(b *testing.B) {
	tagStr := "имя,omitempty,inline"

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		tag := refkit.ParseTag(tagStr)
		if tag.Name != "имя" {
			b.Fatalf("unexpected tag name: %s", tag.Name)
		}
	}
}
