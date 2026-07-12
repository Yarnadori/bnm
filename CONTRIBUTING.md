# Contributing to bnm

Thank you for your interest in contributing! Bug reports, feature requests, and pull requests are all welcome.

## Reporting Issues

- Use the [issue templates](https://github.com/Yarnadori/bnm/issues/new/choose).
- For security vulnerabilities, please do **not** open a public issue — see [SECURITY.md](SECURITY.md).

## Development Setup

Requirements: Go (version matching `go.mod` or newer).

```bash
git clone https://github.com/Yarnadori/bnm.git
cd bnm
go build -o bnm .
```

## Before Submitting a Pull Request

Please make sure the following all pass:

```bash
gofmt -l .        # should print nothing
go vet ./...
go test -race ./...
```

Guidelines:

- Keep changes focused — one topic per pull request.
- Add or update tests for behavior changes.
- bnm is intentionally dependency-light; avoid adding new dependencies unless there is a strong reason.
- bnm supports Linux, macOS, and Windows. Platform-specific code goes in `proc_unix.go` / `proc_windows.go` style build-tagged files.

## Project Layout

| File         | Role                                                        |
| ------------ | ----------------------------------------------------------- |
| `main.go`    | CLI entry point and command dispatch                        |
| `config.go`  | `bnm.json` schema and loading                               |
| `init.go`    | `bnm init` — project scaffolding                            |
| `runner.go`  | `bnm <script>` — parallel/sequential script execution       |
| `exec.go`    | `bnm exec` — ad-hoc command execution in a target directory |
| `list.go`    | `bnm list` — show configured directories and scripts        |
| `process.go` | Process spawning and prefixed output streaming              |
| `color.go`   | ANSI color handling for output prefixes                     |

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
