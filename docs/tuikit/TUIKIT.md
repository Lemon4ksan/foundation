# Terminal UI Presentation & Subcommand Toolkit (`tuikit`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/tuikit)

`tuikit` provides a zero-dependency terminal UI toolkit for building CLI applications, REPLs, formatted tables, styled borders, badges, and progress indicators.

## Core Capabilities

1. **CLI Application Framework (`tuikit.App`)**: Subcommand routing, grouped help generation, signal handling, and execution context.
2. **Terminal Capability Sniffer (`tuikit.ProbeTerminal`)**: Detects TTY support, terminal width/height, color depth (16-color, 256-color, TrueColor 24-bit), and honors `NO_COLOR` standard.
3. **Formatted Tables (`tuikit.Table`)**: Auto-aligned column tables with header styling and cell padding.
4. **Bordered Boxes (`tuikit.Box`)**: Single, double, heavy, and rounded border framing for CLI output.
5. **Interactive Indicators**: Progress bars, spinners, and status badges (`[OK]`, `[FAIL]`, `[WARN]`, `[INFO]`).

## Key APIs & Usage

### 1. Building a Multi-Command CLI App

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lemon4ksan/foundation/tuikit"
)

func main() {
    app := tuikit.NewApp("vortex", "1.0.0", "High-performance network engine")
    
    app.AddCommand(tuikit.Command{
        Name:        "bench",
        Description: "Run latency and throughput benchmark",
        Run: func(ctx context.Context, args []string) error {
            fmt.Println("Running benchmark...")
            return nil
        },
    })

    if err := app.Run(context.Background(), os.Args[1:]); err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
}
```

### 2. Formatted Tables & Boxes

```go
package main

import (
    "os"

    "github.com/lemon4ksan/foundation/tuikit"
)

func main() {
    // Print styled box
    box := tuikit.NewBox(tuikit.BoxRounded)
    box.Render(os.Stdout, "Server online on :8080\nStatus: Healthy")

    // Render table
    table := tuikit.NewTable("Method", "Path", "Latency")
    table.AddRow("GET", "/api/v1/users", "1.2 ms")
    table.AddRow("POST", "/api/v1/login", "4.8 ms")
    table.Render(os.Stdout)
}
```
