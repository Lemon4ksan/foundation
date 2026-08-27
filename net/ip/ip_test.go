// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ip_test

import (
	"net"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/net/ip"
	"github.com/lemon4ksan/foundation/testkit/assert"
	"github.com/lemon4ksan/foundation/testkit/require"
)

func TestSourceIPRotator_RoundRobin(t *testing.T) {
	t.Parallel()

	addrs := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	rot, err := ip.NewSourceIPRotator(addrs)
	require.NoError(t, err)
	assert.Equal(t, 3, rot.Size())

	for i := range 6 {
		expected := addrs[i%3]
		val := rot.Next()
		require.NotNil(t, val)
		assert.Equal(t, expected, val.String())
	}
}

func TestSourceIPRotator_NextOptional(t *testing.T) {
	t.Parallel()

	addrs := []string{"10.0.0.1"}
	rot, err := ip.NewSourceIPRotator(addrs)
	require.NoError(t, err)

	opt := rot.NextOptional()
	assert.True(t, opt.IsPresent())
	val, ok := opt.Value()
	assert.True(t, ok)
	assert.Equal(t, "10.0.0.1", val.String())
}

func TestSourceIPRotator_NextForFamily(t *testing.T) {
	t.Parallel()

	addrs := []string{"192.168.1.1", "2001:db8::1", "10.0.0.1", "2001:db8::2"}
	rot, err := ip.NewSourceIPRotator(addrs)
	require.NoError(t, err)

	v4_1 := rot.NextForFamily(true)
	require.NotNil(t, v4_1)
	assert.Equal(t, "192.168.1.1", v4_1.String())

	v6_1 := rot.NextForFamily(false)
	require.NotNil(t, v6_1)
	assert.Equal(t, "2001:db8::1", v6_1.String())

	v4_2 := rot.NextForFamily(true)
	require.NotNil(t, v4_2)
	assert.Equal(t, "10.0.0.1", v4_2.String())
}

func TestSourceIPRotator_UpdatePool(t *testing.T) {
	t.Parallel()

	rot, err := ip.NewSourceIPRotator([]string{"127.0.0.1"})
	require.NoError(t, err)
	assert.Equal(t, 1, rot.Size())

	err = rot.UpdatePool([]string{"10.0.0.1", "10.0.0.2"})
	require.NoError(t, err)
	assert.Equal(t, 2, rot.Size())
	assert.Equal(t, "10.0.0.1", rot.Next().String())
}

func TestSourceIPRotator_Concurrent(t *testing.T) {
	t.Parallel()

	addrs := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}
	rot, err := ip.NewSourceIPRotator(addrs)
	require.NoError(t, err)

	var wg sync.WaitGroup
	workers := 20
	iterations := 1000

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range iterations {
				val := rot.Next()
				assert.NotNil(t, val)
			}
		}()
	}

	wg.Wait()
}

func TestIPv6SubnetRotator_Next(t *testing.T) {
	t.Parallel()

	rot, err := ip.NewIPv6SubnetRotator("2001:db8:abcd:0012::/64")
	require.NoError(t, err)

	seen := make(map[string]bool)
	for range 100 {
		val := rot.Next()
		require.NotNil(t, val)
		assert.Equal(t, 16, len(val))
		seen[val.String()] = true
	}

	assert.Greater(t, len(seen), 90)
}

func TestIPv6SubnetRotator_NextFast(t *testing.T) {
	t.Parallel()

	rot, err := ip.NewIPv6SubnetRotator("2001:db8:abcd:0012::/64")
	require.NoError(t, err)

	seen := make(map[string]bool)
	for range 100 {
		val := rot.NextFast()
		require.NotNil(t, val)
		assert.Equal(t, 16, len(val))
		seen[val.String()] = true
	}

	assert.Greater(t, len(seen), 90)
}

func TestIsPrivateIP(t *testing.T) {
	t.Parallel()

	assert.True(t, ip.IsPrivateIP(net.ParseIP("127.0.0.1")))
	assert.True(t, ip.IsPrivateIP(net.ParseIP("10.0.0.5")))
	assert.True(t, ip.IsPrivateIP(net.ParseIP("192.168.1.1")))
	assert.True(t, ip.IsPrivateIP(net.ParseIP("172.16.0.1")))
	assert.True(t, ip.IsPrivateIP(net.ParseIP("100.64.0.1")))
	assert.True(t, ip.IsPrivateIP(net.ParseIP("::1")))
	assert.True(t, ip.IsPrivateIP(net.ParseIP("fc00::1")))

	assert.False(t, ip.IsPrivateIP(net.ParseIP("8.8.8.8")))
	assert.False(t, ip.IsPrivateIP(net.ParseIP("1.1.1.1")))
	assert.False(t, ip.IsPrivateIP(net.ParseIP("2606:4700:4700::1111")))
}
