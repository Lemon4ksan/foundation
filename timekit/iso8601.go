// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timekit

import (
	"time"
)

// RFC3339UTCFormatLength is the exact length in bytes of an RFC 3339 UTC string "YYYY-MM-DDTHH:MM:SSZ" (20 bytes).
const RFC3339UTCFormatLength = 20

// AppendRFC3339 appends an RFC 3339 timestamp formatted in UTC to dst without heap allocations.
func AppendRFC3339(dst []byte, t time.Time) []byte {
	t = t.UTC()

	year, month, day := t.Date()
	hour, min, sec := t.Clock()

	var buf [RFC3339UTCFormatLength]byte

	// "YYYY-"
	buf[0] = byte('0' + (year/1000)%10)
	buf[1] = byte('0' + (year/100)%10)
	buf[2] = byte('0' + (year/10)%10)
	buf[3] = byte('0' + year%10)
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

	return append(dst, buf[:]...)
}

// FormatRFC3339 returns an RFC 3339 formatted UTC timestamp string.
func FormatRFC3339(t time.Time) string {
	b := AppendRFC3339(make([]byte, 0, RFC3339UTCFormatLength), t)
	return string(b)
}
