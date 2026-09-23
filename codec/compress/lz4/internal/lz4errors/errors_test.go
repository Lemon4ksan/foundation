// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lz4errors_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/lz4/internal/lz4errors"
)

func TestLZ4ErrorsTable(t *testing.T) {
	tests := []struct {
		err     lz4errors.Error
		wantMsg string
	}{
		{lz4errors.ErrInvalidSourceShortBuffer, "lz4: invalid source or destination buffer too short"},
		{lz4errors.ErrInvalidFrame, "lz4: bad magic number"},
		{lz4errors.ErrInternalUnhandledState, "lz4: unhandled state"},
		{lz4errors.ErrInvalidHeaderChecksum, "lz4: invalid header checksum"},
		{lz4errors.ErrInvalidBlockChecksum, "lz4: invalid block checksum"},
		{lz4errors.ErrInvalidFrameChecksum, "lz4: invalid frame checksum"},
		{lz4errors.ErrOptionInvalidCompressionLevel, "lz4: invalid compression level"},
		{lz4errors.ErrOptionClosedOrError, "lz4: cannot apply options on closed or in error object"},
		{lz4errors.ErrOptionInvalidBlockSize, "lz4: invalid block size"},
		{lz4errors.ErrOptionNotApplicable, "lz4: option not applicable"},
		{lz4errors.ErrWriterNotClosed, "lz4: writer not closed"},
		{lz4errors.ErrEndOfStream, "lz4: end of stream reached"},
	}

	seen := make(map[string]bool)
	for _, tt := range tests {
		t.Run(string(tt.err), func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMsg {
				t.Fatalf("err.Error() = %q; want %q", got, tt.wantMsg)
			}
			if !strings.HasPrefix(got, "lz4: ") {
				t.Fatalf("err.Error() does not start with 'lz4: ': %q", got)
			}
			if seen[got] {
				t.Fatalf("duplicate error message: %q", got)
			}
			seen[got] = true
		})
	}
	if len(seen) != 12 {
		t.Fatalf("expected 12 unique error constants, got %d", len(seen))
	}
}

func TestLZ4ErrorsWrappingAndUnwrapping(t *testing.T) {
	err := lz4errors.ErrInvalidHeaderChecksum

	// Direct match
	if !errors.Is(err, lz4errors.ErrInvalidHeaderChecksum) {
		t.Fatalf("direct errors.Is failed")
	}

	// Single wrapped
	wrapped := fmt.Errorf("context wrapper: %w", err)
	if !errors.Is(wrapped, lz4errors.ErrInvalidHeaderChecksum) {
		t.Fatalf("wrapped errors.Is failed")
	}

	// Multi-wrapped
	deepWrapped := fmt.Errorf("outer: %w", fmt.Errorf("middle: %w", err))
	if !errors.Is(deepWrapped, lz4errors.ErrInvalidHeaderChecksum) {
		t.Fatalf("deepWrapped errors.Is failed")
	}

	// Negative match
	if errors.Is(deepWrapped, lz4errors.ErrInvalidBlockChecksum) {
		t.Fatalf("negative errors.Is incorrectly matched")
	}

	// errors.As
	var target lz4errors.Error
	if !errors.As(deepWrapped, &target) || target != err {
		t.Fatalf("errors.As failed: got %v, want %v", target, err)
	}
}

func TestLZ4CustomAndEmptyError(t *testing.T) {
	empty := lz4errors.Error("")
	if empty.Error() != "" {
		t.Fatalf("empty error should return empty string")
	}

	custom := lz4errors.Error("lz4: custom message")
	if custom.Error() != "lz4: custom message" {
		t.Fatalf("custom error returned %q", custom.Error())
	}
}

func BenchmarkLZ4ErrorString(b *testing.B) {
	err := lz4errors.ErrInvalidHeaderChecksum
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}
