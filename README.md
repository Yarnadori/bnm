# bnm

- [English](README.md)
- [日本語](README.ja.md)

bnm is a task runner designed to streamline command execution and script management in projects with multiple directories, such as monorepos or full-stack applications.

---

## Features

- **Initialize** a project with auto-detected subdirectories
- **Run scripts** defined in `bnm.json` in parallel or sequential mode
- **Execute arbitrary commands** in any configured directory via alias or path
- **Cross-platform** command support (Windows / macOS / Linux)
- **Environment variables** — loads `.env` automatically and exposes `PROJECT_NAME` / `PROJECT_VERSION`
- **Prefixed output** — each process output is labeled with its directory name

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

Initializes the project by creating `bnm.json` in the current directory. Subdirectories are scanned automatically. Hidden directories (`.git`, etc.) are excluded.

### `bnm <script>`

Runs a script defined in `bnm.json`.

```bash
bnm dev
bnm build
```

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

---
