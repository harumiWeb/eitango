# ADR-0001: Local-First な TUI/CLI ランタイム構成

## Status

`accepted`

## Background

初期リリース後の `eitango` は、短い待ち時間で使える単体の学習ツールとして維持する必要がある。現行実装は Cobra を入口にしつつ Bubble Tea で対話 UI を構成し、永続化はユーザーごとのローカル SQLite に閉じている。今後の保守では、なぜ server/web 前提へ寄せないのか、なぜ UI と学習ロジックを分離しているのかを毎回判断し直さない状態にしておきたい。

## Decision

- 実行形態は single-process の CLI/TUI アプリを維持する。
- public entrypoint は Cobra コマンドに集約し、対話体験は Bubble Tea の Model/Update/View で扱う。
- 学習ロジックと永続化は `internal/app` の外へ分離し、`internal/quiz`, `internal/session`, `internal/srs`, `internal/store`, `internal/config` を UI から独立して保守する。
- 永続データはユーザー単位のローカル data dir に保存し、`user.db`, `config.toml`, `logs/`, `update-check.json` を同居させる。
- 通常の学習、復習、統計、辞書保守の主経路はオフラインで完結させる。ネットワークアクセスは補助機能に限定し、学習フローの前提にしない。

## Consequences

- 配布物は単一バイナリ中心で保てるため、クロスプラットフォーム配布とオフライン利用が容易になる。
- UI とドメインロジックの責務境界が残るため、学習ロジックや DB 操作を UI 非依存で検証できる。
- 複数端末同期、共有バックエンド、常時ネットワーク前提の機能は別の判断なしに導入しない。
- ローカル DB migration と保存先互換性は、このリポジトリが継続的に面倒を見る責務になる。
- update check のようなネットワーク機能は例外扱いになり、学習の主経路から分離されたまま保守される。

## Rationale

- Tests:
  - `cmd/eitango/main_test.go`
  - `internal/app/cmds_test.go`
  - `internal/store/store_test.go`
  - `internal/config/config_test.go`
- Code:
  - `cmd/eitango/main.go`
  - `internal/app/model.go`
  - `internal/app/update.go`
  - `internal/config/config.go`
  - `internal/store/db.go`
- Related specs:
  - なし。コード、README、tests を正本とする。

## Supersedes

- None

## Superseded by

- None
