// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mocks

import (
	"testing"

	"github.com/lemon4ksan/foundation/net/quic/internal/protocol"
	"github.com/lemon4ksan/foundation/testing/gomock"
)

func TestMockSendAlgorithmWithDebugInfos(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockAlgo := NewMockSendAlgorithmWithDebugInfos(ctrl)
	mockAlgo.EXPECT().CanSend(protocol.ByteCount(1024)).Return(true)

	if !mockAlgo.CanSend(1024) {
		t.Fatal("expected CanSend(1024) to return true")
	}
}

func TestMockShortHeaderSealer(t *testing.T) {
	ctrl := gomock.NewController(t)

	sealer := NewMockShortHeaderSealer(ctrl)
	sealer.EXPECT().Overhead().Return(16)

	if o := sealer.Overhead(); o != 16 {
		t.Fatalf("expected overhead 16, got %d", o)
	}
}
