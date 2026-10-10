// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ipc provides utilities for IPC (Inter-Process Communication) over Unix domain sockets and Windows Named Pipes.
package ipc

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"
)

// NewUnixTransport creates an [net/http.RoundTripper] bound to a local Unix domain socket path.
//
// Execution requires O(1) time complexity.
// It allocates memory for the returned transport and dialer configuration.
// The returned transport is safe for concurrent use by multiple goroutines.
func NewUnixTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			return dialer.DialContext(ctx, "unix", socketPath)
		},
		DisableKeepAlives: false,
	}
}

// ParseIPCURI parses a URL carrying "unix://" or "npipe://" scheme,
// extracting the target socket path and relative HTTP request URI.
//
// Execution requires O(N) time complexity where N is the length of rawURL.
// It is zero-allocation and operates strictly on the input string slices.
// The operation is safe for concurrent use.
//
// Example:
//
//	"unix:///var/run/docker.sock/v1.41/containers/json"
//	 -> Socket: "/var/run/docker.sock", RequestURI: "/v1.41/containers/json"
func ParseIPCURI(rawURL string) (socketPath, reqURI string, isIPC bool) {
	if path, ok := strings.CutPrefix(rawURL, "unix://"); ok {
		if idx := strings.Index(path, ".sock"); idx != -1 {
			sockEnd := idx + 5
			return path[:sockEnd], path[sockEnd:], true
		}
	}

	if path, ok := strings.CutPrefix(rawURL, "npipe://"); ok {
		if found := strings.Contains(path, "/pipe/"); found {
			return path, "/", true
		}
	}

	return "", rawURL, false
}
