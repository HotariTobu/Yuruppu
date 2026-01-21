# Enhancement: Event List Flex Message

## Overview

`list_events`ツールの出力をFlex Messageに変更し、`get_event`ツールを削除する。

## Background & Purpose

現在、`list_events`と`get_event`はJSON形式でLLMにデータを返し、LLMがユーザーにテキストで伝えている。Flex Messageを使用することで、イベント情報をより視覚的に分かりやすく表示できる。また、`list_events`で詳細情報を含めることで`get_event`が不要になり、ツール構成がシンプルになる。

## Current Behavior

### list_events
- フィルタ条件（created_by_me, start, end）に基づいてイベント一覧を取得
- JSON形式でLLMに返す（chat_room_id, title, start_time, end_time, fee）
- LLMがテキストでユーザーに伝える

### get_event
- 特定のイベントの詳細を取得
- JSON形式でLLMに返す（title, start_time, end_time, fee, capacity, description, creator_name）
- LLMがテキストでユーザーに伝える

## Proposed Changes

- [ ] CH-001: `list_events`ツールがFlex Messageを送信するように変更
- [ ] CH-002: `get_event`ツールを削除

## Acceptance Criteria

### AC-001: Flex Message送信 [CH-001]

- **Given**: ユーザーがイベント一覧をリクエスト
- **When**: `list_events`ツールが実行される
- **Then**:
  - イベントがある場合、ツールが直接Flex MessageをLINEに送信
  - 各イベントのEventフィールド（title, start_time, end_time, fee, capacity, description, show_creatorがtrueなら作成者名）を表示
  - ツールはLLMに送信完了を示すレスポンスを返す
  - Final actionとしてLLMのターンを終了する

### AC-002: イベントなしの場合 [CH-001]

- **Given**: フィルタ条件に一致するイベントがない
- **When**: `list_events`ツールが実行される
- **Then**:
  - LINEには何も送信しない
  - ツールはLLMにイベントなしを示すレスポンスを返す
  - ターンは終了しない（LLMが続けて応答できる）

### AC-003: get_eventツール削除 [CH-002]

- **Given**: コードベース
- **When**: 実装完了後
- **Then**:
  - `get_event`ツールが削除されている

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-21 | 1.0 | Initial version | - |
