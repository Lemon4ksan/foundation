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

var errInvalidHTTPDateLength = errors.New("timekit: invalid HTTP-date length")

// AppendHTTPDate appends an RFC 7231 / RFC 9110 compliant HTTP-date (e.g. "Sun, 06 Nov 1994 08:49:37 GMT")
// to dst without heap allocations.
func AppendHTTPDate(dst []byte, t time.Time) []byte {
	t = t.UTC()

	weekday := days[t.Weekday()]
	month := months[t.Month()]

	year, _, day := t.Date()
	hour, min, sec := t.Clock()

	n := len(dst)
	if cap(dst)-n >= HTTPDateLength {
		dst = dst[:n+HTTPDateLength]
		formatHTTPDateInto(dst[n:], weekday, month, year, day, hour, min, sec)
		return dst
	}

	var buf [HTTPDateLength]byte
	formatHTTPDateInto(buf[:], weekday, month, year, day, hour, min, sec)
	return append(dst, buf[:]...)
}

func formatHTTPDateInto(buf []byte, weekday, month string, year, day, hour, min, sec int) {
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

	y := year
	if y < 0 {
		y = -y
	}

	// "1994 "
	buf[12] = byte('0' + (y/1000)%10)
	buf[13] = byte('0' + (y/100)%10)
	buf[14] = byte('0' + (y/10)%10)
	buf[15] = byte('0' + y%10)
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
}

// FormatHTTPDate returns an RFC 7231 formatted date string.
func FormatHTTPDate(t time.Time) string {
	var buf [HTTPDateLength]byte
	_ = AppendHTTPDate(buf[:0], t)
	return string(buf[:])
}

// ParseHTTPDate parses an RFC 7231 / RFC 9110 HTTP-date string into a [time.Time].
func ParseHTTPDate(s string) (time.Time, error) {
	if len(s) != HTTPDateLength {
		return time.Time{}, errInvalidHTTPDateLength
	}

	// Strictly validated RFC 7231 / RFC 1123 fast-path: "Sun, 06 Nov 1994 08:49:37 GMT"
	if s[3] == ',' && s[4] == ' ' && s[7] == ' ' && s[11] == ' ' && s[16] == ' ' &&
		s[19] == ':' && s[22] == ':' && s[25] == ' ' && s[26] == 'G' && s[27] == 'M' && s[28] == 'T' {

		d0 := s[5] - '0'
		d1 := s[6] - '0'
		if d0 <= 9 && d1 <= 9 {
			var m time.Month
			switch s[8:11] {
			case "Jan":
				m = time.January
			case "Feb":
				m = time.February
			case "Mar":
				m = time.March
			case "Apr":
				m = time.April
			case "May":
				m = time.May
			case "Jun":
				m = time.June
			case "Jul":
				m = time.July
			case "Aug":
				m = time.August
			case "Sep":
				m = time.September
			case "Oct":
				m = time.October
			case "Nov":
				m = time.November
			case "Dec":
				m = time.December
			}

			if m != 0 {
				y0 := s[12] - '0'
				y1 := s[13] - '0'
				y2 := s[14] - '0'
				y3 := s[15] - '0'
				h0 := s[17] - '0'
				h1 := s[18] - '0'
				min0 := s[20] - '0'
				min1 := s[21] - '0'
				sec0 := s[23] - '0'
				sec1 := s[24] - '0'

				if y0 <= 9 && y1 <= 9 && y2 <= 9 && y3 <= 9 &&
					h0 <= 9 && h1 <= 9 && min0 <= 9 && min1 <= 9 && sec0 <= 9 && sec1 <= 9 {
					day := int(d0)*10 + int(d1)
					year := int(y0)*1000 + int(y1)*100 + int(y2)*10 + int(y3)
					hour := int(h0)*10 + int(h1)
					min := int(min0)*10 + int(min1)
					sec := int(sec0)*10 + int(sec1)
					if hour < 24 && min < 60 && sec < 60 && day >= 1 && day <= 31 {
						return time.Date(year, m, day, hour, min, sec, 0, time.UTC), nil
					}
				}
			}
		}
	}

	return time.Parse(time.RFC1123, s)
}
