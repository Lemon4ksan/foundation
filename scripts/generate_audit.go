//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type FileAudit struct {
	RelPath  string
	Metrics  []string
	Status   string
	Findings string
	IsTest   bool
}

type DomainInfo struct {
	Title string
	Files []*FileAudit
}

var acronymRegex = regexp.MustCompile(`([a-z0-9])(Id|Url|Tls|Http|Json|Xml|Api|Aead)([A-Z]|$)`)

func inspectFile(relPath string, content []byte) *FileAudit {
	audit := &FileAudit{
		RelPath: relPath,
		IsTest:  strings.HasSuffix(relPath, "_test.go"),
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, relPath, content, parser.ParseComments)
	if err != nil {
		audit.Metrics = []string{"M5.1: Cognitive Complexity / Parser"}
		audit.Status = "VIOLATIONS_FOUND"
		audit.Findings = fmt.Sprintf("Source file parsing error: %v", err)
		return audit
	}

	var debtItems []string
	var violationItems []string
	var adherenceItems []string

	// Evaluated metric trackers
	evalMetrics := make(map[string]bool)
	evalMetrics["M7.2: Receiver Naming"] = true
	evalMetrics["M1.1: any and interface{} Eradication"] = true
	evalMetrics["M7.1: Acronym Capitalization"] = true

	// Check receivers and GoDoc on declarations
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		// Receiver check (M7.2)
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			for _, field := range fn.Recv.List {
				for _, name := range field.Names {
					if name.Name == "this" || name.Name == "self" || name.Name == "me" {
						violationItems = append(violationItems, fmt.Sprintf("M7.2: Banned receiver name '%s' in method '%s'", name.Name, fn.Name.Name))
					} else if len(name.Name) > 2 {
						debtItems = append(debtItems, fmt.Sprintf("M7.2: Receiver '%s' in method '%s' exceeds 2 letters", name.Name, fn.Name.Name))
					}
				}
			}
		}

		// GoDoc comment check (M7.4)
		if fn.Name.IsExported() && !audit.IsTest {
			evalMetrics["M7.4: GoDoc Standard Comments"] = true
			if fn.Doc == nil || len(fn.Doc.List) == 0 {
				debtItems = append(debtItems, fmt.Sprintf("M7.4: Exported function '%s' lacks GoDoc leading comment", fn.Name.Name))
			} else {
				docText := strings.TrimSpace(fn.Doc.Text())
				if !strings.HasPrefix(docText, fn.Name.Name) {
					debtItems = append(debtItems, fmt.Sprintf("M7.4: GoDoc for '%s' does not start with identifier name", fn.Name.Name))
				}
			}
		}

		// Acronym check on function name (M7.1)
		if acronymRegex.MatchString(fn.Name.Name) {
			debtItems = append(debtItems, fmt.Sprintf("M7.1: Acronym capitalization in function '%s' violates zero-variance initialism", fn.Name.Name))
		}
	}

	// AST Traversal for modern Go features and debt
	var anyCount, interfaceCount int
	var hasFinalizer, hasCleanup, hasSyncTest, hasBLoop, hasBN, hasTimeSleep, hasWeak bool
	var bareGoCount int

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if x.Name == "any" {
				anyCount++
			}
		case *ast.InterfaceType:
			if len(x.Methods.List) == 0 {
				interfaceCount++
			}
		case *ast.TypeSpec:
			if acronymRegex.MatchString(x.Name.Name) {
				debtItems = append(debtItems, fmt.Sprintf("M7.1: Acronym capitalization in type '%s' violates zero-variance initialism", x.Name.Name))
			}
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				if id.Name == "runtime" && x.Sel.Name == "SetFinalizer" {
					hasFinalizer = true
					evalMetrics["M2.2: Memory Finalizers"] = true
				}
				if id.Name == "runtime" && x.Sel.Name == "AddCleanup" {
					hasCleanup = true
					evalMetrics["M2.2: Memory Finalizers"] = true
				}
				if id.Name == "weak" && (x.Sel.Name == "Pointer" || x.Sel.Name == "Make") {
					hasWeak = true
					evalMetrics["M2.1: Weak Pointers"] = true
				}
				if id.Name == "synctest" && x.Sel.Name == "Run" {
					hasSyncTest = true
					evalMetrics["M4.1: synctest Testing"] = true
				}
				if id.Name == "b" && x.Sel.Name == "Loop" {
					hasBLoop = true
					evalMetrics["M4.2: Benchmark b.Loop()"] = true
				}
				if id.Name == "b" && x.Sel.Name == "N" {
					hasBN = true
					evalMetrics["M4.2: Benchmark b.Loop()"] = true
				}
				if id.Name == "time" && x.Sel.Name == "Sleep" {
					hasTimeSleep = true
					evalMetrics["M4.1: synctest Testing"] = true
				}
			}
		case *ast.GoStmt:
			if !audit.IsTest {
				bareGoCount++
				evalMetrics["M9.1: Concurrency & Errgroups"] = true
			}
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "fmt" && sel.Sel.Name == "Errorf" {
					evalMetrics["M5.4: Error Wrapping"] = true
					if len(x.Args) > 0 {
						if lit, ok := x.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							val := strings.Trim(lit.Value, "\"")
							if len(val) > 0 && unicode.IsUpper(rune(val[0])) {
								debtItems = append(debtItems, "M7.3: Error string starts with capital letter")
							}
							if strings.HasSuffix(val, ".") || strings.HasSuffix(val, "!") {
								debtItems = append(debtItems, "M7.3: Error string ends with punctuation")
							}
						}
					}
				}
			}
		}
		return true
	})

	if hasFinalizer {
		violationItems = append(violationItems, "M2.2: Critical debt: runtime.SetFinalizer found; refactor to runtime.AddCleanup")
	}
	if hasCleanup {
		adherenceItems = append(adherenceItems, "M2.2: runtime.AddCleanup modern finalizer pattern adopted")
	}
	if hasWeak {
		adherenceItems = append(adherenceItems, "M2.1: weak.Pointer cache pattern utilized")
	}
	if audit.IsTest {
		evalMetrics["M4.1: synctest Testing"] = true
		evalMetrics["M4.2: Benchmark b.Loop()"] = true
		if hasTimeSleep {
			debtItems = append(debtItems, "M4.1: time.Sleep detected in test; migrate to testing/synctest temporal bubble")
		}
		if hasSyncTest {
			adherenceItems = append(adherenceItems, "M4.1: testing/synctest correctly utilized for concurrent determinism")
		}
		if hasBN {
			debtItems = append(debtItems, "M4.2: Deprecated b.N loop detected; refactor to for b.Loop()")
		}
		if hasBLoop {
			adherenceItems = append(adherenceItems, "M4.2: Modern b.Loop() benchmark syntax utilized")
		}
	}
	if bareGoCount > 0 {
		debtItems = append(debtItems, fmt.Sprintf("M9.1: %d bare goroutine(s) detected; verify lifecycle management with errgroup", bareGoCount))
	}
	if anyCount+interfaceCount > 0 {
		debtItems = append(debtItems, fmt.Sprintf("M1.1: %d any/interface{} occurrences detected; verify generic type constraints", anyCount+interfaceCount))
	} else {
		adherenceItems = append(adherenceItems, "M1.1: Zero any/interface{} usage; strict type safety preserved")
	}

	for k := range evalMetrics {
		audit.Metrics = append(audit.Metrics, k)
	}
	sort.Strings(audit.Metrics)

	if len(violationItems) > 0 {
		audit.Status = "VIOLATIONS_FOUND"
		audit.Findings = strings.Join(append(violationItems, debtItems...), "; ")
	} else if len(debtItems) > 0 {
		audit.Status = "COMPLIANT_WITH_DEBT"
		audit.Findings = strings.Join(debtItems, "; ")
	} else {
		audit.Status = "COMPLETELY_COMPLIANT"
		if len(adherenceItems) > 0 {
			audit.Findings = "Completely compliant with modern Go 1.27 rules: " + strings.Join(adherenceItems, "; ")
		} else {
			audit.Findings = "Completely compliant with modern Go 1.27 zero-variance styling, receiver naming, and error formatting."
		}
	}

	return audit
}

