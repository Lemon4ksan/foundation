// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
)

type AdversarialChild struct {
	Tag  Optional[string] `json:"tag,omitzero"`
	Code Optional[int]    `json:"code,omitzero"`
}

type AdversarialParent struct {
	ID       Optional[string]           `json:"id,omitzero"`
	Child    Optional[AdversarialChild] `json:"child,omitzero"`
	Flag     Optional[bool]             `json:"flag,omitzero"`
	RawBytes Optional[[]byte]           `json:"bytes,omitzero"`
}

func TestOptional_Adversarial_NilPointerReceiver(t *testing.T) {
	var nilOpt *Optional[string]
	err := nilOpt.UnmarshalJSON([]byte(`"fail"`))
	assert.NotNil(t, err)
	assert.True(t, errors.Is(err, ErrNilOptional))
}

func TestOptional_Adversarial_CorruptedData(t *testing.T) {
	corruptedInputs := [][]byte{
		[]byte(`{`),
		[]byte(`}`),
		[]byte(`{"key":`),
		[]byte(`{"key": "val"`),
		[]byte(`[1, 2,`),
		[]byte(`"unterminated`),
		[]byte(`123a`),
		[]byte(`truefalse`),
		[]byte(`nulll`),
		[]byte(`{"nested": {"broken": `),
		[]byte("\x00\x01\x02\xff"),
		[]byte(`{"tag": 123}`), // Type mismatch: int instead of string
	}

	for i, input := range corruptedInputs {
		t.Run(fmt.Sprintf("Input_%d", i), func(t *testing.T) {
			// Test with Optional[string]
			sOpt := Some("original_string")
			err := json.Unmarshal(input, &sOpt)
			assert.NotNil(t, err)
			assert.True(t, sOpt.IsPresent())
			assert.Equal(t, "original_string", sOpt.MustValue())

			// Test with Optional[int]
			iOpt := Some(9999)
			err = json.Unmarshal(input, &iOpt)
			assert.NotNil(t, err)
			assert.True(t, iOpt.IsPresent())
			assert.Equal(t, 9999, iOpt.MustValue())

			// Test with Optional[AdversarialChild]
			cOpt := Some(AdversarialChild{Tag: Some("safe")})
			err = json.Unmarshal(input, &cOpt)
			assert.NotNil(t, err)
			assert.True(t, cOpt.IsPresent())
			assert.Equal(t, "safe", cOpt.MustValue().Tag.MustValue())
		})
	}
}

func TestOptional_Adversarial_WhitespaceAndEmpty(t *testing.T) {
	emptyInputs := [][]byte{
		nil,
		{},
		[]byte(""),
		[]byte(" "),
		[]byte("\t\t"),
		[]byte("\n\r\n"),
		[]byte("   \t  \n  "),
		[]byte("null"),
		[]byte("  null  "),
		[]byte("\n\tnull\r\n"),
		[]byte("\t  null \t\n"),
	}

	for i, input := range emptyInputs {
		t.Run(fmt.Sprintf("Empty_%d", i), func(t *testing.T) {
			var opt Optional[string]
			opt = Some("should_be_cleared")
			err := opt.UnmarshalJSON(input)
			assert.Nil(t, err)
			assert.False(t, opt.IsPresent())
			v, ok := opt.Value()
			assert.False(t, ok)
			assert.Equal(t, "", v)
			assert.True(t, opt.IsZero())
		})
	}
}

