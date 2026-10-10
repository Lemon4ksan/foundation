//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"
)

// Category identifiers for cleanliness violations.
const (
	CatForbiddenFinalizer  = "FORBIDDEN_FINALIZER"
	CatReceiverNameTooLong = "RECEIVER_NAME_TOO_LONG"
	CatAcronymCapital      = "ACRONYM_CAPITALIZATION"
	CatErrorStringFormat   = "ERROR_STRING_FORMAT"
	CatBenchmarkUsesBN     = "BENCHMARK_USES_BN"
	CatLoopVarShadowing    = "LOOP_VAR_SHADOWING"
	CatUnclearedAudit      = "UNCLEARED_AUDIT_VIOLATION"
)

// Violation represents an identified issue.
type Violation struct {
	Category string
	FilePath string
	Line     int
	Message  string
}

// AuditViolationEntry tracks clearance status of files marked VIOLATIONS_FOUND in docs/audit_*.md.
type AuditViolationEntry struct {
	FilePath string
	AuditDoc string
	Findings string
	Cleared  bool
	Reason   string
}

// mixedAcronymRegex detects mixed-case acronyms in identifiers: Tls, Url, Http, Json, Xml, Api, Aead, Tcp, Udp.
var mixedAcronymRegex = regexp.MustCompile(`(Tls|Url|Http|Json|Xml|Api|Aead|Tcp|Udp)([A-Z0-9_]|$)`)

// properNouns contains allowed proper nouns or acronyms in error messages.
var properNouns = map[string]bool{
	"Linux":     true,
	"Windows":   true,
	"Darwin":    true,
	"Unix":      true,
	"POSIX":     true,
	"FreeBSD":   true,
	"OpenBSD":   true,
	"NetBSD":    true,
	"ARM":       true,
	"ARM64":     true,
	"AMD64":     true,
	"x86":       true,
	"x86_64":    true,
	"Google":    true,
	"Microsoft": true,
	"Apple":     true,
	"Go":        true,
}

// isAcronymOrProperNoun returns true if word is an all-caps acronym (len >= 2) or recognized proper noun.
func isAcronymOrProperNoun(word string) bool {
	if properNouns[word] {
		return true
	}
	hasLetter := false
	allUpper := true
	for _, r := range word {
		if unicode.IsLetter(r) {
			hasLetter = true
			if !unicode.IsUpper(r) {
				allUpper = false
				break
			}
		}
	}
	return hasLetter && allUpper && len(word) >= 2
}

// checkAcronym returns the mixed-case acronym name if found.
func checkAcronym(ident string) (string, bool) {
	m := mixedAcronymRegex.FindStringSubmatch(ident)
	if len(m) > 1 {
		return m[1], true
	}
	return "", false
}

