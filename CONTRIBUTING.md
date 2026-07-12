# Contributing

Thanks for your interest in contributing to bnm.

## Development

1. Fork the repository and create a topic branch.
2. Make focused changes with clear commit messages.
3. Format Go files before committing:

```bash
gofmt -w *.go
```

4. Run checks before opening a pull request:

```bash
gofmt -l .        # should print nothing
go vet ./...
go test -race ./...
```

## Pull Requests

- Keep pull requests focused on one bug fix or feature.
- Include a short summary of the change.
- Add or update documentation when behavior changes.
- Mention any manual testing you performed.

## Issues

Use the issue templates when reporting bugs or requesting features. For bugs, include reproduction steps, your operating system, and relevant `bnm.json` content with secrets removed.

## Code Style

- Prefer small, direct functions.
- Keep behavior cross-platform where possible. Platform-specific code goes in build-tagged files (`proc_unix.go` / `proc_windows.go`).
- Avoid adding dependencies unless they clearly reduce complexity.

## Project Layout

| File           | Role                                                        |
| -------------- | ----------------------------------------------------------- |
| `main.go`      | CLI entry point and command dispatch                        |
| `config.go`    | `bnm.json` schema and loading                               |
| `init.go`      | `bnm init` — project scaffolding                            |
| `sync.go`      | `bnm sync` — refresh directory entries in `bnm.json`        |
| `directory.go` | Directory scanning and alias assignment                     |
| `runner.go`    | `bnm <script>` — parallel/sequential script execution       |
| `exec.go`      | `bnm exec` — ad-hoc command execution in a target directory |
| `list.go`      | `bnm list` — show configured directories and scripts        |
| `process.go`   | Process spawning and prefixed output streaming              |
| `color.go`     | ANSI color handling for output prefixes                     |
