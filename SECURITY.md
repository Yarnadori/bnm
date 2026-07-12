# Security Policy

## Supported Versions

Only the latest release of bnm receives security fixes.

## Reporting a Vulnerability

Please do **not** report security vulnerabilities through public GitHub issues.

Instead, use [GitHub private vulnerability reporting](https://github.com/Yarnadori/bnm/security/advisories/new) ("Report a vulnerability" on the Security tab). You should receive a response within a week.

Please include:

- A description of the vulnerability and its impact
- Steps to reproduce
- Affected version(s) and platform(s)

## Security Model

bnm is a task runner: it executes shell commands defined in `bnm.json` and loads environment variables from `.env` in the current directory. This is by design, but it means:

- **Treat `bnm.json` and `.env` as code.** Running bnm in a directory containing an untrusted `bnm.json` executes whatever commands it defines — the same trust model as `npm run` or `make`.
- Release binaries are built by GitHub Actions and published with SHA-256 checksums (`checksums.txt`); the install scripts verify them automatically.
