// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package values_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
	"github.com/lemon4ksan/foundation/types/values"
)

type sampleStruct struct {
	U64  values.Uint64String     `json:"u64"`
	I64  values.Int64String      `json:"i64"`
	F64  values.Float64String    `json:"f64"`
	Bool values.BoolInt          `json:"bool"`
	Unix values.UnixTimestamp    `json:"unix"`
	RFC  values.RFC3339Timestamp `json:"rfc"`
}

func TestValueError_Formatting(t *testing.T) {
	t.Parallel()

	var nilErr *values.ValueError
	assert.Equal(t, "<nil>", nilErr.Error())

	baseErr := errors.New("underlying failure")

	// 1. With Field and Index >= 0
	errWithIndex := &values.ValueError{
		Err:   baseErr,
		Field: "items",
		Index: 3,
	}
	assert.Equal(t, "values: field items[3]: underlying failure", errWithIndex.Error())
	assert.Equal(t, baseErr, errWithIndex.Unwrap())

	// 2. With Field and Index < 0
	errWithoutIndex := &values.ValueError{
		Err:   baseErr,
		Field: "username",
		Index: -1,
	}
	assert.Equal(t, "values: field username: underlying failure", errWithoutIndex.Error())

	// 3. With Type only
	errWithType := &values.ValueError{
		Err:  baseErr,
		Type: "Uint64String",
	}
	assert.Equal(t, "values: Uint64String: underlying failure", errWithType.Error())

	// 4. Without Field and Type
	errBare := &values.ValueError{
		Err: baseErr,
	}
	assert.Equal(t, "values: underlying failure", errBare.Error())
}

func TestUint64String_FullCoverage(t *testing.T) {
	t.Parallel()

	// Null and empty checks
	for _, input := range []string{`""`, `null`, `""`} {
		var u values.Uint64String
		err := json.Unmarshal([]byte(input), &u)
		require.NoError(t, err)
		assert.Equal(t, values.Uint64String(0), u)
	}

	// Valid values
	var u values.Uint64String
	err := json.Unmarshal([]byte(`"18446744073709551615"`), &u)
	require.NoError(t, err)
	assert.Equal(t, values.Uint64String(18446744073709551615), u)

	// Error on invalid
	err = json.Unmarshal([]byte(`"not-a-number"`), &u)
	require.Error(t, err)
	var valErr *values.ValueError
	assert.True(t, errors.As(err, &valErr))
	assert.Equal(t, "Uint64String", valErr.Type)

	// Marshal
	marshaled, err := json.Marshal(values.Uint64String(42))
	require.NoError(t, err)
	assert.Equal(t, `"42"`, string(marshaled))
}

func TestInt64String_FullCoverage(t *testing.T) {
	t.Parallel()

	// Null and empty checks
	for _, input := range []string{`""`, `null`} {
		var i values.Int64String
		err := json.Unmarshal([]byte(input), &i)
		require.NoError(t, err)
		assert.Equal(t, values.Int64String(0), i)
	}

	// Valid negative and positive
	var i values.Int64String
	err := json.Unmarshal([]byte(`"-9223372036854775808"`), &i)
	require.NoError(t, err)
	assert.Equal(t, values.Int64String(-9223372036854775808), i)

	// Error on invalid
	err = json.Unmarshal([]byte(`"bad_int"`), &i)
	require.Error(t, err)
	var valErr *values.ValueError
	assert.True(t, errors.As(err, &valErr))
	assert.Equal(t, "Int64String", valErr.Type)

	// Marshal
	marshaled, err := json.Marshal(values.Int64String(-123))
	require.NoError(t, err)
	assert.Equal(t, `"-123"`, string(marshaled))
}

func TestFloat64String_FullCoverage(t *testing.T) {
	t.Parallel()

	// Null and empty checks
	for _, input := range []string{`""`, `null`} {
		var f values.Float64String
		err := json.Unmarshal([]byte(input), &f)
		require.NoError(t, err)
		assert.Equal(t, values.Float64String(0), f)
	}

	// Valid floats
	var f values.Float64String
	err := json.Unmarshal([]byte(`"3.1415926535"`), &f)
	require.NoError(t, err)
	assert.InDelta(t, 3.1415926535, float64(f), 0.000001)

	// Error on invalid
	err = json.Unmarshal([]byte(`"invalid_float"`), &f)
	require.Error(t, err)
	var valErr *values.ValueError
	assert.True(t, errors.As(err, &valErr))
	assert.Equal(t, "Float64String", valErr.Type)

	// Marshal
	marshaled, err := json.Marshal(values.Float64String(2.718))
	require.NoError(t, err)
	assert.Equal(t, `"2.718"`, string(marshaled))
}

