# Feature: Deployable Server

## Overview

main.goを作成し、HTTPサーバーを起動してCloud Runにデプロイ可能な状態にする。インフラコード（Dockerfileなど）も含める。

## Background & Purpose

`internal/bot/bot.go`にコアロジック（Webhookハンドラー、署名検証など）は実装済みだが、エントリーポイントとなる`main.go`がないため、アプリケーションを起動できない状態。デプロイ可能にするため、main.goを作成しサーバーを起動する。また、Cloud Runにデプロイするためのインフラコードも作成する。

## Requirements

### Functional Requirements

- [ ] FR-001: 環境変数からLINE認証情報を読み込む
  - `LINE_CHANNEL_SECRET`と`LINE_CHANNEL_ACCESS_TOKEN`を`os.Getenv()`で読み込む
  - いずれかが空の場合、エラーメッセージを出力してプログラムを終了する

- [ ] FR-002: Botインスタンスを初期化する
  - `bot.NewBot()`を呼び出してBotを作成する
  - 初期化失敗時はエラーメッセージを出力してプログラムを終了する

- [ ] FR-003: HTTPサーバーを起動する
  - `PORT`環境変数からポートを読み込む（デフォルト: `8080`）
  - `/webhook`エンドポイントに`bot.HandleWebhook`をハンドラーとして登録する
  - サーバー起動時に起動メッセージをログ出力する

- [ ] FR-004: Botとロガーをパッケージレベルで設定する
  - `bot.SetDefaultBot()`でBotインスタンスを設定する
  - `bot.SetLogger()`でロガーを設定する

- [ ] FR-005: インフラコードを作成する
  - Dockerfileを作成する（必須）
  - `docker build`でコンテナイメージをビルドできること
  - CI/CD構成（Cloud Build、GitHub Actionsなど）は`/design`ステップで決定

### Non-Functional Requirements

- [ ] NFR-001: Cloud Runで動作すること
  - `PORT`環境変数に対応すること（Cloud Runが自動設定）
  - graceful shutdownは不要（Cloud Runが管理）

- [ ] NFR-002: 手動デプロイが可能であること
  - ドキュメント化されたコマンドでデプロイできること
  - `docker build`と`gcloud run deploy`でデプロイ可能

## API Design

### main.go Structure

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/takato/yuruppu/internal/bot"
)

func main() {
    // 1. Load configuration from environment variables
    // 2. Initialize bot
    // 3. Set default bot and logger
    // 4. Register webhook handler
    // 5. Start HTTP server
}
```

### Type Definitions

既存の`internal/bot`パッケージを使用。新しい型は不要。

## Usage Examples

```bash
# Run locally
export LINE_CHANNEL_SECRET="your-secret"
export LINE_CHANNEL_ACCESS_TOKEN="your-token"
export PORT="8080"
go run main.go

# Build and run
go build -o yuruppu
./yuruppu

# Docker build and run
docker build -t yuruppu .
docker run -e LINE_CHANNEL_SECRET="your-secret" \
           -e LINE_CHANNEL_ACCESS_TOKEN="your-token" \
           -e PORT=8080 \
           -p 8080:8080 \
           yuruppu

# Deploy to Cloud Run
gcloud run deploy yuruppu \
  --source . \
  --region asia-northeast1 \
  --set-env-vars LINE_CHANNEL_SECRET=...,LINE_CHANNEL_ACCESS_TOKEN=...
```

## Error Handling

| Error Type | Condition | Action |
|------------|-----------|--------|
| ConfigError | LINE_CHANNEL_SECRET が空 | エラーログ出力、exit 1 |
| ConfigError | LINE_CHANNEL_ACCESS_TOKEN が空 | エラーログ出力、exit 1 |
| InitError | bot.NewBot() 失敗 | エラーログ出力、exit 1 |
| ServerError | http.ListenAndServe() 失敗 | エラーログ出力、exit 1 |

## Acceptance Criteria

### AC-001: 環境変数読み込み [FR-001]

- **Given**: `LINE_CHANNEL_SECRET`と`LINE_CHANNEL_ACCESS_TOKEN`が設定されている
- **When**: アプリケーションを起動する
- **Then**:
  - エラーなく起動処理が継続する
  - 環境変数の値がBot初期化に使用される

### AC-002a: LINE_CHANNEL_SECRET不足エラー [FR-001, Error]

- **Given**: `LINE_CHANNEL_SECRET`が設定されていない
- **When**: アプリケーションを起動する
- **Then**:
  - エラーメッセージ「LINE_CHANNEL_SECRET is required」が出力される
  - プログラムが終了する（exit code 1）

### AC-002b: LINE_CHANNEL_ACCESS_TOKEN不足エラー [FR-001, Error]

- **Given**: `LINE_CHANNEL_ACCESS_TOKEN`が設定されていない
- **When**: アプリケーションを起動する
- **Then**:
  - エラーメッセージ「LINE_CHANNEL_ACCESS_TOKEN is required」が出力される
  - プログラムが終了する（exit code 1）

### AC-003: Bot初期化 [FR-002]

- **Given**: 有効な認証情報が設定されている
- **When**: アプリケーションを起動する
- **Then**:
  - `bot.NewBot()`が呼び出される
  - Botインスタンスが正常に作成される

### AC-004: HTTPサーバー起動 [FR-003]

- **Given**: Botが正常に初期化されている
- **When**: アプリケーションを起動する
- **Then**:
  - 指定ポートでHTTPサーバーが起動する
  - `/webhook`エンドポイントがアクセス可能になる
  - 起動メッセージ「Server listening on port {PORT}」がログ出力される

### AC-005: デフォルトポート [FR-003]

- **Given**: `PORT`環境変数が設定されていない
- **When**: アプリケーションを起動する
- **Then**:
  - ポート`8080`でサーバーが起動する

### AC-006: カスタムポート [FR-003]

- **Given**: `PORT`環境変数が`3000`に設定されている
- **When**: アプリケーションを起動する
- **Then**:
  - ポート`3000`でサーバーが起動する

### AC-007: パッケージレベル設定 [FR-004]

- **Given**: Botが正常に初期化されている
- **When**: アプリケーションを起動する
- **Then**:
  - `bot.SetDefaultBot()`が呼び出される
  - `bot.SetLogger()`が呼び出される
  - Webhookハンドラーがこれらの設定を使用できる

### AC-008: Dockerイメージビルド [FR-005]

- **Given**: Dockerfileが存在する
- **When**: `docker build -t yuruppu .`を実行する
- **Then**:
  - イメージが正常にビルドされる
  - イメージにアプリケーションバイナリが含まれる

### AC-009: コンテナ実行 [FR-005, NFR-001]

- **Given**: Dockerイメージがビルド済み
- **When**: `docker run -e LINE_CHANNEL_SECRET=... -e LINE_CHANNEL_ACCESS_TOKEN=... -e PORT=8080 yuruppu`を実行する
- **Then**:
  - コンテナが起動する
  - アプリケーションが`PORT`環境変数で指定されたポートでリッスンする

## Implementation Notes

- ADR 20251217-project-structure.md に従い、`main.go`はルートディレクトリに配置
- ADR 20251217-configuration.md に従い、`os.Getenv()`を直接使用
- ADR 20251217-logging.md に従い、標準ライブラリの`log`または`log/slog`を使用
- `internal/bot`パッケージの既存機能を活用し、main.goは薄いエントリーポイントとする

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-18 | 1.0 | Initial version | - |