// containsSelectorN checks if an AST expression references .N (e.g. b.N).
func containsSelectorN(node ast.Node) bool {
	if node == nil {
		return false
	}
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == "N" {
			if _, ok := sel.X.(*ast.Ident); ok {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// FindAllGoFiles traverses rootDir, skipping hidden directories, vendor, and scripts/.
func FindAllGoFiles(rootDir string) ([]string, error) {
	var files []string
	cleanRoot := filepath.Clean(rootDir)
	err := filepath.WalkDir(cleanRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == cleanRoot {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			if name == "vendor" || name == "scripts" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			rel, err := filepath.Rel(cleanRoot, path)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)
			if strings.HasPrefix(relSlash, "scripts/") || relSlash == "scripts" {
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

// InspectFileAST runs all AST checks on a single Go file.
func InspectFileAST(relPath, fullPath string, fset *token.FileSet) ([]Violation, error) {
	node, err := parser.ParseFile(fset, fullPath, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", relPath, err)
	}

	var violations []Violation
	addViolation := func(cat string, pos token.Pos, msg string) {
		position := fset.Position(pos)
		violations = append(violations, Violation{
			Category: cat,
			FilePath: relPath,
			Line:     position.Line,
			Message:  msg,
		})
	}

	isTest := strings.HasSuffix(relPath, "_test.go")
	loopShadowPositions := make(map[token.Pos]bool)

	// Helper to check loop body for variable shadowing (v := v)
	checkLoopBody := func(body *ast.BlockStmt) {
		if body == nil {
			return
		}
		ast.Inspect(body, func(n ast.Node) bool {
			if assign, ok := n.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
				if len(assign.Lhs) == len(assign.Rhs) {
					for i := range assign.Lhs {
						lhs, ok1 := assign.Lhs[i].(*ast.Ident)
						rhs, ok2 := assign.Rhs[i].(*ast.Ident)
						if ok1 && ok2 && lhs.Name == rhs.Name && lhs.Name != "_" {
							pos := assign.Pos()
							if !loopShadowPositions[pos] {
								loopShadowPositions[pos] = true
								addViolation(CatLoopVarShadowing, pos,
									fmt.Sprintf("redundant loop variable shadowing '%s := %s' (obsolete since Go 1.22)", lhs.Name, rhs.Name))
							}
						}
					}
				}
			}
			return true
		})
	}

	// 1. Declarations check (Receivers, Types, Functions, Benchmark loops)
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// M7.2: Receiver Naming
			if d.Recv != nil && len(d.Recv.List) > 0 {
				for _, field := range d.Recv.List {
					for _, name := range field.Names {
						rName := name.Name
						if rName == "this" || rName == "self" || rName == "me" {
							addViolation(CatReceiverNameTooLong, name.Pos(),
								fmt.Sprintf("banned receiver name '%s' in method '%s'", rName, d.Name.Name))
						} else if len(rName) > 2 && rName != "buf" {
							addViolation(CatReceiverNameTooLong, name.Pos(),
								fmt.Sprintf("receiver '%s' in method '%s' exceeds 2 letters (use 1-2 letters or 'buf')", rName, d.Name.Name))
						}
					}
				}
			}

			// M7.1: Acronym check on function name
			if ac, found := checkAcronym(d.Name.Name); found {
				addViolation(CatAcronymCapital, d.Name.Pos(),
					fmt.Sprintf("function/method '%s' contains mixed-case acronym '%s'; should be uppercase", d.Name.Name, ac))
			}

			// M4.2: Benchmark loops (b.N iteration loops)
			if isTest && strings.HasPrefix(d.Name.Name, "Benchmark") && d.Body != nil {
				ast.Inspect(d.Body, func(n ast.Node) bool {
					switch loop := n.(type) {
					case *ast.ForStmt:
						if loop.Cond != nil && containsSelectorN(loop.Cond) {
							addViolation(CatBenchmarkUsesBN, loop.Pos(),
								fmt.Sprintf("benchmark '%s' uses b.N iteration loop; refactor to 'for b.Loop()'", d.Name.Name))
						}
					case *ast.RangeStmt:
						if loop.X != nil && containsSelectorN(loop.X) {
							addViolation(CatBenchmarkUsesBN, loop.Pos(),
								fmt.Sprintf("benchmark '%s' uses range over b.N; refactor to 'for b.Loop()'", d.Name.Name))
						}
					}
					return true
				})
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					// M7.1: Acronym check on TypeSpec name
					if ac, found := checkAcronym(ts.Name.Name); found {
						addViolation(CatAcronymCapital, ts.Name.Pos(),
							fmt.Sprintf("type '%s' contains mixed-case acronym '%s'; should be uppercase", ts.Name.Name, ac))
					}
				}
			}
		}
	}

	// 2. Full AST Walk for Calls, Fields, and Loops
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok {
					// M2.2: Forbidden Finalizer (runtime.SetFinalizer)
					if id.Name == "runtime" && sel.Sel.Name == "SetFinalizer" {
						addViolation(CatForbiddenFinalizer, x.Pos(),
							"runtime.SetFinalizer is forbidden; use runtime.AddCleanup instead")
					}

					// M7.3: Error Strings (fmt.Errorf format strings)
					if id.Name == "fmt" && sel.Sel.Name == "Errorf" && len(x.Args) > 0 {
						if lit, ok := x.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
							val, err := strconv.Unquote(lit.Value)
							if err != nil {
								val = strings.Trim(lit.Value, "\"`")
							}
							if strings.HasSuffix(val, ".") || strings.HasSuffix(val, "!") {
								addViolation(CatErrorStringFormat, lit.Pos(),
									fmt.Sprintf("error format string %q ends with punctuation", val))
							} else if len(val) > 0 {
								firstRune := []rune(val)[0]
								if unicode.IsUpper(firstRune) {
									fields := strings.FieldsFunc(val, func(r rune) bool {
										return r == ' ' || r == ':' || r == '-' || r == '(' || r == ')' || r == '\'' || r == '"' || r == ','
									})
									firstWord := ""
									if len(fields) > 0 {
										firstWord = fields[0]
									}
									if !isAcronymOrProperNoun(firstWord) {
										addViolation(CatErrorStringFormat, lit.Pos(),
											fmt.Sprintf("error format string %q begins with capitalized letter '%c' (not an acronym/proper noun)", val, firstRune))
									}
								}
							}
						}
					}
				}
			}

		case *ast.StructType:
			// M7.1: Acronym check on struct fields
			if x.Fields != nil {
				for _, f := range x.Fields.List {
					for _, name := range f.Names {
						if ac, found := checkAcronym(name.Name); found {
							addViolation(CatAcronymCapital, name.Pos(),
								fmt.Sprintf("struct field '%s' contains mixed-case acronym '%s'; should be uppercase", name.Name, ac))
						}
					}
				}
			}

		case *ast.InterfaceType:
			// M7.1: Acronym check on interface methods
			if x.Methods != nil {
				for _, m := range x.Methods.List {
					for _, name := range m.Names {
						if ac, found := checkAcronym(name.Name); found {
							addViolation(CatAcronymCapital, name.Pos(),
								fmt.Sprintf("interface method '%s' contains mixed-case acronym '%s'; should be uppercase", name.Name, ac))
						}
					}
				}
			}

		case *ast.ForStmt:
			// M1.4: Loop variable shadowing
			checkLoopBody(x.Body)

		case *ast.RangeStmt:
			// M1.4: Loop variable shadowing
			checkLoopBody(x.Body)
		}
		return true
	})

	return violations, nil
}

