# bnm

[![CI](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml/badge.svg)](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Yarnadori/bnm)](https://github.com/Yarnadori/bnm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

bnm is a task runner designed to streamline command execution and script management in projects with multiple directories, such as monorepos or full-stack applications.

---

## Features

- **Initialize** a project with auto-detected subdirectories
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
- Note: script names that collide with built-in commands (`init`, `list`, `ls`, `exec`, `help`, `version`) cannot be invoked this way.

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

## Security

`bnm.json` and `.env` define commands and environment for execution — treat them as code, just like `npm run` scripts or a `Makefile`. See [SECURITY.md](SECURITY.md) for the security policy and how to report vulnerabilities.

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines. This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).

## License

[MIT](LICENSE)

---

---

## bnm（日本語）

bnm は、モノレポやフルスタックアプリケーションなど、複数ディレクトリを持つプロジェクトでのコマンド実行・スクリプト管理を効率化するタスクランナーです。

---

## 特徴

- サブディレクトリを自動検出してプロジェクトを**初期化**
- `bnm.json` で定義したスクリプトを**並列・直列**で実行
- エイリアスやパスで任意のディレクトリに**コマンドを実行**
- `bnm list` でディレクトリ・スクリプト定義を**一覧表示**
- **クロスプラットフォーム**対応（Windows / macOS / Linux）
- `.env` を自動読み込みし、`PROJECT_NAME` / `PROJECT_VERSION` を**環境変数として提供**
- 各プロセスの出力をディレクトリ名で**色分けプレフィックス表示**
- **CI フレンドリー** — タスク失敗時は非ゼロ終了コード、直列モードは最初の失敗で停止
- **クリーンな終了** — Ctrl+C で各タスクのプロセスツリー全体を確実に終了

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

どちらのインストールスクリプトも、リリースに同梱される `checksums.txt` を使ってダウンロードしたバイナリの SHA-256 チェックサムを検証します。

### バイナリの手動ダウンロード

[Releases](https://github.com/Yarnadori/bnm/releases) ページからお使いのプラットフォーム向けバイナリをダウンロードしてください：

| プラットフォーム      | ファイル名              |
| --------------------- | ----------------------- |
| Linux (x64)           | `bnm-linux-amd64`       |
| Linux (arm64)         | `bnm-linux-arm64`       |
| macOS (x64)           | `bnm-darwin-amd64`      |
| macOS (Apple Silicon) | `bnm-darwin-arm64`      |
| Windows (x64)         | `bnm-windows-amd64.exe` |
| Windows (arm64)       | `bnm-windows-arm64.exe` |

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

カレントディレクトリに `bnm.json` を作成してプロジェクトを初期化します。サブディレクトリが自動的にスキャンされます（`.git` などの隠しディレクトリと、`node_modules` / `dist` / `build` / `vendor` などの依存・ビルド成果物ディレクトリは除外）。

### `bnm list`

`bnm.json` で定義したディレクトリとスクリプトを一覧表示します。エイリアス: `bnm ls`

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

### `bnm <スクリプト名>`

`bnm.json` で定義したスクリプトを実行します。

```bash
bnm dev
bnm build
```

終了コードの挙動:

- `sequential` モードでは、最初に失敗したタスクで実行を停止します。
- いずれかのタスクが失敗した場合、bnm は非ゼロの終了コードで終了します（CI で安全に使えます）。
- 注意: 組み込みコマンド（`init` / `list` / `ls` / `exec` / `help` / `version`）と同名のスクリプトはこの形式では実行できません。

### `bnm exec <ディレクトリ> <コマンド...>`

任意のディレクトリでコマンドをその場で実行します。

```bash
# エイリアス指定（- プレフィックス）
bnm exec -F npm install

# ディレクトリ名指定
bnm exec FRONTEND npm install

# パス指定
bnm exec ./frontend npm install
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

環境変数 `NO_COLOR` を設定すると、プレフィックスの色分けを無効にできます。出力先がターミナルでない場合は自動的に無効になります。

---

## セキュリティ

`bnm.json` と `.env` は実行するコマンドと環境を定義するファイルです。`npm run` のスクリプトや `Makefile` と同様に、コードとして扱ってください。セキュリティポリシーと脆弱性の報告方法は [SECURITY.md](SECURITY.md) を参照してください。

## コントリビュート

コントリビューションを歓迎します！開発環境のセットアップとガイドラインは [CONTRIBUTING.md](CONTRIBUTING.md) を参照してください。本プロジェクトは [Contributor Covenant 行動規範](CODE_OF_CONDUCT.md) に従います。

## ライセンス

[MIT](LICENSE)

---
