// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package argkit

import (
	"flag"
	"fmt"
	"strings"
	"time"
)

type boolFlag interface {
	IsBoolFlag() bool
}

// StringSliceFlag implements [flag.Value] for multi-valued command line arguments.
type StringSliceFlag []string

func (s *StringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *StringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// NormalizeArgs stitches back arguments that were fragmented by shell tokenizers
// (e.g. PowerShell or CMD splitting -out=pkg/api and .go).
func NormalizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}

	result := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		for i+1 < len(args) && strings.HasPrefix(args[i+1], ".") && !strings.Contains(args[i+1], "/") && !strings.Contains(args[i+1], "\\") {
			// Orphaned suffix like .go, .json, .har, .yaml
			arg = strings.Join([]string{arg, args[i+1]}, "")
			i++
		}

		result = append(result, arg)
	}

	return result
}

// ParseInterspersedFlags parses a [flag.FlagSet] correctly with full POSIX semantics:
// - Free interspersing of positional arguments and flags anywhere in the command line.
// - Short flag clumping / stacking (e.g. `-la` -> `-l -a`, `-lAF` -> `-l -A -F`).
// - Attached values for short flags (e.g. `-I*.tmp` -> `-I=*.tmp`).
// - Strict POSIX terminator `--` (all subsequent arguments treated as literal positionals).
// - Fuzzy suggestions ("Did you mean: --flag?") for mistyped options.
func ParseInterspersedFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	args = NormalizeArgs(args)

	// Step 1: Pre-process and decompose clumped short flags or attached values
	expanded := make([]string, 0, len(args))
	hitDoubleDash := false

	for _, arg := range args {
		if hitDoubleDash {
			expanded = append(expanded, arg)
			continue
		}
		if arg == "--" {
			hitDoubleDash = true
			expanded = append(expanded, arg)
			continue
		}

		if clumped, ok := decomposeFlag(fs, arg); ok {
			expanded = append(expanded, clumped...)
		} else {
			expanded = append(expanded, arg)
		}
	}

	args = expanded

	// Step 2: Separate flags from positional arguments while respecting value consumption
	var (
		flagArgs []string
		posArgs  []string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			posArgs = append(posArgs, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "-") && arg != "-" {
			cleanArg := strings.TrimLeft(arg, "-")
			flagName := cleanArg
			hasEqual := false

			if eqIdx := strings.Index(cleanArg, "="); eqIdx != -1 {
				flagName = cleanArg[:eqIdx]
				hasEqual = true
			}

			fl := fs.Lookup(flagName)
			if fl == nil || hasEqual {
				flagArgs = append(flagArgs, arg)
				continue
			}

			// Check if boolean flag
			if bf, ok := fl.Value.(boolFlag); ok && bf.IsBoolFlag() {
				flagArgs = append(flagArgs, arg)
			} else {
				flagArgs = append(flagArgs, arg)
				// Consume next argument as flag value if available and not another flag
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					flagArgs = append(flagArgs, args[i])
				}
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "flag provided but not defined: ") {
			parts := strings.Split(msg, "flag provided but not defined: ")
			if len(parts) == 2 {
				badFlag := strings.TrimSpace(parts[1])
				if match := findClosestFlag(fs, badFlag); match != "" {
					return nil, fmt.Errorf("flag provided but not defined: %s (did you mean: --%s?)", badFlag, match)
				}
			}
		}
		return nil, err
	}

	return posArgs, nil
}

