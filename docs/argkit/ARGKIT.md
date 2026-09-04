# POSIX Command-Line Argument & Flag Parser (`argkit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/argkit)

`argkit` provides a zero-dependency POSIX argument and flag parsing engine with short flag clumping, attached values, shell token normalization, and fuzzy typo suggestions.

## Core Capabilities

1. **Interspersed Arguments & Flags**: Flags and positional arguments can appear in any order on the command line without breaking `flag.FlagSet` parsing.
2. **Short Flag Stacking / Clumping**: Standard POSIX short flag stacking (e.g. `-la` expands to `-l` and `-a`, `-lAF` expands to `-l -A -F`).
3. **Attached Values**: Supports attached flag syntax (e.g. `-I*.tmp` or `-o=output.bin`).
4. **Shell Token Normalization**: Reconnects arguments split by shell tokenizers (e.g. PowerShell or CMD splitting `-out=pkg/api` and `.go`).
5. **Strict POSIX Terminator (`--`)**: Disables flag parsing for all subsequent arguments, treating them as literal positionals.
6. **Typo Suggestions**: Fuzzy matching with Levenshtein distance providing "Did you mean: --flag?" suggestions for unknown options.

## Key APIs & Usage

### 1. Interspersed Flag Parsing with Suggestions (`argkit.ParseInterspersedFlags`)

```go
package main

import (
    "flag"
    "fmt"
    "os"

    "github.com/lemon4ksan/foundation/argkit"
)

func main() {
    fs := flag.NewFlagSet("app", flag.ContinueOnError)
    verbose := fs.Bool("v", false, "verbose output")
    all := fs.Bool("a", false, "process all items")
    output := fs.String("output", "", "output file path")

    // args: ["file1.txt", "-va", "--output=result.json", "file2.txt"]
    posArgs, err := argkit.ParseInterspersedFlags(fs, os.Args[1:])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("Verbose: %v, All: %v, Output: %s\n", *verbose, *all, *output)
    fmt.Printf("Positional files: %v\n", posArgs) // ["file1.txt", "file2.txt"]
}
```

### 2. Multi-Valued Slice Flags (`argkit.StringSliceFlag`)

```go
package main

import (
    "flag"

    "github.com/lemon4ksan/foundation/argkit"
)

func main() {
    fs := flag.NewFlagSet("tool", flag.ExitOnError)
    var includes argkit.StringSliceFlag
    fs.Var(&includes, "include", "directories or files to include (repeatable)")

    // Pass multiple times: --include=src --include=pkg
    _ = fs.Parse([]string{"--include=src", "--include=pkg"})
}
```
