# 5k 初回リリース TODO

このファイルは、初回 OSS リリースに向けた active backlog だけを管理する。
完了済みの長い履歴、旧 30k ロードマップ、設計との差分棚卸しはここには残さない。

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

- [ ] `scripts/vocab/generate_freq_seed.py` を `wordfreq` 依存から Leipzig `tmp/eng_news_2024_1M-words.txt` 読み込みへ置き換える
- [ ] Leipzig parser の正規化ルールを固定する
- [ ] `freq_seed.csv` の中間 schema を `lemma,pos,frequency_rank,frequency_count,source_token,source_corpus` に更新する
- [ ] `scripts/vocab/` の後段スクリプトを新しい `freq_seed.csv` に追従させる
- [ ] 既存 core のうち Leipzig + WordNet で裏付けできる row だけを retained row として再構成する
- [ ] 裏付けできない row を drop し、差分を確認する
- [ ] `frequency_rank` を Leipzig 由来で再採番する
- [ ] `level` を `core-1` / `core-2` / `core-3` / `core-4` に再計算する
- [ ] retained row について既存の `meaning_ja`, `distractor_group`, `example_*` を保持したまま `assets/words_core.jsonl` を clean rebuild する
- [ ] `README.md`, `README.en.md`, `THIRD_PARTY_NOTICES.md`, `docs/design.md` から `wordfreq`, `nltk`, `toeic600`, `toeic800` 前提の記述を除去する
- [ ] `third_party/licenses/` に Leipzig 用のライセンス参照を追加し、bundled data の notice を Leipzig + Japanese WordNet ベースへ更新する
- [ ] `scripts/vocab/` に source manifest を追加し、使用 corpus、入力ファイル名、ライセンス、生成コマンドを固定する
- [ ] `pyproject.toml` から `wordfreq` と `nltk` を削除する
- [ ] fixture とテストデータの `level` 値を `core-*` へ更新する
- [ ] `go test ./...` を通す
- [ ] `go run ./cmd/eitango validate --embedded-core` を通す
- [ ] `go run ./cmd/eitango doctor` で metadata / distractor 周りに新しい問題が出ていないことを確認する
- [ ] `goreleaser check` と snapshot build で法務ファイルの同梱を確認する
- [ ] `dict_version` と `reset --reseed` の最終導線を初回リリース前提で仕上げる

## P1: 初回リリース後でよいもの

- [ ] drop した row の backfill 候補を Leipzig + WordNet review flow で補充する
- [ ] 例文付き row の追加方針を整理する
- [ ] 5k core の `meaning_ja` と `distractor_group` のサンプルレビュー件数を増やす
- [ ] `scripts/vocab/` のローカル再生成手順を README か別文書へ整理する

## Out of Scope

- 10k / 30k への段階拡張
- 複数 frequency source の併用
- 追加 pack の配布設計
- 例文データの大規模拡張
- bundled core schema 自体の破壊的変更
