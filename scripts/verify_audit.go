//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// expectedTotalGoFiles is the expected count of .go files in foundation.
const expectedTotalGoFiles = 1101

// metricRegex matches valid metrics or compliance declarations with strict word boundaries.
var metricRegex = regexp.MustCompile(`(?i)(\bM[1-9]\.[1-4]\b|\bG[1-3]\b|\bcompletely[_\s]+compliant\b|\breceiver\s+nam(?:ing|e)\b|\bfinalizer\b|\bcleanup\b|\bweak(?:\.pointer)?\b|\bsynctest\b|\btime\.sleep\b|\bb\.loop\b|\bb\.n\b|\bomitzero\b|\biter\.seq\b|\bsync\.map\b|\bsync\.pool\b|\bos\.root\b|\bbinary\s+tool\b|\bgocognit\b|\bnestif\b|\bcognitive\b|\berror\s+wrapping\b|\berror\s+string\b|%w|\binterface\s+verification\b|\boptions\s+pattern\b|\bfunctional\s+option\b|\ballocation\s+budget\b|\badr-000[238]\b|\bdoc\.go\b|\bany\b|interface\{\}|\bacronym\b|\berrgroup\b|\bgoroutine\b|\bslices\b|\bmaps\b|\bloop\s+variable\b|\ballocation\b|\ballocs/op\b|\bcontext\b|\bgodoc\b|\blayering\b)`)

// Subsection records parsed information about a single file's audit entry.
type Subsection struct {
	FilePath  string
	SourceDoc string
	Content   string
}

// FindAllGoFiles traverses rootDir, skipping hidden directories and vendor.
func FindAllGoFiles(rootDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name != "." && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			rel, err := filepath.Rel(rootDir, path)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)
			if relSlash == "scripts/verify_audit.go" || relSlash == "scripts/generate_audit.go" {
				return nil
			}
			files = append(files, relSlash)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// ParseAuditDocs reads all docs/audit_*.md files and extracts subsections.
func ParseAuditDocs(docsDir string) (map[string]*Subsection, []string, error) {
	matches, err := filepath.Glob(filepath.Join(docsDir, "audit_*.md"))
	if err != nil {
		return nil, nil, err
	}

	subsections := make(map[string]*Subsection)
	var duplicates []string

	for _, docPath := range matches {
		base := filepath.Base(docPath)
		if base == "audit_report_2026.md" {
			continue // Skip legacy statistical report
		}

		file, err := os.Open(docPath)
		if err != nil {
			return nil, nil, err
		}

		scanner := bufio.NewScanner(file)
		var currentPath string
		var currentLines []string

		flushCurrent := func() {
			if currentPath != "" {
				content := strings.Join(currentLines, "\n")
				if _, exists := subsections[currentPath]; exists {
					duplicates = append(duplicates, currentPath)
				} else {
					subsections[currentPath] = &Subsection{
						FilePath:  currentPath,
						SourceDoc: base,
						Content:   content,
					}
				}
			}
			currentPath = ""
			currentLines = nil
		}

		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "### ") {
				flushCurrent()
				heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
				heading = strings.Trim(heading, "`")
				heading = filepath.ToSlash(heading)
				heading = strings.TrimPrefix(heading, "./")
				if strings.HasSuffix(heading, ".go") {
					currentPath = heading
				}
			} else if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
				flushCurrent()
			} else if currentPath != "" {
				currentLines = append(currentLines, line)
			}
		}
		flushCurrent()
		file.Close()
		if err := scanner.Err(); err != nil {
			return nil, nil, err
		}
	}

	return subsections, duplicates, nil
}

// ValidateSubsection checks that the subsection references at least one metric or declares compliance.
func ValidateSubsection(content string) (bool, string) {
	if metricRegex.MatchString(content) {
		return true, ""
	}
	return false, "missing reference to modern_go_v127.md metric or explicit compliance declaration"
}

func main() {
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	docsDir := filepath.Join(rootDir, "docs")
	failed := false

	fmt.Println("=== Foundation Audit Completeness & Quality Verifier ===")
	repoFiles, err := FindAllGoFiles(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering .go files: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("1. Discovered .go files in repository: %d\n", len(repoFiles))
	if len(repoFiles) != expectedTotalGoFiles {
		fmt.Printf("   FAIL: Found %d files, expected exactly %d\n", len(repoFiles), expectedTotalGoFiles)
		failed = true
	}

	subsections, duplicates, err := ParseAuditDocs(docsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing audit docs: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("2. Parsed dedicated file subsections in docs/audit_*.md: %d\n", len(subsections))
	if len(subsections) == 0 {
		fmt.Printf("   FAIL: No audit subsections parsed from %s\n", docsDir)
		failed = true
	}

	repoMap := make(map[string]bool, len(repoFiles))
	for _, f := range repoFiles {
		repoMap[f] = true
	}

	var missing []string
	var invalid []string

	for _, f := range repoFiles {
		sub, found := subsections[f]
		if !found {
			missing = append(missing, f)
			continue
		}
		valid, reason := ValidateSubsection(sub.Content)
		if !valid {
			invalid = append(invalid, fmt.Sprintf("%s (%s): %s", f, sub.SourceDoc, reason))
		}
	}

	var unknown []string
	for p := range subsections {
		if !repoMap[p] {
			unknown = append(unknown, p)
		}
	}
	sort.Strings(unknown)

	fmt.Println("\n=== Verification Results ===")
	fmt.Printf("- Total Repository Files:   %d\n", len(repoFiles))
	fmt.Printf("- Audited Subsections:       %d\n", len(subsections))
	fmt.Printf("- Missing Files:             %d\n", len(missing))
	fmt.Printf("- Invalid Subsections:       %d\n", len(invalid))
	fmt.Printf("- Duplicate Subsections:     %d\n", len(duplicates))
	fmt.Printf("- Unknown Subsections:       %d\n", len(unknown))

	if len(missing) > 0 {
		failed = true
		fmt.Printf("\nFAIL: %d files are missing dedicated audit subsections:\n", len(missing))
		limit := 10
		if len(missing) < limit {
			limit = len(missing)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  - %s\n", missing[i])
		}
		if len(missing) > limit {
			fmt.Printf("  ... and %d more\n", len(missing)-limit)
		}
	}

	if len(invalid) > 0 {
		failed = true
		fmt.Printf("\nFAIL: %d subsections failed quality validation:\n", len(invalid))
		limit := 10
		if len(invalid) < limit {
			limit = len(invalid)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  - %s\n", invalid[i])
		}
		if len(invalid) > limit {
			fmt.Printf("  ... and %d more\n", len(invalid)-limit)
		}
	}

	if len(duplicates) > 0 {
		failed = true
		fmt.Printf("\nFAIL: %d duplicate subsections found across audit reports.\n", len(duplicates))
	}

	if len(unknown) > 0 {
		failed = true
		fmt.Printf("\nFAIL: %d subsections reference non-existent repository files.\n", len(unknown))
	}

	if failed {
		fmt.Println("\nRESULT: FAILED (Completeness and quality criteria not met)")
		os.Exit(1)
	}

	fmt.Printf("\nRESULT: SUCCESS! All %d files have dedicated, verified audit subsections.\n", len(repoFiles))
	os.Exit(0)
}
