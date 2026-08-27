// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/codec/json"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

type User struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	IsAdmin  bool     `json:"is_admin,omitempty"`
	Score    float64  `json:"score"`
	Tags     []string `json:"tags,omitempty"`
	Ignored  string   `json:"-"`
	QuotedID int      `json:"quoted_id,string"`
}

type Nested struct {
	User    User              `json:"user"`
	Meta    map[string]string `json:"meta"`
	Raw     json.RawMessage   `json:"raw"`
	Pointer *int              `json:"pointer"`
}

func TestMarshalUnmarshal_Primitives(t *testing.T) {
	t.Parallel()

	// Bool
	bData, err := json.Marshal(true)
	require.NoError(t, err)
	assert.Equal(t, "true", string(bData))

	var bVal bool
	require.NoError(t, json.Unmarshal(bData, &bVal))
	assert.True(t, bVal)

	// Ints
	iData, err := json.Marshal(int64(-4294967296))
	require.NoError(t, err)
	assert.Equal(t, "-4294967296", string(iData))

	var iVal int64
	require.NoError(t, json.Unmarshal(iData, &iVal))
	assert.Equal(t, int64(-4294967296), iVal)

	// Uints
	uData, err := json.Marshal(uint64(18446744073709551615))
	require.NoError(t, err)
	assert.Equal(t, "18446744073709551615", string(uData))

	var uVal uint64
	require.NoError(t, json.Unmarshal(uData, &uVal))
	assert.Equal(t, uint64(18446744073709551615), uVal)

	// Floats
	fData, err := json.Marshal(3.14159)
	require.NoError(t, err)
	assert.Equal(t, "3.14159", string(fData))

	var fVal float64
	require.NoError(t, json.Unmarshal(fData, &fVal))
	assert.Equal(t, 3.14159, fVal)

	// String with escapes
	strData, err := json.Marshal("Hello \"World\"\n\t<tag>")
	require.NoError(t, err)
	assert.Contains(t, string(strData), `\"World\"`)
	assert.Contains(t, string(strData), `\n`)
	assert.Contains(t, string(strData), `\u003c`)

	var strVal string
	require.NoError(t, json.Unmarshal(strData, &strVal))
	assert.Equal(t, "Hello \"World\"\n\t<tag>", strVal)

	// Byte slice (base64)
	bytesData, err := json.Marshal([]byte("hello silicon"))
	require.NoError(t, err)
	var bytesVal []byte
	require.NoError(t, json.Unmarshal(bytesData, &bytesVal))
	assert.Equal(t, []byte("hello silicon"), bytesVal)
}

func TestMarshalUnmarshal_Struct(t *testing.T) {
	t.Parallel()

	u := User{
		ID:       1001,
		Name:     "Alice",
		IsAdmin:  true,
		Score:    99.5,
		Tags:     []string{"admin", "core"},
		Ignored:  "secret",
		QuotedID: 12345,
	}

	data, err := json.Marshal(u)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "secret")
	assert.Contains(t, string(data), `"id":1001`)
	assert.Contains(t, string(data), `"name":"Alice"`)
	assert.Contains(t, string(data), `"is_admin":true`)
	assert.Contains(t, string(data), `"quoted_id":"12345"`)

	var u2 User
	require.NoError(t, json.Unmarshal(data, &u2))
	assert.Equal(t, u.ID, u2.ID)
	assert.Equal(t, u.Name, u2.Name)
	assert.Equal(t, u.IsAdmin, u2.IsAdmin)
	assert.Equal(t, u.Score, u2.Score)
	assert.Equal(t, u.Tags, u2.Tags)
	assert.Equal(t, u.QuotedID, u2.QuotedID)
	assert.Empty(t, u2.Ignored)
}

