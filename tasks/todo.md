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

## 2026-03-30 issue #4: curl installer

- [x] `install.sh` を追加し、latest install / `--version` / `--uninstall` / `--purge-data` を実装する
- [x] installer が archive と同じ release の `checksums.txt` を使って SHA256 を必須検証する
- [x] release 同梱の `LICENSE`, `README*`, `THIRD_PARTY_NOTICES.md`, `third_party/licenses/` を `~/.eitango/share/` へ保持する
- [x] Go integration test で install / checksum failure / uninstall の回帰を追加する
- [x] Ubuntu CI に `shellcheck install.sh` を追加する
- [x] README / README.en / CHANGELOG を curl installer 導線へ更新する
- [x] 配布ポリシー ADR を installer 前提へ更新する

## 2026-03-31 PR #7 unresolved review follow-up

- [x] `gh` で PR #7 の未解決 review thread を取得し、妥当性を確認する
- [x] `install.sh` が wrapped / unwrapped archive layout の両方を扱えるようにする
- [x] `install_test.go` に wrapped / unwrapped layout の回帰を反映する
- [x] `go test ./...` を通す

## 2026-03-31 issue #8: update 通知 stale tag

- [x] ホーム画面の update check を cached `Check` ではなく forced `CheckNow` に切り替える
- [x] forced refresh 成功時に cached latest tag を更新する回帰 test を追加する
- [x] forced refresh 失敗時に cached latest tag へ fallback する回帰 test を追加する
- [x] README / README.en / CHANGELOG / ADR の update 通知説明を整合更新する
- [x] `go test ./...` を通す

## 2026-03-31 issue #9: write モード追加

- [x] `sessions` / `reviews` に `answer_mode` を追加し、old row は `choice` default で後方互換にする
- [x] CLI を `play/review [choice|write]` へ拡張し、`learn` alias を維持する
- [x] ホーム画面に `Tab` の回答方式切替を追加する
- [x] write 用の入力、ヒント、skip、auto-rating、feedback 分岐を実装する
- [x] write 中の文字入力と衝突しないよう、hint / skip を `Tab` / `Ctrl+S` に固定する
- [x] write/session/store/CLI の回帰テストを追加する
- [x] README / README.en を新しい操作体系へ更新する
- [x] `go test ./...` を通す

## 2026-04-01 issue #9 follow-up: doctor / write feedback regressions

- [x] `doctor` が pre-005 の read-only DB でも `answer_mode` 不在で落ちず、migration drift だけを報告するようにする
- [x] write feedback の help / quit 無効表示 / status を Enter 専用フローに合わせる
- [x] legacy doctor / write feedback help の回帰テストを追加する
- [x] `go test ./...` を通す

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
