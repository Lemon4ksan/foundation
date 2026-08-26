// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"errors"
	"time"
)

var days = [...]string{
	"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat",
}

var months = [...]string{
	"", "Jan", "Feb", "Mar", "Apr", "May", "Jun",
	"Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
}

// HTTPDateLength is the exact length in bytes of an RFC 7231 HTTP-date string (29 bytes).
const HTTPDateLength = 29

// AppendHTTPDate appends an RFC 7231 / RFC 9110 compliant HTTP-date (e.g. "Sun, 06 Nov 1994 08:49:37 GMT")
// to dst without heap allocations.
func AppendHTTPDate(dst []byte, t time.Time) []byte {
	t = t.UTC()

	weekday := days[t.Weekday()]
	month := months[t.Month()]

	year, _, day := t.Date()
	hour, min, sec := t.Clock()

	var buf [HTTPDateLength]byte

	// "Sun, "
	buf[0] = weekday[0]
	buf[1] = weekday[1]
	buf[2] = weekday[2]
	buf[3] = ','
	buf[4] = ' '

	// "06 "
	buf[5] = byte('0' + day/10)
	buf[6] = byte('0' + day%10)
	buf[7] = ' '

	// "Nov "
	buf[8] = month[0]
	buf[9] = month[1]
	buf[10] = month[2]
	buf[11] = ' '

	// "1994 "
	buf[12] = byte('0' + (year/1000)%10)
	buf[13] = byte('0' + (year/100)%10)
	buf[14] = byte('0' + (year/10)%10)
	buf[15] = byte('0' + year%10)
	buf[16] = ' '

	// "08:49:37 GMT"
	buf[17] = byte('0' + hour/10)
	buf[18] = byte('0' + hour%10)
	buf[19] = ':'
	buf[20] = byte('0' + min/10)
	buf[21] = byte('0' + min%10)
	buf[22] = ':'
	buf[23] = byte('0' + sec/10)
	buf[24] = byte('0' + sec%10)
	buf[25] = ' '
	buf[26] = 'G'
	buf[27] = 'M'
	buf[28] = 'T'

	return append(dst, buf[:]...)
}

// FormatHTTPDate returns an RFC 7231 formatted date string.
func FormatHTTPDate(t time.Time) string {
	b := AppendHTTPDate(make([]byte, 0, HTTPDateLength), t)
	return string(b)
}

// ParseHTTPDate parses an RFC 7231 / RFC 9110 HTTP-date string into a [time.Time].
func ParseHTTPDate(s string) (time.Time, error) {
	if len(s) != HTTPDateLength {
		return time.Time{}, errors.New("timekit: invalid HTTP-date length")
	}
	return time.Parse(time.RFC1123, s)
}
