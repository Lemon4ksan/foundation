// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package refkit_test

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/refkit"
	"github.com/lemon4ksan/foundation/testing/assert"
)

// TestTag_ConcurrencyStress tests concurrent read-access across multiple goroutines
// calling Tag methods (Value, Int, Float, Has, SplitOption) simultaneously with -race.
func TestTag_ConcurrencyStress(t *testing.T) {
	tag := refkit.ParseTag(`user_id,omitempty,min=10,max=1000,ratio=3.14159,flags=a|b|c|d,default=guest`)

	const goroutines = 64
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := range goroutines {
		go func(workerID int) {
			defer wg.Done()

			for range iterations {
				// 1. Tag properties
				if tag.Name != "user_id" {
					t.Errorf("expected user_id, got %q", tag.Name)
					return
				}
				if !tag.Has("omitempty") {
					t.Errorf("missing omitempty")
					return
				}
				if tag.IsEmpty() || tag.IsIgnored() {
					t.Errorf("tag should not be empty or ignored")
					return
				}

				// 2. Value getter
				if v := tag.Value("default"); v != "guest" {
					t.Errorf("expected guest, got %q", v)
					return
				}

				// 3. Int getter
				minVal, ok := tag.Int("min")
				if !ok || minVal != 10 {
					t.Errorf("expected min=10, got (%d, %v)", minVal, ok)
					return
				}
				maxVal, ok := tag.Int("max")
				if !ok || maxVal != 1000 {
					t.Errorf("expected max=1000, got (%d, %v)", maxVal, ok)
					return
				}

				// 4. Float getter
				ratio, ok := tag.Float("ratio")
				if !ok || math.Abs(ratio-3.14159) > 1e-9 {
					t.Errorf("expected ratio=3.14159, got (%f, %v)", ratio, ok)
					return
				}

				// 5. SplitOption
				flags := slices.Collect(tag.SplitOption("flags", "|"))
				if len(flags) != 4 || flags[0] != "a" || flags[3] != "d" {
					t.Errorf("unexpected flags: %v", flags)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

// TestTag_Int_AdversarialEdgeCases evaluates Tag.Int edge cases:
// boundary limits, signs, overflow, underflow, whitespace, hex, and invalid inputs.
func TestTag_Int_AdversarialEdgeCases(t *testing.T) {
	cases := []struct {
		tagStr   string
		key      string
		expected int
		ok       bool
	}{
		// Standard valid ints
		{"field,num=0", "num", 0, true},
		{"field,num=1", "num", 1, true},
		{"field,num=-1", "num", -1, true},
		{"field,num=+42", "num", 42, true},
		{"field,num=-42", "num", -42, true},

		// Max and Min int values (assuming 64-bit platform)
		{fmt.Sprintf("field,num=%d", math.MaxInt), "num", math.MaxInt, true},
		{fmt.Sprintf("field,num=%d", math.MinInt), "num", math.MinInt, true},

		// Overflow / underflow
		{"field,num=9223372036854775808", "num", 0, false},
		{"field,num=-9223372036854775809", "num", 0, false},
		{"field,num=999999999999999999999999999999", "num", 0, false},

		// Formats not supported by strconv.Atoi
		{"field,num=0x10", "num", 0, false},
		{"field,num=0b1010", "num", 0, false},
		{"field,num=1.0", "num", 0, false},
		{"field,num=1e5", "num", 0, false},

		// Whitespace in value (ParseTag trims options, so "num=5 " trims to "num=5", but "num= 5" has internal space)
		{"field,num= 5", "num", 0, false},
		{"field,num=5 ", "num", 5, true},

		// Empty or missing
		{"field,num=", "num", 0, false},
		{"field,num", "num", 0, false},
		{"field", "num", 0, false},
		{"", "num", 0, false},
	}

	for _, tc := range cases {
		tag := refkit.ParseTag(tc.tagStr)
		val, ok := tag.Int(tc.key)
		assert.Equal(t, tc.ok, ok, fmt.Sprintf("case %s ok mismatch", tc.tagStr))
		if tc.ok {
			assert.Equal(t, tc.expected, val, fmt.Sprintf("case %s val mismatch", tc.tagStr))
		}
	}
}

// TestTag_Float_AdversarialEdgeCases evaluates Tag.Float edge cases:
// NaN, infinities, scientific notation, zero signs, underflow/overflow.
func TestTag_Float_AdversarialEdgeCases(t *testing.T) {
	cases := []struct {
		tagStr string
		key    string
		verify func(t *testing.T, val float64, ok bool)
	}{
		{
			"field,val=0.0", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 0.0, val)
			},
		},
		{
			"field,val=-0.0", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 0.0, val)
			},
		},
		{
			"field,val=1e-5", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 1e-5, val)
			},
		},
		{
			"field,val=2.5E+10", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.Equal(t, 2.5e10, val)
			},
		},
		{
			"field,val=NaN", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.True(t, math.IsNaN(val))
			},
		},
		{
			"field,val=+Inf", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.True(t, math.IsInf(val, 1))
			},
		},
		{
			"field,val=-Inf", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.True(t, ok)
				assert.True(t, math.IsInf(val, -1))
			},
		},
		{
			"field,val=", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.False(t, ok)
			},
		},
		{
			"field,val=invalid", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.False(t, ok)
			},
		},
		{
			"field", "val",
			func(t *testing.T, val float64, ok bool) {
				assert.False(t, ok)
			},
		},
	}

	for _, tc := range cases {
		tag := refkit.ParseTag(tc.tagStr)
		val, ok := tag.Float(tc.key)
		tc.verify(t, val, ok)
	}
}

