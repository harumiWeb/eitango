# 2026-04-11 語彙追加 22000 seed batch

- [x] `parallel_review_22000` を `21001-22000` の範囲で作成する
- [x] サブエージェントを使って slice ごとの承認候補をレビューする
- [x] `approved_review_candidates.tsv` / `approved_seed.csv` に 22000 batch をマージする
- [x] `apply_review_batch.py` で `assets/words_core.jsonl` へ反映する
- [x] `validate --embedded-core` と `doctor` で整合性を検証する

# 5k 初回リリース TODO

このファイルは、初回 OSS リリースに向けた active backlog だけを管理する。
完了済みの長い履歴、旧 30k ロードマップ、設計との差分棚卸しはここには残さない。

## 2026-04-04 winget 配布追加

- [x] `.goreleaser.yaml` を dual-archive 化し、Windows zip 専用 `windows-archive` を追加する
- [x] `winget` publish 設定を追加し、`HarumiWeb.Eitango` と `WINGET_GITHUB_TOKEN` を使う
- [x] `.github/workflows/release.yml` で GoReleaser step へ `WINGET_GITHUB_TOKEN` を渡す
- [x] README / README.en / ADR に Windows の winget 導線を反映する
- [x] 実タグ release で fork `harumiWeb/winget-pkgs` への push まで確認した
- [ ] `microsoft/winget-pkgs` への PR を手動で作成し、運用手順を確認する

## 2026-04-06 issue #28: 無色モードとテーマカラー

- [x] `tasks/feature_spec.md` にテーマ仕様を追加する
- [x] `config.toml` に `theme_mode` / `theme_palette` を追加する
- [x] `internal/tui` を preset / custom theme builder 化する
- [x] ホーム設定 overlay に theme mode 行を追加する
- [x] 色以外でも選択状態とエラー状態が判別できる render へ調整する
- [x] locale / README / docs/specs をテーマ設定へ追従させる
- [x] `internal/config` / `internal/tui` / `internal/app` / `internal/i18n` の回帰テストを追加して通す
- [x] review 指摘に合わせて `default` theme を旧配色へ戻し、`theme_palette` の未設定 slot は保存時に omit する

## 2026-04-07 issue #34: キーバインドカスタマイズ

- [x] `tasks/feature_spec.md` に keymap 仕様を追加する
- [x] `config.toml` に `[keymap]` を追加し、strict validation と canonical save を実装する
- [x] runtime keymap を context-aware に差し替え、help / key guide を動的表示へ更新する
- [x] ホーム設定から開く Key Bindings Editor を追加する
- [x] save 後に再起動なしで keymap を再反映する
- [x] `internal/keymap` / `internal/config` / `internal/app` の回帰テストを追加して通す
- [x] README / README.en / `docs/specs/keymap-settings.md` を更新する
- [x] keymap editor が terminal 高を超えないように一覧スクロールを追加する
- [x] review 指摘に合わせて keymap save 時も settings overlay の draft を保持し、help 退路と Esc 録音の回帰を追加する
- [x] review 指摘に合わせて unbound help 表示、`quiz.write` の ASCII 限定 validation、到達不能分岐を修正する
- [x] review 指摘に合わせて `help.back` 必須化、key conflict 置換の原子化、startup keymap error 保持を追加する
- [x] `quiz.choice` のインライン key guide から `select1-4` を外し、詳細は help 画面へ寄せる

## 2026-04-01 PR #12: Codacy follow-up

- [x] `doctor.go` の `PRAGMA table_info(...)` を string formatting ではなく許可テーブルの定数 query に置き換える
- [x] 想定外の table 名を拒否する回帰テストを追加する
- [x] SQLite migration に対する `tsqllint` の誤検知を `.codacy.yaml` で除外する
- [x] review 指摘に合わせて `play/review` が typo 引数を黙って受け付けないようにする
- [x] locale format string の回帰テストを実際の引数数・出力まで検証する

## 2026-04-10 issue #44: reviewed-only fallback review

