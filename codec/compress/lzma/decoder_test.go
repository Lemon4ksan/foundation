// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lzma_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/lzma"
)

func TestDecoderCore_DecodeToSlice(t *testing.T) {
	tests := []struct {
		name string // description of this test case
		// Named input parameters for receiver constructor.
		lc               uint
		lp               uint
		pb               uint
		dictSize         uint32
		uncompressedSize uint64
		// Named input parameters for target function.
		rd        *lzma.RangeDecoder
		dest      []byte
		maxUnpack uint64
		want      int
		wantErr   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := lzma.NewDecoderCore(tt.lc, tt.lp, tt.pb, tt.dictSize, tt.uncompressedSize)
			got, gotErr := d.DecodeToSlice(tt.rd, tt.dest, tt.maxUnpack)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("DecodeToSlice() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("DecodeToSlice() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("DecodeToSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
