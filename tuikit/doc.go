// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tuikit provides a zero-dependency terminal UI toolkit for building CLI applications,
// REPLs, formatted tables, styled borders, badges, and progress indicators.
//
// It includes a CLI application framework ([App]) with subcommand routing and help generation,
// a terminal capability sniffer ([ProbeTerminal]), formatted auto-aligned tables ([Table]),
// bordered boxes ([Box]), and interactive indicators like progress bars and badges.
//
// # Compared to the standard library
//
// Stdlib counterpart: flag
//
// Rejected compromise: The flag package provides no subcommand routing, styling, or TTY detection.
//
// Accepted cost: A larger surface area for simple scripts that only need basic flags.
//
// Allocations: unconstrained
package tuikit
