# Refactor: Codebase Cleanup

## Overview

コードベース全体のリファクタリング。不要なラッパー関数の削除、コード重複の解消、テストコードの分離を行う。

## Background & Purpose

コードレビューにより以下の問題が特定された：
- 不要な抽象化レイヤー（init系ラッパー関数）
- 重複コード（メタデータ取得、型アサーション）
- テスト用モック型が本番コードに混在

これらを解消し、コードの保守性と可読性を向上させる。

## Current Structure

### main.go (lines 82-112)
`initServer`、`initClient`、`initLLM`の3つのラッパー関数が存在。各関数はconfigのnilチェック後、コンストラクタを呼び出すだけ。`loadConfig()`で既にバリデーション済みのためnilチェックは冗長。

### internal/llm/vertexai.go (lines 170-274)
`getRegionFromMetadata`と`getProjectIDFromMetadata`が95%同じコード。HTTPクライアント作成、リクエスト送信、レスポンス読み取りが重複。

### internal/llm/errors.go (lines 13-60)
`MockAPIError`、`MockNetError`、`MockDNSError`などのテスト用モック型がexportedで本番コードに混在。

### internal/line/server.go (lines 85-140)
webhookイベント処理でポインタ型と値型の両方をハンドリングするため、同じロジックが重複。具体的には`webhook.TextMessageContent`と`*webhook.TextMessageContent`を別々のcaseで同一処理している。

## Proposed Structure

### main.go
init系ラッパー関数を削除し、main()内でコンストラクタを直接呼び出す。エラーハンドリングは現状と同様、ログ出力後os.Exit(1)。

### internal/llm/vertexai.go
unexportedな`fetchMetadata(baseURL, endpoint string, parser func(string) string) string`関数を抽出し、重複を解消。

### internal/llm/errors.go
モック型を`internal/llm/vertexai_test.go`に移動し、unexported（小文字）に変更。

### internal/line/server.go
ポインタ型と値型を統一的に処理するヘルパー関数を導入し、各メッセージタイプ（text, image, sticker等）が単一のcase文で処理されるようにする。

## Scope

- [x] SC-001: `main.go` - init系ラッパー関数の削除
- [x] SC-002: `internal/llm/vertexai.go` - メタデータ取得関数の重複解消
- [x] SC-003: `internal/llm/errors.go` - モック型のテストファイル移動
- [x] SC-004: `internal/line/server.go` - 型アサーション重複の整理

## Breaking Changes

None - 外部APIに変更なし。内部リファクタリングのみ。

## Acceptance Criteria

### AC-001: init系ラッパー関数の削除 [SC-001]

- **Given**: main.goにinitServer、initClient、initLLM関数が存在する
- **When**: リファクタリング完了後
- **Then**:
  - initServer、initClient、initLLM関数が削除されている
  - main()内でline.NewServer、line.NewClient、llm.NewVertexAIClientが直接呼び出されている
  - エラーハンドリングは現状維持（ログ出力後os.Exit(1)）
  - 既存のテストがすべてパスする

### AC-002: メタデータ取得関数の統合 [SC-002]

- **Given**: getRegionFromMetadataとgetProjectIDFromMetadataが別々に存在する
- **When**: リファクタリング完了後
- **Then**:
  - unexportedなfetchMetadata関数が存在し、HTTPリクエストロジックを共通化している
  - getRegionFromMetadataとgetProjectIDFromMetadataがfetchMetadataを使用する
  - 重複コード約40行が削減される
  - 既存のテストがすべてパスする

### AC-003: モック型のテストファイル移動 [SC-003]

- **Given**: errors.goにMockAPIError、MockNetError、MockDNSErrorが存在する
- **When**: リファクタリング完了後
- **Then**:
  - errors.goからモック型が削除されている
  - モック型がvertexai_test.goに移動している
  - モック型がunexported（mockAPIError等）になっている
  - 既存のテストがすべてパスする

### AC-004: 型アサーション重複の整理 [SC-004]

- **Given**: server.goでwebhook.TextMessageContentと*webhook.TextMessageContentが別々のcaseで同一処理されている
- **When**: リファクタリング完了後
- **Then**:
  - 各メッセージタイプ（text, image, sticker, video, audio, location）が単一のcase文で処理される
  - ポインタ型と値型の両方を処理するヘルパー関数またはパターンが導入されている
  - 重複していたcase文が統合されている
  - 既存のテストがすべてパスする

## Implementation Notes

- 各変更後にテスト実行を必須とする
- 変更は段階的に行い、各ステップでテストがパスすることを確認
- SC-001から順に実施し、依存関係のある変更は適切な順序で行う

## Out of Scope

- `internal/llm/provider.go`のエラー型ボイラープレート（型アサーションの都合上、現状維持が適切）
- Message構造体の共有化（循環インポート回避のため意図的に重複）

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-26 | 1.0 | Initial version | - |
| 2025-12-26 | 1.1 | Remove SC-005, SC-006 from scope; clarify AC-001, AC-002, AC-004 | - |
