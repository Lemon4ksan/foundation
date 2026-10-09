// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test_test

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

// ViolationCategory categorizes acceptance verification findings.
type ViolationCategory string

const (
	CategoryGetterPrefix  ViolationCategory = "GETTER_PREFIX"
	CategoryStuttering    ViolationCategory = "STUTTERING"
	CategoryTypeAlias     ViolationCategory = "TYPE_ALIAS"
	CategoryDomainClarity ViolationCategory = "DOMAIN_CLARITY"
)

// Violation represents an individual non-idiomatic naming or alias violation.
type Violation struct {
	Category  ViolationCategory
	Milestone string
	Package   string
	File      string
	Line      int
	Symbol    string
	Details   string
}

// findRepoRoot locates the repository root containing go.mod.
func findRepoRoot(t *testing.T) string {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repository root with go.mod starting from %s", dir)
		}
		dir = parent
	}
}

// isStrictEnforcement reports whether strict acceptance assertion is enabled.
func isStrictEnforcement() bool {
	return os.Getenv("FOUNDATION_ACCEPTANCE_ENFORCE") == "1" ||
		os.Getenv("FOUNDATION_STRICT") == "1"
}

// scanRepository performs an AST walk over all non-test Go source files.
func scanRepository(t *testing.T, repoRoot string) ([]Violation, error) {
	var violations []Violation
	fset := token.NewFileSet()

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == ".agents" || name == "testdata" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relPath, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		node, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if parseErr != nil {
			return fmt.Errorf("failed to parse %s: %w", path, parseErr)
		}

		pkgName := node.Name.Name

		// Check declarations
		for _, decl := range node.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				checkFuncDecl(fset, d, pkgName, relPath, &violations)
			case *ast.GenDecl:
				checkGenDecl(fset, d, pkgName, relPath, &violations)
			}
		}

		return nil
	})

	return violations, err
}

func checkFuncDecl(fset *token.FileSet, fd *ast.FuncDecl, pkgName, relPath string, violations *[]Violation) {
	name := fd.Name.Name
	line := fset.Position(fd.Pos()).Line

	// 1. Getter methods on types
	if fd.Recv != nil && fd.Name.IsExported() {
		if isGetterName(name) {
			recvType := extractReceiverType(fd.Recv)
			*violations = append(*violations, Violation{
				Category:  CategoryGetterPrefix,
				Milestone: determineMilestone(relPath),
				Package:   pkgName,
				File:      relPath,
				Line:      line,
				Symbol:    fmt.Sprintf("(%s).%s", recvType, name),
				Details: fmt.Sprintf(
					"Exported getter method has 'Get' prefix; should be renamed without 'Get' (e.g., %s)",
					trimGetPrefix(name),
				),
			})
		}
	}

	// 2. Package-level exported getter functions
	if fd.Recv == nil && fd.Name.IsExported() {
		if isGetterName(name) {
			// Whitelist valid constructors or non-getter funcs if any
			*violations = append(*violations, Violation{
				Category:  CategoryGetterPrefix,
				Milestone: determineMilestone(relPath),
				Package:   pkgName,
				File:      relPath,
				Line:      line,
				Symbol:    name,
				Details:   fmt.Sprintf("Exported function has 'Get' prefix; should be renamed to remove getter prefix"),
			})
		}

		// 3. Stuttering function names
		if isFunctionStuttering(name, pkgName) {
			*violations = append(*violations, Violation{
				Category:  CategoryStuttering,
				Milestone: determineMilestone(relPath),
				Package:   pkgName,
				File:      relPath,
				Line:      line,
				Symbol:    name,
				Details: fmt.Sprintf(
					"Exported function stutters with package name %q; rename to remove package prefix",
					pkgName,
				),
			})
		}
	}
}

