# gotion

Notion API を操作するための CLI ツール。

## インストール

```bash
go install github.com/longkey1/gotion@latest
```

または、リポジトリをクローンしてビルド:

```bash
git clone https://github.com/longkey1/gotion.git
cd gotion
go build -o gotion .
```

## 認証

### 方法1: MCP OAuth (推奨)

事前の設定なしで認証できます。

```bash
gotion auth --mcp
```

ブラウザが開き、Notion アカウントで認証を行います。認証情報は `~/.config/gotion/token.json` に保存されます。

### 方法2: 従来の OAuth

Notion Integration を作成して認証します。

1. [Notion Integrations](https://www.notion.so/my-integrations) で Public Integration を作成
2. Redirect URI に `http://localhost:8080/callback` を設定
3. Client ID と Client Secret を取得

環境変数または設定ファイルで認証情報を設定:

```bash
# 環境変数
export GOTION_CLIENT_ID="your-client-id"
export GOTION_CLIENT_SECRET="your-client-secret"

# または設定ファイル (~/.config/gotion/config.toml)
client_id = "your-client-id"
client_secret = "your-client-secret"
```

認証を実行:

```bash
gotion auth
```

### 方法3: トークン直接指定

Internal Integration のトークンを直接使用:

```bash
export GOTION_TOKEN="secret_xxxxxxxx"
# または
export NOTION_TOKEN="secret_xxxxxxxx"
```

## 使い方

### ページの検索

```bash
# キーワードで検索
gotion list -q "検索キーワード"

# 件数を指定
gotion list -q "検索キーワード" -n 20

# JSON 形式で出力
gotion list -q "検索キーワード" -f json

# テーブル形式で出力 (デフォルト)
gotion list -q "検索キーワード" -f table
```

### ページの取得

```bash
# ページ ID を指定して取得
gotion get <page_id>

# JSON 形式で出力
gotion get <page_id> -f json

# 特定のプロパティのみ取得
gotion get <page_id> --filter-properties "title,status"
```

## コマンド一覧

| コマンド | 説明 |
|---------|------|
| `auth` | Notion アカウントで認証 |
| `list` | ページを検索・一覧表示 |
| `get` | ページの詳細を取得 |
| `version` | バージョン情報を表示 |

## 設定ファイル

認証情報は以下の場所に保存されます:

- トークン: `~/.config/gotion/token.json`
- 設定: `~/.config/gotion/config.toml`

## ライセンス

MIT
