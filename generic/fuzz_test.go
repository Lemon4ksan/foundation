// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic_test

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/generic"
)

func FuzzCollections(f *testing.F) {
	seeds := []string{"apple", "banana", "cherry", "", "a", "12345"}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Set
		set := generic.NewSet(s, "default")
		set.Add(s)
		_ = set.Has(s)
		_ = set.ToSlice()
		_ = set.Intersect(generic.NewSet("default"))

		// Nil Set Safety
		var nilSet generic.Set[string]
		nilSet.Add(s)
		_ = nilSet.Has(s)
		_ = nilSet.ToSlice()
		_ = nilSet.Intersect(set)

		// Cache
		cache := generic.NewCache[string, string]()
		cache.Set(s, s+"_val", 10*time.Millisecond)

		val, ok := cache.Get(s)
		if ok && val != s+"_val" {
			t.Fatalf("cache value mismatch: got %q, want %q", val, s+"_val")
		}

		// Nil Cache Safety
		var nilCache *generic.Cache[string, string]
		nilCache.Set(s, s, time.Second)
		_, _ = nilCache.Get(s)

		// ShardedMap
		sMap := generic.NewShardedMap[int64, string]()
		key := int64(len(s))
		sMap.Set(key, s)

		gotVal, gotOK := sMap.Get(key)
		if gotOK && gotVal != s {
			t.Fatalf("shardedmap value mismatch: got %q, want %q", gotVal, s)
		}

		sMap.Delete(key)

		// Nil ShardedMap Safety
		var nilSMap *generic.ShardedMap[int64, string]
		nilSMap.Set(key, s)
		_, _ = nilSMap.Get(key)
		nilSMap.Delete(key)
		_ = nilSMap.All()
	})
}

func FuzzSliceOps(f *testing.F) {
	seeds := []string{"foo", "bar", "baz", "qux", ""}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, str string) {
		items := []string{str, "static", ""}

		_ = generic.Map(items, strings.ToUpper)
		_ = generic.Map[string, string](nil, strings.ToUpper)
		_ = generic.Map[string, string](items, nil)

		_ = generic.FlatMap(items, func(s string) []string { return []string{s, s} })
		_ = generic.FlatMap[string, string](nil, nil)

		_ = generic.Chunked(items, len(str)%5)
		_ = generic.Chunked(items, -1)

		_ = generic.Unique(items)
		_ = generic.IndexBy(items, func(s string) string { return s })
		_ = generic.GroupBy(items, func(s string) string { return strconv.Itoa(len(s)) })

		_ = generic.Any(items, func(s string) bool { return len(s) > 0 })
		_ = generic.Any(items, nil)

		_ = generic.All(items, func(s string) bool { return true })
		_ = generic.All(items, nil)

		_, _ = generic.Find(items, func(s string) bool { return s == str })
		_, _ = generic.Find[string](items, nil)

		filtered := append([]string(nil), items...)
		_ = generic.FilterInPlace(filtered, func(s string) bool { return len(s) > 0 })
		_ = generic.FilterInPlace[string](items, nil)
	})
}

func FuzzMonads(f *testing.F) {
	f.Add("test_value")

	f.Fuzz(func(t *testing.T, val string) {
		opt := generic.Some(val)
		_ = opt.IsPresent()
		_ = opt.ValueOr("fallback")
		_ = opt.Filter(func(s string) bool { return len(s) > 0 })
		_ = opt.Filter(nil)
		_ = generic.MapOptional(opt, strings.ToUpper)
		_ = generic.MapOptional[string, string](opt, nil)

		none := generic.None[string]()
		_ = none.IsPresent()
		_ = none.ValueOr(val)

		res := generic.Success(val)
		_ = res.IsSuccess()
		_ = res.Recover(func(err error) string { return "recovered" })
		_ = res.Recover(nil)
		_ = generic.MapResult(res, strings.ToUpper)
		_ = generic.MapResult[string, string](res, nil)

		errRes := generic.Failure[string](fmt.Errorf("err: %s", val))
		_ = errRes.IsSuccess()
		_ = errRes.Recover(func(err error) string { return val })
	})
}

func FuzzLazyStreams(f *testing.F) {
	f.Add(5)

	f.Fuzz(func(t *testing.T, n int) {
		slice := []int{1, 2, 3, 4, 5, n}
		seq := generic.ToSeq(slice)

		_ = generic.ToSlice(seq)
		_ = generic.ToSlice[int](nil)

		mapped := generic.MapLazy(seq, func(i int) int { return i * 2 })
		_ = generic.ToSlice(mapped)
		_ = generic.MapLazy[int, int](nil, nil)

		filtered := generic.FilterLazy(seq, func(i int) bool { return i%2 == 0 })
		_ = generic.ToSlice(filtered)
		_ = generic.FilterLazy[int](nil, nil)

		_ = generic.Reduce(seq, 0, func(acc, v int) int { return acc + v })
		_ = generic.Reduce[int, int](nil, 0, nil)

		taken := generic.Take(seq, n)
		_ = generic.ToSlice(taken)
		_ = generic.Take[int](nil, n)

		dropped := generic.Drop(seq, n)
		_ = generic.ToSlice(dropped)
		_ = generic.Drop[int](nil, n)
	})
}
