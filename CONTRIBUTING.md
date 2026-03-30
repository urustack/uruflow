# Contributing to Uruflow

Thanks for your interest in contributing. This document covers how to get started.

---

## Ways to Contribute

- Report bugs via [GitHub Issues](https://github.com/urustack/uruflow/issues)
- Suggest features or improvements
- Submit pull requests for bug fixes or new features
- Improve documentation
- Share the project and spread the word

---

## Development Setup

### Prerequisites

- Go 1.21+
- Git

### Clone and Build

```bash
git clone https://github.com/urustack/uruflow.git
cd uruflow
go mod download
go build ./cmd/...
```

---

## Submitting a Pull Request

1. Fork the repository
2. Create a branch: `git checkout -b fix/your-fix` or `feat/your-feature`
3. Make your changes
4. Test your changes manually against a real server/agent setup
5. Submit a PR with a clear description of what changed and why

Keep PRs focused — one fix or feature per PR.

---

## Reporting Bugs

Open an issue and include:

- Your OS and architecture
- Uruflow version (`uruflow --version`)
- What you did, what you expected, what happened
- Relevant logs if available

---

## Code Style

- Standard Go formatting (`gofmt`)
- No unnecessary abstractions — keep it explicit
- Match the style of the surrounding code

---

## Questions

Open a GitHub Discussion or issue if you have questions before starting a large contribution.
