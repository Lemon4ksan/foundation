// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/lemon4ksan/foundation/codec/json"
	"github.com/lemon4ksan/foundation/testing/assert"
	"github.com/lemon4ksan/foundation/testing/require"
)

func TestObjectEntries_Adversarial_EarlyExit(t *testing.T) {
	t.Parallel()

	input := []byte(`{"k1": 1, "k2": 2, "k3": 3, "k4": 4, "k5": 5}`)

	t.Run("exit after 0 (first yield returns false)", func(t *testing.T) {
		count := 0
		for range json.ObjectEntries(input) {
			count++
			break
		}
		assert.Equal(t, 1, count)

		// Explicit yield call test
		count = 0
		json.ObjectEntries(input)(func(k string, v json.RawMessage) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})

	t.Run("exit after 1 iteration", func(t *testing.T) {
		count := 0
		for k, v := range json.ObjectEntries(input) {
			count++
			assert.Equal(t, "k1", k)
			assert.Equal(t, "1", string(v))
			if count == 1 {
				break
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("exit mid sequence", func(t *testing.T) {
		count := 0
		for range json.ObjectEntries(input) {
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
	})
}

func TestObjectEntries_Adversarial_Malformed(t *testing.T) {
	t.Parallel()

	malformed := [][]byte{
		nil,
		[]byte(""),
		[]byte("   "),
		[]byte("{"),
		[]byte("}"),
		[]byte("{{}}"),
		[]byte("{,}"),
		[]byte("{::}"),
		[]byte(`{"a": 1`),
		[]byte(`{"a":`),
		[]byte(`{"a"`),
		[]byte(`{"a": 1,`),
		[]byte(`{"a": 1, "b"`),
		[]byte(`{"a": 1, "b":`),
		[]byte(`{"a": 1, "b": true`),
		[]byte(`{"key" "val"}`),
		[]byte(`{123: "val"}`),
		[]byte(`{"key": }`),
		[]byte(`{"key": invalid}`),
		[]byte(`{"key": 1.2.3}`),
		[]byte(`{"\uZZZZ": 1}`),
		[]byte(`{"\u": 1}`),
		[]byte(`{"\x00": 1}`),
		[]byte(`{"\uD800": 1}`),
		[]byte(`{"a": [1, 2}}`),
		[]byte(`{"a": {"nested": `),
		[]byte(`{"a": 1, "b": 2, }`),
	}

	for i, m := range malformed {
		t.Run(fmt.Sprintf("malformed_%d", i), func(t *testing.T) {
			count := 0
			defer func() {
				r := recover()
				assert.Nil(t, r, "iterator must not panic on malformed input")
			}()
			for range json.ObjectEntries(m) {
				count++
			}
			assert.True(t, count <= 2, "malformed inputs should yield at most valid prefix entries")
		})
	}

	t.Run("empty string key", func(t *testing.T) {
		input := []byte(`{"": "emptyKey", "normal": 42}`)
		keys := make([]string, 0, 2)
		for k := range json.ObjectEntries(input) {
			keys = append(keys, k)
		}
		require.Equal(t, 2, len(keys))
		assert.Equal(t, "", keys[0])
		assert.Equal(t, "normal", keys[1])
	})
}

func TestArrayElements_Adversarial_EarlyExit(t *testing.T) {
	t.Parallel()

	input := []byte(`["a", "b", "c", "d", "e"]`)

	t.Run("exit after 0 (first yield returns false)", func(t *testing.T) {
		count := 0
		json.ArrayElements(input)(func(i int, rm json.RawMessage) bool {
			count++
			return false
		})
		assert.Equal(t, 1, count)
	})

	t.Run("exit after 1 iteration", func(t *testing.T) {
		count := 0
		for i, v := range json.ArrayElements(input) {
			count++
			assert.Equal(t, 0, i)
			assert.Equal(t, `"a"`, string(v))
			if count == 1 {
				break
			}
		}
		assert.Equal(t, 1, count)
	})

	t.Run("exit mid sequence", func(t *testing.T) {
		count := 0
		for range json.ArrayElements(input) {
			count++
			if count == 3 {
				break
			}
		}
		assert.Equal(t, 3, count)
	})
}

func TestArrayElements_Adversarial_Malformed(t *testing.T) {
	t.Parallel()

	malformed := [][]byte{
		nil,
		[]byte(""),
		[]byte("   "),
		[]byte("["),
		[]byte("]"),
		[]byte("[["),
		[]byte("[,]"),
		[]byte("[,,]"),
		[]byte("[1,"),
		[]byte("[1, 2,"),
		[]byte("[1 2]"),
		[]byte("[1, 2,]"),
		[]byte("[1, invalid]"),
		[]byte(`[1, {"a": `),
		[]byte("[1, [2,"),
		[]byte("[}"),
	}

	for i, m := range malformed {
		t.Run(fmt.Sprintf("malformed_%d", i), func(t *testing.T) {
			count := 0
			defer func() {
				r := recover()
				assert.Nil(t, r, "ArrayElements must not panic on malformed input")
			}()
			for range json.ArrayElements(m) {
				count++
			}
			assert.True(t, count <= 2, "malformed inputs should yield at most valid prefix entries")
		})
	}
}

type errFailingReader struct {
	data []byte
	err  error
}

func (e *errFailingReader) Read(p []byte) (int, error) {
	if len(e.data) > 0 {
		n := copy(p, e.data)
		e.data = e.data[n:]
		return n, nil
	}
	return 0, e.err
}

func TestDecodeSeq_Adversarial(t *testing.T) {
	t.Parallel()

	type Item struct {
		Val int `json:"val"`
	}

	t.Run("empty stream", func(t *testing.T) {
		r := strings.NewReader("")
		count := 0
		for _, err := range json.DecodeSeq[Item](r) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("whitespace only stream", func(t *testing.T) {
		r := strings.NewReader("   \t\n  \r\n   ")
		count := 0
		for _, err := range json.DecodeSeq[Item](r) {
			require.NoError(t, err)
			count++
		}
		assert.Equal(t, 0, count)
	})

	t.Run("immediate early break", func(t *testing.T) {
		r := strings.NewReader(`{"val": 10}{"val": 20}{"val": 30}`)
		count := 0
		for item, err := range json.DecodeSeq[Item](r) {
			require.NoError(t, err)
			assert.Equal(t, 10, item.Val)
			count++
			break
		}
		assert.Equal(t, 1, count)
	})

	t.Run("valid then corrupted item", func(t *testing.T) {
		r := strings.NewReader(`{"val": 100}{"val": `)
		var items []Item
		var errs []error
		for item, err := range json.DecodeSeq[Item](r) {
			if err != nil {
				errs = append(errs, err)
			} else {
				items = append(items, item)
			}
		}
		require.Equal(t, 1, len(items))
		assert.Equal(t, 100, items[0].Val)
		require.Equal(t, 1, len(errs))
	})

	t.Run("stream error yields error and halts", func(t *testing.T) {
		customErr := errors.New("simulated network read failure")
		r := &errFailingReader{
			data: []byte(`{"val": 42}`),
			err:  customErr,
		}
		var errs []error
		for _, err := range json.DecodeSeq[Item](r) {
			if err != nil {
				errs = append(errs, err)
			}
		}
		require.Equal(t, 1, len(errs))
		assert.True(t, errors.Is(errs[0], customErr))
	})
}

func TestJSONIter_Adversarial_ExtremeInputs(t *testing.T) {
	t.Parallel()

	t.Run("massive array 10000 elements", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("[")
		for i := range 10000 {
			if i > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, "%d", i)
		}
		sb.WriteString("]")
		data := []byte(sb.String())

		count := 0
		lastIdx := -1
		for idx, raw := range json.ArrayElements(data) {
			assert.Equal(t, lastIdx+1, idx)
			lastIdx = idx
			assert.True(t, len(raw) > 0)
			count++
		}
		assert.Equal(t, 10000, count)
	})

	t.Run("massive object 5000 entries", func(t *testing.T) {
		var sb strings.Builder
		sb.WriteString("{")
		for i := range 5000 {
			if i > 0 {
				sb.WriteString(",")
			}
			fmt.Fprintf(&sb, `"key_%d": %d`, i, i)
		}
		sb.WriteString("}")
		data := []byte(sb.String())

		count := 0
		for k, v := range json.ObjectEntries(data) {
			expectedKey := fmt.Sprintf("key_%d", count)
			assert.Equal(t, expectedKey, k)
			assert.True(t, len(v) > 0)
			count++
		}
		assert.Equal(t, 5000, count)
	})

	t.Run("deeply nested array 200 levels", func(t *testing.T) {
		depth := 200
		open := strings.Repeat("[", depth)
		close := strings.Repeat("]", depth)
		data := []byte(open + "42" + close)

		count := 0
		for idx, v := range json.ArrayElements(data) {
			assert.Equal(t, 0, idx)
			assert.True(t, len(v) > 0)
			count++
		}
		assert.Equal(t, 1, count)
	})

	t.Run("deeply nested object 200 levels", func(t *testing.T) {
		depth := 200
		var sb strings.Builder
		for range depth {
			sb.WriteString(`{"a": `)
		}
		sb.WriteString("100")
		sb.WriteString(strings.Repeat("}", depth))
		data := []byte(sb.String())

		count := 0
		for k, v := range json.ObjectEntries(data) {
			assert.Equal(t, "a", k)
			assert.True(t, len(v) > 0)
			count++
		}
		assert.Equal(t, 1, count)
	})
}

func TestJSONIter_Adversarial_Concurrency(t *testing.T) {
	t.Parallel()

	objData := []byte(`{"worker": 1, "status": "active", "payload": [1, 2, 3]}`)
	arrData := []byte(`[10, 20, 30, 40, 50]`)

	var wg sync.WaitGroup
	goroutines := 30

	for g := range goroutines {
		wg.Add(3)

		go func(id int) {
			defer wg.Done()
			count := 0
			for range json.ObjectEntries(objData) {
				count++
			}
			assert.Equal(t, 3, count)
		}(g)

		go func(id int) {
			defer wg.Done()
			count := 0
			for range json.ArrayElements(arrData) {
				count++
			}
			assert.Equal(t, 5, count)
		}(g)

		go func(id int) {
			defer wg.Done()
			r := strings.NewReader(`{"k": 1}{"k": 2}`)
			type Record struct {
				K int `json:"k"`
			}
			count := 0
			for rec, err := range json.DecodeSeq[Record](r) {
				assert.Nil(t, err)
				assert.True(t, rec.K > 0)
				count++
			}
			assert.Equal(t, 2, count)
		}(g)
	}

	wg.Wait()
}

func TestJSONIter_Adversarial_ZeroAlloc(t *testing.T) {
	objData := []byte(`{"a": 1, "b": 2, "c": 3, "d": 4}`)
	arrData := []byte(`[1, 2, 3, 4, 5, 6, 7, 8]`)

	allocsObj := testing.AllocsPerRun(100, func() {
		for k, v := range json.ObjectEntries(objData) {
			if len(k) == 0 || len(v) == 0 {
				t.Fatal("empty")
			}
		}
	})
	assert.Equal(t, float64(0), allocsObj, "ObjectEntries must be 0 allocs/op for ASCII keys")

	allocsArr := testing.AllocsPerRun(100, func() {
		for i, v := range json.ArrayElements(arrData) {
			if i < 0 || len(v) == 0 {
				t.Fatal("empty")
			}
		}
	})
	assert.Equal(t, float64(0), allocsArr, "ArrayElements must be 0 allocs/op")
}
