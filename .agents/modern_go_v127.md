---
name: foundation-modern-go
description: Strict, deterministic Go 1.24-1.27 architectural guidelines, linter configs, and refactoring heuristics to ensure perfectly uniform code generation across all agents.
trigger: always_on
---

# Modern Go (1.24 - 1.27) Architecture & Audit Guidelines

This skill provides mathematically strict, deterministic templates for evaluating and refactoring modern Go code. **The primary objective is that code generation must be unified and identical regardless of the specific agent or model.** Ambiguity is not permitted. 

When working on `foundation`, you MUST follow these specific "Before/After" transformations and rules.

## 1. Syntactic and Semantic Modernization

### 1.1. Eradicate `any` and `interface{}`
**Rule:** The use of `any` or `interface{}` for bypassing type safety in methods is strictly forbidden. You must use Go 1.24 generic methods and generic type aliases.
*   **BAD:** `func (c *Cache) Set(key string, val any)`
*   **GOOD:** `func (c *Cache[V]) Set(key string, val V)`
*   **GOOD (Type Alias):** `type Set[P comparable] = map[P]bool`

### 1.2. Collection Traversal and Iterators
**Rule:** Manual `for` loops for basic search, clone, or filter operations are forbidden. You must use `slices` and `maps`. For parsing, use Go 1.25+ `iter.Seq` from `strings` and `bytes`.
*   **BAD:** `for _, v := range arr { if v == target { return true } }`
*   **GOOD:** `slices.Contains(arr, target)`
*   **BAD:** `func clone(m map[string]int) map[string]int { ... }`
*   **GOOD:** `maps.Clone(m)`

### 1.3. JSON Zero-Value Handling (Go 1.27+)
**Rule:** The `omitempty` tag is obsolete for modern structs. Use `omitzero`. Any struct using `omitzero` must implement `IsZero() bool`. Do not write custom `MarshalJSON` methods solely for zero-value checks.
*   **BAD:** `Field []string ` + "`" + `json:"field,omitempty"` + "`" + `
*   **GOOD:** `Field []string ` + "`" + `json:"field,omitzero"` + "`" + `

### 1.4. Loop Variables
**Rule:** Never shadow loop variables (`v := v`). Go 1.22+ handles loop variable capture natively. Delete all legacy shadowing.

## 2. Memory Management (Zero-Leak Policy)

### 2.1. Weak Pointers (Go 1.24+)
**Rule:** Any long-lived cache holding objects MUST use `weak.Pointer` to prevent GC leaks.
*   **BAD:** Storing `*Session` directly in a global map.
*   **GOOD:** Storing `weak.Pointer[Session]` using `weak.Make(sess)`. Always check `if val := ptr.Value(); val != nil { ... }`.

### 2.2. Finalizers vs Cleanups
**Rule:** `runtime.SetFinalizer` is classified as a critical technical debt. It must be refactored to `runtime.AddCleanup`.
*   **BAD:** `runtime.SetFinalizer(obj, cleanupFunc)`
*   **GOOD:** `runtime.AddCleanup(obj, cleanupFunc, arg)`

### 2.3. SwissTable `sync.Map`
**Rule:** Custom caches using `sync.RWMutex` + `map` must be replaced with `sync.Map` (which uses SwissTable as of Go 1.24). Always wrap `sync.Map` in a generic struct to preserve type safety.

### 2.4. `sync.Pool` Demolition
**Rule:** Do not use `sync.Pool` for structs under 32 bytes. The Go 1.27 allocator handles small allocations efficiently.

## 3. Security and Path Isolation

### 3.1. File System Path Traversal
**Rule:** Never concatenate user input to create file paths. Use `os.Root` (Go 1.24+) to sandbox file operations.
*   **BAD:** `os.Open(filepath.Join(base, userInput))`
*   **GOOD:**
    ```go
    root, _ := os.OpenRoot(base)
    defer root.Close()
    f, _ := root.Open(userInput)
    ```

### 3.2. Binary Dependencies
**Rule:** Track binary tools in `go.mod` using the `tool` directive. Use `go get -tool <pkg>` instead of `//go:generate` or `Makefile` downloads.

## 4. Modern Testing (Go 1.24+)

### 4.1. Temporal Bubbles (`testing/synctest`)
**Rule:** Network timeouts, retries, and concurrent backoff algorithms must NEVER use `time.Sleep` in tests. Wrap them in `synctest.Run` and sync with `synctest.Wait()`.
*   **BAD:** `time.Sleep(1 * time.Second)` in a test.
*   **GOOD:**
    ```go
    synctest.Run(func() {
        // test logic
        synctest.Wait()
    })
    ```

### 4.2. Benchmark Loops
**Rule:** `b.N` loops are banned due to compiler over-optimization. Use `b.Loop()`.
*   **BAD:** `for i := 0; i < b.N; i++ { ... }`
*   **GOOD:** `for b.Loop() { ... }`

