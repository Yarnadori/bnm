# bnm

- [English](README.md)
- [日本語](README.ja.md)

[![CI](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml/badge.svg)](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Yarnadori/bnm)](https://github.com/Yarnadori/bnm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

bnm is a task runner designed to streamline command execution and script management in projects with multiple directories, such as monorepos or full-stack applications.

---

## Features

- **Initialize** a project with auto-detected subdirectories
- **Sync directories** in `bnm.json` when project folders are added or removed
- **Run scripts** defined in `bnm.json` in parallel or sequential mode
- **Script dependencies** — `dependsOn` runs prerequisite scripts first (cycles are rejected)
- **Directory filters** — `bnm dev -F` runs only the tasks of the given directories
- **Pass-through arguments** — `bnm dev -- --port 3000` appends extra args to every task command
- **Watch mode** — `bnm dev --watch` reruns the script when files in task directories change
- **Dry run** — `bnm deploy --dry-run` shows the execution plan without running anything
- **Timeouts & retries** — per-task `timeout` kills slow tasks, `retries` reruns flaky ones
- **Config validation** — `bnm check` catches broken paths, aliases, and dependencies before anything runs
- **Concurrency limit** — `maxParallel` caps how many tasks run at once
- **Execute arbitrary commands** in any configured directory via alias or path, or in all of them with `exec --all`
- **List** configured directories and scripts with `bnm list`
- **Cross-platform** command support (Windows / macOS / Linux)
- **Environment variables** — loads `.env` automatically (project root and per-directory), supports per-task `env`, and exposes `PROJECT_NAME` / `PROJECT_VERSION`
- **Prefixed, color-coded output** — each process output is labeled with its directory name (`--no-color` / `NO_COLOR` to disable)
- **Per-task log files** — `--log-dir logs` writes each task's output to its own file
- **Run summary** — per-task status and duration after every script run, also available as JSON (`--summary json`)
- **Shell completion** — `bnm completion bash|zsh|fish`, plus "Did you mean ...?" typo suggestions
- **Editor support** — published [JSON Schema](schema/bnm.schema.json) for `bnm.json` (`bnm init` adds `$schema` automatically)
- **CI-friendly** — non-zero exit code on failure; sequential scripts stop at the first failing task
- **Clean shutdown** — Ctrl+C terminates the whole process tree of every task

---

## Installation

### Linux / macOS (one-liner)

```bash
curl -fsSL https://raw.githubusercontent.com/Yarnadori/bnm/main/install.sh | bash
```

### Windows (PowerShell one-liner)

```powershell
irm https://raw.githubusercontent.com/Yarnadori/bnm/main/install.ps1 | iex
```

Both install scripts verify the SHA-256 checksum of the downloaded binary against the `checksums.txt` published with each release.

### Manual download

Download the pre-built binary for your platform from the [Releases](https://github.com/Yarnadori/bnm/releases) page:

| Platform              | File                    |
| --------------------- | ----------------------- |
| Linux (x64)           | `bnm-linux-amd64`       |
| Linux (arm64)         | `bnm-linux-arm64`       |
| macOS (x64)           | `bnm-darwin-amd64`      |
| macOS (Apple Silicon) | `bnm-darwin-arm64`      |
| Windows (x64)         | `bnm-windows-amd64.exe` |
| Windows (arm64)       | `bnm-windows-arm64.exe` |

Move the binary to a directory in your `PATH` (e.g. `/usr/local/bin` on Linux/macOS).

### Build from source

```bash
git clone https://github.com/Yarnadori/bnm.git
cd bnm
go build -o bnm .
```

---

## Getting Started

### 1. Initialize

Run `bnm init` in your project root. bnm scans subdirectories automatically and generates `bnm.json`.

```bash
bnm init
```

**Example — project with `frontend/` and `backend/` directories:**

```json
{
  "name": "my-app",
  "version": "0.0.0",
  "directories": {
    "BACKEND": {
      "alias": "B",
      "path": "./backend"
    },
    "FRONTEND": {
      "alias": "F",
      "path": "./frontend"
    }
  },
  "scripts": {}
}
```

### 2. Define Scripts

Add scripts to `bnm.json`:

```json
{
  "name": "my-app",
  "version": "1.0.0",
  "directories": {
    "FRONTEND": { "alias": "F", "path": "./frontend" },
    "BACKEND": { "alias": "B", "path": "./backend" }
  },
  "scripts": {
    "dev": {
      "mode": "parallel",
      "tasks": [
        { "dir": "FRONTEND", "command": "npm run dev" },
        { "dir": "BACKEND", "command": "npm run dev" }
      ]
    },
    "build": {
      "mode": "sequential",
      "tasks": [
        { "dir": "FRONTEND", "command": "npm run build" },
        { "dir": "BACKEND", "command": "npm run build" }
      ]
    }
  }
}
```

### 3. Run Scripts

```bash
bnm dev    # runs all "dev" tasks in parallel
bnm build  # runs all "build" tasks sequentially
```

---

## Commands

### `bnm init`

Initializes the project by creating `bnm.json` in the current directory. Subdirectories are scanned automatically. Hidden directories (`.git`, etc.) and dependency/build directories (`node_modules`, `dist`, `build`, `vendor`, etc.) are excluded.

### `bnm sync`

Updates the `directories` section in `bnm.json` to match the current subdirectories. Existing aliases are kept for unchanged directories, new directories get generated aliases, and removed directories are deleted from `directories`.

### `bnm check`

Validates `bnm.json` and exits non-zero if anything is wrong: JSON syntax, `mode` / `maxParallel` / `timeout` / `retries` values, `dependsOn` references and cycles, directory paths that don't exist on disk, duplicate aliases, task `dir` entries that resolve nowhere, and tasks with no command for the current OS. Useful as an early step in CI.

```bash
$ bnm check
[bnm] Found 2 problem(s) in bnm.json:
  - directory 'FRONTEND': path './frontend' does not exist
  - script 'dev' task 2: no command for this OS (linux)
```

### `bnm list`

Shows the directories and scripts defined in `bnm.json`. Alias: `bnm ls`.

```bash
$ bnm list
my-app v1.0.0

Directories:
  BACKEND      -B  ./backend
  FRONTEND     -F  ./frontend

Scripts:
  dev (parallel)
    FRONTEND     npm run dev
    BACKEND      npm run dev
```

### `bnm <script> [dir...] [-- args...]`

Runs a script defined in `bnm.json`.

```bash
bnm dev
bnm build
```

**Directory filters** — run only the tasks of specific directories (alias with `-`, directory key, or path):

```bash
bnm dev -F                # only the FRONTEND task
bnm dev FRONTEND BACKEND  # multiple directories
bnm dev ./frontend        # by path, and '.' matches root tasks
```

**Pass-through arguments** — everything after `--` is appended to every task command of the script (dependencies are not affected):

```bash
bnm test -- --watch
bnm dev -F -- --port 3000
```

**Watch mode** — `--watch` (or `-w`) reruns the script whenever a file under any task directory changes. Running tasks are terminated and restarted, so it also works with dev servers. Hidden directories and dependency/build directories (`node_modules`, `dist`, `target`, etc.) are ignored. Stop with Ctrl+C.

```bash
bnm test --watch
bnm dev -F --watch
```

**Dry run** — `--dry-run` (or `-n`) prints the execution plan (order, mode, resolved directories, and commands, including dependencies and filters) without running anything:

```bash
bnm deploy --dry-run
```

**Log files** — `--log-dir <dir>` also writes each task's output (without prefixes or colors) to `<dir>/<script>/<task>.log`, which makes parallel output easy to inspect in CI. Files are truncated at the start of each invocation; retries and watch-mode reruns append.

```bash
bnm test --log-dir logs   # → logs/test/FRONTEND.log, logs/test/BACKEND.log, ...
```

**JSON summary** — `--summary json` replaces the summary table with a single JSON line, so CI can parse per-task results:

```bash
$ bnm dev --summary json
...
{"script":"dev","ok":true,"tasks":[{"name":"FRONTEND","status":"ok","durationMs":812}]}
```

**Colors** — output colors are disabled automatically when stdout is not a TTY or the [`NO_COLOR`](https://no-color.org/) environment variable is set; `--no-color` forces them off.

If the script has `dependsOn`, those scripts run to completion first (see [Script group](#script-group)).

After the run, bnm prints a summary with per-task status (`ok` / `failed` / `skipped` / `canceled`) and duration.

Exit code behavior:

- In `sequential` mode, execution stops at the first failing task.
- If a dependency script fails, the remaining scripts are skipped.
- bnm exits with a non-zero code if any task fails, so scripts are safe to use in CI.
- Note: script names that collide with built-in commands (`init`, `sync`, `list`, `ls`, `check`, `exec`, `completion`, `help`, `version`) cannot be invoked this way.

### `bnm exec <dir> <command...>`

Executes an arbitrary command in a specific directory.

```bash
# By alias (prefix with -)
bnm exec -F npm install

# By directory name
bnm exec FRONTEND npm install

# By path
bnm exec ./frontend npm install
```

### `bnm exec --all <command...>`

Executes a command in every configured directory, one after another (sorted by directory key). Failures don't stop the remaining directories, but any failure makes bnm exit non-zero.

```bash
bnm exec --all git status
bnm exec --all npm install
```

### `bnm completion <bash|zsh|fish>`

Prints a shell completion script that completes commands, script names, and directories (for `bnm exec`).

```bash
# bash (~/.bashrc)
source <(bnm completion bash)

# zsh (~/.zshrc)
source <(bnm completion zsh)

# fish
bnm completion fish > ~/.config/fish/completions/bnm.fish
```

---

## bnm.json Reference

| Field         | Type   | Description                                            |
| ------------- | ------ | ------------------------------------------------------ |
| `$schema`     | string | Optional JSON Schema URL for editor validation         |
| `name`        | string | Project name. Exposed as `PROJECT_NAME`                |
| `version`     | string | Project version. Exposed as `PROJECT_VERSION`          |
| `directories` | object | Named directory entries with alias and path            |
| `scripts`     | object | Named script groups with mode, dependencies, and tasks |

A JSON Schema is published at [`schema/bnm.schema.json`](schema/bnm.schema.json); `bnm init` writes the `$schema` reference automatically so editors like VS Code validate and autocomplete `bnm.json`.

### Directory entry

| Field   | Type   | Description                         |
| ------- | ------ | ----------------------------------- |
| `alias` | string | Short alias used with `bnm exec -X` |
| `path`  | string | Relative path to the directory      |

### Script group

| Field         | Type    | Description                                                             |
| ------------- | ------- | ----------------------------------------------------------------------- |
| `mode`        | string  | `"parallel"` (default) or `"sequential"`                                 |
| `dependsOn`   | array   | Scripts that run to completion before this one. Cycles are rejected     |
| `maxParallel` | integer | Max tasks running at once in parallel mode. `0` or omitted is unlimited |
| `tasks`       | array   | List of tasks to run                                                    |

```json
{
  "scripts": {
    "build": { "tasks": [{ "dir": "FRONTEND", "command": "npm run build" }] },
    "deploy": {
      "dependsOn": ["build"],
      "tasks": [{ "dir": "BACKEND", "command": "npm run deploy" }]
    }
  }
}
```

### Task

| Field     | Type             | Description                                                                  |
| --------- | ---------------- | ---------------------------------------------------------------------------- |
| `dir`     | string           | Directory key from `directories`                                             |
| `command` | string or object | Command to run. Can be OS-specific (see below)                               |
| `env`     | object           | Extra environment variables applied only to this task                        |
| `timeout` | string           | Per-attempt time limit (e.g. `"30s"`, `"2m"`). The task is killed when it elapses |
| `retries` | integer          | How many times to rerun the task after a failure. Default `0`                |

```json
{
  "tasks": [
    { "dir": "BACKEND", "command": "npm run test:e2e", "timeout": "5m", "retries": 2 }
  ]
}
```

### OS-specific commands

```json
{
  "command": {
    "windows": "echo Running on Windows",
    "mac": "echo Running on macOS",
    "linux": "echo Running on Linux",
    "default": "echo Fallback command"
  }
}
```

---

## Environment Variables

bnm automatically loads `.env` from the project root and passes the following variables to every process:

| Variable          | Value                         |
| ----------------- | ----------------------------- |
| `PROJECT_NAME`    | `name` field in `bnm.json`    |
| `PROJECT_VERSION` | `version` field in `bnm.json` |

Each task additionally receives, in order of increasing precedence:

1. The process environment plus the project root `.env`
2. `.env` in the task's directory (e.g. `frontend/.env`)
3. The task's own `env` entries in `bnm.json`

Set `NO_COLOR` to any value to disable colored output prefixes. Colors are also disabled automatically when output is not a terminal.

---

## Community

- [Contributing](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License](LICENSE)
