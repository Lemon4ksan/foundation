// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json_test

import (
	stdjson "encoding/json"
	"testing"

	"github.com/lemon4ksan/foundation/codec/json"
)

var benchUser = User{
	ID:       1001,
	Name:     "BenchmarkUser",
	IsAdmin:  true,
	Score:    98.765,
	Tags:     []string{"bench", "json", "silicon", "aoni"},
	QuotedID: 5544,
}

var benchUserData, _ = stdjson.Marshal(benchUser)

var (
	benchSinkBytes []byte
	benchSinkUser  User
)

func BenchmarkMarshal_StdJSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, _ := stdjson.Marshal(benchUser)
		benchSinkBytes = data
	}
}

func BenchmarkMarshal_SiliconJSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(benchUser)
		benchSinkBytes = data
	}
}

func BenchmarkMarshalTo_SiliconJSON(b *testing.B) {
	dst := make([]byte, 0, 512)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		dst, _ = json.MarshalTo(dst[:0], benchUser)
		benchSinkBytes = dst
	}
}

func BenchmarkUnmarshal_StdJSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var u User
		_ = stdjson.Unmarshal(benchUserData, &u)
		benchSinkUser = u
	}
}

func BenchmarkUnmarshal_SiliconJSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var u User
		_ = json.Unmarshal(benchUserData, &u)
		benchSinkUser = u
	}
}

func BenchmarkUnmarshalNoCopy_SiliconJSON(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var u User
		_ = json.UnmarshalNoCopy(benchUserData, &u)
		benchSinkUser = u
	}
}
