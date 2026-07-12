# Security Policy

## Reporting a Vulnerability

Please do not report security vulnerabilities through public GitHub issues.

Report security concerns privately through GitHub Security Advisories:

https://github.com/Yarnadori/bnm/security/advisories/new

Include:

- A description of the issue.
- Steps to reproduce or a proof of concept.
- The affected version or commit.
- Any suggested mitigation, if known.

## Supported Versions

Security fixes are handled on the default branch. If releases are published, users should upgrade to the latest release.

## Security Model

bnm is a task runner: it executes shell commands defined in `bnm.json` and loads environment variables from `.env` in the current directory. This is by design, but it means:

- **Treat `bnm.json` and `.env` as code.** Running bnm in a directory containing an untrusted `bnm.json` executes whatever commands it defines — the same trust model as `npm run` or `make`.
- Release binaries are built by GitHub Actions and published with SHA-256 checksums (`checksums.txt`); the install scripts verify them automatically.