func TestOptional_Adversarial_NestedStructsRoundtrip(t *testing.T) {
	// 1. All fields present
	p1 := AdversarialParent{
		ID: Some("root-101"),
		Child: Some(AdversarialChild{
			Tag:  Some("alpha"),
			Code: Some(404),
		}),
		Flag:     Some(false), // Explicit zero-value boolean
		RawBytes: Some([]byte("binary-payload")),
	}

	data1, err := json.Marshal(p1)
	assert.Nil(t, err)

	var p1Decoded AdversarialParent
	err = json.Unmarshal(data1, &p1Decoded)
	assert.Nil(t, err)
	assert.True(t, p1Decoded.ID.IsPresent())
	assert.Equal(t, "root-101", p1Decoded.ID.MustValue())
	assert.True(t, p1Decoded.Child.IsPresent())
	assert.Equal(t, "alpha", p1Decoded.Child.MustValue().Tag.MustValue())
	assert.Equal(t, 404, p1Decoded.Child.MustValue().Code.MustValue())
	assert.True(t, p1Decoded.Flag.IsPresent())
	assert.Equal(t, false, p1Decoded.Flag.MustValue())
	assert.True(t, p1Decoded.RawBytes.IsPresent())
	assert.Equal(t, []byte("binary-payload"), p1Decoded.RawBytes.MustValue())

	// 2. Inner child partially present
	p2 := AdversarialParent{
		ID: Some("root-202"),
		Child: Some(AdversarialChild{
			Tag:  Some(""), // Explicit empty string
			Code: None[int](),
		}),
		Flag:     None[bool](),
		RawBytes: None[[]byte](),
	}

	data2, err := json.Marshal(p2)
	assert.Nil(t, err)
	// Code, Flag, RawBytes should be omitted due to omitzero!
	assert.Equal(t, `{"id":"root-202","child":{"tag":""}}`, string(data2))

	var p2Decoded AdversarialParent
	err = json.Unmarshal(data2, &p2Decoded)
	assert.Nil(t, err)
	assert.True(t, p2Decoded.ID.IsPresent())
	assert.True(t, p2Decoded.Child.IsPresent())
	assert.True(t, p2Decoded.Child.MustValue().Tag.IsPresent())
	assert.Equal(t, "", p2Decoded.Child.MustValue().Tag.MustValue())
	assert.False(t, p2Decoded.Child.MustValue().Code.IsPresent())
	assert.False(t, p2Decoded.Flag.IsPresent())
	assert.False(t, p2Decoded.RawBytes.IsPresent())

	// 3. All fields None() -> should serialize to {}
	p3 := AdversarialParent{
		ID:       None[string](),
		Child:    None[AdversarialChild](),
		Flag:     None[bool](),
		RawBytes: None[[]byte](),
	}
	data3, err := json.Marshal(p3)
	assert.Nil(t, err)
	assert.Equal(t, `{}`, string(data3))

	var p3Decoded AdversarialParent
	// Initialize with Some values to verify unmarshaling into empty JSON clears/keeps fields
	p3Decoded = AdversarialParent{ID: Some("preset")}
	err = json.Unmarshal(data3, &p3Decoded)
	assert.Nil(t, err)
	// Note: in standard Go json.Unmarshal into existing struct, missing keys in JSON do not overwrite existing fields.
	assert.Equal(t, "preset", p3Decoded.ID.MustValue())
}

func TestOptional_Adversarial_NestedOptional(t *testing.T) {
	// Optional[Optional[string]]
	nestedSome := Some(Some("inner_value"))
	data, err := json.Marshal(nestedSome)
	assert.Nil(t, err)
	assert.Equal(t, `"inner_value"`, string(data))

	var decoded Optional[Optional[string]]
	err = json.Unmarshal(data, &decoded)
	assert.Nil(t, err)
	assert.True(t, decoded.IsPresent())
	assert.True(t, decoded.MustValue().IsPresent())
	assert.Equal(t, "inner_value", decoded.MustValue().MustValue())

	// Optional[Optional[string]] where outer is Some(None())
	nestedSomeNone := Some(None[string]())
	data, err = json.Marshal(nestedSomeNone)
	assert.Nil(t, err)
	assert.Equal(t, `null`, string(data))

	// When outer is None
	nestedNone := None[Optional[string]]()
	data, err = json.Marshal(nestedNone)
	assert.Nil(t, err)
	assert.Equal(t, `null`, string(data))
}

func TestOptional_Adversarial_OmitZeroExhaustive(t *testing.T) {
	type BoundaryModel struct {
		EmptyStr  Optional[string]         `json:"empty_str,omitzero"`
		ZeroInt   Optional[int]            `json:"zero_int,omitzero"`
		ZeroFloat Optional[float64]        `json:"zero_float,omitzero"`
		FalseBool Optional[bool]           `json:"false_bool,omitzero"`
		ZeroTime  Optional[time.Time]      `json:"zero_time,omitzero"`
		EmptyMap  Optional[map[string]int] `json:"empty_map,omitzero"`
		EmptyList Optional[[]string]       `json:"empty_list,omitzero"`
		NilOpt    Optional[*int]           `json:"nil_opt,omitzero"`
	}

	// 1. All explicitly Some(zero-value)
	bm := BoundaryModel{
		EmptyStr:  Some(""),
		ZeroInt:   Some(0),
		ZeroFloat: Some(0.0),
		FalseBool: Some(false),
		ZeroTime:  Some(time.Time{}),
		EmptyMap:  Some(map[string]int{}),
		EmptyList: Some([]string{}),
		NilOpt:    Some[*int](nil),
	}

	data, err := json.Marshal(bm)
	assert.Nil(t, err)

	// Since all are Some(...), IsZero() returns false, meaning none are omitted!
	var rawMap map[string]any
	err = json.Unmarshal(data, &rawMap)
	assert.Nil(t, err)

	assert.True(t, rawMap["empty_str"] != nil)
	assert.Equal(t, "", rawMap["empty_str"])

	assert.True(t, rawMap["zero_int"] != nil)
	assert.Equal(t, float64(0), rawMap["zero_int"])

	assert.True(t, rawMap["zero_float"] != nil)
	assert.Equal(t, float64(0), rawMap["zero_float"])

	assert.True(t, rawMap["false_bool"] != nil)
	assert.Equal(t, false, rawMap["false_bool"])

	// 2. All explicitly None()
	bmNone := BoundaryModel{}
	dataNone, err := json.Marshal(bmNone)
	assert.Nil(t, err)
	assert.Equal(t, `{}`, string(dataNone))
}

