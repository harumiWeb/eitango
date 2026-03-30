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

---

# 2026-03-30 issue #3: go install 時の version 表示仕様

## Goal

- `go install github.com/harumiWeb/eitango/cmd/eitango@latest` で導入したバイナリでも `eitango version` と update check 比較対象が `dev` ではなく実バージョンを使う。

## Scope

- `cmd/eitango/main.go` の build version 解決ロジック
- 版情報を使う CLI 表示と update check 呼び出し
- 回帰テスト
- README の install / update 説明の最小補足

## Non-Goals

- release build の `ldflags` 方針変更
- local checkout 上の `go run ./cmd/eitango` や `go install ./cmd/eitango` を release version 扱いすること
- `commit` / `date` の解決方式変更

## Required Behavior

- `main.version` が `ldflags` で上書き済みなら、その値を最優先で使う。
- `main.version` が `dev` の場合だけ `runtime/debug.ReadBuildInfo()` を参照し、`Main.Version` が semver や pseudo-version のような実値ならそれを使う。
- `Main.Version` が空文字または `"(devel)"` の場合は `dev` を維持する。
- `--version`、`version` サブコマンド、update check への `CurrentVersion` は同じ解決済み version を使う。

## Acceptance

- `go install github.com/harumiWeb/eitango/cmd/eitango@latest` 由来の build info を模したテストで `dev` ではなく `vX.Y.Z` が表示される。
- build info がないケースでは従来どおり `dev` が表示される。
- `go test ./...` が通る。
