# Contributing

Thanks for your interest in contributing to bnm.

## Development

1. Fork the repository and create a topic branch.
2. Make focused changes with clear commit messages.
3. Format Go files before committing:

```bash
gofmt -w *.go
```

4. Run tests before opening a pull request:

```bash
go test ./...
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
- Keep behavior cross-platform where possible.
- Avoid adding dependencies unless they clearly reduce complexity.
