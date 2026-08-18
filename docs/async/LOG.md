# Structured Logger Facade (`async/log`)

[![Go Reference](https://img.shields.io/badge/go-reference-007d9c?logo=go&logoColor=white&style=flat-square)](https://pkg.go.dev/github.com/lemon4ksan/foundation/async/log)

`async/log` provides an ultra-high-speed structured logging facade with zero memory allocations on hot paths and asynchronous channel-based writer decoupling.

## Motivation & Problem Context

In high-throughput network applications, synchronous disk writes during log emission block application worker threads during I/O latency spikes. Formatting structured log entries with standard reflection-based loggers generates thousands of heap allocations per second, increasing GC mark-assist overhead. Decoupling log serialization through non-blocking buffer channels ensures that logging never interferes with application line-rate throughput.

## Comparison

### Standard Implementation (Synchronous & Allocating)

```go
logger := slog.Default()
logger.Info("user request",
    slog.String("user", "100"),
    slog.Int("status", 200),
)
```

### Foundation Implementation (Asynchronous & Zero-Alloc)

```go
logger := log.New(os.Stdout, log.LevelDebug)
compLog := logger.With(log.Component("auth"))

compLog.Info("user request",
    log.String("user", "100"),
    log.Int("status", 200),
)
```

## Architecture & Mechanics

```mermaid
graph LR
    CALLER["Application Goroutine"] -- "0 B/op Push" --> BUF["sync.Pool Buffer"]
    BUF --> CHAN["Buffered Channel Queue"]
    CHAN --> WRITER["Dedicated Background Flusher"]
    WRITER --> DISK["os.Stdout / Disk File"]
```

* **Zero-Allocation Buffer Pools**: Log lines are formatted into pooled memory buffers (`sync.Pool`) before being pushed to an asynchronous background worker.
* **Non-Blocking Drop Strategy**: If the queue fills up during disk stalls, logs can be dropped to protect application liveness rather than hanging the system.

## Practical Recipes

### 1. Component-Scoped Subsystem Logging

```go
package main

import (
	"os"

	"github.com/lemon4ksan/foundation/async/log"
)

func main() {
	baseLog := log.New(os.Stdout, log.LevelInfo)

	authLog := baseLog.With(log.Component("auth"))
	dbLog := baseLog.With(log.Component("postgres"))

	authLog.Info("token verified", log.String("sub", "usr_100"))
	dbLog.Warn("slow query detected", log.Int("duration_ms", 125))
}
```