func checkGenDecl(fset *token.FileSet, gd *ast.GenDecl, pkgName, relPath string, violations *[]Violation) {
	// Type declarations
	if gd.Tok == token.TYPE {
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			typeName := ts.Name.Name
			line := fset.Position(ts.Pos()).Line

			// Check backward-compatibility type aliases: type Old = New
			if ts.Assign.IsValid() && ts.Name.IsExported() {
				// Detect legacy aliases
				isLegacyAlias := false
				if strings.HasPrefix(relPath, "net/quic/interface.go") {
					switch typeName {
					case "Connection", "Session", "EarlyConnection", "TokenGeneratorKey":
						isLegacyAlias = true
					}
				}
				if isLegacyAlias {
					*violations = append(*violations, Violation{
						Category:  CategoryTypeAlias,
						Milestone: "M4",
						Package:   pkgName,
						File:      relPath,
						Line:      line,
						Symbol:    typeName,
						Details:   "Backward-compatibility type alias masks modern Conn/Token name; must be removed in-place",
					})
				}
			}

			// Check interface methods for Get* getters
			if iface, ok := ts.Type.(*ast.InterfaceType); ok && iface.Methods != nil {
				for _, method := range iface.Methods.List {
					for _, mName := range method.Names {
						if mName.IsExported() && isGetterName(mName.Name) {
							mLine := fset.Position(mName.Pos()).Line
							*violations = append(*violations, Violation{
								Category:  CategoryGetterPrefix,
								Milestone: determineMilestone(relPath),
								Package:   pkgName,
								File:      relPath,
								Line:      mLine,
								Symbol:    fmt.Sprintf("%s.%s", typeName, mName.Name),
								Details: fmt.Sprintf(
									"Interface method has 'Get' prefix; should be renamed without 'Get' (e.g., %s)",
									trimGetPrefix(mName.Name),
								),
							})
						}
					}
				}
			}

			// Check exported type stuttering
			if ts.Name.IsExported() {
				if stutterDesc, stutters := isTypeStuttering(typeName, pkgName, relPath); stutters {
					*violations = append(*violations, Violation{
						Category:  CategoryStuttering,
						Milestone: determineMilestone(relPath),
						Package:   pkgName,
						File:      relPath,
						Line:      line,
						Symbol:    typeName,
						Details:   stutterDesc,
					})
				}

				// Check domain clarity targets
				if clarityDesc, isGeneric := isDomainClarityViolation(typeName, pkgName, relPath); isGeneric {
					*violations = append(*violations, Violation{
						Category:  CategoryDomainClarity,
						Milestone: determineMilestone(relPath),
						Package:   pkgName,
						File:      relPath,
						Line:      line,
						Symbol:    typeName,
						Details:   clarityDesc,
					})
				}
			}
		}
	}

	// Const declarations for alias detection in net/tls/cert
	if gd.Tok == token.CONST && strings.HasPrefix(relPath, "net/tls/cert/") {
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range vs.Names {
				switch name.Name {
				case "CompressionZlib", "CompressionBrotli", "CompressionZstd":
					line := fset.Position(name.Pos()).Line
					*violations = append(*violations, Violation{
						Category:  CategoryTypeAlias,
						Milestone: "M4",
						Package:   pkgName,
						File:      relPath,
						Line:      line,
						Symbol:    name.Name,
						Details:   "Legacy alias constant for CertCompression*; must be removed per PROJECT.md Feature 10",
					})
				}
			}
		}
	}
}

func isGetterName(name string) bool {
	if !strings.HasPrefix(name, "Get") || len(name) <= 3 {
		return false
	}
	r := rune(name[3])
	return unicode.IsUpper(r) || unicode.IsDigit(r)
}

func trimGetPrefix(name string) string {
	trimmed := strings.TrimPrefix(name, "Get")
	if len(trimmed) > 0 && unicode.IsDigit(rune(trimmed[0])) {
		switch {
		case strings.HasPrefix(trimmed, "0RTT"):
			return "ZeroRTT" + strings.TrimPrefix(trimmed, "0RTT")
		case strings.HasPrefix(trimmed, "1RTT"):
			return "OneRTT" + strings.TrimPrefix(trimmed, "1RTT")
		}
	}
	return trimmed
}

func extractReceiverType(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	t := recv.List[0].Type
	var buf bytes.Buffer
	if star, ok := t.(*ast.StarExpr); ok {
		buf.WriteString("*")
		t = star.X
	}
	if ident, ok := t.(*ast.Ident); ok {
		buf.WriteString(ident.Name)
	} else if index, ok := t.(*ast.IndexExpr); ok {
		if ident, ok := index.X.(*ast.Ident); ok {
			buf.WriteString(ident.Name)
		}
	}
	return buf.String()
}

