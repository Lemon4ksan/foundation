package timekit_test

import (
	"math"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
	"github.com/lemon4ksan/foundation/timekit"
)

func TestRepro_ParseHTTPDate_InvalidMonthDay(t *testing.T) {
	// February 31 does not exist in any calendar. Must return an error.
	_, err := timekit.ParseHTTPDate("Sun, 31 Feb 1994 08:49:37 GMT")
	assert.Error(t, err, "expected error for invalid day in February")
}

func TestRepro_ParseHTTPDate_InvalidWeekday(t *testing.T) {
	// "XYZ" is not a valid 3-letter RFC 1123 weekday abbreviation.
	_, err := timekit.ParseHTTPDate("XYZ, 06 Nov 1994 08:49:37 GMT")
	assert.Error(t, err, "expected error for invalid weekday")
}

func TestRepro_ParseHTTPDate_LeapSecond(t *testing.T) {
	// RFC 7231 §5.6.7 explicitly permits leap seconds: second = 60
	tm, err := timekit.ParseHTTPDate("Sun, 06 Nov 1994 08:49:60 GMT")
	assert.NoError(t, err)
	assert.Equal(t, 1994, tm.Year())
	assert.Equal(t, time.November, tm.Month())
	assert.Equal(t, 6, tm.Day())
	assert.Equal(t, 8, tm.Hour())
	assert.Equal(t, 50, tm.Minute()) // normalized 49:60 -> 50:00
	assert.Equal(t, 0, tm.Second())
}

func TestRepro_YearBoundaries_Formatting(t *testing.T) {
	// Extreme negative years (e.g. math.MinInt) must not produce invalid ASCII characters
	minTm := time.Date(math.MinInt/1000000000000, 1, 1, 0, 0, 0, 0, time.UTC)
	s1 := timekit.FormatRFC3339(minTm)
	assert.Len(t, s1, 20)
	for i := range 4 {
		assert.True(t, s1[i] >= '0' && s1[i] <= '9', "expected year digit in RFC3339")
	}

	// Extreme positive year (> 9999) must format 4 digits cleanly
	maxTm := time.Date(12345, 1, 1, 0, 0, 0, 0, time.UTC)
	s2 := timekit.FormatRFC3339(maxTm)
	assert.Len(t, s2, 20)
	for i := range 4 {
		assert.True(t, s2[i] >= '0' && s2[i] <= '9', "expected year digit in RFC3339")
	}
}

func TestRepro_ParseRFC3339_TimezoneBoundaries(t *testing.T) {
	// Valid RFC3339 UTC
	tm, err := timekit.ParseRFC3339("2026-08-26T16:30:45Z")
	require.NoError(t, err)
	assert.Equal(t, 2026, tm.Year())
	assert.Equal(t, 16, tm.Hour())

	// Valid with timezone offset
	tm2, err := timekit.ParseRFC3339("2026-08-26T16:30:45+02:00")
	require.NoError(t, err)
	assert.Equal(t, 14, tm2.UTC().Hour())

	// Invalid offset boundaries: +25:00
	_, err = timekit.ParseRFC3339("2026-08-26T16:30:45+25:00")
	assert.Error(t, err, "expected error for offset +25:00")

	// Invalid offset boundaries: -99:99
	_, err = timekit.ParseRFC3339("2026-08-26T16:30:45-99:99")
	assert.Error(t, err, "expected error for offset -99:99")

	// Invalid minute offset: +01:60
	_, err = timekit.ParseRFC3339("2026-08-26T16:30:45+01:60")
	assert.Error(t, err, "expected error for minute offset 60")
}

func TestRepro_ParseRFC3339_FractionalSeconds(t *testing.T) {
	// Fractional seconds with 9 digits (nanoseconds)
	tm, err := timekit.ParseRFC3339("2026-08-26T16:30:45.123456789Z")
	require.NoError(t, err)
	assert.Equal(t, 123456789, tm.Nanosecond())

	// Fractional seconds with more than 9 digits (sub-nanosecond truncation without overflow)
	tm2, err := timekit.ParseRFC3339("2026-08-26T16:30:45.12345678999999Z")
	require.NoError(t, err)
	assert.Equal(t, 123456789, tm2.Nanosecond())

	// Fractional seconds with fewer than 9 digits (e.g. milliseconds)
	tm3, err := timekit.ParseRFC3339("2026-08-26T16:30:45.5Z")
	require.NoError(t, err)
	assert.Equal(t, 500000000, tm3.Nanosecond())
}

func TestRepro_ParseRFC3339_LeapSeconds(t *testing.T) {
	// Leap second 23:59:60
	tm, err := timekit.ParseRFC3339("2026-06-30T23:59:60Z")
	require.NoError(t, err)
	assert.Equal(t, 2026, tm.Year())
	assert.Equal(t, 23, tm.Hour())
	assert.Equal(t, 59, tm.Minute())
}

func TestRepro_ParseRFC3339_ZeroAlloc(t *testing.T) {
	s := "2026-08-26T16:30:45Z"
	allocs := testing.AllocsPerRun(100, func() {
		_, _ = timekit.ParseRFC3339(s)
	})
	assert.Equal(t, float64(0), allocs, "expected 0 allocations for ParseRFC3339 UTC")
}

func TestRepro_ParseRFC3339_InvalidInputs(t *testing.T) {
	invalidCases := []string{
		"",
		"2026-08-26",
		"2026-08-26T16:30:45",
		"2026-13-01T10:00:00Z",      // month 13
		"2026-00-01T10:00:00Z",      // month 0
		"2026-02-31T10:00:00Z",      // Feb 31
		"2026-01-32T10:00:00Z",      // day 32
		"2026-01-00T10:00:00Z",      // day 0
		"2026-01-01T24:00:00Z",      // hour 24
		"2026-01-01T10:60:00Z",      // min 60
		"2026-01-01T10:00:61Z",      // sec 61
		"2026-01-01T10:00:00.Z",     // dot without digits
		"2026-01-01T10:00:00+00",    // incomplete offset
		"2026-01-01T10:00:00+00:60", // minute offset 60
		"2026-01-01T10:00:00+24:00", // hour offset 24
		"2026/01/01T10:00:00Z",      // wrong separator
		"2026-01-01X10:00:00Z",      // wrong separator
	}

	for _, s := range invalidCases {
		_, err := timekit.ParseRFC3339(s)
		assert.Error(t, err, "expected error for input: "+s)
	}
}
