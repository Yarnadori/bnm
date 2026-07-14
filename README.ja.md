# bnm

- [English](README.md)
- [日本語](README.ja.md)

[![CI](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml/badge.svg)](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Yarnadori/bnm)](https://github.com/Yarnadori/bnm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

bnm は、モノレポやフルスタックアプリケーションなど、複数ディレクトリを持つプロジェクトでのコマンド実行・スクリプト管理を効率化するタスクランナーです。

---

## 特徴

- サブディレクトリを自動検出してプロジェクトを**初期化**
- プロジェクトフォルダの追加・削除に合わせて `bnm.json` のディレクトリ定義を**同期**
- `bnm.json` で定義したスクリプトを**並列・直列**で実行
- **スクリプト依存関係** — `dependsOn` で前提スクリプトを先に実行（循環は検出してエラー）
- **ディレクトリ絞り込み** — `bnm dev -F` で特定ディレクトリのタスクのみ実行
- **引数パススルー** — `bnm dev -- --port 3000` で `--` 以降を各タスクコマンドに追加
- **ウォッチモード** — `bnm dev --watch` でタスクディレクトリのファイル変更時にスクリプトを再実行
- **ドライラン** — `bnm deploy --dry-run` で実行せずに実行計画を表示
- **タイムアウト・リトライ** — タスク単位の `timeout` で長すぎるタスクを強制終了、`retries` で失敗時に再試行
- **設定の検証** — `bnm check` でパス・エイリアス・依存関係の問題を実行前に検出
- **並列度制限** — `maxParallel` で同時実行タスク数を制御
- エイリアスやパスで任意のディレクトリに**コマンドを実行**（`exec --all` で全ディレクトリ一括も可能）
- `bnm list` でディレクトリ・スクリプト定義を**一覧表示**
- **クロスプラットフォーム**対応（Windows / macOS / Linux）
- `.env` を自動読み込み（プロジェクトルート＋各ディレクトリ）し、タスク単位の `env` にも対応。`PROJECT_NAME` / `PROJECT_VERSION` を**環境変数として提供**
- 各プロセスの出力をディレクトリ名で**色分けプレフィックス表示**
- **実行サマリー** — スクリプト実行後にタスクごとの成否と所要時間を表示
- **シェル補完** — `bnm completion bash|zsh|fish`、typo 時の「Did you mean ...?」サジェスト付き
- **エディタ対応** — `bnm.json` の [JSON Schema](schema/bnm.schema.json) を公開（`bnm init` が `$schema` を自動付与）
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

### `bnm init`

カレントディレクトリに `bnm.json` を作成してプロジェクトを初期化します。サブディレクトリが自動的にスキャンされます（`.git` などの隠しディレクトリと、`node_modules` / `dist` / `build` / `vendor` などの依存・ビルド成果物ディレクトリは除外）。

### `bnm sync`

現在のサブディレクトリに合わせて、`bnm.json` の `directories` を更新します。変更のないディレクトリは既存のエイリアスを維持し、新しいディレクトリにはエイリアスを自動生成し、削除されたディレクトリは `directories` から削除します。

### `bnm check`

`bnm.json` を検証し、問題があれば非ゼロ終了コードで終了します。チェック内容: JSON 構文、`mode` / `maxParallel` / `timeout` / `retries` の値、`dependsOn` の参照と循環、実在しないディレクトリパス、エイリアスの重複、解決できないタスクの `dir`、現在の OS 向けコマンドがないタスク。CI の最初のステップとしても便利です。

```bash
$ bnm check
[bnm] Found 2 problem(s) in bnm.json:
  - directory 'FRONTEND': path './frontend' does not exist
  - script 'dev' task 2: no command for this OS (linux)
```

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

### `bnm <スクリプト名> [ディレクトリ...] [-- 引数...]`

`bnm.json` で定義したスクリプトを実行します。

```bash
bnm dev
bnm build
```

**ディレクトリ絞り込み** — 特定ディレクトリのタスクのみ実行します（`-` 付きエイリアス、ディレクトリキー、パスのいずれでも指定可能）：

```bash
bnm dev -F                # FRONTEND のタスクのみ
bnm dev FRONTEND BACKEND  # 複数指定
bnm dev ./frontend        # パス指定。'.' はルートのタスクにマッチ
```

**引数パススルー** — `--` 以降はスクリプトの各タスクコマンドの末尾に追加されます（依存スクリプトには影響しません）：

```bash
bnm test -- --watch
bnm dev -F -- --port 3000
```

**ウォッチモード** — `--watch`（または `-w`）を付けると、タスクディレクトリ配下のファイルが変更されるたびにスクリプトを再実行します。実行中のタスクは終了させてから再起動するので、dev サーバーにも使えます。隠しディレクトリや依存・ビルドディレクトリ（`node_modules`、`dist`、`target` など）は監視対象外です。Ctrl+C で終了します。

```bash
bnm test --watch
bnm dev -F --watch
```

**ドライラン** — `--dry-run`（または `-n`）は、何も実行せずに実行計画（順序・モード・解決後のディレクトリとコマンド。依存スクリプトや絞り込みも反映）を表示します：

```bash
bnm deploy --dry-run
```

スクリプトに `dependsOn` がある場合、それらのスクリプトが先に最後まで実行されます（[スクリプトグループ](#スクリプトグループ)参照）。

実行後には、タスクごとの状態（`ok` / `failed` / `skipped` / `canceled`）と所要時間のサマリーが表示されます。

終了コードの挙動:

- `sequential` モードでは、最初に失敗したタスクで実行を停止します。
- 依存スクリプトが失敗した場合、残りのスクリプトはスキップされます。
- いずれかのタスクが失敗した場合、bnm は非ゼロの終了コードで終了します（CI で安全に使えます）。
- 注意: 組み込みコマンド（`init` / `sync` / `list` / `ls` / `check` / `exec` / `completion` / `help` / `version`）と同名のスクリプトはこの形式では実行できません。

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

### `bnm exec --all <コマンド...>`

設定されたすべてのディレクトリで同じコマンドを順番に実行します（ディレクトリキーのソート順）。途中で失敗しても残りのディレクトリは実行され、いずれかが失敗すると非ゼロ終了コードになります。

```bash
bnm exec --all git status
bnm exec --all npm install
```

### `bnm completion <bash|zsh|fish>`

シェル補完スクリプトを出力します。コマンド名・スクリプト名・（`bnm exec` の）ディレクトリを補完できます。

```bash
# bash（~/.bashrc）
source <(bnm completion bash)

# zsh（~/.zshrc）
source <(bnm completion zsh)

# fish
bnm completion fish > ~/.config/fish/completions/bnm.fish
```

---

## bnm.json リファレンス

| フィールド    | 型     | 説明                                             |
| ------------- | ------ | ------------------------------------------------ |
| `$schema`     | string | エディタ検証用の JSON Schema URL（任意）         |
| `name`        | string | プロジェクト名。`PROJECT_NAME` として渡される    |
| `version`     | string | バージョン。`PROJECT_VERSION` として渡される     |
| `directories` | object | ディレクトリ定義（エイリアスとパス）             |
| `scripts`     | object | スクリプト定義（モード・依存関係・タスク一覧）   |

JSON Schema は [`schema/bnm.schema.json`](schema/bnm.schema.json) として公開されています。`bnm init` は `$schema` を自動で書き込むため、VS Code などのエディタで `bnm.json` の検証・補完が効きます。

### ディレクトリエントリ

| フィールド | 型     | 説明                               |
| ---------- | ------ | ---------------------------------- |
| `alias`    | string | `bnm exec -X` で使う短縮エイリアス |
| `path`     | string | ディレクトリへの相対パス           |

### スクリプトグループ

| フィールド    | 型      | 説明                                                              |
| ------------- | ------- | ----------------------------------------------------------------- |
| `mode`        | string  | `"parallel"`（デフォルト）または `"sequential"`                   |
| `dependsOn`   | array   | このスクリプトの前に実行するスクリプト名。循環は検出してエラー   |
| `maxParallel` | integer | 並列モードでの同時実行タスク数の上限。`0` または省略で無制限     |
| `tasks`       | array   | タスクの一覧                                                      |

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

### タスク

| フィールド | 型                   | 説明                                                                     |
| ---------- | -------------------- | ------------------------------------------------------------------------ |
| `dir`      | string               | `directories` のキー名                                                   |
| `command`  | string または object | OS 別指定も可能（下記参照）                                              |
| `env`      | object               | このタスクだけに追加で渡す環境変数                                       |
| `timeout`  | string               | 1 回の実行あたりの制限時間（例: `"30s"`、`"2m"`）。超過するとタスクを終了 |
| `retries`  | integer              | 失敗時に再実行する回数。省略時は `0`（再試行なし）                       |

```json
{
  "tasks": [
    { "dir": "BACKEND", "command": "npm run test:e2e", "timeout": "5m", "retries": 2 }
  ]
}
```

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

各タスクには、さらに以下が優先度の低い順に適用されます（後のものが優先）：

1. プロセスの環境変数＋プロジェクトルートの `.env`
2. タスクの実行ディレクトリの `.env`（例: `frontend/.env`）
3. `bnm.json` のタスク `env` エントリ

環境変数 `NO_COLOR` を設定すると、プレフィックスの色分けを無効にできます。出力先がターミナルでない場合は自動的に無効になります。

---

## コミュニティ

- [Contributing](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License](LICENSE)