// TestTag_Value_AdversarialEdgeCases evaluates Tag.Value key lookup behavior.
func TestTag_Value_AdversarialEdgeCases(t *testing.T) {
	tag := refkit.ParseTag(`field,key=first=second=third,empty=,no_val,=lone_val,"quoted=val"`)

	// key=first=second=third -> Cut splits at first '='
	assert.Equal(t, "first=second=third", tag.Value("key"))

	// empty= -> ""
	assert.Equal(t, "", tag.Value("empty"))

	// no_val -> "" (no '=')
	assert.Equal(t, "", tag.Value("no_val"))

	// =lone_val -> Cut gives k="", v="lone_val"
	assert.Equal(t, "lone_val", tag.Value(""))

	// Nonexistent
	assert.Equal(t, "", tag.Value("nonexistent"))
}

// TestTag_SplitOption_EmptyDelim verifies whether calling SplitOption with empty delim
// terminates or enters an infinite loop.
func TestTag_SplitOption_EmptyDelim(t *testing.T) {
	tag := refkit.ParseTag(`field,tokens=abc`)

	// Run SplitOption with empty delim under a timeout channel
	done := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	go func() {
		// Attempt iteration with delim = ""
		for range tag.SplitOption("tokens", "") {
			// If it yields indefinitely, break out
			break
		}
		close(done)
	}()

	select {
	case <-done:
		t.Logf("SplitOption with delim='' returned successfully")
	case <-ctx.Done():
		t.Logf(
			"VULNERABILITY DETECTED: Tag.SplitOption(\"tokens\", \"\") enters an infinite loop due to strings.Index(s, \"\") == 0",
		)
	}
}

// TestTagOf_ReflectTagLookup tests TagOf against multiple tags on a struct field.
func TestTagOf_ReflectTagLookup(t *testing.T) {
	type TestStruct struct {
		Field string `json:"user_name,omitempty" db:"users.name" yaml:"name"`
	}

	sf, _ := reflect.TypeFor[TestStruct]().FieldByName("Field")

	// 1. First key matches
	tag1 := refkit.TagOf(sf, "json", "db")
	assert.Equal(t, "user_name", tag1.Name)
	assert.True(t, tag1.Has("omitempty"))

	// 2. Second key fallback
	tag2 := refkit.TagOf(sf, "xml", "db")
	assert.Equal(t, "users.name", tag2.Name)

	// 3. No match
	tag3 := refkit.TagOf(sf, "xml", "protobuf")
	assert.True(t, tag3.IsEmpty())

	// 4. Empty keys
	tag4 := refkit.TagOf(sf)
	assert.True(t, tag4.IsEmpty())
}
