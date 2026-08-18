// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOptionalChainingAndFilter(t *testing.T) {
	// Filter
	opt := Some(10)
	filtered := opt.Filter(func(v int) bool { return v > 5 })
	assert.True(t, filtered.IsPresent())

	filteredNone := opt.Filter(func(v int) bool { return v > 15 })
	assert.False(t, filteredNone.IsPresent())

	// Map and FlatMap (Optional Chaining equivalent)
	mapped := MapOptional(opt, func(v int) string { return "val" })
	assert.Equal(t, "val", mapped.ValueOr("default"))

	flatMap := FlatMapOptional(opt, func(v int) Optional[string] {
		return Some("flat")
	})
	assert.Equal(t, "flat", flatMap.ValueOr("default"))
}

type CustomErr struct{ msg string }

func (c *CustomErr) Error() string { return c.msg }

func TestTypedResult(t *testing.T) {
	// Success
	res := SuccessTyped[int, *CustomErr](42)
	assert.True(t, res.IsSuccess())
	val, err := res.Unwrap()
	assert.Equal(t, 42, val)
	assert.Nil(t, err)

	// Failure
	customErr := &CustomErr{msg: "oops"}
	failRes := FailureTyped[int, *CustomErr](customErr)
	assert.False(t, failRes.IsSuccess())
	_, errVal := failRes.Unwrap()
	assert.Equal(t, "oops", errVal.Error())

	// Recover
	recovered := failRes.Recover(func(e *CustomErr) int {
		return 99
	})
	assert.Equal(t, 99, recovered)

	// Map & FlatMap
	mapped := MapTypedResult(res, func(v int) string { return "success" })
	assert.True(t, mapped.IsSuccess())
	v, _ := mapped.Unwrap()
	assert.Equal(t, "success", v)
}

func TestRangeAndStride(t *testing.T) {
	r := NewRange(1, 5)
	assert.True(t, r.Contains(3))
	assert.False(t, r.Contains(5))

	var values []int
	r.Stride(1, func(val int) bool {
		values = append(values, val)
		return true
	})
	assert.Equal(t, []int{1, 2, 3, 4}, values)

	assert.Equal(t, []int{1, 2, 3, 4}, r.ToSlice())
}

func TestTaskGroup(t *testing.T) {
	ctx := context.Background()
	tg := NewTaskGroup[int](ctx)

	tg.Add(func(c context.Context) Result[int] {
		return Success(10)
	})
	tg.Add(func(c context.Context) Result[int] {
		return Success(20)
	})

	results := tg.Wait()
	assert.Len(t, results, 2)

	var sum int
	for _, r := range results {
		val, _ := r.Unwrap()
		sum += val
	}

	assert.Equal(t, 30, sum)
}

func TestTaskGroupFailureCancellation(t *testing.T) {
	ctx := context.Background()
	tg := NewTaskGroup[int](ctx)

	tg.Add(func(c context.Context) Result[int] {
		return Failure[int](errors.New("fail"))
	})
	tg.Add(func(c context.Context) Result[int] {
		select {
		case <-c.Done():
			// Succesfully cancelled early
			return Failure[int](c.Err())
		case <-time.After(1 * time.Second):
			return Success(100)
		}
	})

	results := tg.Wait()
	assert.Len(t, results, 2)

	cancelledCount := 0

	failureCount := 0
	for _, r := range results {
		if !r.IsSuccess() {
			_, err := r.Unwrap()
			if errors.Is(err, context.Canceled) {
				cancelledCount++
			} else {
				failureCount++
			}
		}
	}

	assert.Equal(t, 1, failureCount)
	assert.Equal(t, 1, cancelledCount)
}
