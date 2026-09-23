// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package identifier_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/text/encoding/internal/identifier"
)

func TestMIBConstants(t *testing.T) {
	// Verify Unofficial base and iota progression.
	if identifier.Unofficial != 10000 {
		t.Fatalf("identifier.Unofficial = %d, want 10000", identifier.Unofficial)
	}
	if identifier.Replacement != 10001 {
		t.Fatalf("identifier.Replacement = %d, want 10001", identifier.Replacement)
	}
	if identifier.XUserDefined != 10002 {
		t.Fatalf("identifier.XUserDefined = %d, want 10002", identifier.XUserDefined)
	}
	if identifier.MacintoshCyrillic != 10003 {
		t.Fatalf("identifier.MacintoshCyrillic = %d, want 10003", identifier.MacintoshCyrillic)
	}

	// Verify standard IANA MIB assignments.
	standardMIBs := []struct {
		name string
		mib  identifier.MIB
		want uint16
	}{
		{"ASCII", identifier.ASCII, 3},
		{"ISOLatin1", identifier.ISOLatin1, 4},
		{"ISOLatin2", identifier.ISOLatin2, 5},
		{"ISOLatin3", identifier.ISOLatin3, 6},
		{"ISOLatin4", identifier.ISOLatin4, 7},
		{"ISOLatinCyrillic", identifier.ISOLatinCyrillic, 8},
		{"ISOLatinArabic", identifier.ISOLatinArabic, 9},
		{"ISOLatinGreek", identifier.ISOLatinGreek, 10},
		{"ISOLatinHebrew", identifier.ISOLatinHebrew, 11},
		{"ISOLatin5", identifier.ISOLatin5, 12},
		{"ISOLatin6", identifier.ISOLatin6, 13},
		{"ISOTextComm", identifier.ISOTextComm, 14},
		{"HalfWidthKatakana", identifier.HalfWidthKatakana, 15},
		{"JISEncoding", identifier.JISEncoding, 16},
		{"ShiftJIS", identifier.ShiftJIS, 17},
		{"EUCPkdFmtJapanese", identifier.EUCPkdFmtJapanese, 18},
		{"EUCFixWidJapanese", identifier.EUCFixWidJapanese, 19},
		{"EUCKR", identifier.EUCKR, 38},
		{"ISO2022JP", identifier.ISO2022JP, 39},
		{"ISO88598I", identifier.ISO88598I, 85},
		{"UTF8", identifier.UTF8, 106},
		{"ISO885913", identifier.ISO885913, 109},
		{"ISO885914", identifier.ISO885914, 110},
		{"ISO885915", identifier.ISO885915, 111},
		{"ISO885916", identifier.ISO885916, 112},
		{"GBK", identifier.GBK, 113},
		{"GB18030", identifier.GB18030, 114},
		{"UTF16BE", identifier.UTF16BE, 1013},
		{"UTF16LE", identifier.UTF16LE, 1014},
		{"UTF16", identifier.UTF16, 1015},
		{"UTF32", identifier.UTF32, 1017},
		{"UTF32BE", identifier.UTF32BE, 1018},
		{"UTF32LE", identifier.UTF32LE, 1019},
		{"Big5", identifier.Big5, 2026},
		{"Macintosh", identifier.Macintosh, 2027},
		{"KOI8R", identifier.KOI8R, 2084},
		{"IBM866", identifier.IBM866, 2086},
		{"KOI8U", identifier.KOI8U, 2088},
		{"Windows874", identifier.Windows874, 2109},
		{"Windows1250", identifier.Windows1250, 2250},
		{"Windows1251", identifier.Windows1251, 2251},
		{"Windows1252", identifier.Windows1252, 2252},
		{"Windows1253", identifier.Windows1253, 2253},
		{"Windows1254", identifier.Windows1254, 2254},
		{"Windows1255", identifier.Windows1255, 2255},
		{"Windows1256", identifier.Windows1256, 2256},
		{"Windows1257", identifier.Windows1257, 2257},
		{"Windows1258", identifier.Windows1258, 2258},
		{"TIS620", identifier.TIS620, 2259},
		{"CP50220", identifier.CP50220, 2260},
	}

	for _, tc := range standardMIBs {
		if uint16(tc.mib) != tc.want {
			t.Errorf("MIB %s = %d, want %d", tc.name, tc.mib, tc.want)
		}
	}
}

func TestWindowsContiguity(t *testing.T) {
	// Verify Microsoft Windows 1250-1258 contiguous MIB allocation.
	windowsList := []identifier.MIB{
		identifier.Windows1250,
		identifier.Windows1251,
		identifier.Windows1252,
		identifier.Windows1253,
		identifier.Windows1254,
		identifier.Windows1255,
		identifier.Windows1256,
		identifier.Windows1257,
		identifier.Windows1258,
	}

	for i, mib := range windowsList {
		expected := uint16(2250 + i)
		if uint16(mib) != expected {
			t.Errorf("Windows code page index %d: got %d, want %d", i, mib, expected)
		}
	}
}

