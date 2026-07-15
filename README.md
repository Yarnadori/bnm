# bnm

- [English](README.md)
- [日本語](README.ja.md)

[![CI](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml/badge.svg)](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Yarnadori/bnm)](https://github.com/Yarnadori/bnm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**bnm is a lightweight task runner for running commands across multiple directories, regardless of language or package manager.**

Describe your tasks in one small JSON file:

```json
{
  "scripts": {
    "dev": {
      "frontend": "npm run dev",
      "backend": "go run ."
    }
  }
}
```

Run them all at once:

```bash
$ bnm dev
[frontend] $ npm run dev
[backend]  $ go run .
```

That's it — both processes run in parallel with color-coded output, and Ctrl+C shuts down the whole tree cleanly.

## When to use bnm

- You want to start `frontend` and `backend` (and more) with one command
- Your project mixes Node.js, Go, Python, Rust, ... — bnm doesn't care
- A full monorepo build system (Turborepo, Bazel, Nx) is more than you need
- You want tasks in a single, simple JSON file that anyone can read

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

Run `bnm init` in your project root. It scans subdirectories, asks which ones to include, and suggests commands based on what it finds (`package.json`, `go.mod`, `Cargo.toml`, ...):

```
$ bnm init
Include 'frontend'? [Y/n]:
Include 'backend'? [Y/n]:
Detected package.json in frontend.
Use "npm run dev" for the dev task? [Y/n]:
Detected go.mod in backend.
Use "go run ." for the dev task? [Y/n]:
Created bnm.json with 2 directories.
```

Non-interactive environments (CI, scripts) can skip the questions:

```bash
bnm init --yes                        # accept all defaults
bnm init --include frontend,backend   # only these directories
bnm init --exclude docs,tmp           # everything except these
bnm init --dry-run                    # print the config without writing it
```

`bnm init` never overwrites an existing `bnm.json`; use `--force` to replace it (a `bnm.json.bak` backup is written first), or `bnm sync` to just update the directory list.

Then run your scripts:

```bash
bnm dev                # every task of "dev", in parallel
bnm dev frontend       # only the frontend task
bnm test -- --verbose  # append arguments to every task command
```

---

## Commands

### `bnm <script> [dir...] [options] [-- args...]`

Runs a script defined in `bnm.json`.

**Directory filters** — positional names run only the matching tasks. A name is a `directories` key, its alias, or a path; `--filter` / `-F` is the explicit equivalent and can repeat:

```bash
bnm dev frontend backend
bnm dev --filter frontend -F backend
bnm dev ./frontend        # by path, and '.' matches root tasks
```

**Task filters** — `--task` / `-T` runs only the tasks with the given names (case-insensitive exact match, repeatable). Combined with a directory filter, both conditions must match:

```bash
bnm check --task lint
bnm check -T lint -T typecheck
bnm check --filter frontend --task lint   # AND: the lint task in frontend
```

**Pass-through arguments** — everything after `--` is appended to every task command of the script (dependencies are not affected):

```bash
bnm dev -F frontend -- --port 3000
```

**Watch mode** — `--watch` (or `-w`) reruns the script whenever a file under any task directory changes. Running tasks are terminated and restarted, so it also works with dev servers. Hidden directories and dependency/build directories (`node_modules`, `dist`, `target`, etc.) are ignored. Stop with Ctrl+C.

A script that should always watch can set `"watch": true` in its detailed form — then plain `bnm dev` watches, and `--no-watch` runs it once. The setting only applies to the script you invoke; a dependency's `watch` is ignored:

```json
{
  "scripts": {
    "dev": {
      "watch": true,
      "tasks": {
        "frontend": "npm run dev",
        "backend": "go run ."
      }
    }
  }
}
```

**Per-task watch** — `"watch": true` on a *task* restarts just that task when files in its directory change; the other tasks keep running. Dependencies run once up front, and tasks sharing the directory all restart together:

```json
{
  "scripts": {
    "dev": {
      "tasks": {
        "frontend": {
          "command": "npm run dev",
          "watch": true
        },
        "backend": "go run ."
      }
    }
  }
}
```

```text
$ bnm dev
[frontend] $ npm run dev
[backend] $ go run .
... edit frontend/src/app.js ...
[bnm] Task 'frontend' changed. Restarting...
[frontend] $ npm run dev
```

Per-task watch requires parallel mode (restarting one task contradicts a fixed order), and `--no-watch` disables it. Script-level watch (`--watch` or the script's `"watch": true`) takes precedence and reruns everything.

**Dry run** — `--dry-run` (or `-n`) prints the execution plan (order, mode, resolved directories, and commands, including dependencies and filters) without running anything.

**Log files** — `--log-dir <dir>` also writes each task's output (without prefixes or colors) to `<dir>/<script>/<task>.log`. Files are truncated at the start of each invocation; retries and watch-mode reruns append.

**JSON summary** — `--summary json` replaces the summary table with a single JSON line, so CI can parse per-task results:

```json
{"script":"dev","ok":true,"tasks":[{"name":"frontend","dir":"frontend","status":"ok","durationMs":812}]}
```

**Colors** — disabled automatically when stdout is not a TTY or [`NO_COLOR`](https://no-color.org/) is set; `--no-color` forces them off.

Exit code behavior:

- In `sequential` mode, execution stops at the first failing task.
- If a dependency script fails, the remaining scripts are skipped.
- bnm exits with a non-zero code if any task fails, so scripts are safe to use in CI.
- Note: script names that collide with built-in commands (`init`, `sync`, `list`, `ls`, `doctor`, `exec`, `completion`, `help`, `version`) cannot be invoked this way. `check` is the exception: a script named `check` takes priority, and the validator stays available as `bnm doctor`.

### `bnm init`

Creates `bnm.json` interactively (see [Getting Started](#getting-started)). Flags: `--yes`, `--force`, `--dry-run`, `--include a,b`, `--exclude a,b`.

### `bnm sync`

Updates the `directories` section in `bnm.json` to match the current subdirectories. Existing entries are kept for unchanged paths, new directories are added, and removed directories are deleted.

### `bnm doctor`

Validates `bnm.json` and exits non-zero if anything is wrong: JSON syntax, `mode` / `maxParallel` / `timeout` / `retries` values, `dependsOn` references and cycles, directory paths that don't exist on disk, duplicate aliases, task directories that resolve nowhere, duplicate task names within a script, and tasks with no command for the current OS. Useful as an early step in CI.

```bash
$ bnm doctor
[bnm] Found 2 problem(s) in bnm.json:
  - directory 'frontend': path './frontend' does not exist
  - script 'dev' task 2: no command for this OS (linux)
```

`bnm check` does the same — unless your config defines a script named `check`, in which case it runs that script like any other. Existing CI setups keep working; use `bnm doctor` when you want validation unambiguously.

### `bnm list`

Shows the directories and scripts defined in `bnm.json`. Alias: `bnm ls`.

### `bnm exec <dir> <command...>`

Executes an arbitrary command in a specific directory, addressed by name, alias, or path:

```bash
bnm exec frontend npm install
bnm exec ./frontend npm install
bnm exec . git status          # project root
```

### `bnm exec --all <command...>`

Executes a command in every configured directory, one after another (sorted by name). Failures don't stop the remaining directories, but any failure makes bnm exit non-zero.

```bash
bnm exec --all git status
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

## Configuration

A JSON Schema is published at [`schema/bnm.schema.json`](schema/bnm.schema.json); `bnm init` writes the `$schema` reference automatically so editors like VS Code validate and autocomplete `bnm.json`.

### Scripts

Most scripts are just a map from task name to command. The key doubles as the directory — a `directories` name, an alias, or a plain path (`./` optional, `.` is the project root) — and tasks run in parallel by default:

```json
{
  "scripts": {
    "dev": {
      "frontend": "npm run dev",
      "backend": "go run ."
    }
  }
}
```

To run the same command in every configured directory, use a plain string:

```json
{
  "scripts": {
    "lint": "npx eslint ."
  }
}
```

When you need more control — sequential order, dependencies, concurrency limits, or per-task settings — use the detailed form. Tasks stay keyed the same way and can carry `dir`, `env`, `timeout`, and `retries`:

```json
{
  "scripts": {
    "test": {
      "mode": "sequential",
      "dependsOn": ["build"],
      "maxParallel": 2,
      "tasks": {
        "frontend": {
          "command": "npm test",
          "timeout": "2m"
        },
        "backend": {
          "command": "go test ./...",
          "retries": 1,
          "env": { "APP_ENV": "test" }
        }
      }
    }
  }
}
```

| Script field  | Type    | Description                                                              |
| ------------- | ------- | ------------------------------------------------------------------------ |
| `mode`        | string  | `"parallel"` (default) or `"sequential"`                                  |
| `dependsOn`   | array   | Scripts that run to completion before this one. Cycles are rejected      |
| `maxParallel` | integer | Max tasks running at once in parallel mode. `0` or omitted is unlimited  |
| `watch`       | boolean | Watch mode by default when this script is invoked; `--no-watch` overrides |
| `tasks`       | object  | Tasks keyed by task name (an array of `{dir, command, ...}` also works)  |

| Task field | Type             | Description                                                                        |
| ---------- | ---------------- | ---------------------------------------------------------------------------------- |
| `dir`      | string           | Directory the command runs in: a `directories` name, alias, or path. Defaults to the task's key |
| `command`  | string or object | Command to run. Can be OS-specific (see below)                                     |
| `env`      | object           | Extra environment variables applied only to this task                              |
| `timeout`  | string           | Per-attempt time limit (e.g. `"30s"`, `"2m"`). The task is killed when it elapses  |
| `retries`  | integer          | How many times to rerun the task after a failure. Default `0`                      |
| `watch`    | boolean          | Restart just this task when files in its directory change (parallel mode only)     |

### Multiple tasks in one directory

Task names are independent of directories, so `dir` lets several tasks share one. Task names must be unique within a script:

```json
{
  "scripts": {
    "check": {
      "tasks": {
        "lint": {
          "dir": "frontend",
          "command": "npm run lint"
        },
        "typecheck": {
          "dir": "frontend",
          "command": "npm run typecheck"
        }
      }
    }
  }
}
```

Run a single task by name, or every task of a directory:

```bash
bnm check --task lint     # only the lint task (-T typecheck also works)
bnm check frontend        # all tasks whose dir is frontend: lint and typecheck
```

Logs, the summary, and `--log-dir` files are labeled with the task name, so tasks sharing a directory stay distinguishable:

```text
[lint] $ npm run lint
[typecheck] $ npm run typecheck
```

### Directories

`directories` is optional — script keys that aren't listed there are used as paths directly. Define it when you want short names for longer paths (the key doubles as the name for filters and `bnm exec`):

```json
{
  "directories": {
    "web": "./apps/frontend",
    "api": "./services/backend"
  },
  "scripts": {
    "dev": {
      "web": "npm run dev",
      "api": "go run ."
    }
  }
}
```

The legacy `{"alias": ..., "path": ...}` object form is still accepted; its alias works as an alternative name.

### OS-specific commands

Any `command` can be an object keyed by OS:

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

---

## All features at a glance

- **Parallel & sequential** script execution with `dependsOn` dependencies and `maxParallel` limits
- **Directory filters** — `bnm dev frontend` or `--filter` / `-F`
- **Task filters** — `--task` / `-T` runs a single task by name; `dir` lets several named tasks share one directory
- **Pass-through arguments** — `bnm dev -- --port 3000`
- **Watch mode** — `--watch` or `"watch": true` in the config reruns the script when files change (`--no-watch` overrides); `"watch": true` on a task restarts just that task
- **Dry run** — `--dry-run` shows the execution plan
- **Timeouts & retries** — per-task `timeout` kills slow tasks, `retries` reruns flaky ones
- **Config validation** — `bnm doctor` catches broken paths and dependencies before anything runs
- **Interactive init** — `bnm init` detects directories and suggests commands
- **Prefixed, color-coded output** (`--no-color` / `NO_COLOR` to disable), plus per-task log files (`--log-dir`)
- **Run summary** — per-task status and duration, also as JSON (`--summary json`)
- **Cross-platform** commands (Windows / macOS / Linux) and `.env` support
- **Shell completion** for bash / zsh / fish, with "Did you mean ...?" typo suggestions
- **CI-friendly** — non-zero exit code on failure; **clean shutdown** — Ctrl+C terminates the whole process tree

---

## Community

- [Contributing](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License](LICENSE)
