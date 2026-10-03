# Documentation

Package documentation lives in the code. Read it with `go doc github.com/lemon4ksan/foundation/<package>` or on pkg.go.dev. Every package has a `doc.go` that explains what it is for and how it differs from the standard library, and runnable examples in `example_test.go`.

This directory holds only what spans the whole repository:

- [VISION.md](VISION.md): why `foundation` exists, who uses it, what is in and out of scope.
- [ARCHITECTURE.md](ARCHITECTURE.md): techniques used across packages, and how to choose between similar primitives.
- [adr/](adr/README.md): the decisions behind the layering, allocation tiers, documentation and release process.
