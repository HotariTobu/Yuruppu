# Feature: Chat History Storage

## Overview

チャット履歴を保存し、過去の会話コンテキストを考慮したLLM応答を生成できるようにする機能。

## Background & Purpose

現在のYuruppuボットは各メッセージを独立して処理しており、過去の会話内容を記憶していない。これにより、ユーザーが「さっき話したこと」を参照したり、継続的な会話をしたりすることができない。

チャット履歴を保存し、LLMへのリクエストに含めることで、よりコンテキストに沿った自然な会話が可能になる。

## Requirements

### Functional Requirements

- [ ] FR-001: ユーザーからのメッセージと、ボットからの応答を保存する
- [ ] FR-002: 保存された履歴をLLMリクエストのコンテキストとして含める
- [ ] FR-003: 会話ソース（1:1チャット/グループ/ルーム）ごとに会話履歴を分離して管理する

### Non-Functional Requirements

- [ ] NFR-001: 履歴の読み書きはメッセージ処理のレイテンシに大きな影響を与えない（+100ms以内）
- [ ] NFR-002: ストレージ障害時は応答を生成しない（エラーログのみ）

## Design Decisions

### Storage Backend

Google Cloud Storage を使用する。詳細は [ADR-20251228-chat-history-storage](../../adr/20251228-chat-history-storage.md) を参照。

- JSONL形式で会話履歴を保存（1ファイル = 1 SourceID、1行 = 1メッセージ）
- Read-Modify-Write パターンでメッセージを追加
- Generation preconditions で競合を検出

### History Scope

会話履歴は **ソースID** 単位で管理する。

- **SourceID**: 会話が行われる場所を識別するID
  - 1:1チャット: 相手のユーザーID
  - グループチャット: グループID
  - トークルーム: ルームID

```
ユーザーAとの1:1 → SourceID: "U123abc..."（ユーザーAのID）
ユーザーBとの1:1 → SourceID: "U456def..."（ユーザーBのID）
グループX → SourceID: "C789ghi..."（グループXのID）
```

グループチャットでは、グループ内の全員の発言が1つの履歴として共有される。

### History Format

```go
// Message represents a single message in conversation history.
type Message struct {
    Role      string    // "user" or "assistant"
    Content   string
    Timestamp time.Time
}

// ConversationHistory holds messages for a specific source.
type ConversationHistory struct {
    SourceID string    // Where the conversation takes place
    Messages []Message
}
```

## Error Handling

| Error Type | Condition | Behavior |
|------------|-----------|----------|
| StorageReadError | 履歴の読み込みに失敗 | 応答を生成しない、エラーログ出力 |
| StorageWriteError | 履歴の保存に失敗 | 応答を生成しない、エラーログ出力 |
| StorageTimeoutError | ストレージ操作がタイムアウト | 応答を生成しない、エラーログ出力 |

## Acceptance Criteria

### AC-001: メッセージ履歴の保存 [FR-001]

- **Given**: ユーザーがメッセージを送信する
- **When**: ボットが応答を返す
- **Then**:
  - ユーザーのメッセージが履歴に保存される
  - ボットの応答が履歴に保存される
  - 各メッセージにタイムスタンプが記録される

### AC-002: コンテキストを含む応答生成 [FR-002]

- **Given**: ユーザーが過去に「私の名前は太郎です」と送信している
- **When**: ユーザーが「私の名前を覚えてる？」と送信する
- **Then**:
  - LLMリクエストに過去の会話履歴が含まれる
  - ボットは「太郎」という名前を認識した応答を返す

### AC-003: 会話ソース間の履歴分離 [FR-003]

- **Given**: ユーザーAとの1:1チャットで「好きな食べ物はラーメン」という会話があり、グループXで「好きな食べ物は寿司」という会話がある
- **When**: ユーザーAとの1:1チャットで「好きな食べ物は？」と送信する
- **Then**:
  - ユーザーAとの1:1チャットの履歴のみがLLMリクエストに含まれる
  - ボットは「ラーメン」と回答する（寿司ではない）

### AC-004: ストレージ障害時の動作 [NFR-002]

- **Given**: ストレージが利用不可能
- **When**: ユーザーがメッセージを送信する
- **Then**:
  - ボットは応答を生成しない
  - エラーがログに記録される

## Implementation Notes

- 現在の `Agent` は単発のメッセージのみを処理
- `Responder` インターフェースを拡張するか、新しいコンポーネントを追加する必要がある
- Gemini APIの `Content` に履歴を含める形式で実装

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-28 | 1.0 | Initial version | - |