func TestOptional_Adversarial_ConcurrencyStress(t *testing.T) {
	const goroutines = 100
	const iterations = 500

	sharedValue := Some("thread_safe_shared_payload")
	var wg sync.WaitGroup

	// 1. Concurrent readers of the same Optional instance
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				assert.True(t, sharedValue.IsPresent())
				assert.Equal(t, "thread_safe_shared_payload", sharedValue.MustValue())
				assert.False(t, sharedValue.IsZero())

				b, err := sharedValue.MarshalJSON()
				assert.Nil(t, err)
				assert.True(t, bytes.Equal(b, []byte(`"thread_safe_shared_payload"`)))
			}
		}()
	}

	// 2. Concurrent workers unmarshaling into independent Optional instances
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				var opt Optional[int]
				payload := fmt.Sprintf("%d", gid*1000+i)
				err := json.Unmarshal([]byte(payload), &opt)
				assert.Nil(t, err)
				assert.True(t, opt.IsPresent())
				assert.Equal(t, gid*1000+i, opt.MustValue())

				// Reset with null
				err = json.Unmarshal([]byte("null"), &opt)
				assert.Nil(t, err)
				assert.False(t, opt.IsPresent())
				assert.True(t, opt.IsZero())
			}
		}(g)
	}

	wg.Wait()
}

func TestOptional_Adversarial_PrimitiveBoundariesRoundtrip(t *testing.T) {
	// 1. Extreme Int64 Boundaries
	int64Cases := []int64{
		math.MinInt64,
		-9223372036854775807,
		-1000000000,
		-1,
		0,
		1,
		1000000000,
		math.MaxInt64,
	}
	for _, val := range int64Cases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%d", val), string(b))

		var decoded Optional[int64]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 2. Extreme Uint64 Boundaries
	uint64Cases := []uint64{
		0,
		1,
		math.MaxUint32,
		math.MaxUint64,
	}
	for _, val := range uint64Cases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%d", val), string(b))

		var decoded Optional[uint64]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 3. Float64 Extreme Boundaries
	float64Cases := []float64{
		-12345.6789,
		-1.0,
		-0.0,
		0.0,
		1.0,
		3.141592653589793,
		math.MaxFloat64,
		math.SmallestNonzeroFloat64,
	}
	for _, val := range float64Cases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)

		var decoded Optional[float64]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 4. Unicode, Emojis, and Escape Strings
	strCases := []string{
		"",
		"simple",
		"spaces in between",
		"alpha & beta = gamma ? # 100% / test+case",
		"日本語のクエリ",
		"Привет мир",
		"Größe Überprüfung",
		"مرحبا بالعالم",
		"🚀🔥💻🎉",
		"line1\nline2\r\nline3\ttab",
		"\"quoted\" and \\backslash\\",
	}
	for _, val := range strCases {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)

		var decoded Optional[string]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}

	// 5. Booleans
	for _, val := range []bool{true, false} {
		opt := Some(val)
		b, err := json.Marshal(opt)
		assert.Nil(t, err)
		assert.Equal(t, fmt.Sprintf("%t", val), string(b))

		var decoded Optional[bool]
		err = json.Unmarshal(b, &decoded)
		assert.Nil(t, err)
		assert.True(t, decoded.IsPresent())
		assert.Equal(t, val, decoded.MustValue())
		assert.False(t, decoded.IsZero())
	}
}

