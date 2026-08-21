// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sse_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lemon4ksan/foundation/net/sse"
)

func TestSSE_Parser_Basic(t *testing.T) {
	t.Parallel()

	raw := "event: message\ndata: hello world\nid: 1\nretry: 5000\n\n" +
		"event: update\ndata: first line\ndata: second line\n\n"

	r := sse.NewReader[sse.Event](strings.NewReader(raw))

	ev1, err := r.NextEvent()
	require.NoError(t, err)
	assert.Equal(t, "message", ev1.Event)
	assert.Equal(t, "hello world", ev1.Data)
	assert.Equal(t, "1", ev1.ID)
	assert.Equal(t, 5000, ev1.Retry)

	ev2, err := r.NextEvent()
	require.NoError(t, err)
	assert.Equal(t, "update", ev2.Event)
	assert.Equal(t, "first line\nsecond line", ev2.Data)
}

func TestSSE_Iterator_JSON(t *testing.T) {
	t.Parallel()

	type Item struct {
		Name string `json:"name"`
		Val  int    `json:"val"`
	}

	raw := "data: {\"name\":\"first\",\"val\":1}\n\n" +
		"data: {\"name\":\"second\",\"val\":2}\n\n"

	r := sse.NewReader[Item](strings.NewReader(raw))

	var items []Item
	for item, err := range r.All() {
		require.NoError(t, err)
		items = append(items, item)
	}

	require.Len(t, items, 2)
	assert.Equal(t, "first", items[0].Name)
	assert.Equal(t, 1, items[0].Val)
	assert.Equal(t, "second", items[1].Name)
	assert.Equal(t, 2, items[1].Val)
}
