# 2026-03-29 ドキュメント再編仕様

## Goal

- 旧設計書を廃止し、初期リリース後も参照価値がある判断だけを ADR に残す。
- `docs/specs/` は増やさず、コード・README・tests を正本として維持する。

## Deliverables

- `docs/adr/` に `accepted` な ADR を 3 本追加する。
- 旧設計書を削除する。
- 削除した設計書への残存参照を解消する。

## ADR Scope

- ADR 1: ローカル完結の TUI/CLI ランタイム構成
- ADR 2: bundled core 辞書のライフサイクルと由来管理
- ADR 3: 配布物と update check の運用方針

## Non-Goals

- CLI、env、DB schema、学習ロジックの挙動変更
- `docs/specs/` の新設や index/README の追加
- 旧設計書の全文アーカイブ

## Acceptance

- `docs/adr/` の 3 本が `Status`, `Background`, `Decision`, `Consequences`, `Rationale`, `Supersedes`, `Superseded by` を持つ。
- `rg -n "docs/design\.md|design\.md"` で削除済み設計書への壊れた参照が残らない。
- `go test ./...` が通る。
