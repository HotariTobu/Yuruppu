# Feature: Tool Calling with Weather Tool

## Overview

botにツールコール機能を追加し、外部ツールを呼び出して情報を取得・活用できるようにする。最初のサンプルツールとして天気取得ツールを実装する。

## Background & Purpose

現在のYuruppuは、テキストベースの応答のみ可能である。ツールコール機能を追加することで、外部APIやサービスと連携し、より実用的な情報（天気、時刻、ニュースなど）を提供できるようになる。

天気ツールは、ユーザーが「今日の東京の天気は？」などと聞いた際に、実際の天気予報を取得して応答できるようにするためのサンプル実装である。

## Out of Scope

- 天気以外のツール実装（今後別specで追加）
- ツールの動的登録機能
- ユーザーごとのツール設定
- 有料の天気APIの利用
- ツールコールのループ回数制限（LLMの判断に委ねる）

## Notes

- 天気APIの選定は実装フェーズ（/tech-research）で決定する
- ロケーション指定方法はAPI選定後に決定する
- ACのテストはLLMからパラメータが指定されたと仮定して動作を検証する

## Requirements

### Functional Requirements

- [ ] FR-001: 指定された地域の天気予報を取得できる

### Non-Functional Requirements

- [ ] NFR-001: 天気API呼び出しは3秒以内にタイムアウトする
- [ ] NFR-002: APIエラー時はエラー情報を補足としてLLMに渡す

## Acceptance Criteria

### AC-001: Weather forecast retrieval [FR-001]

- **Given**: botが起動している
- **When**: ユーザーが「東京の天気は？」と質問する
- **Then**: 東京の天気予報が応答に反映される

### AC-002: Error handling [NFR-001, NFR-002]

- **Given**: 天気APIが利用不可の状態
- **When**: ユーザーが天気を質問する
- **Then**: エラー情報が補足としてLLMに渡される

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-31 | 1.0 | Initial version | - |
