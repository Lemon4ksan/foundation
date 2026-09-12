// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package packet

import (
	"encoding/binary"
	"net/netip"
)

// CalculateInternetChecksum calculates standard 16-bit 1's complement Internet Checksum (RFC 1071).
func CalculateInternetChecksum(b []byte) uint16 {
	var sum uint32

	for i := 0; i < len(b)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(b[i : i+2]))
	}

	if len(b)%2 == 1 {
		sum += uint32(b[len(b)-1]) << 8
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}

// CalculateICMPv6Checksum calculates ICMPv6 checksum with IPv6 pseudo-header (RFC 4443 & RFC 8200 Section 8.1).
func CalculateICMPv6Checksum(srcIP, dstIP netip.Addr, icmpMessage []byte) uint16 {
	var ph [40]byte

	srcBytes := srcIP.As16()
	dstBytes := dstIP.As16()

	copy(ph[0:16], srcBytes[:])
	copy(ph[16:32], dstBytes[:])
	binary.BigEndian.PutUint32(ph[32:36], uint32(len(icmpMessage)))
	ph[39] = 58

	var sum uint32

	for i := 0; i < 40; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(ph[i : i+2]))
	}

	for i := 0; i < len(icmpMessage)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(icmpMessage[i : i+2]))
	}

	if len(icmpMessage)%2 == 1 {
		sum += uint32(icmpMessage[len(icmpMessage)-1]) << 8
	}

	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}

	return ^uint16(sum)
}
