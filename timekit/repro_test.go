package timekit_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/testing/assert"
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
