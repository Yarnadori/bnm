# bnm

- [English](README.md)
- [日本語](README.ja.md)

[![CI](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml/badge.svg)](https://github.com/Yarnadori/bnm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Yarnadori/bnm)](https://github.com/Yarnadori/bnm/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**bnm は、複数のディレクトリにまたがるコマンドを、言語やパッケージマネージャーに関係なくまとめて実行するタスクランナーです。**

タスクは小さな JSON ファイル 1 つで定義します:

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

あとは 1 コマンドでまとめて実行するだけです:

```bash
$ bnm dev
[frontend] $ npm run dev
[backend]  $ go run .
```

両方のプロセスが並列に起動し、出力は色分けプレフィックスで区別され、Ctrl+C でプロセスツリーごとクリーンに終了します。

## bnm が向いているケース

- `frontend` と `backend`(あるいはもっと多く)を 1 コマンドで同時に起動したい
- Node.js、Go、Python、Rust などが混在している — bnm は言語を問いません
- 大規模なモノレポツール(Turborepo、Bazel、Nx)ほどの仕組みは必要ない
- 誰でも読める単純な JSON 設定でタスクを管理したい

---

## インストール

### Linux / macOS(ワンライナー)

```bash
curl -fsSL https://raw.githubusercontent.com/Yarnadori/bnm/main/install.sh | bash
```

### Windows(PowerShell ワンライナー)

```powershell
irm https://raw.githubusercontent.com/Yarnadori/bnm/main/install.ps1 | iex
```

どちらのインストールスクリプトも、リリースに同梱される `checksums.txt` を使ってダウンロードしたバイナリの SHA-256 チェックサムを検証します。

### バイナリの手動ダウンロード

[Releases](https://github.com/Yarnadori/bnm/releases) ページからプラットフォームに合ったビルド済みバイナリをダウンロードします:

| プラットフォーム      | ファイル                |
| --------------------- | ----------------------- |
| Linux (x64)           | `bnm-linux-amd64`       |
| Linux (arm64)         | `bnm-linux-arm64`       |
| macOS (x64)           | `bnm-darwin-amd64`      |
| macOS (Apple Silicon) | `bnm-darwin-arm64`      |
| Windows (x64)         | `bnm-windows-amd64.exe` |
| Windows (arm64)       | `bnm-windows-arm64.exe` |

`PATH` の通ったディレクトリ(Linux/macOS なら `/usr/local/bin` など)にバイナリを配置してください。

### ソースからビルド

```bash
git clone https://github.com/Yarnadori/bnm.git
cd bnm
go build -o bnm .
```

---

## はじめかた

プロジェクトルートで `bnm init` を実行します。サブディレクトリをスキャンし、含めるかどうかを尋ね、見つかったファイル(`package.json`、`go.mod`、`Cargo.toml` など)からコマンドを提案します:

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

CI やスクリプトなどの非対話環境では質問をスキップできます:

```bash
bnm init --yes                        # すべてデフォルトで生成
bnm init --include frontend,backend   # このディレクトリだけを対象にする
bnm init --exclude docs,tmp           # これらを除外する
bnm init --dry-run                    # 書き込まずに生成結果を表示
```

`bnm init` は既存の `bnm.json` を上書きしません。置き換えるには `--force` を使います(先に `bnm.json.bak` としてバックアップされます)。ディレクトリ一覧だけ更新したい場合は `bnm sync` を使ってください。

あとはスクリプトを実行するだけです:

```bash
bnm dev                # "dev" の全タスクを並列実行
bnm dev frontend       # frontend のタスクのみ
bnm test -- --verbose  # 各タスクコマンドに引数を追加
```

---

## コマンド

### `bnm <スクリプト名> [ディレクトリ...] [オプション] [-- 引数...]`

`bnm.json` で定義したスクリプトを実行します。

**ディレクトリ絞り込み** — 位置引数で名前を渡すと、一致するタスクだけが実行されます。名前は `directories` のキー、そのエイリアス、またはパスです。`--filter` / `-F` は同じ意味の明示的なオプションで、複数回指定できます:

```bash
bnm dev frontend backend
bnm dev --filter frontend -F backend
bnm dev ./frontend        # パス指定。'.' はルートのタスクにマッチ
```

**引数パススルー** — `--` 以降はスクリプトの各タスクコマンドの末尾に追加されます(依存スクリプトには影響しません):

```bash
bnm dev -F frontend -- --port 3000
```

**ウォッチモード** — `--watch`(または `-w`)を付けると、タスクディレクトリ配下のファイルが変更されるたびにスクリプトを再実行します。実行中のタスクは終了させてから再起動するので、dev サーバーにも使えます。隠しディレクトリや依存・ビルドディレクトリ(`node_modules`、`dist`、`target` など)は監視対象外です。Ctrl+C で終了します。

**ドライラン** — `--dry-run`(または `-n`)は、何も実行せずに実行計画(順序・モード・解決後のディレクトリとコマンド。依存スクリプトや絞り込みも反映)を表示します。

**ログファイル出力** — `--log-dir <ディレクトリ>` を付けると、各タスクの出力(プレフィックス・色なし)を `<ディレクトリ>/<スクリプト>/<タスク>.log` にも書き出します。ファイルは実行開始時に空にされ、リトライやウォッチモードの再実行では追記されます。

**JSON サマリー** — `--summary json` を付けると、サマリーの表の代わりに 1 行の JSON を出力します。CI からタスクごとの結果をパースできます:

```json
{"script":"dev","ok":true,"tasks":[{"name":"frontend","status":"ok","durationMs":812}]}
```

**色付き出力** — stdout が TTY でない場合や [`NO_COLOR`](https://no-color.org/) 環境変数が設定されている場合は自動的に無効になります。`--no-color` で強制的に無効化できます。

終了コードの挙動:

- `sequential` モードでは、最初に失敗したタスクで実行を停止します。
- 依存スクリプトが失敗した場合、残りのスクリプトはスキップされます。
- いずれかのタスクが失敗した場合、bnm は非ゼロの終了コードで終了します(CI で安全に使えます)。
- 注意: 組み込みコマンド(`init` / `sync` / `list` / `ls` / `check` / `exec` / `completion` / `help` / `version`)と同名のスクリプトはこの形式では実行できません。

### `bnm init`

`bnm.json` を対話形式で生成します([はじめかた](#はじめかた)参照)。オプション: `--yes`、`--force`、`--dry-run`、`--include a,b`、`--exclude a,b`

### `bnm sync`

`bnm.json` の `directories` セクションを現在のサブディレクトリに合わせて更新します。パスが変わっていない既存エントリは維持され、新しいディレクトリは追加、削除されたディレクトリはエントリごと削除されます。

### `bnm check`

`bnm.json` を検証し、問題があれば非ゼロ終了コードで終了します。チェック内容: JSON 構文、`mode` / `maxParallel` / `timeout` / `retries` の値、`dependsOn` の参照と循環、実在しないディレクトリパス、エイリアスの重複、解決できないタスクのディレクトリ、現在の OS 向けコマンドがないタスク。CI の最初のステップとしても便利です。

```bash
$ bnm check
[bnm] Found 2 problem(s) in bnm.json:
  - directory 'frontend': path './frontend' does not exist
  - script 'dev' task 2: no command for this OS (linux)
```

### `bnm list`

`bnm.json` で定義したディレクトリとスクリプトを一覧表示します。エイリアス: `bnm ls`

### `bnm exec <ディレクトリ> <コマンド...>`

特定のディレクトリで任意のコマンドを実行します。名前・エイリアス・パスのどれでも指定できます:

```bash
bnm exec frontend npm install
bnm exec ./frontend npm install
bnm exec . git status          # プロジェクトルート
```

### `bnm exec --all <コマンド...>`

設定済みのすべてのディレクトリで、コマンドを順番に(名前順)実行します。途中で失敗しても残りのディレクトリの実行は続き、いずれかが失敗すると bnm は非ゼロで終了します。

```bash
bnm exec --all git status
```

### `bnm completion <bash|zsh|fish>`

コマンド名・スクリプト名・ディレクトリ(`bnm exec` 用)を補完するシェル補完スクリプトを出力します。

```bash
# bash (~/.bashrc)
source <(bnm completion bash)

# zsh (~/.zshrc)
source <(bnm completion zsh)

# fish
bnm completion fish > ~/.config/fish/completions/bnm.fish
```

---

## 設定リファレンス

JSON Schema を [`schema/bnm.schema.json`](schema/bnm.schema.json) で公開しています。`bnm init` は `$schema` を自動で書き込むので、VS Code などのエディタで `bnm.json` の検証と補完が効きます。

### スクリプト

ほとんどのスクリプトは「ディレクトリ → コマンド」のマップだけで書けます。キーは `directories` の名前か、そのままのパス(`./` は省略可能、`.` はプロジェクトルート)で、デフォルトは並列実行です:

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

設定済みのすべてのディレクトリで同じコマンドを実行するなら、文字列 1 つで書けます:

```json
{
  "scripts": {
    "lint": "npx eslint ."
  }
}
```

直列実行・依存関係・並列度制限・タスク単位の設定が必要なときだけ、詳細形式を使います。タスクはディレクトリをキーにしたまま、`env` / `timeout` / `retries` を指定できます:

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

| スクリプトのフィールド | 型      | 説明                                                                      |
| ---------------------- | ------- | ------------------------------------------------------------------------- |
| `mode`                 | string  | `"parallel"`(デフォルト)または `"sequential"`                           |
| `dependsOn`            | array   | このスクリプトの前に最後まで実行されるスクリプト。循環は検出してエラー    |
| `maxParallel`          | integer | 並列モードでの同時実行タスク数の上限。`0` または省略で無制限              |
| `tasks`                | object  | ディレクトリをキーにしたタスク定義(`{dir, command, ...}` の配列も可)    |

| タスクのフィールド | 型                   | 説明                                                                      |
| ------------------ | -------------------- | -------------------------------------------------------------------------- |
| `command`          | string または object | 実行するコマンド。OS 別指定も可能(下記参照)                              |
| `env`              | object               | このタスクだけに追加で渡す環境変数                                        |
| `timeout`          | string               | 1 回の実行あたりの制限時間(例: `"30s"`、`"2m"`)。超過するとタスクを終了 |
| `retries`          | integer              | 失敗時に再実行する回数。省略時は `0`(再試行なし)                        |

### directories

`directories` は省略できます。スクリプトのキーが `directories` にない場合は、そのままパスとして扱われます。長いパスに短い名前を付けたいときだけ定義してください(キーがそのまま絞り込みや `bnm exec` の名前になります):

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

従来の `{"alias": ..., "path": ...}` オブジェクト形式も引き続き使えます。alias は別名として機能します。

### OS 別コマンド指定

`command` は OS をキーにしたオブジェクトでも書けます:

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

bnm はプロジェクトルートの `.env` を自動で読み込み、以下の変数をすべてのプロセスに渡します:

| 変数名            | 値                      |
| ----------------- | ----------------------- |
| `PROJECT_NAME`    | `bnm.json` の `name`    |
| `PROJECT_VERSION` | `bnm.json` の `version` |

各タスクには、さらに以下が優先度の低い順に適用されます(後のものが優先):

1. プロセスの環境変数+プロジェクトルートの `.env`
2. タスクの実行ディレクトリの `.env`(例: `frontend/.env`)
3. `bnm.json` のタスク `env` エントリ

---

## 機能一覧

- **並列・直列**実行、`dependsOn` によるスクリプト依存、`maxParallel` による並列度制限
- **ディレクトリ絞り込み** — `bnm dev frontend` または `--filter` / `-F`
- **引数パススルー** — `bnm dev -- --port 3000`
- **ウォッチモード** — `--watch` でファイル変更時に再実行
- **ドライラン** — `--dry-run` で実行計画を表示
- **タイムアウト・リトライ** — タスク単位の `timeout` と `retries`
- **設定の検証** — `bnm check` で実行前に問題を検出
- **対話式の初期化** — `bnm init` がディレクトリとコマンドを自動検出
- 出力の**色分けプレフィックス表示**(`--no-color` / `NO_COLOR` で無効化)と、タスク別**ログファイル**(`--log-dir`)
- **実行サマリー** — タスクごとの成否と所要時間。JSON 出力(`--summary json`)にも対応
- **クロスプラットフォーム**対応(Windows / macOS / Linux)と `.env` サポート
- bash / zsh / fish の**シェル補完**、typo 時の「Did you mean ...?」サジェスト
- **CI フレンドリー** — 失敗時は非ゼロ終了コード。**クリーンな終了** — Ctrl+C でプロセスツリー全体を終了

---

## コミュニティ

- [Contributing](CONTRIBUTING.md)
- [Security Policy](SECURITY.md)
- [Support](SUPPORT.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [License](LICENSE)