- [x] `tasks/feature_spec.md` に due 0 件時の review fallback 仕様を追加する
- [x] store に review 済み語の random 候補取得を追加する
- [x] `sessionCmd` に due 0 件時の fallback 確認 message を追加する
- [x] home / startup review 開始時に fallback 確認 overlay を出せるようにする
- [x] locale / README / README.en の review 説明を更新する
- [x] `internal/app` / `internal/store` の回帰テストを追加して通す
- [x] reviewed-only fallback session を通常 review と別 mode で保持する
- [x] reviewed-only fallback 回答で SRS progress を更新しない
- [x] reviewed-only fallback の choice feedback から 4 段階評価を外し Enter-only にする
- [x] reviewed-only fallback の write feedback でも rating 保存を行わない
- [x] fallback practice の UI/README と回帰テストを追従させる

## 2026-04-02 issue #14: Write モード難易度緩和

- [x] `config.toml` に `write_mode_difficulty` (`basic` / `hard`) を追加する
- [x] ホーム設定 UI に Write 難易度行を追加する
- [x] Learn + Write + `basic` で Choice 既出語だけを新規候補に使う
- [x] `ListWordsSeenInChoice` の store query と回帰テストを追加する
- [x] `hard` と既定値 `basic` の回帰テストを追加する
- [x] README / README.en に設定と挙動差分を反映する

## 2026-04-03 issue #15: 進行中セッション時のモード切替確認

- [x] `tasks/feature_spec.md` にホーム開始競合の仕様を追加する
- [x] ホーム画面に active session 破棄確認 overlay を追加する
- [x] `Enter` / `r` / `n` の競合時に pending request を確認経由へ切り替える
- [x] locale 文言と help 表示を確認 overlay に対応させる
- [x] `internal/app` / `internal/i18n` の回帰テストを追加して通す

## 2026-04-03 issue #19: macOS / Windows 音声再生

- [x] `tasks/feature_spec.md` に音声再生仕様を追加する
- [x] `internal/audio` に speaker abstraction と macOS / Windows / noop backend を追加する
- [x] `config.toml` とホーム設定に `audio_enabled` / `audio_autoplay` を追加する
- [x] quiz / feedback に `Ctrl+P` 手動再生と `Shift+Tab` セッション内 autoplay toggle を追加する
- [x] session 開始直後と次問題読込直後の autoplay を追加する
- [x] locale / README / help 表示を音声操作へ追従させる
- [x] `internal/audio` / `internal/config` / `internal/app` / `internal/i18n` の回帰テストを追加して通す

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
- [x] issue #29 v1 として狭幅 terminal 向け narrow-width guard を描画入口へ追加する
- [x] locale / README / `docs/specs/` を狭幅 guard の恒久仕様へ追従させる
- [x] 狭幅 guard の回帰テストを追加し、`go test ./internal/app` で検証する
- [x] issue #29 v2 として主要画面に `normal / compact / narrow` の 3 段階描画を導入する
- [x] compact layout 用の key guide / help / label-value wrap helper を追加する
- [x] compact layout の回帰テストと docs を更新し、`go test ./internal/app` で検証する
- [x] issue #29 v3 として主要画面の compact panel を border 付き shrink layout へ切り替える
- [x] 単一行 UI 向けの `...` 省略 helper を追加し、key guide / settings / results / keymap editor に適用する
- [x] issue #29 v3 の回帰テストと docs を更新し、`go test ./internal/app` で検証する
- [x] issue #29 v4 として主要画面を `adaptive / narrow` の 2 段描画へ整理する
- [x] `normal / compact` 閾値切替をやめ、最小幅以上では同じ renderer が terminal 幅へ連続追従するようにする
- [x] issue #29 v4 の回帰テストと docs を更新し、`go test ./internal/app` で検証する
- [x] adaptive 化で落ちた `home` の選択色、`quiz.write` の固定幅ラベル整列、panel の上下余白を復元する
- [x] `home` のメトリクス行を縦並びの固定幅ラベル整列へ戻し、adaptive panel の左右余白を復元する
- [x] adaptive panel の左右余白を「外側 0 / 内側 2」へ調整し、terminal 幅内に収まることを維持する
- [x] review 指摘に合わせて `quiz.choice` の選択肢本文と `results` の hard words は省略せず wrap し、`width == 0` では旧 renderer 経路を維持する
- [x] review 指摘に合わせて `width == 0` の legacy renderer 経路を `results` / `stats` / `keymap editor` まで揃え、回帰テストで固定する