## 5. Mathematical Refactoring Triggers (Strict Heuristics)

Agents must autonomously trigger structural changes when these thresholds are met:
1.  **Cognitive Thresholds**: If `gocognit` > 15 or `nestif` > 3, you MUST decompose the function. Apply "Guard Clauses" (Early Returns) to flatten `if/else` logic to 0 levels of successful-path nesting.
2.  **Memory Hot Paths**: If `allocs/op > 0` in transport parsing, you MUST replace `string(byteSlice)` with `unsafe.String`, or use `bytes.Buffer`. Use **In-Place Modification** (e.g., rewriting elements inside the original array without allocating a new slice).
3.  **Context**: I/O functions missing `context.Context` as the first argument must be rewritten.
4.  **Error Wrapping**: Always use `fmt.Errorf("...: %w", err)` or `errors.Join`. Never use `%v` for errors.

## 6. Linter Master Config
Agents must ensure the project passes this strict config without warnings:
- `staticcheck` (All categories)
- `errcheck` + `nilerr`
- `modernize` (with `--fix`)
- `sloglint`
- `gocognit` (min-complexity: 15)
- `nestif` + `funlen` (nesting max 3)
- `gosec` (all except G204)
- `spancheck` + `musttag`
- `revive` + `sqlclosecheck`


## 7. Deterministic Naming & Style (Zero-Variance Policy)
To ensure identical code generation across all agents, you must adhere to strict naming and formatting rules without deviation.

### 7.1. Acronym Capitalization
**Rule:** Initialisms and acronyms must always be fully capitalized or fully lowercased. Mixed case is forbidden.
*   **BAD:** `TlsConfig`, `HttpServer`, `ParseUrl`, `Ja4Fingerprint`
*   **GOOD:** `TLSConfig`, `HTTPServer`, `ParseURL`, `JA4Fingerprint` (or `tlsConfig` for unexported).

### 7.2. Receiver Naming
**Rule:** Method receivers must strictly be 1 or 2 letters derived from the struct name. Banned words: `this`, `self`, `me`.
*   **BAD:** `func (client *Client) Do()` / `func (self *Client) Do()`
*   **GOOD:** `func (c *Client) Do()` 

### 7.3. Error Messages Formatting
**Rule:** Error strings must strictly be lowercase and have no trailing punctuation. They must describe the failed action.
*   **BAD:** `fmt.Errorf("Failed to parse JA4! %w", err)`
*   **GOOD:** `fmt.Errorf("parse ja4 fingerprint: %w", err)`

### 7.4. GoDoc Comments Strictness
**Rule:** Every exported identifier must have a GoDoc comment that strictly begins with the identifier's exact name. No HTML.
*   **BAD:** `// This function parses the URL.`
*   **GOOD:** `// ParseURL extracts the domain and port from the raw string.`

## 8. Structural & Instantiation Patterns

### 8.1. The Functional Options Pattern
**Rule:** If a struct has more than 3 configuration fields, or fields that might grow in the future (like `TLSConfig`), you MUST use the Functional Options pattern. Banned: Massive configuration structs passed by value.
*   **GOOD Template:**
    ```go
    type Option func(*Server)
    func WithTimeout(t time.Duration) Option { ... }
    func NewServer(opts ...Option) *Server { ... }
    ```

### 8.2. Explicit Interface Verification
**Rule:** When implementing an interface, you must include a compile-time assertion at the top of the file (or right after the struct).
*   **GOOD:** `var _ io.Reader = (*MyReader)(nil)`

## 9. Concurrency Strictness

### 9.1. Goroutine Lifecycle (No Bare Goroutines)
**Rule:** Bare `go func()` calls are forbidden unless they are daemon processes attached to the application lifecycle. For any concurrent tasks that return errors or need synchronization, you MUST use `golang.org/x/sync/errgroup`.
*   **BAD:** Using `sync.WaitGroup` and manually passing error channels.
*   **GOOD:** `eg, ctx := errgroup.WithContext(ctx)` + `eg.Go(func() error { ... })`

## 10. AI Agent Execution Directives (Output Determinism)
**Rule:** When asked to write, refactor, or output Go code based on these guidelines, the AI must obey these output constraints:
1.  **No Placeholders:** Never use `// ... implementation here ...` or `TODO`. Write the complete, functional code.
2.  **No Conversational Filler:** Start directly with the Go code or the requested explanation. Do not say "Here is your modernized code."
3.  **In-Place Refactoring:** When fixing a specific function, output ONLY the corrected function and its necessary imports, do not hallucinate the rest of the file unless requested.
4.  **Self-Correction Gate:** Before outputting, internally verify: Are there any `interface{}`? Are loops shadowing? Are errors capitalized? If yes, fix before returning the response.