type mibTestMapping struct {
	mib           identifier.MIB
	canonicalName string
	aliases       []string
}

var mibTable = []mibTestMapping{
	{identifier.ASCII, "us-ascii", []string{"ascii", "iso-ir-6", "ansi_x3.4-1968"}},
	{identifier.ISOLatin1, "iso-8859-1", []string{"latin1", "iso_8859-1", "cp819"}},
	{identifier.ISOLatin2, "iso-8859-2", []string{"latin2", "iso_8859-2"}},
	{identifier.ISOLatin3, "iso-8859-3", []string{"latin3", "iso_8859-3"}},
	{identifier.ISOLatin4, "iso-8859-4", []string{"latin4", "iso_8859-4"}},
	{identifier.ISOLatinCyrillic, "iso-8859-5", []string{"cyrillic", "iso_8859-5"}},
	{identifier.ISOLatinArabic, "iso-8859-6", []string{"arabic", "iso_8859-6"}},
	{identifier.ISOLatinGreek, "iso-8859-7", []string{"greek", "greek8", "iso_8859-7"}},
	{identifier.ISOLatinHebrew, "iso-8859-8", []string{"hebrew", "iso_8859-8"}},
	{identifier.ISO88598I, "iso-8859-8-i", []string{"csiso88598i", "logical"}},
	{identifier.ISOLatin5, "iso-8859-9", []string{"latin5", "iso_8859-9"}},
	{identifier.ISOLatin6, "iso-8859-10", []string{"latin6", "iso-8859-10"}},
	{identifier.ISO885913, "iso-8859-13", []string{"latin7"}},
	{identifier.ISO885914, "iso-8859-14", []string{"latin8", "iso-celtic"}},
	{identifier.ISO885915, "iso-8859-15", []string{"latin9"}},
	{identifier.ISO885916, "iso-8859-16", []string{"latin10"}},
	{identifier.KOI8R, "koi8-r", []string{"cskoi8r"}},
	{identifier.KOI8U, "koi8-u", []string{}},
	{identifier.IBM866, "ibm866", []string{"cp866", "866"}},
	{identifier.Macintosh, "macintosh", []string{"mac"}},
	{identifier.Windows874, "windows-874", []string{"dos-874", "cp874"}},
	{identifier.Windows1250, "windows-1250", []string{"cp1250"}},
	{identifier.Windows1251, "windows-1251", []string{"cp1251"}},
	{identifier.Windows1252, "windows-1252", []string{"cp1252"}},
	{identifier.Windows1253, "windows-1253", []string{"cp1253"}},
	{identifier.Windows1254, "windows-1254", []string{"cp1254"}},
	{identifier.Windows1255, "windows-1255", []string{"cp1255"}},
	{identifier.Windows1256, "windows-1256", []string{"cp1256"}},
	{identifier.Windows1257, "windows-1257", []string{"cp1257"}},
	{identifier.Windows1258, "windows-1258", []string{"cp1258"}},
	{identifier.UTF8, "utf-8", []string{"utf8", "unicode-1-1-utf-8"}},
	{identifier.UTF16BE, "utf-16be", []string{"unicodefeff"}},
	{identifier.UTF16LE, "utf-16le", []string{"unicodefffe"}},
	{identifier.UTF16, "utf-16", []string{}},
	{identifier.UTF32, "utf-32", []string{}},
	{identifier.UTF32BE, "utf-32be", []string{}},
	{identifier.UTF32LE, "utf-32le", []string{}},
	{identifier.ShiftJIS, "shift_jis", []string{"sjis", "ms_kanji"}},
	{identifier.EUCPkdFmtJapanese, "euc-jp", []string{"eucjp", "x-euc-jp"}},
	{identifier.ISO2022JP, "iso-2022-jp", []string{"csiso2022jp"}},
	{identifier.EUCKR, "euc-kr", []string{"korean"}},
	{identifier.GBK, "gbk", []string{"cp936", "windows-936"}},
	{identifier.GB18030, "gb18030", []string{}},
	{identifier.Big5, "big5", []string{"big5-hkscs", "cn-big5"}},
	{identifier.TIS620, "tis-620", []string{}},
	{identifier.CP50220, "cp50220", []string{}},
	{identifier.Replacement, "replacement", []string{"csreplacement"}},
	{identifier.XUserDefined, "x-user-defined", []string{}},
	{identifier.MacintoshCyrillic, "x-mac-cyrillic", []string{"maccyrillic"}},
}

