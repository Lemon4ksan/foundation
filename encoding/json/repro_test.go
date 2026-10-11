package json

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
)

func TestRepro_RecursionDepthBomb_Valid(t *testing.T) {
	// Deeply nested JSON array must return an error (or false) without crashing with stack overflow.
	depth := 15000
	payload := make([]byte, depth*2)
	for i := range depth {
		payload[i] = '['
		payload[depth+i] = ']'
	}

	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- false
			}
		}()
		res := Valid(payload)
		done <- res
	}()

	select {
	case ok := <-done:
		assert.False(t, ok, "expected Valid to reject depth exceeding limit")
	case <-time.After(2 * time.Second):
		t.Fatal("Valid hung or stack overflowed on recursion depth bomb")
	}
}

func TestRepro_RecursionDepthBomb_Unmarshal(t *testing.T) {
	depth := 15000
	payload := make([]byte, depth*2)
	for i := range depth {
		payload[i] = '['
		payload[depth+i] = ']'
	}

	var v any
	err := Unmarshal(payload, &v)
	assert.Error(t, err, "expected error on deeply nested input")
}

func TestRepro_IntegerOverflow_ParseUint(t *testing.T) {
	// 2^64 = 18446744073709551616 exceeds uint64 max (18446744073709551615).
	// Must fail with error rather than wrapping around to 0.
	var u uint64
	err := Unmarshal([]byte("18446744073709551616"), &u)
	assert.Error(t, err, "expected error for uint64 overflow")
}

func TestRepro_IntegerOverflow_ParseInt_PositiveSign(t *testing.T) {
	// 9223372036854775808 = math.MaxInt64 + 1.
	// In int64, this must return an error rather than silently becoming negative -9223372036854775808.
	var n int64
	err := Unmarshal([]byte("9223372036854775808"), &n)
	assert.Error(t, err, "expected error for int64 overflow")
}

func TestRepro_IntegerTruncation_Int8(t *testing.T) {
	// 1000 exceeds int8 max (127). Must fail with error rather than truncating to -24.
	var i8 int8
	err := Unmarshal([]byte("1000"), &i8)
	assert.Error(t, err, "expected error for int8 out of range")
}

func TestRepro_MalformedNumber_Validation(t *testing.T) {
	invalidNumbers := []string{
		"1.....",
		"1e",
		"1e+",
		"1.",
		"-.",
	}

	for _, s := range invalidNumbers {
		assert.False(t, Valid([]byte(s)), s)
	}
}

func TestRepro_Unicode_TruncatedHexEscape(t *testing.T) {
	// Truncated \u escape before quote must not cause scanner to jump over quote.
	var s string
	err := Unmarshal([]byte(`"\u00"extra"`), &s)
	assert.Error(t, err, "expected error on truncated \\u escape")
}

func TestRepro_Unicode_SurrogatePairs(t *testing.T) {
	// Valid surrogate pair (U+1F600 😀)
	var s string
	err := Unmarshal([]byte(`"\uD83D\uDE00"`), &s)
	assert.NoError(t, err)
	assert.Equal(t, "😀", s)

	// Unpaired high surrogate \uD800 alone should produce replacement char U+FFFD
	var s2 string
	err = Unmarshal([]byte(`"\uD800"`), &s2)
	assert.NoError(t, err)
	assert.Equal(t, "\ufffd", s2)

	// Two high surrogates \uD800\uD800 should produce \ufffd\ufffd
	var s3 string
	err = Unmarshal([]byte(`"\uD800\uD800"`), &s3)
	assert.NoError(t, err)
	assert.Equal(t, "\ufffd\ufffd", s3)

	// Unpaired low surrogate \uDC00 alone should produce replacement char U+FFFD
	var s4 string
	err = Unmarshal([]byte(`"\uDC00"`), &s4)
	assert.NoError(t, err)
	assert.Equal(t, "\ufffd", s4)
}

func TestRepro_ExtremeFloats_Overflow(t *testing.T) {
	// 1e309 exceeds float64 range (+Inf)
	var f float64
	err := Unmarshal([]byte("1e309"), &f)
	assert.Error(t, err, "expected range error for float64 overflow")

	// -1e309 exceeds float64 range (-Inf)
	err = Unmarshal([]byte("-1e309"), &f)
	assert.Error(t, err, "expected range error for float64 underflow to -Inf")
}

func TestRepro_ExtremeFloats_NegativeZero(t *testing.T) {
	// -0.0 decoded into any should preserve the negative sign bit in float64
	var v any
	err := Unmarshal([]byte("-0.0"), &v)
	assert.NoError(t, err)
	f, ok := v.(float64)
	assert.True(t, ok)
	assert.True(t, math.Signbit(f), "expected signbit to be true for -0.0")

	// -0 decoded into any should also preserve negative sign bit
	err = Unmarshal([]byte("-0"), &v)
	assert.NoError(t, err)
	f2, ok := v.(float64)
	assert.True(t, ok)
	assert.True(t, math.Signbit(f2), "expected signbit to be true for -0")
}

func TestRepro_TrailingGarbage(t *testing.T) {
	// Top-level unmarshal with trailing non-whitespace characters must error.
	var m map[string]int
	err := Unmarshal([]byte(`{"a": 1} trailing`), &m)
	assert.Error(t, err, "expected error on trailing garbage")

	// Multi-document concatenation without whitespace
	err = Unmarshal([]byte(`{"a": 1}{"b": 2}`), &m)
	assert.Error(t, err, "expected error on concatenated documents without whitespace in Unmarshal")
}

func TestRepro_StreamMultiDocument_Concatenated(t *testing.T) {
	// Stream decoder should correctly decode concatenated documents without whitespace separation
	dec := NewDecoder(strings.NewReader(`{"a":1}{"b":2}`))
	var m1, m2 map[string]int
	err := dec.Decode(&m1)
	assert.NoError(t, err)
	assert.Equal(t, 1, m1["a"])

	err = dec.Decode(&m2)
	assert.NoError(t, err)
	assert.Equal(t, 2, m2["b"])
}
