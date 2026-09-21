// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tcp_test

import (
	"encoding/binary"
	"testing"

	"github.com/lemon4ksan/foundation/net/packet/tcp"
)

func TestClampMSSInPlace(t *testing.T) {
	// Synthesize IPv4 + TCP SYN packet with MSS Option
	pkt := make([]byte, 44)
	pkt[0] = 0x45 // IPv4, 20-byte IP header
	pkt[9] = 6    // TCP protocol
	binary.BigEndian.PutUint16(pkt[2:4], 44)

	// TCP header at offset 20
	tcpHdr := pkt[20:]
	tcpHdr[12] = 0x60 // data offset = 6 (24 bytes)
	tcpHdr[13] = 0x02 // SYN flag

	// MSS Option: Kind=2, Len=4, Value=1460 (0x05b4)
	tcpHdr[20] = 2
	tcpHdr[21] = 4
	binary.BigEndian.PutUint16(tcpHdr[22:24], 1460)

	// Clamp with lower MTU = 1300 -> max MSS = 1300 - 40 = 1260 (0x04ec)
	tcp.ClampMSSInPlace(pkt, 1300)

	clampedMSS := binary.BigEndian.Uint16(tcpHdr[22:24])
	if clampedMSS != 1260 {
		t.Fatalf("Clamped MSS = %d, want 1260", clampedMSS)
	}

	// Clamp with higher MTU = 1500 -> should remain 1260 (no increase)
	tcp.ClampMSSInPlace(pkt, 1500)
	clampedMSS2 := binary.BigEndian.Uint16(tcpHdr[22:24])
	if clampedMSS2 != 1260 {
		t.Fatalf("Clamped MSS changed to %d, want 1260", clampedMSS2)
	}
}

func BenchmarkClampMSSInPlace(b *testing.B) {
	pkt := make([]byte, 44)
	pkt[0] = 0x45
	pkt[9] = 6
	binary.BigEndian.PutUint16(pkt[2:4], 44)
	tcpHdr := pkt[20:]
	tcpHdr[12] = 0x60
	tcpHdr[13] = 0x02
	tcpHdr[20] = 2
	tcpHdr[21] = 4
	binary.BigEndian.PutUint16(tcpHdr[22:24], 1460)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		tcp.ClampMSSInPlace(pkt, 1300)
	}
}