func TestMarshalUnmarshal_Nested(t *testing.T) {
	t.Parallel()

	pVal := 42
	n := Nested{
		User: User{
			ID:   1,
			Name: "Bob",
		},
		Meta: map[string]string{
			"env": "production",
		},
		Raw:     json.RawMessage(`{"status":"ok"}`),
		Pointer: &pVal,
	}

	data, err := json.Marshal(n)
	require.NoError(t, err)

	var n2 Nested
	require.NoError(t, json.Unmarshal(data, &n2))
	assert.Equal(t, int64(1), n2.User.ID)
	assert.Equal(t, "Bob", n2.User.Name)
	assert.Equal(t, "production", n2.Meta["env"])
	assert.JSONEq(t, `{"status":"ok"}`, string(n2.Raw))
	require.NotNil(t, n2.Pointer)
	assert.Equal(t, 42, *n2.Pointer)
}

type customType struct {
	Val string
}

func (c customType) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"custom:%s"`, c.Val)), nil
}

func (c *customType) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	c.Val = strings.TrimPrefix(s, "custom:")
	return nil
}

func TestCustomMarshaler(t *testing.T) {
	t.Parallel()

	c := customType{Val: "engine"}
	data, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Equal(t, `"custom:engine"`, string(data))

	var c2 customType
	require.NoError(t, json.Unmarshal(data, &c2))
	assert.Equal(t, "engine", c2.Val)
}

func TestDynamicInterface(t *testing.T) {
	t.Parallel()

	input := `{"name":"Aoni","count":10,"active":true,"list":[1,2,"three"]}`
	var val any
	require.NoError(t, json.Unmarshal([]byte(input), &val))

	m, ok := val.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Aoni", m["name"])
	assert.Equal(t, float64(10), m["count"])
	assert.Equal(t, true, m["active"])

	list, ok := m["list"].([]any)
	require.True(t, ok)
	assert.Len(t, list, 3)
}

func TestDecoderOptions(t *testing.T) {
	t.Parallel()

	// DisallowUnknownFields
	var u User
	err := json.UnmarshalWithConfig([]byte(`{"id":1,"unknown":123}`), &u, json.DecoderConfig{
		DisallowUnknownFields: true,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown field")

	// UseNumber
	var numVal any
	err = json.UnmarshalWithConfig([]byte(`12345678901234567890`), &numVal, json.DecoderConfig{
		UseNumber: true,
	})
	require.NoError(t, err)
	num, ok := numVal.(json.Number)
	require.True(t, ok)
	assert.Equal(t, "12345678901234567890", num.String())
}

func TestStreaming(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")

	u := User{ID: 55, Name: "Stream"}
	require.NoError(t, enc.Encode(u))
	assert.Contains(t, buf.String(), "  \"name\": \"Stream\"")

	dec := json.NewDecoder(&buf)
	var u2 User
	require.NoError(t, dec.Decode(&u2))
	assert.Equal(t, int64(55), u2.ID)
	assert.Equal(t, "Stream", u2.Name)
}

func TestValid(t *testing.T) {
	t.Parallel()

	assert.True(t, json.Valid([]byte(`{"key": "value", "list": [1, 2, 3]}`)))
	assert.True(t, json.Valid([]byte(`"hello"`)))
	assert.True(t, json.Valid([]byte(`123.456`)))
	assert.True(t, json.Valid([]byte(`true`)))
	assert.True(t, json.Valid([]byte(`null`)))

	assert.False(t, json.Valid([]byte(`{invalid`)))
	assert.False(t, json.Valid([]byte(`[1, 2,]`)))
	assert.False(t, json.Valid([]byte(``)))
}

func TestMarshalTo(t *testing.T) {
	t.Parallel()

	dst := make([]byte, 0, 128)
	u := User{ID: 777, Name: "ZeroAlloc"}

	res, err := json.MarshalTo(dst, u)
	require.NoError(t, err)
	assert.Contains(t, string(res), `"id":777`)
	assert.Contains(t, string(res), `"name":"ZeroAlloc"`)
}

func TestConcurrentMarshalUnmarshal(t *testing.T) {
	t.Parallel()

	const goroutines = 16
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				u := User{
					ID:       int64(id*1000 + i),
					Name:     "ConcurrentTester",
					IsAdmin:  i%2 == 0,
					Score:    float64(i) * 1.5,
					Tags:     []string{"go", "fast", "aoni"},
					QuotedID: 99,
				}

				data, err := json.Marshal(u)
				if err != nil {
					t.Errorf("Marshal error: %v", err)
					return
				}

				var u2 User
				if err := json.Unmarshal(data, &u2); err != nil {
					t.Errorf("Unmarshal error: %v", err)
					return
				}

				if u.ID != u2.ID || u.Name != u2.Name {
					t.Errorf("Mismatch: expected %v, got %v", u, u2)
					return
				}
			}
		}(g)
	}

	wg.Wait()
}

type TextType struct {
	Val string
}

func (t TextType) MarshalText() ([]byte, error) {
	return []byte("text:" + t.Val), nil
}

func (t *TextType) UnmarshalText(data []byte) error {
	t.Val = strings.TrimPrefix(string(data), "text:")
	return nil
}

type AllScalars struct {
	I8   int8            `json:"i8"`
	I16  int16           `json:"i16"`
	I32  int32           `json:"i32"`
	U    uint            `json:"u"`
	U8   uint8           `json:"u8"`
	U16  uint16          `json:"u16"`
	U32  uint32          `json:"u32"`
	F32  float32         `json:"f32"`
	Arr  [3]int          `json:"arr"`
	Text TextType        `json:"text"`
	Raw  json.RawMessage `json:"raw"`
}

func TestAllScalars_And_Arrays_And_TextMarshaler(t *testing.T) {
	t.Parallel()

	in := AllScalars{
		I8:   -8,
		I16:  -16,
		I32:  -32,
		U:    100,
		U8:   8,
		U16:  16,
		U32:  32,
		F32:  1.25,
		Arr:  [3]int{10, 20, 30},
		Text: TextType{Val: "custom"},
		Raw:  json.RawMessage(`{"nested":true}`),
	}

	data, err := json.Marshal(in)
	require.NoError(t, err)

	var out AllScalars
	require.NoError(t, json.Unmarshal(data, &out))
	assert.Equal(t, in.I8, out.I8)
	assert.Equal(t, in.I16, out.I16)
	assert.Equal(t, in.I32, out.I32)
	assert.Equal(t, in.U, out.U)
	assert.Equal(t, in.U8, out.U8)
	assert.Equal(t, in.U16, out.U16)
	assert.Equal(t, in.U32, out.U32)
	assert.Equal(t, in.F32, out.F32)
	assert.Equal(t, in.Arr, out.Arr)
	assert.Equal(t, in.Text.Val, out.Text.Val)
	assert.Equal(t, string(in.Raw), string(out.Raw))

	// UnmarshalNoCopy
	var outNoCopy AllScalars
	require.NoError(t, json.UnmarshalNoCopy(data, &outNoCopy))
	assert.Equal(t, in.I8, outNoCopy.I8)
}

func TestStreamHelpers_Compact_HTMLEscape(t *testing.T) {
	t.Parallel()

	// 1. Compact
	var compactBuf bytes.Buffer
	srcJSON := []byte("{\n  \"key\":   \"value\"\n}\n")
	require.NoError(t, json.Compact(&compactBuf, srcJSON))
	assert.Equal(t, `{"key":"value"}`, compactBuf.String())

	// 2. HTMLEscape
	var htmlBuf bytes.Buffer
	srcHTML := []byte(`{"tag":"<script>&foo</script>"}`)
	json.HTMLEscape(&htmlBuf, srcHTML)
	assert.Contains(t, htmlBuf.String(), `\u003cscript\u003e\u0026foo\u003c/script\u003e`)

	// 3. Decoder More and InputOffset
	dec := json.NewDecoder(strings.NewReader(`[1, 2, 3]`))
	dec.UseNumber()
	dec.DisallowUnknownFields()
	assert.Equal(t, int64(0), dec.InputOffset())

	var nums []int
	require.NoError(t, dec.Decode(&nums))
	assert.Equal(t, []int{1, 2, 3}, nums)
	assert.Greater(t, dec.InputOffset(), int64(0))

	// 4. Encoder SetEscapeHTML
	var encBuf bytes.Buffer
	enc := json.NewEncoder(&encBuf)
	enc.SetEscapeHTML(false)
	require.NoError(t, enc.Encode(map[string]string{"tag": "test"}))
	assert.Contains(t, encBuf.String(), `"tag":"test"`)
}
