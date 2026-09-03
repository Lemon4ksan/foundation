// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit

import (
	"context"
	"io"
)

// Command represents an isolated, testable CLI subcommand contract.
type Command interface {
	// Name returns the primary subcommand invocation name (e.g. "ls", "scan", "pack").
	Name() string
	// Aliases returns alternative command names (e.g. "dir" or "list" for "ls").
	Aliases() []string
	// Synopsis returns a one-line summary for help listings.
	Synopsis() string
	// Usage returns detailed syntax instructions.
	Usage() string
	// Run executes the command logic with isolated arguments and output streams.
	Run(ctx context.Context, args []string, stdout, stderr io.Writer) error
}

// SimpleCommand is a lightweight adapter creating a [Command] from functional closures.
type SimpleCommand struct {
	CmdName     string
	CmdAliases  []string
	CmdSynopsis string
	CmdUsage    string
	CmdRun      func(ctx context.Context, args []string, stdout, stderr io.Writer) error
}

func (c *SimpleCommand) Name() string      { return c.CmdName }
func (c *SimpleCommand) Aliases() []string { return c.CmdAliases }
func (c *SimpleCommand) Synopsis() string  { return c.CmdSynopsis }
func (c *SimpleCommand) Usage() string     { return c.CmdUsage }
func (c *SimpleCommand) Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if c.CmdRun != nil {
		return c.CmdRun(ctx, args, stdout, stderr)
	}
	return nil
}
