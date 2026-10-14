// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package wire_test

import (
	"encoding/binary"
	"testing"

	"github.com/lemon4ksan/foundation/net/dns/wire"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestRepro_SkipDomainName_NegativeOffset(t *testing.T) {
	msg := []byte{0x03, 'f', 'o', 'o', 0x00}
	_, err := wire.SkipDomainName(msg, -1)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)

	_, err = wire.SkipDomainName(msg, -999)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
}

func TestRepro_SkipDomainName_PointerBomb_ForwardJump(t *testing.T) {
	// Offset 12 points forward to offset 16. Forward pointer jumps are illegal per RFC 1035 §4.1.4.
	msg := make([]byte, 20)
	msg[12] = 0xc0
	msg[13] = 16
	_, err := wire.SkipDomainName(msg, 12)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
}

func TestRepro_SkipDomainName_PointerBomb_SelfLoop(t *testing.T) {
	// Offset 12 points to offset 12 (self-loop).
	msg := make([]byte, 20)
	msg[12] = 0xc0
	msg[13] = 12
	_, err := wire.SkipDomainName(msg, 12)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
}

func TestRepro_SkipDomainName_PointerBomb_Cycle(t *testing.T) {
	msg := make([]byte, 20)
	// Offset 14 points to 12
	msg[14] = 0xc0
	msg[15] = 12
	// Offset 12 points to 14
	msg[12] = 0xc0
	msg[13] = 14
	_, err := wire.SkipDomainName(msg, 14)
	assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
}

func TestRepro_PackDNSQuery_EmbeddedNullByte(t *testing.T) {
	_, err := wire.PackDNSQuery(1, "foo\x00bar.com", wire.TypeA)
	require.Error(t, err)
	assert.ErrorIs(t, err, wire.ErrInvalidDomain)

	_, err = wire.PackDNSQuery(1, "\x00.example.com", wire.TypeA)
	require.Error(t, err)
	assert.ErrorIs(t, err, wire.ErrInvalidDomain)
}

func TestRepro_ParseDNSResponseRecords_MismatchedLengths(t *testing.T) {
	// Build a valid 12-byte header with qdCount=0, anCount=1
	header := buildMockDNSHeader(0x1234, 0x8100, 0, 1, 0, 0)

	t.Run("TypeA with rdLength != 4", func(t *testing.T) {
		// Answer: Name="." (0x00), Type=TypeA, Class=ClassIN, TTL=60, rdLength=3 (truncated!), IP=1.2.3
		ans := []byte{
			0x00,       // root domain
			0x00, 0x01, // TypeA
			0x00, 0x01, // ClassIN
			0x00, 0x00, 0x00, 0x3c, // TTL
			0x00, 0x03, // rdLength 3 != 4
			1, 2, 3,
		}
		msg := append(append([]byte{}, header...), ans...)
		_, err := wire.ParseDNSResponseRecords(msg, 0x1234)
		assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
	})

	t.Run("TypeAAAA with rdLength != 16", func(t *testing.T) {
		// Answer: Name="." (0x00), Type=TypeAAAA, Class=ClassIN, TTL=60, rdLength=15 (truncated!)
		ans := []byte{
			0x00,       // root domain
			0x00, 0x1c, // TypeAAAA
			0x00, 0x01, // ClassIN
			0x00, 0x00, 0x00, 0x3c, // TTL
			0x00, 0x04, // rdLength 4 != 16
			1, 2, 3, 4,
		}
		msg := append(append([]byte{}, header...), ans...)
		_, err := wire.ParseDNSResponseRecords(msg, 0x1234)
		assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
	})

	t.Run("TypeTXT with mismatched character string length", func(t *testing.T) {
		// Answer: Name="." (0x00), Type=TypeTXT, Class=ClassIN, TTL=60, rdLength=5, string length byte=10 (> remaining!)
		ans := []byte{
			0x00,       // root domain
			0x00, 0x10, // TypeTXT
			0x00, 0x01, // ClassIN
			0x00, 0x00, 0x00, 0x3c, // TTL
			0x00, 0x05, // rdLength 5
			0x0a, 'h', 'e', 'l', 'l', // string length 10 but only 4 bytes follow!
		}
		msg := append(append([]byte{}, header...), ans...)
		_, err := wire.ParseDNSResponseRecords(msg, 0x1234)
		assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
	})

	t.Run("TypeOPT with mismatched option length", func(t *testing.T) {
		// Answer: Name="." (0x00), Type=TypeOPT, Class=4096, TTL=0, rdLength=6
		// OptionCode=8, OptionLength=10 (exceeds rdLength!)
		var ans [11]byte
		ans[0] = 0x00
		binary.BigEndian.PutUint16(ans[1:3], wire.TypeOPT)
		binary.BigEndian.PutUint16(ans[3:5], 4096)
		binary.BigEndian.PutUint32(ans[5:9], 0)
		binary.BigEndian.PutUint16(ans[9:11], 6) // rdLength 6

		optPayload := []byte{0x00, 0x08, 0x00, 0x0a, 0x01, 0x02} // option len 10 > 2 remaining
		msg := append(append([]byte{}, header...), ans[:]...)
		msg = append(msg, optPayload...)
		_, err := wire.ParseDNSResponseRecords(msg, 0x1234)
		assert.ErrorIs(t, err, wire.ErrTruncatedDNSMessage)
	})
}
