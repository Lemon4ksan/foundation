// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qpack

// StreamSenderDelegate writes unidirectional encoder/decoder control stream data to the peer.
// Direct 1:1 structural translation of Chromium's quiche::StreamSenderDelegate.
type StreamSenderDelegate interface {
	// WriteStreamData writes data on the unidirectional stream.
	WriteStreamData(data []byte)

	// NumBytesBuffered returns the number of bytes buffered due to underlying stream being blocked.
	NumBytesBuffered() uint64
}