// CheckAuditClearance reads all docs/audit_*.md files and validates clearance of previously reported violations.
func CheckAuditClearance(docsDir string, repoViolations []Violation) ([]AuditViolationEntry, error) {
	matches, err := filepath.Glob(filepath.Join(docsDir, "audit_*.md"))
	if err != nil {
		return nil, err
	}

	// Index AST violations by relative file path
	fileViolations := make(map[string][]Violation)
	for _, v := range repoViolations {
		fileViolations[v.FilePath] = append(fileViolations[v.FilePath], v)
	}

	var entries []AuditViolationEntry

	for _, docPath := range matches {
		base := filepath.Base(docPath)
		if base == "audit_report_2026.md" {
			continue // skip legacy summary
		}

		f, err := os.Open(docPath)
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(f)
		var currentFile string
		var isViolationFound bool
		var findings string

		flush := func() {
			if currentFile != "" && isViolationFound {
				vList := fileViolations[currentFile]
				cleared := len(vList) == 0
				reason := ""
				if !cleared {
					var msgs []string
					for _, v := range vList {
						msgs = append(msgs, fmt.Sprintf("%s (line %d)", v.Category, v.Line))
					}
					reason = strings.Join(msgs, "; ")
				} else {
					reason = "all reported violations resolved in current AST"
				}

				entries = append(entries, AuditViolationEntry{
					FilePath: currentFile,
					AuditDoc: base,
					Findings: findings,
					Cleared:  cleared,
					Reason:   reason,
				})
			}
			currentFile = ""
			isViolationFound = false
			findings = ""
		}

		for scanner.Scan() {
			line := scanner.Text()
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "### ") {
				flush()
				heading := strings.TrimSpace(strings.TrimPrefix(trimmed, "### "))
				heading = strings.Trim(heading, "`")
				heading = filepath.ToSlash(heading)
				heading = strings.TrimPrefix(heading, "./")
				if strings.HasSuffix(heading, ".go") {
					currentFile = heading
				}
			} else if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
				flush()
			} else if currentFile != "" {
				if strings.HasPrefix(trimmed, "- **Compliance Status**:") {
					if strings.Contains(trimmed, "VIOLATIONS_FOUND") {
						isViolationFound = true
					}
				} else if strings.HasPrefix(trimmed, "- **Findings & Technical Debt**:") {
					findings = strings.TrimSpace(strings.TrimPrefix(trimmed, "- **Findings & Technical Debt**:"))
				}
			}
		}
		flush()
		f.Close()
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FilePath < entries[j].FilePath
	})

	return entries, nil
}

