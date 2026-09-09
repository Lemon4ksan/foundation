// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package generic

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/lemon4ksan/foundation/testkit/assert"
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

func TestMonads_RemainingBranches(t *testing.T) {
	// Optional Value on None
	none := None[int]()
	val, ok := none.Value()
	assert.False(t, ok)
	assert.Equal(t, 0, val)

	// Result MapResult & FlatMapResult
	successRes := Success(10)
	failRes := Failure[int](errors.New("fail"))

	mappedSuccess := MapResult(successRes, func(v int) string { return "ten" })
	assert.True(t, mappedSuccess.IsSuccess())
	valStr, _ := mappedSuccess.Unwrap()
	assert.Equal(t, "ten", valStr)

	mappedFail := MapResult(failRes, func(v int) string { return "none" })
	assert.False(t, mappedFail.IsSuccess())

	flatSuccess := FlatMapResult(successRes, func(v int) Result[string] { return Success("flat-ten") })
	assert.True(t, flatSuccess.IsSuccess())

	flatFail := FlatMapResult(failRes, func(v int) Result[string] { return Success("none") })
	assert.False(t, flatFail.IsSuccess())

	// Recover & RecoverWith
	recovered := failRes.Recover(func(err error) int { return 99 })
	assert.Equal(t, 99, recovered)

	recoveredSuccess := successRes.Recover(func(err error) int { return 99 })
	assert.Equal(t, 10, recoveredSuccess)

	recWith := failRes.RecoverWith(func(err error) Result[int] { return Success(77) })
	assert.True(t, recWith.IsSuccess())
	v, _ := recWith.Unwrap()
	assert.Equal(t, 77, v)

	recWithSuccess := successRes.RecoverWith(func(err error) Result[int] { return Success(77) })
	assert.True(t, recWithSuccess.IsSuccess())
	v, _ = recWithSuccess.Unwrap()
	assert.Equal(t, 10, v)

	// TypedResult RecoverWith & FlatMap
	typedFail := FailureTyped[int, *CustomErr](&CustomErr{msg: "err"})
	typedSuccess := SuccessTyped[int, *CustomErr](50)

	recTypedWith := typedFail.RecoverWith(func(e *CustomErr) TypedResult[int, *CustomErr] {
		return SuccessTyped[int, *CustomErr](88)
	})
	assert.True(t, recTypedWith.IsSuccess())

	recTypedSuccess := typedSuccess.RecoverWith(func(e *CustomErr) TypedResult[int, *CustomErr] {
		return SuccessTyped[int, *CustomErr](88)
	})
	assert.True(t, recTypedSuccess.IsSuccess())

	flatTyped := FlatMapTypedResult(typedSuccess, func(v int) TypedResult[string, *CustomErr] {
		return SuccessTyped[string, *CustomErr]("typed-ok")
	})
	assert.True(t, flatTyped.IsSuccess())

	flatTypedFail := FlatMapTypedResult(typedFail, func(v int) TypedResult[string, *CustomErr] {
		return SuccessTyped[string, *CustomErr]("none")
	})
	assert.False(t, flatTypedFail.IsSuccess())
}

func TestFromAndFromPtr(t *testing.T) {
	// From (comma-ok)
	optOk := From(100, true)
	assert.True(t, optOk.IsPresent())
	val, ok := optOk.Value()
	assert.True(t, ok)
	assert.Equal(t, 100, val)

	optNone := From(100, false)
	assert.False(t, optNone.IsPresent())
	valNone, okNone := optNone.Value()
	assert.False(t, okNone)
	assert.Equal(t, 0, valNone)

	// FromPtr
	x := 42
	optPtr := FromPtr(&x)
	assert.True(t, optPtr.IsPresent())
	valPtr, okPtr := optPtr.Value()
	assert.True(t, okPtr)
	assert.Equal(t, &x, valPtr)

	var nilPtr *int
	optNilPtr := FromPtr(nilPtr)
	assert.False(t, optNilPtr.IsPresent())
	_, okNil := optNilPtr.Value()
	assert.False(t, okNil)
}

func TestFromResultAndFromTypedResult(t *testing.T) {
	// FromResult success
	resSuccess := ToResult("hello", nil)
	assert.True(t, resSuccess.IsSuccess())
	val, err := resSuccess.Unwrap()
	assert.Equal(t, "hello", val)
	assert.Nil(t, err)

	// FromResult failure
	dummyErr := errors.New("something went wrong")
	resFail := ToResult("hello", dummyErr)
	assert.False(t, resFail.IsSuccess())
	valFail, errFail := resFail.Unwrap()
	assert.Equal(t, "", valFail)
	assert.Equal(t, dummyErr, errFail)

	// FromTypedResult success
	typedSuccess := FromTypedResult[int, *CustomErr](123, nil)
	assert.True(t, typedSuccess.IsSuccess())
	tVal, tErr := typedSuccess.Unwrap()
	assert.Equal(t, 123, tVal)
	assert.Nil(t, tErr)

	// FromTypedResult failure
	customErr := &CustomErr{msg: "typed fail"}
	typedFail := FromTypedResult[int, *CustomErr](123, customErr)
	assert.False(t, typedFail.IsSuccess())
	tValFail, tErrFail := typedFail.Unwrap()
	assert.Equal(t, 0, tValFail)
	assert.Equal(t, customErr, tErrFail)
}

