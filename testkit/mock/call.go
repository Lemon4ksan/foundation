// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"fmt"
	"reflect"
)

// Call represents an expected mock method invocation.
type Call struct {
	ctrl        *Controller
	receiver    any
	method      string
	methodType  reflect.Type
	args        []any
	returns     []any
	action      any // func
	doAndReturn any // func
	setArgs     map[int]any
	prereq      *Call
	minCalls    int
	maxCalls    int
	actualCalls int
}

func newCall(ctrl *Controller, receiver any, method string, methodType reflect.Type, args ...any) *Call {
	return &Call{
		ctrl:       ctrl,
		receiver:   receiver,
		method:     method,
		methodType: methodType,
		args:       args,
		minCalls:   1,
		maxCalls:   1,
		setArgs:    make(map[int]any),
	}
}

func (c *Call) satisfied() bool {
	return c.actualCalls >= c.minCalls && (c.maxCalls < 0 || c.actualCalls <= c.maxCalls)
}

func (c *Call) matches(receiver any, method string, args []any) bool {
	if c.receiver != receiver || c.method != method {
		return false
	}

	if c.maxCalls >= 0 && c.actualCalls >= c.maxCalls {
		return false
	}

	if c.prereq != nil && !c.prereq.satisfied() {
		return false
	}

	if len(c.args) != len(args) {
		return false
	}

	for i, expectedArg := range c.args {
		actualArg := args[i]
		if matcher, ok := expectedArg.(Matcher); ok {
			if !matcher.Matches(actualArg) {
				return false
			}
		} else if !reflect.DeepEqual(expectedArg, actualArg) {
			return false
		}
	}

	return true
}

func (c *Call) execute(args []any) []any {
	for idx, val := range c.setArgs {
		if idx < len(args) {
			target := reflect.ValueOf(args[idx])
			if target.Kind() == reflect.Pointer && !target.IsNil() {
				src := reflect.ValueOf(val)
				if src.Type().AssignableTo(target.Elem().Type()) {
					target.Elem().Set(src)
				}
			}
		}
	}

	if c.doAndReturn != nil {
		fnVal := reflect.ValueOf(c.doAndReturn)
		in := make([]reflect.Value, len(args))

		for i, arg := range args {
			if arg == nil {
				in[i] = reflect.Zero(fnVal.Type().In(i))
			} else {
				in[i] = reflect.ValueOf(arg)
			}
		}

		outVals := fnVal.Call(in)
		rets := make([]any, len(outVals))

		for i, out := range outVals {
			rets[i] = out.Interface()
		}

		return rets
	}

	if c.action != nil {
		fnVal := reflect.ValueOf(c.action)
		in := make([]reflect.Value, len(args))

		for i, arg := range args {
			if arg == nil {
				in[i] = reflect.Zero(fnVal.Type().In(i))
			} else {
				in[i] = reflect.ValueOf(arg)
			}
		}

		fnVal.Call(in)
	}

	if len(c.returns) > 0 {
		return c.returns
	}

	if c.methodType != nil && c.methodType.Kind() == reflect.Func {
		numOut := c.methodType.NumOut()
		if numOut > 0 {
			rets := make([]any, numOut)
			for i := 0; i < numOut; i++ {
				rets[i] = reflect.Zero(c.methodType.Out(i)).Interface()
			}

			return rets
		}
	}

	return []any{nil, nil, nil, nil}
}

// Return specifies return values for the expected call.
func (c *Call) Return(rets ...any) *Call {
	c.returns = rets
	return c
}

// Do specifies a function to execute when the call is made.
func (c *Call) Do(f any) *Call {
	c.action = f
	return c
}

// DoAndReturn specifies a function to execute and return its values.
func (c *Call) DoAndReturn(f any) *Call {
	c.doAndReturn = f
	return c
}

// Times specifies the exact number of times the call is expected.
func (c *Call) Times(n int) *Call {
	c.minCalls = n
	c.maxCalls = n

	return c
}

// AnyTimes allows the call to be executed 0 or more times.
func (c *Call) AnyTimes() *Call {
	c.minCalls = 0
	c.maxCalls = -1

	return c
}

// MinTimes sets the minimum number of expected calls.
func (c *Call) MinTimes(n int) *Call {
	c.minCalls = n
	c.maxCalls = -1

	return c
}

// MaxTimes sets the maximum number of expected calls (between 0 and n calls).
func (c *Call) MaxTimes(n int) *Call {
	c.minCalls = 0
	c.maxCalls = n

	return c
}

// SetArg sets an out-parameter by pointer when the call is matched.
func (c *Call) SetArg(index int, val any) *Call {
	c.setArgs[index] = val
	return c
}

func (c *Call) String() string {
	return fmt.Sprintf("%T.%s(%v)", c.receiver, c.method, c.args)
}
