// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grpcweb_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lemon4ksan/foundation/net/grpcweb"
)

func TestGRPCWeb_Framing_Roundtrip(t *testing.T) {
	t.Parallel()

	framer := grpcweb.NewFramer(0)
	var buf bytes.Buffer

	payload := []byte("hello grpc-web payload")
	n, err := framer.WriteFrame(&buf, 0x00, payload)
	require.NoError(t, err)
	assert.Equal(t, 5+len(payload), n)

	flags, readPayload, err := framer.ReadFrame(&buf)
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), flags)
	assert.Equal(t, payload, readPayload)
}

func TestGRPCWeb_Trailer_Validation(t *testing.T) {
	t.Parallel()

	// OK status (0)
	okTrailer := []byte("grpc-status: 0\r\ngrpc-message: OK\r\n")
	require.NoError(t, grpcweb.VerifyTrailer(okTrailer))

	// Error status (7 = PERMISSION_DENIED)
	errTrailer := []byte("grpc-status: 7\r\ngrpc-message: Permission denied\r\n")
	err := grpcweb.VerifyTrailer(errTrailer)
	require.Error(t, err)

	var grpcErr *grpcweb.GRPCWebError
	require.ErrorAs(t, err, &grpcErr)
	assert.Equal(t, "7", grpcErr.StatusCode)
	assert.Equal(t, "Permission denied", grpcErr.StatusMsg)
}
