// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/generic"
	"github.com/lemon4ksan/foundation/testkit/assert"
)

func TestShardedMap_Basic(t *testing.T) {
	m := generic.NewShardedMap[string, int]()

	m.Set("alpha", 1)
	m.Set("beta", 2)
	m.Set("gamma", 3)

	val, ok := m.Get("alpha")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = m.Get("unknown")
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	assert.Equal(t, 3, m.Len())

	all := m.All()
	assert.Equal(t, 3, len(all))
	assert.Equal(t, 1, all["alpha"])

	m.Delete("beta")
	assert.Equal(t, 2, m.Len())
	_, ok = m.Get("beta")
	assert.False(t, ok)
}

func TestShardedMap_TryGetSet(t *testing.T) {
	m := generic.NewShardedMap[int, string]()

	ok := m.TrySet(42, "answer")
	assert.True(t, ok)

	val, exists, acquired := m.TryGet(42)
	assert.True(t, acquired)
	assert.True(t, exists)
	assert.Equal(t, "answer", val)
}

func TestShardedMap_Concurrent(t *testing.T) {
	m := generic.NewShardedMap[string, int]()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", id)
			m.Set(key, id)
			v, ok := m.Get(key)
			assert.True(t, ok)
			assert.Equal(t, id, v)
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 100, m.Len())
}

func BenchmarkShardedMap_Parallel(b *testing.B) {
	m := generic.NewShardedMap[int, int]()
	for i := range 1000 {
		m.Set(i, i)
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := i % 1000
			if i%4 == 0 {
				m.Set(key, i)
			} else {
				_, _ = m.Get(key)
			}
			i++
		}
	})
}