// decomposeFlag unpacks POSIX short flag clumps (e.g. -la -> [-l, -a]) and attached values (-I*.tmp -> [-I=*.tmp]).
func decomposeFlag(fs *flag.FlagSet, arg string) ([]string, bool) {
	// Must start with single '-' and not '--'
	if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") || arg == "-" {
		return nil, false
	}

	clean := arg[1:]
	if len(clean) <= 1 {
		return nil, false
	}

	// If the entire string is already registered (e.g. -where, -all, -sort), keep as is
	flagName := clean
	if eqIdx := strings.Index(clean, "="); eqIdx != -1 {
		flagName = clean[:eqIdx]
	}
	if fs.Lookup(flagName) != nil {
		return nil, false
	}

	var result []string
	curr := clean

	for len(curr) > 0 {
		firstChar := string(curr[0])
		fl := fs.Lookup(firstChar)
		if fl == nil {
			// First character is not a known flag, cannot decompose
			return nil, false
		}

		if bf, ok := fl.Value.(boolFlag); ok && bf.IsBoolFlag() {
			result = append(result, "-"+firstChar)
			curr = curr[1:]
			if strings.HasPrefix(curr, "=") {
				return nil, false
			}
		} else {
			// Non-boolean flag: rest of current string is an attached value
			rest := curr[1:]
			if strings.HasPrefix(rest, "=") {
				rest = rest[1:]
			}
			result = append(result, fmt.Sprintf("-%s=%s", firstChar, rest))
			curr = ""
		}
	}

	return result, true
}

// findClosestFlag calculates Levenshtein distance to suggest the closest defined flag for typos.
func findClosestFlag(fs *flag.FlagSet, unknown string) string {
	clean := strings.TrimLeft(unknown, "-")
	clean = strings.Split(clean, "=")[0]
	if len(clean) == 0 {
		return ""
	}

	bestMatch := ""
	minDist := 3 // max allowed edit distance threshold

	fs.VisitAll(func(fl *flag.Flag) {
		dist := levenshtein(strings.ToLower(clean), strings.ToLower(fl.Name))
		if dist < minDist {
			minDist = dist
			bestMatch = fl.Name
		}
	})

	return bestMatch
}

func levenshtein(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	n1, n2 := len(r1), len(r2)
	if n1 == 0 {
		return n2
	}
	if n2 == 0 {
		return n1
	}

	row := make([]int, n2+1)
	for j := 0; j <= n2; j++ {
		row[j] = j
	}

	for i := 1; i <= n1; i++ {
		prev := i
		for j := 1; j <= n2; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			cur := row[j] + 1
			if prev+1 < cur {
				cur = prev + 1
			}
			if row[j-1]+cost < cur {
				cur = row[j-1] + cost
			}
			row[j-1] = prev
			prev = cur
		}
		row[n2] = prev
	}
	return row[n2]
}

// StringVar binds a string flag with optional short alias.
func StringVar(fs *flag.FlagSet, p *string, name, short, value, usage string) {
	fs.StringVar(p, name, value, usage)

	if short != "" && short != name {
		fs.StringVar(p, short, value, usage)
	}
}

// BoolVar binds a boolean flag with optional short alias.
func BoolVar(fs *flag.FlagSet, p *bool, name, short string, value bool, usage string) {
	fs.BoolVar(p, name, value, usage)

	if short != "" && short != name {
		fs.BoolVar(p, short, value, usage)
	}
}

// IntVar binds an integer flag with optional short alias.
func IntVar(fs *flag.FlagSet, p *int, name, short string, value int, usage string) {
	fs.IntVar(p, name, value, usage)

	if short != "" && short != name {
		fs.IntVar(p, short, value, usage)
	}
}

// Int64Var binds an int64 flag with optional short alias.
func Int64Var(fs *flag.FlagSet, p *int64, name, short string, value int64, usage string) {
	fs.Int64Var(p, name, value, usage)

	if short != "" && short != name {
		fs.Int64Var(p, short, value, usage)
	}
}

// Float64Var binds a float64 flag with optional short alias.
func Float64Var(fs *flag.FlagSet, p *float64, name, short string, value float64, usage string) {
	fs.Float64Var(p, name, value, usage)

	if short != "" && short != name {
		fs.Float64Var(p, short, value, usage)
	}
}

// DurationVar binds a time.Duration flag with optional short alias.
func DurationVar(fs *flag.FlagSet, p *time.Duration, name, short string, value time.Duration, usage string) {
	fs.DurationVar(p, name, value, usage)

	if short != "" && short != name {
		fs.DurationVar(p, short, value, usage)
	}
}
