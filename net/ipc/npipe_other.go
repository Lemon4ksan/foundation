// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows

package ipc

import (
	"net/http"
)

// NewNamedPipeTransport creates a fallback [net/http.RoundTripper] for non-Windows platforms.
// It executes in O(1) time and allocates memory for the returned empty transport.
// It is safe for concurrent use.
func NewNamedPipeTransport(_ string) *http.Transport {
	return &http.Transport{}
}