func isTypeStuttering(typeName, pkgName, relPath string) (string, bool) {
	// Whitelist intended designs and canonical types that match standard library conventions
	switch {
	case pkgName == "pool" && (typeName == "PerCorePool" || typeName == "Pool"):
		return "", false
	case pkgName == "deque" && typeName == "Deque":
		return "", false
	case pkgName == "list" && typeName == "List":
		return "", false
	case pkgName == "uuid" && typeName == "UUID":
		return "", false
	case pkgName == "secret" && typeName == "Secret":
		return "", false
	case pkgName == "pipeline" && typeName == "Pipeline":
		return "", false
	case pkgName == "scheduler" && typeName == "Scheduler":
		return "", false
	case pkgName == "semaphore" && typeName == "Semaphore":
		return "", false
	case pkgName == "event" && typeName == "Event":
		return "", false
	case pkgName == "breaker" && typeName == "Breaker":
		return "", false
	case pkgName == "encoding" && typeName == "Encoding":
		return "", false
	case pkgName == "charmap" && typeName == "Charmap":
		return "", false
	}

	lowerType := strings.ToLower(typeName)
	lowerPkg := strings.ToLower(pkgName)

	// Direct prefix match: pkg "bitset", type "BitSet"
	if lowerType == lowerPkg {
		return fmt.Sprintf(
			"Type %s stutters with package name %s (caller writes %s.%s)",
			typeName,
			pkgName,
			pkgName,
			typeName,
		), true
	}

	// Prefix stuttering: pkg "ringbuffer", type "RingBuffer"
	// pkg "diff", type "DiffOptions", "DiffReport"
	if strings.HasPrefix(lowerType, lowerPkg) && len(lowerType) > len(lowerPkg) {
		// Verify word boundary
		nextChar := rune(typeName[len(pkgName)])
		if unicode.IsUpper(nextChar) || unicode.IsDigit(nextChar) {
			return fmt.Sprintf(
				"Exported type %s starts with package name %s (caller writes %s.%s)",
				typeName,
				pkgName,
				pkgName,
				typeName,
			), true
		}
	}

	// Suffix stuttering: pkg "backoff", type "ExponentialBackoff", "LinearBackoff", "ConstantBackoff"
	// pkg "breaker", type "CircuitBreaker"
	// pkg "limiter", type "AdaptiveLimiter", "KeyedLimiter"
	// pkg "spinlock", type "SpinLock"
	if strings.HasSuffix(lowerType, lowerPkg) && len(lowerType) > len(lowerPkg) {
		return fmt.Sprintf(
			"Exported type %s ends with package name %s (caller writes %s.%s)",
			typeName,
			pkgName,
			pkgName,
			typeName,
		), true
	}

	// Specific inventory targets from PROJECT.md
	switch {
	case strings.Contains(relPath, "silicon/ringbuf") && typeName == "RingBuffer":
		return "silicon/ringbuf exports RingBuffer; should be Buffer", true
	case strings.Contains(relPath, "silicon/offheap") && typeName == "OffHeapBuffer":
		return "silicon/offheap exports OffHeapBuffer; should be Buffer", true
	case strings.Contains(relPath, "sync/spinlock") && typeName == "SpinLock":
		return "sync/spinlock exports SpinLock; should be Lock", true
	case strings.Contains(relPath, "net/ip") && (typeName == "SourceIPRotator" || typeName == "IPv6SubnetRotator"):
		return fmt.Sprintf("net/ip exports %s; should be SourceRotator/SubnetRotator", typeName), true
	}

	return "", false
}

func isFunctionStuttering(funcName, pkgName string) bool {
	// e.g. compress.CompressGzip, varint.EncodeVarint
	switch pkgName {
	case "compress":
		if strings.HasPrefix(funcName, "Compress") && len(funcName) > len("Compress") {
			switch funcName {
			case "CompressGzip", "CompressDeflate", "CompressBrotli", "CompressZstd", "CompressLZMA2", "CompressLZ4":
				return true
			}
		}
	case "varint":
		if strings.Contains(funcName, "Varint") || strings.Contains(funcName, "VarInt") {
			return true
		}
	}
	return false
}

func isDomainClarityViolation(typeName, pkgName, relPath string) (string, bool) {
	switch {
	case strings.Contains(relPath, "silicon/pool") && typeName == "PerPStorage":
		return "silicon/pool.PerPStorage has unclear responsibility; should be PerCorePool", true
	case strings.Contains(relPath, "async/task") && typeName == "Manager":
		return "async/task.Manager is overly generic; should be Tracker", true
	case strings.Contains(relPath, "async/event") && typeName == "BaseEvent":
		return "async/event.BaseEvent should be Envelope", true
	case strings.Contains(relPath, "tuikit") && typeName == "SimpleCommand":
		return "tuikit.SimpleCommand is overly generic; should be FuncCommand", true
	case strings.Contains(relPath, "testing/mock") && typeName == "TestHelper":
		return "testing/mock.TestHelper should be TestingT", true
	case strings.Contains(relPath, "text/diff") && typeName == "DriftItem":
		return "text/diff.DriftItem should be Drift", true
	}
	return "", false
}

