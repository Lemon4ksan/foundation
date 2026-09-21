// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mockackhandler

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/quic/internal/protocol"
	"github.com/lemon4ksan/foundation/testing/gomock"
)

func TestMockSentPacketHandler(t *testing.T) {
	ctrl := gomock.NewController(t)

	handler := NewMockSentPacketHandler(ctrl)
	handler.EXPECT().AmplificationAllowance().Return(protocol.ByteCount(1200))

	if allowance := handler.AmplificationAllowance(); allowance != 1200 {
		t.Fatalf("expected allowance 1200, got %d", allowance)
	}
}
