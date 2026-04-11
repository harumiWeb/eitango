# ADR-0002: Bundled Core 辞書のライフサイクルと由来管理

## Status

`accepted`

## Background

`eitango` は標準の語彙データを配布物に含めるが、そのデータは単なる static file ではなく、出題品質、進捗互換性、再生成手順、第三者データ由来の説明責任と結びついている。初期リリース後は、core 辞書の更新時に何を守るかを、旧設計書ではなく current code と整合する判断として残しておく必要がある。

## Decision

- `assets/words_core.jsonl` を bundled core の正規 runtime asset とし、起動時は embed された内容を読み込んで利用する。
- bundled core は `dict.LoadCoreWords()` で parse と validation を通したうえで seed する。core では `lemma`, `meaning_ja`, `pos`, `level`, `frequency_rank`, `distractor_group` を必須とし、`(lemma, pos)` と `frequency_rank` の重複を許さず、各 `distractor_group` は最低 4 語を要求する。
- bundled core の版は `dict.CoreWordsVersion` で管理し、DB 内では `app_meta.dict_version` と `source = "core"` を使って現在の seed 状態を追跡する。
- 初回 seed で core 語彙を投入し、`dict_version` が変わった場合は `source = "core"` の既存 row を `normalized(lemma, pos)` 単位で差分同期する。matched row は同じ `word_id` を維持したまま metadata を更新し、新語だけを追加し、消えた旧 core row は inactive として退役させる。
- `reset --reseed` は bundled core の明示的な破壊的再投入導線として残し、core source を全置換したうえで学習履歴テーブルもリセットする。
- import 語彙は `import:*` source として core から分離し、`core` は予約済み source として扱う。
- raw の Leipzig / Japanese WordNet 入力は配布物へ含めず、再生成条件は `scripts/vocab/source_manifest.json` と repository tooling に固定する。

## Consequences

- 学習時に参照する core 辞書の品質と整合性を、アプリ起動時 validation と DB metadata の両方で担保できる。
- core 更新時も matched row の `word_id` と SRS 履歴を保持できるため、bundled core の語彙追加や軽微な metadata 更新を学習進捗の全消去なしで配布できる。
- 退役した core 語彙を inactive row として残すため、過去 session・review・export の参照整合性を壊さずに新規計画対象からだけ除外できる。
- core と import を source で分けるため、標準辞書の更新とユーザー追加データの保守方針を分離できる。
- core の版更新は `normalized(lemma, pos)` を軸にした互換性契約を伴うため、同一 key の語は同じ学習対象として扱える範囲の更新に留める必要がある。
- 破壊的な全リセットは `reset --reseed` に限定されるため、CLI とテストの両方でこの導線を明示的に維持し続ける必要がある。
- データ由来と再配布条件を repository 内で維持し続ける責務が残る。

## Rationale

- Tests:
  - `internal/dict/validate_test.go`
  - `internal/store/embedded_core_words_test.go`
  - `internal/store/store_test.go`
  - `cmd/eitango/main_test.go`
  - `cmd/eitango/export_test.go`
- Code:
  - `internal/dict/embed.go`
  - `internal/dict/loader.go`
  - `internal/store/migrate.go`
  - `internal/store/reset.go`
  - `internal/store/word_write.go`
  - `scripts/vocab/source_manifest.json`
- Related specs:
  - なし。コード、README、tests を正本とする。

## Supersedes

- None

## Superseded by

- None
