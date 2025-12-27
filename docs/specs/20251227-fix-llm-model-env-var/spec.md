# Fix: LLM Model Name Hardcoded Instead of Environment Variable

> Template for bug fixes.

## Overview

LLMモデル名 (`gemini-2.5-flash-lite`) がハードコードされており、環境変数で指定できない。LINE認証情報と同様に必須環境変数にすべき。

## Current Behavior (Bug)

- `internal/llm/vertexai.go` で `geminiModel = "gemini-2.5-flash-lite"` が定数として定義されている
- モデルを変更するにはコード変更が必要

## Expected Behavior

- 環境変数 `LLM_MODEL` でモデル名を指定できる
- `LLM_MODEL` は必須（未設定時はアプリ起動エラー）
- デフォルト値なし（LINE認証情報と同様の扱い）

## Root Cause

設計時にモデル名を固定と想定し、設定可能にする要件が漏れていた。

## Proposed Fix

- [ ] FX-001: `Config` 構造体に `LLMModel` フィールドを追加
- [ ] FX-002: `loadConfig()` で環境変数 `LLM_MODEL` を必須として読み込む
- [ ] FX-003: `NewVertexAIClient()` に `model` パラメータを追加
- [ ] FX-004: ハードコードされた `geminiModel` 定数を削除
- [ ] FX-005: `LLM_MODEL` が空文字列または空白のみの場合、アプリ起動エラー

## Acceptance Criteria

### AC-001: [Linked to FX-001, FX-002]

- **Given**: `LLM_MODEL` 環境変数が設定されていない
- **When**: アプリケーションを起動する
- **Then**:
  - アプリケーションが起動に失敗する
  - エラーメッセージに `LLM_MODEL` が含まれる

### AC-002: [Linked to FX-001, FX-002, FX-003]

- **Given**: `LLM_MODEL` に有効なモデル名が設定されている
- **When**: アプリケーションを起動してLLMを呼び出す
- **Then**:
  - LLM呼び出し成功時に `resp.ModelVersion` をログ出力する
  - ログに出力されたモデル名が指定したモデルを含むことを検証できる

### AC-003: [Linked to FX-004, Regression]

- **Given**: 既存のLLM呼び出し機能
- **When**: 環境変数を正しく設定してアプリケーションを使用する
- **Then**:
  - 既存の機能が正常に動作する
  - 新しいバグが導入されない

## Implementation Notes

- `LINE_CHANNEL_SECRET` と `LINE_CHANNEL_ACCESS_TOKEN` の実装パターンに従う
- ADR `20251225-gemini-model-selection.md` の更新は不要（選定理由は有効なまま）

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-12-27 | 1.0 | Initial version | - |