func TestBoolInt_FullCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected bool
	}{
		{`"1"`, true},
		{`1`, true},
		{`"true"`, true},
		{`"TRUE"`, true},
		{`"0"`, false},
		{`0`, false},
		{`"false"`, false},
		{`"FALSE"`, false},
		{`""`, false},
		{`null`, false},
		{`"42"`, true},
		{`"-1"`, true},
		{`"invalid"`, false},
	}

	for _, tt := range tests {
		var bi values.BoolInt
		err := json.Unmarshal([]byte(tt.input), &bi)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, bool(bi))
	}

	// Marshal true
	bTrue, err := json.Marshal(values.BoolInt(true))
	require.NoError(t, err)
	assert.Equal(t, "1", string(bTrue))

	// Marshal false
	bFalse, err := json.Marshal(values.BoolInt(false))
	require.NoError(t, err)
	assert.Equal(t, "0", string(bFalse))
}

func TestUnixTimestamp_FullCoverage(t *testing.T) {
	t.Parallel()

	// Zero / empty / null / "0" values
	for _, input := range []string{`""`, `null`, `"0"`, `0`} {
		var ts values.UnixTimestamp
		err := json.Unmarshal([]byte(input), &ts)
		require.NoError(t, err)
		assert.True(t, ts.Time().IsZero())

		marshaled, err := json.Marshal(ts)
		require.NoError(t, err)
		assert.Equal(t, "0", string(marshaled))
	}

	// Valid timestamp
	var ts values.UnixTimestamp
	err := json.Unmarshal([]byte(`"1700000000"`), &ts)
	require.NoError(t, err)
	assert.Equal(t, int64(1700000000), ts.Time().Unix())

	marshaled, err := json.Marshal(ts)
	require.NoError(t, err)
	assert.Equal(t, "1700000000", string(marshaled))

	// Invalid timestamp
	err = json.Unmarshal([]byte(`"invalid_time"`), &ts)
	require.Error(t, err)
	var valErr *values.ValueError
	assert.True(t, errors.As(err, &valErr))
	assert.Equal(t, "UnixTimestamp", valErr.Type)
}

func TestRFC3339Timestamp_FullCoverage(t *testing.T) {
	t.Parallel()

	// Zero / empty / null values
	for _, input := range []string{`""`, `null`} {
		var rfc values.RFC3339Timestamp
		err := json.Unmarshal([]byte(input), &rfc)
		require.NoError(t, err)
		assert.True(t, rfc.Time().IsZero())
		assert.Equal(t, "", rfc.String())

		marshaled, err := json.Marshal(rfc)
		require.NoError(t, err)
		assert.Equal(t, "null", string(marshaled))
	}

	// Valid RFC3339
	var rfc values.RFC3339Timestamp
	err := json.Unmarshal([]byte(`"2026-08-27T22:00:00Z"`), &rfc)
	require.NoError(t, err)
	assert.Equal(t, 2026, rfc.Time().Year())
	assert.Equal(t, "2026-08-27T22:00:00Z", rfc.String())

	marshaled, err := json.Marshal(rfc)
	require.NoError(t, err)
	assert.Equal(t, `"2026-08-27T22:00:00Z"`, string(marshaled))

	// Invalid RFC3339
	err = json.Unmarshal([]byte(`"not-a-valid-date"`), &rfc)
	require.Error(t, err)
	var valErr *values.ValueError
	assert.True(t, errors.As(err, &valErr))
	assert.Equal(t, "RFC3339Timestamp", valErr.Type)
}

func TestValues_Unmarshaling_NumericAndString(t *testing.T) {
	t.Parallel()

	// Test with string-wrapped values
	jsonStr := `{
		"u64": "1234567890",
		"i64": "-987654321",
		"f64": "123.456",
		"bool": "true",
		"unix": "1700000000",
		"rfc": "2026-08-21T12:00:00Z"
	}`

	var s sampleStruct
	err := json.Unmarshal([]byte(jsonStr), &s)
	require.NoError(t, err)

	assert.Equal(t, values.Uint64String(1234567890), s.U64)
	assert.Equal(t, values.Int64String(-987654321), s.I64)
	assert.InDelta(t, 123.456, float64(s.F64), 0.001)
	assert.True(t, bool(s.Bool))
	assert.Equal(t, int64(1700000000), s.Unix.Time().Unix())
	assert.Equal(t, 2026, s.RFC.Time().Year())

	// Test with native raw JSON values
	jsonRaw := `{
		"u64": 1234567890,
		"i64": -987654321,
		"f64": 123.456,
		"bool": 1,
		"unix": 1700000000
	}`

	var s2 sampleStruct
	err = json.Unmarshal([]byte(jsonRaw), &s2)
	require.NoError(t, err)

	assert.Equal(t, values.Uint64String(1234567890), s2.U64)
	assert.Equal(t, values.Int64String(-987654321), s2.I64)
	assert.InDelta(t, 123.456, float64(s2.F64), 0.001)
	assert.True(t, bool(s2.Bool))
}