func TestMustValue(t *testing.T) {
	// Optional MustValue
	optSome := Some("data")
	assert.Equal(t, "data", optSome.MustValue())
	optNone := None[string]()
	assert.Panics(t, func() {
		optNone.MustValue()
	})

	// Result MustValue
	resOk := Success(55)
	assert.Equal(t, 55, resOk.MustValue())
	resErr := Failure[int](errors.New("bang"))
	assert.Panics(t, func() {
		resErr.MustValue()
	})

	// TypedResult MustValue
	typedOk := SuccessTyped[string, *CustomErr]("typed-value")
	assert.Equal(t, "typed-value", typedOk.MustValue())
	typedErr := FailureTyped[string, *CustomErr](&CustomErr{msg: "typed-err"})
	assert.Panics(t, func() {
		typedErr.MustValue()
	})
}

func TestEither(t *testing.T) {
	// Constructors & inspection
	l := Left[string, int]("failure")
	assert.True(t, l.IsLeft())
	assert.False(t, l.IsRight())
	assert.Equal(t, "failure", l.Left())
	assert.Equal(t, 0, l.Right())
	assert.Equal(t, 42, l.ValueOr(42))

	r := Right[string, int](100)
	assert.False(t, r.IsLeft())
	assert.True(t, r.IsRight())
	assert.Equal(t, "", r.Left())
	assert.Equal(t, 100, r.Right())
	assert.Equal(t, 100, r.ValueOr(42))

	// FromError
	errSample := errors.New("sample error")
	eFromErr := FromError(50, errSample)
	assert.True(t, eFromErr.IsLeft())
	assert.Equal(t, errSample, eFromErr.Left())

	eFromNilErr := FromError(50, nil)
	assert.True(t, eFromNilErr.IsRight())
	assert.Equal(t, 50, eFromNilErr.Right())

	// Optionals
	assert.True(t, l.LeftOptional().IsPresent())
	assert.False(t, l.RightOptional().IsPresent())
	assert.False(t, r.LeftOptional().IsPresent())
	assert.True(t, r.RightOptional().IsPresent())

	// Fold
	var foldedLeft string
	l.Fold(func(s string) { foldedLeft = s }, func(i int) { t.Fatal("unexpected right call") })
	assert.Equal(t, "failure", foldedLeft)

	var foldedRight int
	r.Fold(func(s string) { t.Fatal("unexpected left call") }, func(i int) { foldedRight = i })
	assert.Equal(t, 100, foldedRight)

	// Swap
	swappedL := l.Swap()
	assert.True(t, swappedL.IsRight())
	assert.Equal(t, "failure", swappedL.Right())

	swappedR := r.Swap()
	assert.True(t, swappedR.IsLeft())
	assert.Equal(t, 100, swappedR.Left())

	// AsResult
	resSuccess := r.AsResult()
	assert.True(t, resSuccess.IsSuccess())
	val, err := resSuccess.Unwrap()
	assert.Equal(t, 100, val)
	assert.Nil(t, err)

	resFail := l.AsResult()
	assert.False(t, resFail.IsSuccess())
	_, err = resFail.Unwrap()
	assert.Equal(t, "failure", err.Error())

	errEither := Left[error, int](errSample)
	resFromErr := errEither.AsResult()
	assert.False(t, resFromErr.IsSuccess())
	_, err = resFromErr.Unwrap()
	assert.Equal(t, errSample, err)

	// MapEither & MapLeft
	mappedRight := MapEither(r, func(v int) string { return fmt.Sprintf("val:%d", v) })
	assert.True(t, mappedRight.IsRight())
	assert.Equal(t, "val:100", mappedRight.Right())

	mappedLeft := MapLeft(l, func(s string) string { return "prefix:" + s })
	assert.True(t, mappedLeft.IsLeft())
	assert.Equal(t, "prefix:failure", mappedLeft.Left())

	// FlatMapEither
	flatMapRight := FlatMapEither(r, func(v int) Either[string, string] {
		return Right[string, string](fmt.Sprintf("flat:%d", v))
	})
	assert.True(t, flatMapRight.IsRight())
	assert.Equal(t, "flat:100", flatMapRight.Right())

	flatMapLeft := FlatMapEither(l, func(v int) Either[string, string] {
		return Right[string, string]("unexpected")
	})
	assert.True(t, flatMapLeft.IsLeft())
	assert.Equal(t, "failure", flatMapLeft.Left())

	// FoldMap
	valLeft := FoldMap(l, func(s string) int { return len(s) }, func(i int) int { return i * 2 })
	assert.Equal(t, 7, valLeft)

	valRight := FoldMap(r, func(s string) int { return len(s) }, func(i int) int { return i * 2 })
	assert.Equal(t, 200, valRight)
}
