// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package loadbalance_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lemon4ksan/foundation/async/loadbalance"
)

func TestBalancer_RoundRobin(t *testing.T) {
	t.Parallel()

	t1 := loadbalance.NewTarget("http://server1", 1)
	t2 := loadbalance.NewTarget("http://server2", 1)
	t3 := loadbalance.NewTarget("http://server3", 1)

	lb, err := loadbalance.New(loadbalance.RoundRobin, 10*time.Second, t1, t2, t3)
	require.NoError(t, err)

	selected := make([]string, 6)
	for i := 0; i < 6; i++ {
		target, err := lb.Select()
		require.NoError(t, err)
		selected[i] = target.Value
	}

	assert.Equal(t, []string{
		"http://server1", "http://server2", "http://server3",
		"http://server1", "http://server2", "http://server3",
	}, selected)
}

func TestBalancer_LeastConn(t *testing.T) {
	t.Parallel()

	t1 := loadbalance.NewTarget("srv1", 1)
	t2 := loadbalance.NewTarget("srv2", 1)

	t1.Acquire()
	t1.Acquire() // t1 has 2 active conns
	t2.Acquire() // t2 has 1 active conn

	lb, err := loadbalance.New(loadbalance.LeastConn, 10*time.Second, t1, t2)
	require.NoError(t, err)

	target, err := lb.Select()
	require.NoError(t, err)
	assert.Equal(t, "srv2", target.Value)
}

func TestBalancer_HealthAndCooldown(t *testing.T) {
	t.Parallel()

	t1 := loadbalance.NewTarget("srv1", 1)
	t2 := loadbalance.NewTarget("srv2", 1)

	lb, err := loadbalance.New(loadbalance.RoundRobin, 50*time.Millisecond, t1, t2)
	require.NoError(t, err)

	// Fail t1
	t1.RecordFailure(1)
	assert.False(t, t1.IsHealthy(50*time.Millisecond))

	// All selections should now go to t2
	target, err := lb.Select()
	require.NoError(t, err)
	assert.Equal(t, "srv2", target.Value)

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)
	assert.True(t, t1.IsHealthy(50*time.Millisecond))
}
