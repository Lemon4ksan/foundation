package json

import (
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
