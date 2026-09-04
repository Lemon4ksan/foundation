# Zero-Dependency Test & Mocking Toolkit (`testkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/testkit)

`testkit` provides a zero-dependency testing toolkit containing full `assert`, `require`, and `mock` primitives, replacing bulky external testing packages with minimal CPU overhead.

## Architectural Components

```text
foundation/testkit/
├── assert/              # Non-terminating assertion library (Equal, True, Nil, Len, ElementsMatch...)
├── require/             # Immediate-terminating assertions (calls t.FailNow on failure)
├── mock/                # Fluent method mocking and call expectations
└── gomock/              # Interface mocking engine
```

## Core Capabilities

1. **Zero External Dependencies**: Completely eliminates third-party testing dependencies from `go.mod`.
2. **High-Speed Value Comparisons**: Optimized value comparisons with detailed color-coded failure diffs.
3. **Formatted Assertion Variants**: Every assertion provides an `*f` variant (e.g. `assert.Equalf`, `require.NotNilf`).
4. **Mocking Primitives (`mock.Mock`)**: Method expectations, return values, and call counts.

## Key APIs & Usage

### 1. Assertions & Requirements

```go
package main

import (
    "testing"

    "github.com/lemon4ksan/foundation/testkit/assert"
    "github.com/lemon4ksan/foundation/testkit/require"
)

func TestExample(t *testing.T) {
    val, err := computeValue()
    require.NoError(t, err)
    require.NotNil(t, val)

    assert.Equal(t, 42, val.Status)
    assert.ElementsMatch(t, []string{"a", "b"}, val.Tags)
}
```

### 2. Method Mocking (`testkit/mock`)

```go
package main

import (
    "testing"

    "github.com/lemon4ksan/foundation/testkit/mock"
)

type DatabaseMock struct {
    mock.Mock
}

func (m *DatabaseMock) GetUser(id int64) (string, error) {
    args := m.Called(id)
    return args.String(0), args.Error(1)
}

func TestDatabase(t *testing.T) {
    db := new(DatabaseMock)
    db.On("GetUser", int64(10)).Return("alice", nil)

    name, err := db.GetUser(10)
    db.AssertExpectations(t)
    _ = name
    _ = err
}
```
