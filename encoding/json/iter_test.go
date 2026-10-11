// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/codec/json"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestObjectEntries(t *testing.T) {
	t.Parallel()

	t.Run("basic object", func(t *testing.T) {
		input := []byte(`{"name": "Alice", "age": 30, "admin": true, "score": null}`)
		keys := make([]string, 0, 4)
		vals := make([]string, 0, 4)

		for k, v := range json.ObjectEntries(input) {
			keys = append(keys, k)
			vals = append(vals, string(v))
		}

		require.Equal(t, 4, len(keys))
		assert.Equal(t, "name", keys[0])
		assert.Equal(t, `"Alice"`, vals[0])
		assert.Equal(t, "age", keys[1])
		assert.Equal(t, "30", vals[1])
		assert.Equal(t, "admin", keys[2])
		assert.Equal(t, "true", vals[2])
		assert.Equal(t, "score", keys[3])
		assert.Equal(t, "null", vals[3])
	})

	t.Run("empty object", func(t *testing.T) {
		input := []byte(`{   }`)
		count := 0
		for range json.ObjectEntries(input) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("escaped keys", func(t *testing.T) {
		input := []byte(`{"hello\nworld": 1, "quote\"test": 2}`)
		keys := make([]string, 0, 2)
		for k := range json.ObjectEntries(input) {
			keys = append(keys, k)
		}
		require.Equal(t, 2, len(keys))
		assert.Equal(t, "hello\nworld", keys[0])
		assert.Equal(t, "quote\"test", keys[1])
	})

	t.Run("early exit", func(t *testing.T) {
		input := []byte(`{"a": 1, "b": 2, "c": 3}`)
		count := 0
		for range json.ObjectEntries(input) {
			count++
			if count == 2 {
				break
			}
		}
		assert.Equal(t, 2, count)
	})

	t.Run("invalid inputs", func(t *testing.T) {
		invalidInputs := [][]byte{
			[]byte(`[1, 2, 3]`),
			[]byte(`"not an object"`),
			[]byte(`{`),
			[]byte(`{"a": 1,}`),
			[]byte(`{"a" 1}`),
			[]byte(``),
		}
		for _, inv := range invalidInputs {
			count := 0
			for range json.ObjectEntries(inv) {
				count++
			}
			assert.True(t, count <= 1)
		}
	})
}

func TestArrayElements(t *testing.T) {
	t.Parallel()

	t.Run("basic array", func(t *testing.T) {
		input := []byte(`[10, "twenty", {"key": "value"}, true, null]`)
		indices := make([]int, 0, 5)
		vals := make([]string, 0, 5)

		for i, v := range json.ArrayElements(input) {
			indices = append(indices, i)
			vals = append(vals, string(v))
		}

		require.Equal(t, 5, len(indices))
		assert.Equal(t, 0, indices[0])
		assert.Equal(t, "10", vals[0])
		assert.Equal(t, 1, indices[1])
		assert.Equal(t, `"twenty"`, vals[1])
		assert.Equal(t, 2, indices[2])
		assert.Equal(t, `{"key": "value"}`, vals[2])
		assert.Equal(t, 3, indices[3])
		assert.Equal(t, "true", vals[3])
		assert.Equal(t, 4, indices[4])
		assert.Equal(t, "null", vals[4])
	})

	t.Run("empty array", func(t *testing.T) {
		input := []byte(`[   ]`)
		count := 0
		for range json.ArrayElements(input) {
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("early exit", func(t *testing.T) {
		input := []byte(`[1, 2, 3, 4, 5]`)
		count := 0
		for range json.ArrayElements(input) {
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
	})

	t.Run("invalid inputs", func(t *testing.T) {
		invalidInputs := [][]byte{
			[]byte(`{"a": 1}`),
			[]byte(`123`),
			[]byte(`[`),
			[]byte(`[1, , 2]`),
			[]byte(``),
		}
		for _, inv := range invalidInputs {
			count := 0
			for range json.ArrayElements(inv) {
				count++
			}
			assert.True(t, count <= 1)
		}
	})
}

func TestDecodeSeq(t *testing.T) {
	t.Parallel()

	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	t.Run("multiple values", func(t *testing.T) {
		input := `{"id": 1, "name": "one"}{"id": 2, "name": "two"}{"id": 3, "name": "three"}`
		r := strings.NewReader(input)

		var items []Item
		for item, err := range json.DecodeSeq[Item](r) {
			require.NoError(t, err)
			items = append(items, item)
		}

		require.Equal(t, 3, len(items))
		assert.Equal(t, 1, items[0].ID)
		assert.Equal(t, "one", items[0].Name)
		assert.Equal(t, 2, items[1].ID)
		assert.Equal(t, "two", items[1].Name)
		assert.Equal(t, 3, items[2].ID)
		assert.Equal(t, "three", items[2].Name)
	})

	t.Run("newline delimited", func(t *testing.T) {
		input := `{"id": 10, "name": "ten"}
{"id": 20, "name": "twenty"}
`
		r := strings.NewReader(input)
		var items []Item
		for item, err := range json.DecodeSeq[Item](r) {
			require.NoError(t, err)
			items = append(items, item)
		}

		require.Equal(t, 2, len(items))
		assert.Equal(t, 10, items[0].ID)
		assert.Equal(t, 20, items[1].ID)
	})

	t.Run("early exit", func(t *testing.T) {
		input := `{"id": 1, "name": "one"}{"id": 2, "name": "two"}`
		r := strings.NewReader(input)
		count := 0
		for _, err := range json.DecodeSeq[Item](r) {
			require.NoError(t, err)
			count++
			break
		}
		assert.Equal(t, 1, count)
	})

	t.Run("syntax error terminates", func(t *testing.T) {
		input := `{"id": 1} {invalid json`
		r := strings.NewReader(input)
		var errs []error
		for _, err := range json.DecodeSeq[Item](r) {
			if err != nil {
				errs = append(errs, err)
			}
		}
		assert.Equal(t, 1, len(errs))
	})
}

func BenchmarkObjectEntries_ZeroAlloc(b *testing.B) {
	data := []byte(`{"key1": "value1", "key2": 12345, "key3": true, "key4": null}`)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for k, v := range json.ObjectEntries(data) {
			if len(k) == 0 || len(v) == 0 {
				b.Fatal("unexpected empty")
			}
		}
	}
}

func BenchmarkArrayElements_ZeroAlloc(b *testing.B) {
	data := []byte(`[100, 200, 300, 400, 500, 600, 700, 800]`)
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		for i, v := range json.ArrayElements(data) {
			if i < 0 || len(v) == 0 {
				b.Fatal("unexpected empty")
			}
		}
	}
}

func TestDelim(t *testing.T) {
	t.Parallel()
	d := json.Delim('{')
	assert.Equal(t, "{", d.String())
	var tok json.Token = d
	assert.NotNil(t, tok)
}
