// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ip_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/ip"
)

func BenchmarkSourceIPRotator_Next_Serial(b *testing.B) {
	addrs := []string{
		"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4",
		"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4",
	}
	rot, err := ip.NewSourceIPRotator(addrs)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = rot.Next()
	}
}

func BenchmarkSourceIPRotator_Next_Parallel(b *testing.B) {
	addrs := []string{
		"192.168.1.1", "192.168.1.2", "192.168.1.3", "192.168.1.4",
		"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4",
	}
	rot, err := ip.NewSourceIPRotator(addrs)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rot.Next()
		}
	})
}

func BenchmarkIPv6SubnetRotator_NextFast(b *testing.B) {
	rot, err := ip.NewIPv6SubnetRotator("2001:db8:abcd:0012::/64")
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = rot.NextFast()
		}
	})
}
