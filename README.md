# bnm

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

# In current directory
bnm exec . npm install
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

---

## bnm（日本語）

bnm は、モノレポやフルスタックアプリケーションなど、複数ディレクトリを持つプロジェクトでのコマンド実行・スクリプト管理を効率化するタスクランナーです。

---

## 特徴

- サブディレクトリを自動検出してプロジェクトを**初期化**
- `bnm.json` で定義したスクリプトを**並列・直列**で実行
- エイリアスやパスで任意のディレクトリに**コマンドを実行**
- **クロスプラットフォーム**対応（Windows / macOS / Linux）
- `.env` を自動読み込みし、`PROJECT_NAME` / `PROJECT_VERSION` を**環境変数として提供**
- 各プロセスの出力をディレクトリ名で**プレフィックス表示**

---

## インストール

### Linux / macOS（ワンライナー）

```bash
curl -fsSL https://raw.githubusercontent.com/Yarnadori/bnm/main/install.sh | bash
```

### Windows（PowerShell ワンライナー）

```powershell
irm https://raw.githubusercontent.com/Yarnadori/bnm/main/install.ps1 | iex
```

### Go ユーザー向け

```bash
go install github.com/Yarnadori/bnm@latest
```

### バイナリの手動ダウンロード

[Releases](https://github.com/Yarnadori/bnm/releases) ページからお使いのプラットフォーム向けバイナリをダウンロードしてください：

| プラットフォーム      | ファイル名              |
| --------------------- | ----------------------- |
| Linux (x64)           | `bnm-linux-amd64`       |
| Linux (arm64)         | `bnm-linux-arm64`       |
| macOS (x64)           | `bnm-darwin-amd64`      |
| macOS (Apple Silicon) | `bnm-darwin-arm64`      |
| Windows (x64)         | `bnm-windows-amd64.exe` |

ダウンロード後、バイナリを `PATH` の通ったディレクトリに移動してください（Linux/macOS では `/usr/local/bin` など）。

### ソースからビルド

```bash
git clone https://github.com/Yarnadori/bnm.git
cd bnm
go build -o bnm .
```

---

## はじめかた

### 1. 初期化

プロジェクトルートで `bnm init` を実行します。サブディレクトリが自動検出され、`bnm.json` が生成されます。

```bash
bnm init
```

**例 — `frontend/` と `backend/` があるプロジェクト：**

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

### 2. スクリプトを定義する

`bnm.json` にスクリプトを追加します：

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

### 3. スクリプトを実行する

```bash
bnm dev    # "dev" タスクを並列実行
bnm build  # "build" タスクを直列実行
```

---

## コマンド

### bnm init（初期化コマンド）

カレントディレクトリに `bnm.json` を作成してプロジェクトを初期化します。サブディレクトリが自動的にスキャンされます（`.git` などの隠しディレクトリは除外）。

### `bnm <スクリプト名>`

`bnm.json` で定義したスクリプトを実行します。

```bash
bnm dev
bnm build
```

### `bnm exec <ディレクトリ> <コマンド...>`

任意のディレクトリでコマンドをその場で実行します。

```bash
# エイリアス指定（- プレフィックス）
bnm exec -F npm install

# ディレクトリ名指定
bnm exec FRONTEND npm install

# パス指定
bnm exec ./frontend npm install

# カレントディレクトリ
bnm exec . npm install
```

---

## bnm.json リファレンス

| フィールド    | 型     | 説明                                          |
| ------------- | ------ | --------------------------------------------- |
| `name`        | string | プロジェクト名。`PROJECT_NAME` として渡される |
| `version`     | string | バージョン。`PROJECT_VERSION` として渡される  |
| `directories` | object | ディレクトリ定義（エイリアスとパス）          |
| `scripts`     | object | スクリプト定義（モードとタスク一覧）          |

### ディレクトリエントリ

| フィールド | 型     | 説明                               |
| ---------- | ------ | ---------------------------------- |
| `alias`    | string | `bnm exec -X` で使う短縮エイリアス |
| `path`     | string | ディレクトリへの相対パス           |

### スクリプトグループ

| フィールド | 型     | 説明                                            |
| ---------- | ------ | ----------------------------------------------- |
| `mode`     | string | `"parallel"`（デフォルト）または `"sequential"` |
| `tasks`    | array  | タスクの一覧                                    |

### タスク

| フィールド | 型                   | 説明                        |
| ---------- | -------------------- | --------------------------- |
| `dir`      | string               | `directories` のキー名      |
| `command`  | string または object | OS 別指定も可能（下記参照） |

### OS 別コマンド指定

```json
{
  "command": {
    "windows": "echo Windows で実行",
    "mac": "echo macOS で実行",
    "linux": "echo Linux で実行",
    "default": "echo フォールバック"
  }
}
```

---

## 環境変数

bnm はプロジェクトルートの `.env` を自動で読み込み、以下の変数をすべてのプロセスに渡します：

| 変数名            | 値                      |
| ----------------- | ----------------------- |
| `PROJECT_NAME`    | `bnm.json` の `name`    |
| `PROJECT_VERSION` | `bnm.json` の `version` |

---