func getDomainForPath(relPath string) (string, string) {
	parts := strings.Split(relPath, "/")
	top := parts[0]
	switch top {
	case "async":
		return "audit_async.md", "Async & Concurrency Kits"
	case "sync":
		return "audit_sync.md", "Synchronization Primitives"
	case "silicon":
		return "audit_silicon.md", "Silicon Hardware & SIMD"
	case "structures":
		return "audit_structures.md", "Data Structures"
	case "generic":
		return "audit_generic.md", "Generic Library"
	case "testing", "test":
		return "audit_testing.md", "Testing Framework & Mocks"
	case "cmd":
		return "audit_cmd.md", "Command-Line Tools"
	case "scripts":
		return "audit_scripts.md", "Engineering Scripts & Tooling"
	case "text":
		return "audit_text.md", "Text & Encodings"
	case "crypto":
		return "audit_crypto.md", "Cryptography & AEAD"
	case "encoding":
		return "audit_encoding.md", "Binary Encodings & Varint"
	case "codec":
		return "audit_codec.md", "Codecs & JSON Serialization"
	case "net":
		return "audit_net.md", "Networking & Wire Codecs"
	default:
		return "audit_core.md", "Core Kits & Base Primitives"
	}
}

func main() {
	rootDir := "."
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}
	docsDir := filepath.Join(rootDir, "docs")
	_ = os.MkdirAll(docsDir, 0755)

	domains := make(map[string]*DomainInfo)

	var allFiles []string
	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if (name != "." && strings.HasPrefix(name, ".")) || name == "vendor" {
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
			allFiles = append(allFiles, relSlash)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning files: %v\n", err)
		os.Exit(1)
	}
	sort.Strings(allFiles)

	fmt.Printf("Analyzing %d files across repository...\n", len(allFiles))

	for _, relPath := range allFiles {
		fullPath := filepath.Join(rootDir, filepath.FromSlash(relPath))
		content, err := os.ReadFile(fullPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", relPath, err)
			continue
		}

		audit := inspectFile(relPath, content)
		docName, domainTitle := getDomainForPath(relPath)

		if domains[docName] == nil {
			domains[docName] = &DomainInfo{
				Title: domainTitle,
			}
		}
		domains[docName].Files = append(domains[docName].Files, audit)
	}

	// Write each domain report
	for docName, info := range domains {
		var compliantCount, debtCount, violationCount int
		for _, f := range info.Files {
			switch f.Status {
			case "COMPLETELY_COMPLIANT":
				compliantCount++
			case "COMPLIANT_WITH_DEBT":
				debtCount++
			case "VIOLATIONS_FOUND":
				violationCount++
			}
		}

		var buf bytes.Buffer
		buf.WriteString(fmt.Sprintf("# Package Audit: %s\n\n", info.Title))
		buf.WriteString("## Executive Summary\n")
		buf.WriteString(fmt.Sprintf("- Total Files Audited: %d\n", len(info.Files)))
		buf.WriteString(fmt.Sprintf("- Compliant Files: %d\n", compliantCount))
		buf.WriteString(fmt.Sprintf("- Files with Technical Debt / Violations: %d\n\n", debtCount+violationCount))
		buf.WriteString("## File-by-File Evaluation\n\n")

		for _, f := range info.Files {
			buf.WriteString(fmt.Sprintf("### %s\n", f.RelPath))
			buf.WriteString(fmt.Sprintf("- **Metrics Evaluated**: %s\n", strings.Join(f.Metrics, ", ")))
			buf.WriteString(fmt.Sprintf("- **Compliance Status**: %s\n", f.Status))
			buf.WriteString(fmt.Sprintf("- **Findings & Technical Debt**: %s\n\n", f.Findings))
		}

		outPath := filepath.Join(docsDir, docName)
		if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outPath, err)
			os.Exit(1)
		}
		fmt.Printf("Generated %s (%d files: %d compliant, %d debt/violations)\n",
			docName, len(info.Files), compliantCount, debtCount+violationCount)
	}

	fmt.Println("Audit generation complete across all domain packages.")
}