func TestOptional_Adversarial_CollectionsAndPointersRoundtrip(t *testing.T) {
	// 1. Slices
	optSlicePopulated := Some([]int{10, -20, 30})
	b, err := json.Marshal(optSlicePopulated)
	assert.Nil(t, err)
	assert.Equal(t, "[10,-20,30]", string(b))

	var decodedSlice Optional[[]int]
	err = json.Unmarshal(b, &decodedSlice)
	assert.Nil(t, err)
	assert.True(t, decodedSlice.IsPresent())
	assert.Equal(t, []int{10, -20, 30}, decodedSlice.MustValue())
	assert.False(t, decodedSlice.IsZero())

	// Empty slice
	optSliceEmpty := Some([]int{})
	b, err = json.Marshal(optSliceEmpty)
	assert.Nil(t, err)
	assert.Equal(t, "[]", string(b))

	var decodedEmptySlice Optional[[]int]
	err = json.Unmarshal(b, &decodedEmptySlice)
	assert.Nil(t, err)
	assert.True(t, decodedEmptySlice.IsPresent())
	assert.Equal(t, 0, len(decodedEmptySlice.MustValue()))
	assert.False(t, decodedEmptySlice.IsZero())

	// None slice
	optSliceNone := None[[]int]()
	b, err = json.Marshal(optSliceNone)
	assert.Nil(t, err)
	assert.Equal(t, "null", string(b))
	assert.True(t, optSliceNone.IsZero())

	// 2. Maps
	optMap := Some(map[string]int{"a": 1, "b": 2})
	b, err = json.Marshal(optMap)
	assert.Nil(t, err)

	var decodedMap Optional[map[string]int]
	err = json.Unmarshal(b, &decodedMap)
	assert.Nil(t, err)
	assert.True(t, decodedMap.IsPresent())
	assert.Equal(t, 2, len(decodedMap.MustValue()))
	assert.Equal(t, 1, decodedMap.MustValue()["a"])
	assert.Equal(t, 2, decodedMap.MustValue()["b"])
	assert.False(t, decodedMap.IsZero())

	// None map
	optMapNone := None[map[string]int]()
	b, err = json.Marshal(optMapNone)
	assert.Nil(t, err)
	assert.Equal(t, "null", string(b))
	assert.True(t, optMapNone.IsZero())

	// 3. Pointers
	x := 42
	optPtr := Some(&x)
	b, err = json.Marshal(optPtr)
	assert.Nil(t, err)
	assert.Equal(t, "42", string(b))
	assert.False(t, optPtr.IsZero())

	var decodedPtr Optional[*int]
	err = json.Unmarshal(b, &decodedPtr)
	assert.Nil(t, err)
	assert.True(t, decodedPtr.IsPresent())
	assert.Equal(t, 42, *decodedPtr.MustValue())

	// Nil pointer inside Some
	optNilPtr := Some[*int](nil)
	b, err = json.Marshal(optNilPtr)
	assert.Nil(t, err)
	assert.Equal(t, "null", string(b))
	assert.True(t, optNilPtr.IsPresent())
	assert.False(t, optNilPtr.IsZero())
}

func TestOptional_Adversarial_DirectIsZeroMatrix(t *testing.T) {
	// Explicit Some(zeroValue) MUST report IsZero() == false
	assert.False(t, Some("").IsZero())
	assert.False(t, Some(0).IsZero())
	assert.False(t, Some(int64(0)).IsZero())
	assert.False(t, Some(uint(0)).IsZero())
	assert.False(t, Some(uint64(0)).IsZero())
	assert.False(t, Some(0.0).IsZero())
	assert.False(t, Some(false).IsZero())
	assert.False(t, Some(time.Time{}).IsZero())
	assert.False(t, Some([]int{}).IsZero())
	assert.False(t, Some(map[string]int{}).IsZero())
	assert.False(t, Some[*int](nil).IsZero())

	// Unset None[T]() MUST report IsZero() == true
	assert.True(t, None[string]().IsZero())
	assert.True(t, None[int]().IsZero())
	assert.True(t, None[int64]().IsZero())
	assert.True(t, None[uint]().IsZero())
	assert.True(t, None[uint64]().IsZero())
	assert.True(t, None[float64]().IsZero())
	assert.True(t, None[bool]().IsZero())
	assert.True(t, None[time.Time]().IsZero())
	assert.True(t, None[[]int]().IsZero())
	assert.True(t, None[map[string]int]().IsZero())
	assert.True(t, None[*int]().IsZero())
}
