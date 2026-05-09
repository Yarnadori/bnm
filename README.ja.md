# bnm

- [English](README.md)
- [日本語](README.ja.md)

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
