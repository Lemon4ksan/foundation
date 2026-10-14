# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| latest  | ✅        |

Only the latest release receives security fixes and vulnerability patches. We recommend always running on the most recent version of `foundation`.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Report security issues privately via GitHub's built-in
[Security Advisories](https://github.com/lemon4ksan/foundation/security/advisories/new)
feature (Security → Advisories → New draft advisory).

Include as much of the following as you can:

- A description of the vulnerability and its potential impact (e.g. memory safety violation, out-of-bounds arena access, data race, cryptographic weakness)
- Steps to reproduce (minimal Go code reproduction snippet or test case)
- The version of `foundation` and Go you are using
- Any relevant environment details (OS, CPU architecture, memory layout)

We will acknowledge receipt within **72 hours** and aim to release a fix within
**14 days** for confirmed critical issues.