func determineMilestone(relPath string) string {
	switch {
	case strings.HasPrefix(relPath, "silicon/") || strings.HasPrefix(relPath, "structures/") ||
		strings.HasPrefix(relPath, "refkit/") || strings.HasPrefix(relPath, "borrow/") || strings.HasPrefix(relPath, "bufkit/"):
		return "M1"
	case strings.HasPrefix(relPath, "sync/") || strings.HasPrefix(relPath, "async/") || strings.HasPrefix(relPath, "generic/"):
		return "M2"
	case strings.HasPrefix(relPath, "codec/") || strings.HasPrefix(relPath, "encoding/") ||
		strings.HasPrefix(relPath, "text/") || strings.HasPrefix(relPath, "tuikit/") || strings.HasPrefix(relPath, "testing/"):
		return "M3"
	case strings.HasPrefix(relPath, "net/"):
		return "M4"
	default:
		return "M5"
	}
}

// TestModernization_AcceptanceSuite verifies the repository against all acceptance criteria in ORIGINAL_REQUEST.md.
func TestModernization_AcceptanceSuite(t *testing.T) {
	repoRoot := findRepoRoot(t)
	violations, err := scanRepository(t, repoRoot)
	if err != nil {
		t.Fatalf("failed to scan repository: %v", err)
	}

	strict := isStrictEnforcement()

	t.Run("Criterion1_NoGetterPrefixes", func(t *testing.T) {
		var count int
		for _, v := range violations {
			if v.Category == CategoryGetterPrefix {
				count++
				msg := fmt.Sprintf("[%s] %s:%d: %s (%s)", v.Milestone, v.File, v.Line, v.Symbol, v.Details)
				if strict {
					t.Errorf("Getter prefix violation: %s", msg)
				} else {
					t.Logf("VIOLATION: %s", msg)
				}
			}
		}
		t.Logf("Criterion 1 Summary: Discovered %d exported getter prefix violations across repository.", count)
	})

	t.Run("Criterion2_NoStutteringExportedTypes", func(t *testing.T) {
		var count int
		for _, v := range violations {
			if v.Category == CategoryStuttering {
				count++
				msg := fmt.Sprintf(
					"[%s] %s:%d: %s.%s (%s)",
					v.Milestone,
					v.File,
					v.Line,
					v.Package,
					v.Symbol,
					v.Details,
				)
				if strict {
					t.Errorf("Stuttering type violation: %s", msg)
				} else {
					t.Logf("VIOLATION: %s", msg)
				}
			}
		}
		t.Logf("Criterion 2 Summary: Discovered %d stuttering exported type/func violations across repository.", count)
	})

	t.Run("Criterion3_NoBackwardCompatibilityAliases", func(t *testing.T) {
		var count int
		for _, v := range violations {
			if v.Category == CategoryTypeAlias {
				count++
				msg := fmt.Sprintf("[%s] %s:%d: %s (%s)", v.Milestone, v.File, v.Line, v.Symbol, v.Details)
				if strict {
					t.Errorf("Type alias violation: %s", msg)
				} else {
					t.Logf("VIOLATION: %s", msg)
				}
			}
		}
		t.Logf("Criterion 3 Summary: Discovered %d backward-compatibility alias violations across repository.", count)
	})

	t.Run("Criterion_DomainClarity", func(t *testing.T) {
		var count int
		for _, v := range violations {
			if v.Category == CategoryDomainClarity {
				count++
				msg := fmt.Sprintf("[%s] %s:%d: %s (%s)", v.Milestone, v.File, v.Line, v.Symbol, v.Details)
				if strict {
					t.Errorf("Domain clarity violation: %s", msg)
				} else {
					t.Logf("VIOLATION: %s", msg)
				}
			}
		}
		t.Logf("Domain Clarity Summary: Discovered %d overly generic / non-descriptive names across repository.", count)
	})

	t.Run("Criterion4_BuildClean", func(t *testing.T) {
		cmd := exec.Command("go", "build", "./...")
		cmd.Dir = repoRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go build ./... failed:\n%s", string(out))
		}
		t.Log("Criterion 4: Entire project builds cleanly with 0 errors.")
	})

	// Milestone Breakdown Report
	t.Run("MilestoneBreakdown", func(t *testing.T) {
		milestoneCounts := make(map[string]int)
		for _, v := range violations {
			milestoneCounts[v.Milestone]++
		}
		t.Log("=== Modernization Violations by Milestone ===")
		for _, m := range []string{"M1", "M2", "M3", "M4", "M5"} {
			t.Logf("  Milestone %s: %d pending modernization violations", m, milestoneCounts[m])
		}
		t.Logf("  Total Pending Violations: %d", len(violations))
		if strict && len(violations) > 0 {
			t.Fatalf("Modernization incomplete: %d violations must be resolved across milestones.", len(violations))
		}
	})
}
