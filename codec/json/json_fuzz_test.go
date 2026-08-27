// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	stdjson "encoding/json"
	"testing"

	"github.com/lemon4ksan/foundation/codec/json"
)

type FuzzPayload struct {
	ID      int64             `json:"id"`
	Name    string            `json:"name"`
	Active  bool              `json:"active"`
	Values  []float64         `json:"values"`
	Meta    map[string]string `json:"meta"`
	RawData json.RawMessage   `json:"raw_data"`
}

func FuzzJSONUnmarshal(f *testing.F) {
	f.Add([]byte(`{"id": 12345, "name": "alice", "active": true, "values": [1.5, 2.5], "meta": {"k": "v"}}`))
	f.Add([]byte(`{"id": -1, "name": "\u003cscript\u003e", "active": false}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"plain string"`))
	f.Add([]byte(`invalid json {{{`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var p FuzzPayload
		var stdP FuzzPayload

		err := json.Unmarshal(data, &p)
		stdErr := stdjson.Unmarshal(data, &stdP)

		if err == nil && stdErr == nil {
			if p.ID != stdP.ID || p.Name != stdP.Name || p.Active != stdP.Active {
				t.Fatalf("Unmarshal mismatch: got %+v, want %+v", p, stdP)
			}
		}
	})
}