func main() {
	rootDir := "."
	verbose := false

	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
		} else if arg == "-h" || arg == "--help" {
			fmt.Println("Usage: go run scripts/verify_cleanliness.go [-v] [path_to_repo_root]")
			os.Exit(0)
		} else if !strings.HasPrefix(arg, "-") {
			rootDir = arg
		}
	}

	fmt.Println("=== Foundation Cleanliness & Modern Go 1.27 Verifier ===")
	cleanRoot := filepath.Clean(rootDir)
	fmt.Printf("Scanning repository at: %s\n", cleanRoot)

	files, err := FindAllGoFiles(cleanRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering .go files: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d Go source files across repository.\n\n", len(files))

	fset := token.NewFileSet()
	var allViolations []Violation

	for _, relPath := range files {
		fullPath := filepath.Join(cleanRoot, filepath.FromSlash(relPath))
		fileViolations, err := InspectFileAST(relPath, fullPath, fset)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error inspecting %s: %v\n", relPath, err)
			allViolations = append(allViolations, Violation{
				Category: "PARSER_ERROR",
				FilePath: relPath,
				Line:     1,
				Message:  err.Error(),
			})
			continue
		}
		allViolations = append(allViolations, fileViolations...)
	}

	// Audit Clearance Check (Phase 1)
	docsDir := filepath.Join(cleanRoot, "docs")
	auditEntries, err := CheckAuditClearance(docsDir, allViolations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: error reading audit docs in %s: %v\n", docsDir, err)
	}

	// Record uncleared audit entries as UNCLEARED_AUDIT_VIOLATION
	var unclearedCount int
	for _, entry := range auditEntries {
		if !entry.Cleared {
			unclearedCount++
			allViolations = append(allViolations, Violation{
				Category: CatUnclearedAudit,
				FilePath: entry.FilePath,
				Line:     1,
				Message:  fmt.Sprintf("audit doc %s flagged VIOLATIONS_FOUND; still present in AST: %s", entry.AuditDoc, entry.Reason),
			})
		}
	}

	// Print Audit Clearance Status
	if len(auditEntries) > 0 {
		fmt.Println("=== Audit Clearance Check (docs/audit_*.md) ===")
		fmt.Printf("Files previously flagged with VIOLATIONS_FOUND: %d (Cleared: %d, Uncleared: %d)\n",
			len(auditEntries), len(auditEntries)-unclearedCount, unclearedCount)
		for _, entry := range auditEntries {
			if entry.Cleared {
				fmt.Printf("  [CLEARED]   %s (%s)\n", entry.FilePath, entry.AuditDoc)
			} else {
				fmt.Printf("  [UNCLEARED] %s (%s)\n              Current AST Violations: %s\n",
					entry.FilePath, entry.AuditDoc, entry.Reason)
			}
		}
		fmt.Println()
	}

	// If verbose or <= 30 violations, print detailed list
	if verbose || len(allViolations) <= 30 {
		if len(allViolations) > 0 {
			fmt.Println("=== Discovered Cleanliness Violations ===")
			for _, v := range allViolations {
				fmt.Printf("  [%s] %s:%d: %s\n", v.Category, v.FilePath, v.Line, v.Message)
			}
			fmt.Println()
		}
	} else {
		// Print a sample of top violations if verbose is false
		fmt.Println("=== Sample Cleanliness Violations (First 20) ===")
		for i, v := range allViolations {
			if i >= 20 {
				fmt.Printf("  ... and %d more violations (run with -v to view all)\n", len(allViolations)-20)
				break
			}
			fmt.Printf("  [%s] %s:%d: %s\n", v.Category, v.FilePath, v.Line, v.Message)
		}
		fmt.Println()
	}

	// Summary Table
	catCounts := make(map[string]int)
	for _, v := range allViolations {
		catCounts[v.Category]++
	}

	categories := []string{
		CatForbiddenFinalizer,
		CatReceiverNameTooLong,
		CatAcronymCapital,
		CatErrorStringFormat,
		CatBenchmarkUsesBN,
		CatLoopVarShadowing,
		CatUnclearedAudit,
	}

	fmt.Println("=== Cleanliness Verification Summary ===")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	fmt.Fprintln(w, "CATEGORY\tVIOLATIONS")
	fmt.Fprintln(w, "--------\t----------")
	for _, cat := range categories {
		fmt.Fprintf(w, "%s\t%d\n", cat, catCounts[cat])
	}
	w.Flush()

	fmt.Println("----------------------------------------")
	fmt.Printf("Total Files Checked:      %d\n", len(files))
	fmt.Printf("Total Violations Found:   %d\n\n", len(allViolations))

	if len(allViolations) > 0 {
		fmt.Printf("[FAIL] Cleanliness verification failed with %d violation(s)\n", len(allViolations))
		os.Exit(1)
	}

	fmt.Println("[PASS] Repository is 100% clean and compliant with Go 1.27 rules")
	os.Exit(0)
}
