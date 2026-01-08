# Feature: Typing Indicator

## Overview

ユーザーがメッセージを送信した後、ボットが応答を生成している間、LINE上でタイピングインジケーター（ローディングアニメーション）を表示する。

## Background & Purpose

現在、ユーザーがメッセージを送信してからボットが応答するまで2〜10秒程度かかる（Gemini APIによる応答生成のため）。この間、ユーザーには何のフィードバックもなく、ボットが動作しているかどうかわからない。

タイピングインジケーターを表示することで：
- ボットがメッセージを受け取り、処理中であることをユーザーに伝える
- ユーザー体験を向上させる
- 応答待ち時間のストレスを軽減する

## Out of Scope

- グループチャットやマルチパーソンチャットでのタイピングインジケーター表示（LINE APIの制限により1:1チャットのみ対応）
- ローディング時間のカスタマイズ設定（管理画面等での設定変更）
- 複数回のローディングインジケーター更新（LINEの仕様により、進行中のインジケーターに対する追加リクエストは時間を上書きするのみ）

## Definitions

- **遅延時間（delay）**: メッセージ受信からローディングインジケーター表示までの待機時間。デフォルト3秒
- **表示時間（loadingSeconds）**: ローディングインジケーターの表示継続時間。5〜60秒の範囲（LINE API制限）
- **処理完了**: メッセージハンドラーの処理が終了した状態（reply tool呼び出し、skip tool呼び出し、またはエラー発生のいずれか）

## Requirements

### Functional Requirements

- [x] FR-001: メッセージ受信後、遅延時間が経過しても処理完了していない場合に、ローディングインジケーターAPIを呼び出す
- [x] FR-002: ローディングインジケーターは1:1チャット（UserSource）でのみ呼び出す。グループチャット（GroupSource）やルームチャット（RoomSource）では呼び出さない
- [x] FR-003: ボットからの応答メッセージ送信時に、ローディングインジケーターは自動的に消える（LINE APIの仕様）
- [x] FR-004: ローディングインジケーターAPI呼び出しが失敗しても、メッセージ処理は継続する
- [x] FR-005: 表示時間（loadingSeconds）は設定可能とする。5〜60秒の範囲外の場合はアプリケーション起動時にエラーとする
- [x] FR-006: 遅延時間内に処理完了した場合は、ローディングインジケーターを表示しない

### Non-Functional Requirements

- [x] NFR-001: ローディングインジケーターAPI呼び出しは非同期で行い、メッセージ処理をブロックしない
- [x] NFR-002: ローディングインジケーターAPI呼び出しのエラーはWARNレベルでログに記録し、ユーザーにはエラーを表示しない

## Acceptance Criteria

### AC-001: 遅延後にローディングインジケーターが表示される [FR-001, FR-002, FR-005]

- **Given**: ユーザーがボットと1:1チャットをしている
- **When**: ユーザーがメッセージを送信し、遅延時間が経過してもまだ処理中
- **Then**:
  - LINE上でローディングアニメーションが表示される
  - ボットが応答を返すまでアニメーションが継続する

### AC-002: 応答送信でローディングインジケーターが消える [FR-003]

- **Given**: ローディングインジケーターが表示されている
- **When**: ボットが応答メッセージを送信する
- **Then**:
  - ローディングインジケーターが自動的に消える
  - 応答メッセージが表示される
  - ユーザーにエラーメッセージは表示されない

### AC-003: グループチャットではローディングインジケーターを呼び出さない [FR-002]

- **Given**: ユーザーがグループチャット（GroupSourceまたはRoomSource）でボットにメンションしている
- **When**: ユーザーがメッセージを送信する
- **Then**:
  - ローディングインジケーターAPIは呼び出されない
  - メッセージ処理は通常通り完了する

### AC-004: API呼び出し失敗時もメッセージ処理は継続 [FR-004, NFR-002]

- **Given**: ローディングインジケーターAPIが利用不可（ネットワークエラー等）
- **When**: ユーザーがメッセージを送信する
- **Then**:
  - エラーがWARNレベルでログに記録される
  - メッセージ処理は中断されずに継続する
  - ユーザーには通常通り応答が返される

### AC-005: API呼び出しがメッセージ処理をブロックしない [NFR-001]

- **Given**: ユーザーがメッセージを送信する
- **When**: ローディングインジケーターAPIが呼び出される
- **Then**:
  - メッセージ処理がAPI呼び出しの完了を待たない
  - ローディングインジケーターAPIのレスポンスが遅延しても、メッセージ処理は正常に完了する

### AC-006: 高速処理時はローディングインジケーターを表示しない [FR-006]

- **Given**: ユーザーがボットと1:1チャットをしている
- **When**: ユーザーがメッセージを送信し、遅延時間内に処理完了する（skip tool使用を含む）
- **Then**:
  - ローディングインジケーターAPIは呼び出されない
  - 応答またはskipが正常に処理される

### AC-007: 設定値が範囲外の場合は起動エラー [FR-005]

- **Given**: 表示時間（loadingSeconds）が5秒未満または60秒超に設定されている
- **When**: アプリケーションを起動する
- **Then**:
  - アプリケーションがエラーで起動に失敗する
  - エラーメッセージに設定値の有効範囲が含まれる

## Change History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2026-01-08 | 1.0 | Initial version | - |
| 2026-01-08 | 1.1 | Address spec-reviewer feedback: add loadingSeconds parameter (FR-005), clarify source type definitions, make acceptance criteria testable | - |
| 2026-01-08 | 1.2 | Remove implementation details (specific function/layer names) per spec guidelines - leave to /design phase | - |
| 2026-01-08 | 2.0 | Major revision: delayed loading indicator approach - show indicator only if processing takes longer than delay time; separate env var for timeout; error on invalid range instead of clamping; add FR-006, AC-006, AC-007 | - |
| 2026-01-08 | 2.1 | Add Definitions section (delay, loadingSeconds, processing complete); clarify AC-006 skip tool reference | - |
