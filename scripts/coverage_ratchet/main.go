// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

type pkgStats struct {
	TotalStmts   int
	CoveredStmts int
}

func (s *pkgStats) Coverage() float64 {
	if s.TotalStmts == 0 {
		return 100.0
	}
	return float64(s.CoveredStmts) / float64(s.TotalStmts) * 100.0
}

func parseCoverageProfile(profilePath string) (map[string]*pkgStats, error) {
	f, err := os.Open(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open coverage profile: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	stats := make(map[string]*pkgStats)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "mode:") {
			continue
		}

		// Format: filePath:startLine.startCol,endLine.endCol numStmt count
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}

		filePathParts := strings.Split(fields[0], ":")
		filePath := filePathParts[0]
		pkgPath := path.Dir(filePath)

		numStmt, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}

		st, exists := stats[pkgPath]
		if !exists {
			st = &pkgStats{}
			stats[pkgPath] = st
		}

		st.TotalStmts += numStmt
		if count > 0 {
			st.CoveredStmts += numStmt
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading coverage profile: %w", err)
	}

	return stats, nil
}

func parseBaseline(baselinePath string) (map[string]float64, error) {
	baseline := make(map[string]float64)
	f, err := os.Open(baselinePath)
	if err != nil {
		if os.IsNotExist(err) {
			return baseline, nil
		}
		return nil, fmt.Errorf("failed to open baseline file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pkg := parts[0]
			valStr := strings.TrimSuffix(parts[1], "%")
			val, err := strconv.ParseFloat(valStr, 64)
			if err == nil {
				baseline[pkg] = val
			}
		}
	}

	return baseline, scanner.Err()
}

func writeBaseline(baselinePath string, stats map[string]*pkgStats) error {
	f, err := os.Create(baselinePath)
	if err != nil {
		return fmt.Errorf("failed to create baseline file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	_, _ = w.WriteString("# Foundation Statement Test Coverage Ratchet Baseline\n")
	_, _ = w.WriteString("# Generated automatically. Coverage must never drop below these values.\n")
	_, _ = w.WriteString("# Package                                                            Coverage\n")

	var pkgs []string
	for p := range stats {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)

	for _, p := range pkgs {
		st := stats[p]
		cov := st.Coverage()
		_, _ = fmt.Fprintf(w, "%-66s %.1f%%\n", p, cov)
	}

	return w.Flush()
}

func main() {
	profileFlag := flag.String("profile", "coverage.out", "Path to coverage profile")
	baselineFlag := flag.String("baseline", "scripts/coverage_baseline.txt", "Path to baseline file")
	updateFlag := flag.Bool("update", false, "Update baseline file with current coverage")
	checkFlag := flag.Bool("check", true, "Check current coverage against baseline ratchet")
	minNewPkgCov := flag.Float64("min-new-coverage", 90.0, "Minimum coverage required for brand new packages")
	flag.Parse()

	stats, err := parseCoverageProfile(*profileFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *updateFlag {
		if err := writeBaseline(*baselineFlag, stats); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing baseline: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully updated coverage baseline in %s (%d packages recorded)\n", *baselineFlag, len(stats))
		return
	}

	if *checkFlag {
		baseline, err := parseBaseline(*baselineFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading baseline: %v\n", err)
			os.Exit(1)
		}

		if len(baseline) == 0 {
			fmt.Fprintf(os.Stderr, "Baseline is empty or missing (%s). Run with -update first.\n", *baselineFlag)
			os.Exit(1)
		}

		var regressions []string
		var improvements []string
		var newViolations []string

		var pkgs []string
		for p := range stats {
			pkgs = append(pkgs, p)
		}
		sort.Strings(pkgs)

		for _, p := range pkgs {
			curr := stats[p].Coverage()
			base, found := baseline[p]
			if !found {
				// New package
				if curr < *minNewPkgCov {
					newViolations = append(
						newViolations,
						fmt.Sprintf("%-60s %.1f%% (minimum required: %.1f%%)", p, curr, *minNewPkgCov),
					)
				}
				continue
			}

			// Tolerance of 0.05% for floating point rounding
			if curr < base-0.05 {
				regressions = append(
					regressions,
					fmt.Sprintf("%-60s dropped from %.1f%% to %.1f%% (delta: %+.1f%%)", p, base, curr, curr-base),
				)
			} else if curr > base+0.1 {
				improvements = append(
					improvements,
					fmt.Sprintf("%-60s improved from %.1f%% to %.1f%% (delta: %+.1f%%)", p, base, curr, curr-base),
				)
			}
		}

		if len(regressions) > 0 {
			fmt.Fprintln(os.Stderr, "\n=======================================================")
			fmt.Fprintln(os.Stderr, "FAIL: Statement Coverage Ratchet Regressions Detected:")
			fmt.Fprintln(os.Stderr, "=======================================================")
			for _, r := range regressions {
				fmt.Fprintln(os.Stderr, "  [REGRESSION] "+r)
			}
			fmt.Fprintln(os.Stderr, "\nRule: Package statement coverage must never decrease.")
		}

		if len(newViolations) > 0 {
			fmt.Fprintln(os.Stderr, "\n=======================================================")
			fmt.Fprintln(os.Stderr, "FAIL: New Packages Below Coverage Threshold:")
			fmt.Fprintln(os.Stderr, "=======================================================")
			for _, v := range newViolations {
				fmt.Fprintln(os.Stderr, "  [NEW PACKAGE] "+v)
			}
		}

		if len(regressions) > 0 || len(newViolations) > 0 {
			os.Exit(1)
		}

		if len(improvements) > 0 {
			fmt.Printf(
				"Coverage improved in %d package(s)! Consider running 'go run ./scripts/coverage_ratchet -update' to raise the ratchet.\n",
				len(improvements),
			)
		}
		fmt.Printf("Coverage ratchet check passed cleanly across %d packages!\n", len(stats))
	}
}
