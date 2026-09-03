// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit

import (
	"io"
)

// fdOwner is an interface implemented by os.File and custom descriptors that expose their underlying file descriptor.
type fdOwner interface {
	Fd() uintptr
}

// ProbeTerminal reports whether w is attached to an interactive terminal or console device
// without relying on any third-party external dependencies.
func ProbeTerminal(w any) bool {
	if w == nil {
		return false
	}

	if f, ok := w.(fdOwner); ok {
		return ProbeTerminalFd(f.Fd())
	}

	return false
}

// ProbeTerminalFd reports whether the given integer file descriptor is connected to a terminal/console.
func ProbeTerminalFd(fd uintptr) bool {
	return probeFd(fd)
}

// IsInteractive reports whether output sent to w should be treated as interactive human UI.
// It returns true only if w is a terminal and ANSI colors are not explicitly disabled via NO_COLOR.
func IsInteractive(w io.Writer) bool {
	return ProbeTerminal(w) && ColorEnabled()
}
