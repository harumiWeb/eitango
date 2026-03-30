# 5k 初回リリース TODO

このファイルは、初回 OSS リリースに向けた active backlog だけを管理する。
完了済みの長い履歴、旧 30k ロードマップ、設計との差分棚卸しはここには残さない。

## 2026-03-30 issue #3: go install version 表示

- [x] `dev` のときだけ build info の `Main.Version` を使う解決ロジックを追加する
- [x] `--version` / `version` / update check の参照 version を統一する
- [x] `go install @latest` 相当の回帰テストを追加する
- [x] README の install / update 説明を最小限補足する
- [x] `go test ./...` を通す
- [ ] 公開 release 更新後に `go install github.com/harumiWeb/eitango/cmd/eitango@latest` 実機で `dev` 解消を再確認する

## 2026-03-29 ドキュメント再編

- [x] 旧設計書のうち current code に効いている判断だけを抽出する
- [x] `docs/adr/` に runtime / core dictionary / release-update policy の ADR を追加する
- [x] `docs/specs/` は空のまま維持し、コード正本の方針で固定する
- [x] 削除済み設計書への参照を整理し、検証手順を更新する

## 固定方針

- 初回リリースの bundled core は約 5k 語で固定する
- `wordfreq` は生成パイプラインと配布説明から完全に外す
- bundled data の語彙由来は `Leipzig` と `Japanese WordNet` に限定する
- `level` は `core-1` から `core-4` の自前バケットに置き換える
- raw の Leipzig / WordNet 入力は `tmp/` のローカル生成入力として扱い、配布物には含めない

## 現在の土台

- [x] `learn`, `review`, `stats`, `doctor`, `reset`, `import`, `export`, `validate` の CLI は揃っている
- [x] embedded core words の seed、`reset --reseed`、`doctor`、`goreleaser` の導線は動いている
- [x] 現在の `assets/words_core.jsonl` は約 5k 語まで拡張済み

## P0: Release Blockers

- [x] `scripts/vocab/generate_freq_seed.py` を `wordfreq` 依存から Leipzig `tmp/eng_news_2024_1M-words.txt` 読み込みへ置き換える
- [x] Leipzig parser の正規化ルールを固定する
- [x] `freq_seed.csv` の中間 schema を `lemma,pos,frequency_rank,frequency_count,source_token,source_corpus` に更新する
- [x] `scripts/vocab/` の後段スクリプトを新しい `freq_seed.csv` に追従させる
- [x] 既存 core のうち Leipzig + WordNet で裏付けできる row だけを retained row として再構成する
- [x] 裏付けできない row を drop し、差分を確認する
- [x] `frequency_rank` を Leipzig 由来で再採番する
- [x] `level` を `core-1` / `core-2` / `core-3` / `core-4` に再計算する
- [x] retained row について既存の `meaning_ja`, `distractor_group`, `example_*` を保持したまま `assets/words_core.jsonl` を clean rebuild する
- [x] `README.md`, `README.en.md`, `THIRD_PARTY_NOTICES.md`, 旧設計書から `wordfreq`, `nltk`, `toeic600`, `toeic800` 前提の記述を除去する
- [x] `third_party/licenses/` に Leipzig 用のライセンス参照を追加し、bundled data の notice を Leipzig + Japanese WordNet ベースへ更新する
- [x] `scripts/vocab/` に source manifest を追加し、使用 corpus、入力ファイル名、ライセンス、生成コマンドを固定する
- [x] `pyproject.toml` から `wordfreq` と `nltk` を削除する
- [x] fixture とテストデータの `level` 値を `core-*` へ更新する
- [x] `go test ./...` を通す
- [x] `go run ./cmd/eitango validate --embedded-core` を通す
- [x] `go run ./cmd/eitango doctor` で metadata / distractor 周りに新しい問題が出ていないことを確認する
- [x] `goreleaser check` と snapshot build で法務ファイルの同梱を確認する
- [x] `dict_version` と `reset --reseed` の最終導線を初回リリース前提で仕上げる
