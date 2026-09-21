// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lz4errors_test

import (
	"testing"

	"github.com/lemon4ksan/foundation/codec/compress/lz4/internal/lz4errors"
)

func TestLZ4Errors(t *testing.T) {
	errs := []lz4errors.Error{
		lz4errors.ErrInvalidSourceShortBuffer,
		lz4errors.ErrInvalidFrame,
		lz4errors.ErrInternalUnhandledState,
		lz4errors.ErrInvalidHeaderChecksum,
		lz4errors.ErrInvalidBlockChecksum,
		lz4errors.ErrInvalidFrameChecksum,
		lz4errors.ErrOptionInvalidCompressionLevel,
		lz4errors.ErrOptionClosedOrError,
		lz4errors.ErrOptionInvalidBlockSize,
		lz4errors.ErrOptionNotApplicable,
		lz4errors.ErrWriterNotClosed,
		lz4errors.ErrEndOfStream,
	}

	for _, err := range errs {
		if err.Error() == "" {
			t.Fatalf("empty error string for %v", err)
		}
	}
}
