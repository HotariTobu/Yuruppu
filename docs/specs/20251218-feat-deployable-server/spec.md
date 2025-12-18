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
  - コンテナイメージをビルドできること（ADR 20251218-container-build）
  - CI/CDで自動デプロイできること（ADR 20251218-cicd）

### Non-Functional Requirements

- [ ] NFR-001: Cloud Runで動作すること
  - `PORT`環境変数に対応すること（Cloud Runが自動設定）
  - graceful shutdownは不要（Cloud Runが管理）

- [ ] NFR-002: 手動デプロイが可能であること
  - ドキュメント化されたコマンドでデプロイできること

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
go run .
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

### AC-008: コンテナイメージビルド [FR-005]

- **Given**: ビルド環境が整っている
- **When**: コンテナイメージをビルドする
- **Then**: イメージが正常にビルドされる

### AC-009: 手動デプロイ [FR-005, NFR-002]

- **Given**: コンテナイメージがビルド済み
- **When**: READMEの手順に従ってデプロイする
- **Then**: Cloud Runでアプリケーションが動作する

### AC-010: 自動デプロイ [FR-005]

- **Given**: CI/CDが設定済み
- **When**: mainブランチにpushする
- **Then**: Cloud Runサービスが自動的に更新される

## Implementation Notes

関連ADR:
- 20251217-project-structure.md
- 20251217-configuration.md
- 20251217-logging.md
- 20251218-container-build.md
- 20251218-cicd.md

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-18 | 1.0 | Initial version | - |
| 2025-12-18 | 1.1 | Update FR-005 to use ko instead of Dockerfile, add Cloud Build CI/CD | - |
