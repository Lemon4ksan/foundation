// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"strings"
)

// CommandGroup organizes subcommands under logical sections in help listings.
type CommandGroup struct {
	Title    string
	Commands []string
}

// App manages the lifecycle, subcommand routing, help formatting, and execution of CLI tools.
type App struct {
	Name          string
	Version       string
	Description   string
	Commands      []Command
	CommandGroups []CommandGroup
	Examples      []string
	DefaultCmd    Command
	Stdout        io.Writer
	Stderr        io.Writer
	cmdMap        map[string]Command
}

// NewApp constructs a new CLI application instance.
func NewApp(name, version, description string, commands ...Command) *App {
	app := &App{
		Name:        name,
		Version:     version,
		Description: description,
		cmdMap:      make(map[string]Command),
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}

	for _, c := range commands {
		app.RegisterCommand(c)
	}

	return app
}

// RegisterCommand registers a command and its aliases.
func (a *App) RegisterCommand(c Command) *App {
	a.Commands = append(a.Commands, c)
	a.cmdMap[c.Name()] = c
	for _, alias := range c.Aliases() {
		a.cmdMap[alias] = c
	}
	return a
}

// SetDefaultCommand sets the default fallback command when no subcommands are provided.
func (a *App) SetDefaultCommand(c Command) *App {
	a.DefaultCmd = c
	return a
}

// RunCommand runs a specific registered command or alias by name.
func (a *App) RunCommand(ctx context.Context, name string, args []string, stdout, stderr io.Writer) error {
	cmd, ok := a.cmdMap[name]
	if !ok {
		return fmt.Errorf("unknown command %q", name)
	}

	return cmd.Run(ctx, args, stdout, stderr)
}

// Run parses command line arguments and executes the appropriate subcommand.
func (a *App) Run(ctx context.Context, args []string) error {
	stdout := a.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	stderr := a.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	if len(args) == 0 {
		if a.DefaultCmd != nil {
			return a.DefaultCmd.Run(ctx, nil, stdout, stderr)
		}

		a.PrintUsage(stderr)
		return nil
	}

	first := args[0]

	switch first {
	case "help", "--help", "-h", "-help":
		if len(args) > 1 {
			subName := args[1]
			if cmd, ok := a.cmdMap[subName]; ok {
				err := cmd.Run(ctx, []string{"-h"}, stdout, stderr)
				if errors.Is(err, flag.ErrHelp) {
					return nil
				}
				return err
			}
			return fmt.Errorf("unknown command %q for help. Run '%s help' for available commands", subName, a.Name)
		}

		a.PrintUsage(stdout)
		return nil

	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "%s version %s %s/%s\n", a.Name, a.Version, runtime.GOOS, runtime.GOARCH)
		return nil
	}

	// Direct match against registered command or alias
	if cmd, ok := a.cmdMap[first]; ok {
		err := cmd.Run(ctx, args[1:], stdout, stderr)
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// Check if flag passed directly without command
	if strings.HasPrefix(first, "-") && a.DefaultCmd != nil {
		return a.DefaultCmd.Run(ctx, args, stdout, stderr)
	}

	return fmt.Errorf("unknown command %q. Run '%s help' for available commands", first, a.Name)
}

// PrintUsage renders the application help screen cleanly and naturally without ASCII table grids or color noise.
func (a *App) PrintUsage(w io.Writer) {
	if a.Description != "" {
		if a.Version != "" && !strings.Contains(a.Description, a.Version) {
			fmt.Fprintf(w, "%s %s — %s\n\n", a.Name, a.Version, a.Description)
		} else {
			fmt.Fprintf(w, "%s\n\n", a.Description)
		}
	} else if a.Version != "" {
		fmt.Fprintf(w, "%s %s\n\n", a.Name, a.Version)
	}

	fmt.Fprintf(w, "Usage:\n  %s <command> [flags] [arguments...]\n\n", a.Name)

	cmdDisplay := func(c Command) string {
		name := c.Name()
		if len(c.Aliases()) > 0 {
			name = fmt.Sprintf("%s, %s", c.Name(), strings.Join(c.Aliases(), ", "))
		}
		return name
	}

	maxLen := 12
	for _, c := range a.Commands {
		d := cmdDisplay(c)
		if len(d) > maxLen && len(d) <= 30 {
			maxLen = len(d)
		}
	}

	if len(a.CommandGroups) > 0 {
		rendered := make(map[string]bool)

		for _, grp := range a.CommandGroups {
			fmt.Fprintf(w, "%s:\n", grp.Title)

			for _, name := range grp.Commands {
				if cmd, ok := a.cmdMap[name]; ok && !rendered[cmd.Name()] {
					rendered[cmd.Name()] = true
					d := cmdDisplay(cmd)
					fmt.Fprintf(w, "  %-*s  %s\n", maxLen, d, cmd.Synopsis())
				}
			}

			fmt.Fprintln(w)
		}

		// Any remaining ungrouped commands
		var remaining []Command
		for _, c := range a.Commands {
			if !rendered[c.Name()] {
				remaining = append(remaining, c)
				rendered[c.Name()] = true
			}
		}

		if len(remaining) > 0 {
			fmt.Fprintf(w, "Additional Commands:\n")
			for _, cmd := range remaining {
				d := cmdDisplay(cmd)
				fmt.Fprintf(w, "  %-*s  %s\n", maxLen, d, cmd.Synopsis())
			}
			fmt.Fprintln(w)
		}
	} else if len(a.Commands) > 0 {
		fmt.Fprintf(w, "Available Commands:\n")

		sortedCmds := make([]Command, len(a.Commands))
		copy(sortedCmds, a.Commands)
		slices.SortFunc(sortedCmds, func(a, b Command) int {
			return strings.Compare(a.Name(), b.Name())
		})

		for _, cmd := range sortedCmds {
			d := cmdDisplay(cmd)
			fmt.Fprintf(w, "  %-*s  %s\n", maxLen, d, cmd.Synopsis())
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Global Flags:\n")
	fmt.Fprintf(w, "  -h, --help       Show help context for any command or %s itself\n", a.Name)
	fmt.Fprintf(w, "  -v, --version    Show application version\n\n")

	if len(a.Examples) > 0 {
		fmt.Fprintf(w, "Quick Start Examples:\n")
		for _, ex := range a.Examples {
			fmt.Fprintf(w, "  %s\n", ex)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintf(w, "Run '%s help <command>' or '%s <command> --help' for detailed documentation.\n", a.Name, a.Name)
}
