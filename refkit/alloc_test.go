package refkit

// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"reflect"
	"testing"
)

func TestPutWalker_ClearsPointers(t *testing.T) {
	w := getWalker()

	// Populate walker with mock entries, visiting types, and struct fields holding pointer references.
	mockType := reflect.TypeFor[string]()
	w.visiting = append(w.visiting, mockType)
	w.entries = append(w.entries, nameEntry{name: "FieldA", index: 0})
	w.index = append(w.index, 1, 2)
	w.fields = append(w.fields, reflect.StructField{
		Name:  "FieldA",
		Type:  mockType,
		Index: []int{1, 2},
	})

	// Pre-condition check
	if len(w.fields) != 1 || w.fields[0].Type == nil {
		t.Fatal("expected populated field before putWalker")
	}

	// Capture backing array capacity
	fieldsCap := cap(w.fields)
	visitingCap := cap(w.visiting)

	putWalker(w)

	// Verify lengths are reset
	if len(w.fields) != 0 {
		t.Errorf("expected w.fields length 0, got %d", len(w.fields))
	}
	if len(w.visiting) != 0 {
		t.Errorf("expected w.visiting length 0, got %d", len(w.visiting))
	}
	if len(w.entries) != 0 {
		t.Errorf("expected w.entries length 0, got %d", len(w.entries))
	}
	if len(w.index) != 0 {
		t.Errorf("expected w.index length 0, got %d", len(w.index))
	}

	// Verify pointer clearing in backing capacity (no memory leaks)
	fieldsSlice := w.fields[:fieldsCap]
	for i, f := range fieldsSlice {
		if f.Type != nil {
			t.Errorf("backing array field %d retained Type pointer %v", i, f.Type)
		}
		if f.Index != nil {
			t.Errorf("backing array field %d retained Index slice %v", i, f.Index)
		}
		if f.Name != "" {
			t.Errorf("backing array field %d retained Name %q", i, f.Name)
		}
	}

	visitingSlice := w.visiting[:visitingCap]
	for i, v := range visitingSlice {
		if v != nil {
			t.Errorf("backing array visiting %d retained Type pointer %v", i, v)
		}
	}
}

func TestParseFieldTag_EscapedQuotes(t *testing.T) {
	tests := []struct {
		name       string
		tag        reflect.StructTag
		wantName   string
		wantOption string
	}{
		{
			name:       "escaped quote in value",
			tag:        `custom:"val\"quote" other:"val"`,
			wantName:   `val\"quote`,
			wantOption: `custom=val\"quote`,
		},
		{
			name:       "tag ending with escaped quote",
			tag:        `custom:"val\"" other:"next"`,
			wantName:   `val\"`,
			wantOption: `custom=val\"`,
		},
		{
			name:       "escaped backslash before escaped quote",
			tag:        `custom:"val\\\"" next:"ok"`,
			wantName:   `val\\\"`,
			wantOption: `custom=val\\\"`,
		},
		{
			name:       "escaped backslash before closing quote",
			tag:        `custom:"val\\" next:"ok"`,
			wantName:   `val\\`,
			wantOption: `custom=val\\`,
		},
		{
			name:       "empty tag value",
			tag:        `empty:"" other:"1"`,
			wantOption: `empty=`,
		},
		{
			name:       "multiple tags with complex escapes",
			tag:        `tag1:"v1\"" tag2:"v2\\" tag3:"v3"`,
			wantName:   `v1\"`,
			wantOption: `tag1=v1\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parseFieldTag(tt.tag)
			if tt.wantName != "" && parsed.Name != tt.wantName {
				t.Errorf("got Name %q, want %q", parsed.Name, tt.wantName)
			}
			if tt.wantOption != "" && !parsed.Has(tt.wantOption) {
				t.Errorf("options %v missing expected %q", parsed.Options, tt.wantOption)
			}
		})
	}
}

func TestParseFieldTag_MalformedBoundaryNeverPanics(t *testing.T) {
	malformed := []reflect.StructTag{
		`custom:"val\"`,
		`custom:"val\`,
		`custom:"val\\`,
		`custom:"`,
		`custom:\`,
		`custom:`,
		`"`,
		`\`,
		`custom:"val\" other:"next"`,
		`custom:"val\ `,
		`custom:"val\\\"`,
		`a:"\`,
		`a:"\"`,
	}
	for _, tag := range malformed {
		// Must never panic on out of bounds or boundary conditions
		parsed := parseFieldTag(tag)
		_ = parsed.Name
		_ = parsed.Options
	}
}
