// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package values_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

func TestValues_BoolInt(t *testing.T) {
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
	}

	for _, tt := range tests {
		var bi values.BoolInt
		err := json.Unmarshal([]byte(tt.input), &bi)
		require.NoError(t, err)
		assert.Equal(t, tt.expected, bool(bi))
	}
}

func TestValues_Timestamps(t *testing.T) {
	t.Parallel()

	// Zero values
	var unix values.UnixTimestamp
	marshaledUnix, err := json.Marshal(unix)
	require.NoError(t, err)
	assert.Equal(t, "0", string(marshaledUnix))

	var rfc values.RFC3339Timestamp
	marshaledRFC, err := json.Marshal(rfc)
	require.NoError(t, err)
	assert.Equal(t, "null", string(marshaledRFC))

	// Non-zero values
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	unix = values.UnixTimestamp(now)
	marshaledUnix, err = json.Marshal(unix)
	require.NoError(t, err)
	assert.Equal(t, "1787313600", string(marshaledUnix))

	rfc = values.RFC3339Timestamp(now)
	marshaledRFC, err = json.Marshal(rfc)
	require.NoError(t, err)
	assert.Equal(t, `"2026-08-21T12:00:00Z"`, string(marshaledRFC))
}