func TestBidirectionalMIBMapping(t *testing.T) {
	mibToName := make(map[identifier.MIB]string)
	nameToMIB := make(map[string]identifier.MIB)

	for _, entry := range mibTable {
		// Verify no duplicate MIB.
		if existing, ok := mibToName[entry.mib]; ok {
			t.Fatalf("duplicate MIB entry %d: mapped to %q, new %q",
				entry.mib, existing, entry.canonicalName)
		}
		// Verify no duplicate canonical name.
		if existingMIB, ok := nameToMIB[entry.canonicalName]; ok {
			t.Fatalf("duplicate canonical name %q: mapped to both %d and %d",
				entry.canonicalName, existingMIB, entry.mib)
		}

		mibToName[entry.mib] = entry.canonicalName
		nameToMIB[strings.ToLower(entry.canonicalName)] = entry.mib

		for _, alias := range entry.aliases {
			normAlias := strings.ToLower(alias)
			if _, ok := nameToMIB[normAlias]; !ok {
				nameToMIB[normAlias] = entry.mib
			}
		}
	}

	// 1. Forward lookup: MIB -> Canonical Name
	for _, entry := range mibTable {
		name, ok := mibToName[entry.mib]
		if !ok || name != entry.canonicalName {
			t.Errorf("Forward lookup MIB %d: got %q, want %q", entry.mib, name, entry.canonicalName)
		}
	}

	// 2. Reverse lookup: Canonical Name -> MIB
	for _, entry := range mibTable {
		mib, ok := nameToMIB[entry.canonicalName]
		if !ok || mib != entry.mib {
			t.Errorf("Reverse lookup Name %q: got %d, want %d", entry.canonicalName, mib, entry.mib)
		}
	}

	// 3. Bidirectional round-trip: MIB -> Name -> MIB
	for _, entry := range mibTable {
		name := mibToName[entry.mib]
		mib := nameToMIB[name]
		if mib != entry.mib {
			t.Errorf("Roundtrip failed for MIB %d: got %d", entry.mib, mib)
		}
	}

	// 4. Case-insensitivity reverse lookup.
	for _, entry := range mibTable {
		upper := strings.ToUpper(entry.canonicalName)
		mib, ok := nameToMIB[strings.ToLower(upper)]
		if !ok || mib != entry.mib {
			t.Errorf("Case-insensitive lookup for %q: got %d, want %d", upper, mib, entry.mib)
		}
	}

	// 5. Alias lookup.
	for _, entry := range mibTable {
		for _, alias := range entry.aliases {
			mib, ok := nameToMIB[strings.ToLower(alias)]
			if !ok || mib != entry.mib {
				t.Errorf("Alias lookup %q: got %d, want %d", alias, mib, entry.mib)
			}
		}
	}
}

// otherRegex validates the permitted charset for Interface.ID() 'other' string: [a-zA-Z0-9_-]
var otherRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateIdentifierContract(id identifier.Interface) error {
	mib, other := id.ID()
	if (mib != 0 && other != "") || (mib == 0 && other == "") {
		return fmt.Errorf("contract violation: exactly one of mib and other must be non-zero (mib=%d, other=%q)",
			mib, other)
	}
	if other != "" && !otherRegex.MatchString(other) {
		return fmt.Errorf("contract violation: other string %q contains invalid characters", other)
	}
	return nil
}

type mockIdentifier struct {
	mib   identifier.MIB
	other string
}

func (m mockIdentifier) ID() (identifier.MIB, string) {
	return m.mib, m.other
}

func TestInterfaceContract(t *testing.T) {
	tests := []struct {
		name    string
		id      mockIdentifier
		wantErr bool
	}{
		{
			name:    "Valid standard MIB",
			id:      mockIdentifier{mib: identifier.UTF8, other: ""},
			wantErr: false,
		},
		{
			name:    "Valid unofficial other string",
			id:      mockIdentifier{mib: 0, other: "x-mac-dingbat"},
			wantErr: false,
		},
		{
			name:    "Valid other string with underscores and numbers",
			id:      mockIdentifier{mib: 0, other: "custom_enc-123"},
			wantErr: false,
		},
		{
			name:    "Invalid: both mib and other non-zero",
			id:      mockIdentifier{mib: identifier.UTF8, other: "utf-8"},
			wantErr: true,
		},
		{
			name:    "Invalid: both zero",
			id:      mockIdentifier{mib: 0, other: ""},
			wantErr: true,
		},
		{
			name:    "Invalid: other contains spaces",
			id:      mockIdentifier{mib: 0, other: "x mac dingbat"},
			wantErr: true,
		},
		{
			name:    "Invalid: other contains slashes",
			id:      mockIdentifier{mib: 0, other: "x/mac/dingbat"},
			wantErr: true,
		},
		{
			name:    "Invalid: other contains @",
			id:      mockIdentifier{mib: 0, other: "custom@enc"},
			wantErr: true,
		},
		{
			name:    "Invalid: other contains dots",
			id:      mockIdentifier{mib: 0, other: "iso.8859.1"},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateIdentifierContract(tc.id)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateIdentifierContract(%+v) error = %v, wantErr = %v", tc.id, err, tc.wantErr)
			}
		})
	}
}
