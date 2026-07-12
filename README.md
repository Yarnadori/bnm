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
- **Execute arbitrary commands** in any configured directory via alias or path
- **List** configured directories and scripts with `bnm list`
- **Cross-platform** command support (Windows / macOS / Linux)
- **Environment variables** — loads `.env` automatically and exposes `PROJECT_NAME` / `PROJECT_VERSION`
- **Prefixed, color-coded output** — each process output is labeled with its directory name
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

### `bnm <script>`

Runs a script defined in `bnm.json`.

```bash
bnm dev
bnm build
```

Exit code behavior:

- In `sequential` mode, execution stops at the first failing task.
- bnm exits with a non-zero code if any task fails, so scripts are safe to use in CI.
- Note: script names that collide with built-in commands (`init`, `sync`, `list`, `ls`, `exec`, `help`, `version`) cannot be invoked this way.

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

---

## bnm.json Reference

| Field         | Type   | Description                                   |
| ------------- | ------ | --------------------------------------------- |
| `name`        | string | Project name. Exposed as `PROJECT_NAME`       |
| `version`     | string | Project version. Exposed as `PROJECT_VERSION` |
| `directories` | object | Named directory entries with alias and path   |
| `scripts`     | object | Named script groups with mode and tasks       |

### Directory entry

| Field   | Type   | Description                         |
| ------- | ------ | ----------------------------------- |
| `alias` | string | Short alias used with `bnm exec -X` |
| `path`  | string | Relative path to the directory      |

### Script group

| Field   | Type   | Description                              |
| ------- | ------ | ---------------------------------------- |
| `mode`  | string | `"parallel"` (default) or `"sequential"` |
| `tasks` | array  | List of tasks to run                     |

### Task

| Field     | Type             | Description                                    |
| --------- | ---------------- | ---------------------------------------------- |
| `dir`     | string           | Directory key from `directories`               |
| `command` | string or object | Command to run. Can be OS-specific (see below) |

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

Set `NO_COLOR` to any value to disable colored output prefixes. Colors are also disabled automatically when output is not a terminal.

---

## Community

- [Contributing](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License](LICENSE)
