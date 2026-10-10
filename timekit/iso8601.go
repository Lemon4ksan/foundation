// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"errors"
	"time"
)

// RFC3339UTCFormatLength is the exact length in bytes of an RFC 3339 UTC string "YYYY-MM-DDTHH:MM:SSZ" (20 bytes).
const RFC3339UTCFormatLength = 20

// AppendRFC3339 appends an RFC 3339 timestamp formatted in UTC to dst without heap allocations.
func AppendRFC3339(dst []byte, t time.Time) []byte {
	t = t.UTC()

	year, month, day := t.Date()
	hour, min, sec := t.Clock()

	n := len(dst)
	if cap(dst)-n >= RFC3339UTCFormatLength {
		dst = dst[:n+RFC3339UTCFormatLength]
		formatRFC3339Into(dst[n:], year, month, day, hour, min, sec)
		return dst
	}

	var buf [RFC3339UTCFormatLength]byte
	formatRFC3339Into(buf[:], year, month, day, hour, min, sec)
	return append(dst, buf[:]...)
}

func formatRFC3339Into(buf []byte, year int, month time.Month, day, hour, min, sec int) {
	y := year % 10000
	if y < 0 {
		y = -y
	}

	// "YYYY-"
	buf[0] = byte('0' + (y/1000)%10)
	buf[1] = byte('0' + (y/100)%10)
	buf[2] = byte('0' + (y/10)%10)
	buf[3] = byte('0' + y%10)
	buf[4] = '-'

	// "MM-"
	m := int(month)
	buf[5] = byte('0' + m/10)
	buf[6] = byte('0' + m%10)
	buf[7] = '-'

	// "DD"
	buf[8] = byte('0' + day/10)
	buf[9] = byte('0' + day%10)

	// "T"
	buf[10] = 'T'

	// "HH:"
	buf[11] = byte('0' + hour/10)
	buf[12] = byte('0' + hour%10)
	buf[13] = ':'

	// "MM:"
	buf[14] = byte('0' + min/10)
	buf[15] = byte('0' + min%10)
	buf[16] = ':'

	// "SS"
	buf[17] = byte('0' + sec/10)
	buf[18] = byte('0' + sec%10)

	// "Z"
	buf[19] = 'Z'
}

// FormatRFC3339 returns an RFC 3339 formatted UTC timestamp string.
func FormatRFC3339(t time.Time) string {
	var buf [RFC3339UTCFormatLength]byte
	_ = AppendRFC3339(buf[:0], t)
	return string(buf[:])
}

// AppendISO8601 appends an ISO 8601 timestamp formatted in UTC to dst without heap allocations.
func AppendISO8601(dst []byte, t time.Time) []byte {
	return AppendRFC3339(dst, t)
}

// FormatISO8601 returns an ISO 8601 formatted UTC timestamp string.
func FormatISO8601(t time.Time) string {
	return FormatRFC3339(t)
}

// ErrInvalidRFC3339 is returned when an RFC 3339 / ISO 8601 timestamp cannot be parsed.
var ErrInvalidRFC3339 = errors.New("timekit: invalid RFC 3339 timestamp")

// ParseRFC3339 parses an RFC 3339 formatted timestamp into a [time.Time] without heap allocations.
func ParseRFC3339(s string) (time.Time, error) {
	n := len(s)
	if n < RFC3339UTCFormatLength {
		return time.Time{}, ErrInvalidRFC3339
	}

	if s[4] != '-' || s[7] != '-' || (s[10] != 'T' && s[10] != 't' && s[10] != ' ') || s[13] != ':' || s[16] != ':' {
		return time.Time{}, ErrInvalidRFC3339
	}

	y0, y1, y2, y3 := s[0]-'0', s[1]-'0', s[2]-'0', s[3]-'0'
	m0, m1 := s[5]-'0', s[6]-'0'
	d0, d1 := s[8]-'0', s[9]-'0'
	h0, h1 := s[11]-'0', s[12]-'0'
	min0, min1 := s[14]-'0', s[15]-'0'
	s0, s1 := s[17]-'0', s[18]-'0'

	if y0 > 9 || y1 > 9 || y2 > 9 || y3 > 9 ||
		m0 > 9 || m1 > 9 ||
		d0 > 9 || d1 > 9 ||
		h0 > 9 || h1 > 9 ||
		min0 > 9 || min1 > 9 ||
		s0 > 9 || s1 > 9 {
		return time.Time{}, ErrInvalidRFC3339
	}

	year := int(y0)*1000 + int(y1)*100 + int(y2)*10 + int(y3)
	month := int(m0)*10 + int(m1)
	day := int(d0)*10 + int(d1)
	hour := int(h0)*10 + int(h1)
	min := int(min0)*10 + int(min1)
	sec := int(s0)*10 + int(s1)

	if month < 1 || month > 12 || day < 1 || day > 31 || hour > 23 || min > 59 || sec > 60 {
		return time.Time{}, ErrInvalidRFC3339
	}

	dCheck := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if dCheck.Day() != day || int(dCheck.Month()) != month {
		return time.Time{}, ErrInvalidRFC3339
	}

	idx := 19
	var nsec int

	if idx < n && s[idx] == '.' {
		idx++
		if idx >= n || s[idx] < '0' || s[idx] > '9' {
			return time.Time{}, ErrInvalidRFC3339
		}
		scale := 100000000 // 1e8
		for idx < n && s[idx] >= '0' && s[idx] <= '9' {
			digit := int(s[idx] - '0')
			if scale > 0 {
				nsec += digit * scale
				scale /= 10
			}
			idx++
		}
	}

	if idx >= n {
		return time.Time{}, ErrInvalidRFC3339
	}

	var loc *time.Location
	switch s[idx] {
	case 'Z', 'z':
		if idx+1 != n {
			return time.Time{}, ErrInvalidRFC3339
		}
		loc = time.UTC
	case '+', '-':
		sign := s[idx]
		if idx+6 != n || s[idx+3] != ':' {
			return time.Time{}, ErrInvalidRFC3339
		}
		oh0, oh1 := s[idx+1]-'0', s[idx+2]-'0'
		om0, om1 := s[idx+4]-'0', s[idx+5]-'0'
		if oh0 > 9 || oh1 > 9 || om0 > 9 || om1 > 9 {
			return time.Time{}, ErrInvalidRFC3339
		}
		offsetHour := int(oh0)*10 + int(oh1)
		offsetMin := int(om0)*10 + int(om1)
		if offsetHour > 23 || offsetMin > 59 {
			return time.Time{}, ErrInvalidRFC3339
		}
		offsetSec := offsetHour*3600 + offsetMin*60
		if sign == '-' {
			offsetSec = -offsetSec
		}
		if offsetSec == 0 {
			loc = time.UTC
		} else {
			loc = time.FixedZone("", offsetSec)
		}
	default:
		return time.Time{}, ErrInvalidRFC3339
	}

	if sec == 60 {
		sec = 59
	}

	return time.Date(year, time.Month(month), day, hour, min, sec, nsec, loc), nil
}

// ParseISO8601 parses an ISO 8601 formatted timestamp into a [time.Time] without heap allocations.
func ParseISO8601(s string) (time.Time, error) {
	return ParseRFC3339(s)
}
