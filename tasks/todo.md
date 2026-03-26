# `docs/design.md` 未着手タスク整理

更新基準: `docs/design.md` と現行実装（`fc4fa8f` 時点）を突き合わせた整理。

このファイルは、まだ未着手または一部着手の項目を次の実装候補として残す。完了済みになったものは進捗を追いやすいようにチェック済みで残す。すでに動いている `eitango learn` / `eitango stats` / Bubble Tea の基本学習ループ / SQLite schema / SRS / resume / retry / embedded core words はここでは再掲しない。

## 判定ルール

- 未着手: CLI 入口・画面・永続化・テストの土台がまだ無い
- 一部着手: 内部ロジックや保存先はあるが、CLI 契約・設定・データ・UX が設計書に届いていない

## 実装ポリシー

- 既存の `store` / `session` / `quiz` を再利用し、TUI 層へドメインロジックを増やしすぎない
- 破壊的な操作は必ず明示フラグか確認付きにする
- 新しい CLI コマンドは Cobra 側の入口だけでなく、store 経由の統合テストまでセットで足す
- 画面追加は Bubble Tea の screen 遷移を増やす形で実装し、状態管理を status line の分岐で肥大化させない

## P0: 設計と現実の差分を先に埋める

- [x] `eitango review` コマンドを追加する
  - 種別: 一部着手
  - 現状: home 画面の `r` では review セッションを開始できるが、Cobra の `eitango review` は無い
  - 実装方針:
    - `cmd/eitango/main.go` に `review` サブコマンドを追加する
    - `app.NewModel` / `sessionCmd` に初期起動モードを渡せるようにし、home を経由せず review 開始できるようにする
    - active session がある場合の優先順位を明文化する。基本は resume 優先、強制開始は別フラグで切る
  - 完了条件: `eitango review` だけで due 単語のみのセッションを開始できる

- [x] `config.toml` の実読み込みを入れる
  - 種別: 一部着手
  - 現状: `internal/config` は保存先パスだけ解決しており、設定ロード/保存は未実装
  - 実装方針:
    - まずは `session_size`, `review_ratio`, `focus_mode_default` だけを対象にする
    - 設定ファイルが無ければ既存デフォルトを使う
    - 最初の段階では読み込みだけ実装し、設定編集コマンドは後回しにする
  - 完了条件: 設定値がセッション計画に反映される

- [x] `--focus-mode` を `learn` / `review` に追加する
  - 種別: 一部着手
  - 現状: `session.MakePlan` は総問題数を受け取れるが、呼び出し側が常に `DefaultQuestionCount = 10` を使っている
  - 実装方針:
    - `--focus-mode` は 5 問固定として CLI フラグで追加する
    - `session.DefaultQuestionCount` の書き換えではなく、明示的な session option を渡す
    - `config.toml` が入ったら default を設定可能にする
  - 依存: `config.toml` 読み込み（任意）
  - 完了条件: `eitango learn --focus-mode` で 5 問固定になる

- [x] Phase 1 の辞書パックを 1000〜3000 語へ拡張する
  - 種別: 一部着手
  - 現状: `assets/words_core.jsonl` は `1051` 語の Phase 1 コアパックになり、`lemma`, `pos`, `meaning_ja`, `level`, `frequency_rank`, `distractor_group` を全件で保持する
  - 実装方針:
    - `lemma`, `pos`, `meaning_ja`, `level`, `frequency_rank`, `distractor_group` を優先し、まずは例文より出題品質を優先する
    - 同一 `pos` / 近い `level` / 同一 `distractor_group` で 4 択が成立するよう、グループ単位でデータを増やす
    - データ拡張とセットで loader / doctor 用の検証を追加する
  - 実装メモ:
    - `ListDistractorCandidates` を追加し、quiz / doctor が `distractor_group`, `level`, `frequency_rank` を踏まえた runtime 候補集合を使うようにした
    - `internal/dict/embed_test.go` で件数・必須項目・重複・distractor group 最小件数を固定し、`internal/store/embedded_core_words_test.go` で seed + doctor 健全性を固定した
  - 完了条件: 設計書 Phase 1 相当の語彙量と distractor の最低品質を満たす

## P1: まだ入口が無い運用コマンド群

- [x] `eitango doctor` を追加する
  - 種別: 未着手
  - 実装方針:
    - 最初は read-only 診断に限定する
    - DB open/migrate, core words version, orphan progress/reviews/session_items, active session の不整合, 4 択を作れない単語群を検査する
    - テキスト出力と終了コードで CI / 手元調査の両方に使える形にする
  - 完了条件: DB/辞書の壊れ方を CLI から切り分けられる

- [x] `eitango reset` を追加する
  - 種別: 未着手
  - 実装方針:
    - 破壊的操作をデフォルトにしない
    - まずは `--progress`（学習履歴だけ初期化）と `--reseed`（組み込み辞書の再投入）に分ける
    - 将来の import 実装に備えて、core 辞書と user 辞書を分離できるデータモデルを前提にする
  - 実装メモ:
    - `cmd/eitango/main.go` に `reset` サブコマンドを追加し、フラグ未指定時は DB を触らず拒否する
    - 破壊操作は `store.Reset` に集約し、`--progress` は `sessions` / `session_items` / `reviews` / `progress` のみを消す
    - `--reseed` は同一 `dict_version` でも組み込み core words を再投入し、結果サマリを CLI へ返す
  - 依存: import 用の source 管理が入ると拡張しやすい
  - 完了条件: 学習履歴のリセットと辞書再投入を安全に実行できる

- [x] `eitango export` を追加する
  - 種別: 未着手
  - 実装方針:
    - 先に `wrong-words` CSV と `progress` JSON の 2 経路だけ実装する
    - 出力契約を先に固定し、Anki 取り込みやバックアップ用途を優先する
    - `reviews`, `progress`, `words` を join する read model を store 側に追加する
  - 実装メモ:
    - `cmd/eitango/export.go` に `export wrong-words` / `export progress` を追加し、現段階ではそれぞれ `--format csv` / `--format json` のみを許可する
    - export は `config.Resolve()` + `store.OpenReadOnly()` で DB を開き、migrate / seed / DB 作成を行わない
    - `internal/store/export.go` に `words` + `progress` + 集計済み `reviews` を束ねた export snapshot を追加し、CSV/JSON の両方が同じ read model を使うようにした
  - 完了条件: 苦手語 CSV と進捗 JSON を外部へ持ち出せる

- [x] `eitango import` を追加する
  - 種別: 未着手
  - 実装方針:
    - CSV から始め、必須列は `lemma`, `meaning_ja` に限定する
    - import と reset / doctor を安全にするため、`words` に `source`（`core` / `import:<name>`）相当の識別子を持たせる migration を先に入れる
    - duplicate policy は「同じ source 内は upsert、source を跨ぐ重複は許可して doctor で警告」に寄せる
  - 実装メモ:
    - `assets/migrations/004_words_source.sql` で `words.source` を追加し、既存 core rows をそのまま `source = core` として扱えるようにした
    - core seed / `reset --reseed` は `source = core` だけを置き換えるように変え、imported words 自体は保持したまま学習履歴だけをリセットするようにした
    - `internal/dict.ParseCSV`, `internal/dict.ParseJSONL`, `Store.ImportWords`, `cmd/eitango/import.go` を追加し、`eitango import --file ... --format csv|jsonl [--source ...]` を実装した
    - import source のデフォルトはファイル名由来で、同一 source 内は `(source, lemma, pos)` キー相当で upsert、source を跨ぐ重複は `doctor` の `word sources` check が warning として報告する
  - 依存: `words.source` 追加 migration, `doctor`, `reset`
  - 完了条件: CSV から追加辞書を取り込める

## P2: UI / 体験の未着手項目

- [x] 専用の help 画面を追加する
  - 種別: 一部着手
  - 現状: `?` は status line のヒントだけで、設計書の `ScreenHelp` はまだ無い
  - 実装方針:
    - 画面ごとの keymap を共通の help renderer に寄せる
    - `?` で開く / `Esc` で戻るの往復だけにする
  - 実装メモ:
    - `internal/app` に `ScreenHelp` と共通 help renderer を追加し、home / quiz / feedback / results / stats から `?` で開いて `Esc` で戻れるようにした
  - 完了条件: home / quiz / feedback / results / stats から一貫した help を見られる

- [x] 例文表示を quiz / feedback に載せる
  - 種別: 一部着手
  - 現状: schema に `example_en` / `example_ja` はあるが、seed データと画面表示では未使用
  - 実装方針:
    - まずは quiz では非表示、feedback で正答確認と一緒に出す
    - seed データ拡張では例文が無い単語を許容し、表示側で空欄を自然に扱う
  - 依存: 辞書パック拡張
  - 実装メモ:
    - `feedback` 画面で `example_en` / `example_ja` を表示し、例文が無い単語ではラベルごと省略するようにした
  - 完了条件: 例文付き単語が自然に学習フローへ出る

- [x] `streak by waiting` を stats / home に追加する
  - 種別: 未着手
  - 実装方針:
    - 既存の `reviews.response_ms` から待機時間変換メトリクスを導出し、最初は追加テーブルを作らず集計で出す
    - 指標名は streak よりも `wait minutes` / `waiting converted` のように誤読しない形に寄せる
  - 実装メモ:
    - `stats.Window` に `WaitMinutes` を追加し、`reviews.response_ms` の合計から today / 7 days / 30 days / total の待機分数を集計表示するようにした
    - home 画面でも `Wait today` として当日分の値を見られるようにした
  - 完了条件: 通常の連続日数とは別に、待機時間由来の学習量を表示できる

## P3: 配布・将来機能の保留タスク

- [x] `.goreleaser.yaml` を実運用向けに固める
  - 種別: 一部着手
  - 現状: テンプレートはあるが、成果物名・アーカイブ構成・リリース運用は未整理
  - 実装方針:
    - `cmd/eitango` を前提に成果物名を固定し、Windows zip / Unix tar.gz を最終仕様に合わせる
    - `go generate ./...` が不要なら hook から外す
  - 実装メモ:
    - `.goreleaser.yaml` を `cmd/eitango` / `binary: eitango` 前提へ更新し、darwin/linux/windows 向けの archive 名・checksum・ldflags を固定した
    - `cmd/eitango/main.go` に build metadata と `--version` 表示を追加し、GoReleaser から注入した情報を表示できるようにした
    - `goreleaser release --snapshot --clean` を実行し、Windows zip / Unix tar.gz / `checksums.txt` の生成を確認した
  - 完了条件: ローカル dry-run で配布アーカイブを確認できる


## P4: 30,000語拡張

- [x] 30k 語彙の配布方針を固める
  - 種別: 完了
  - 現状: `core` の埋め込み seed、`import`、`doctor`、`reset --reseed` は揃っており、`source` 付きで複数語彙セットを共存できる
  - 実装方針:
    - 最終目標は単一バイナリで完結する `core` 同梱 30,000 語とする
    - ただし実装と検証は 5k → 10k → 30k の段階投入で進める
    - `source` モデルは維持し、将来の追加パック運用へ戻れる余地も残す
  - 実装メモ:
    - `docs/design.md` に 30k 拡張時の配布方針を追記し、`core` 同梱を最終目標、段階投入を実装方針として固定した
    - `words.source` を維持して import pack と共存できる前提も文書化した
  - 完了条件: 30k 到達時の配布、reseed、進捗リセット方針が後続タスクの前提として参照できる

- [x] 30k 語彙のデータ契約を定義する
  - 種別: 完了
  - 現状: loader/import は最低限の列で動くが、30k では `pos`、`level`、`frequency_rank`、`distractor_group` の品質が出題品質を左右する
  - 実装方針:
    - 必須/任意項目、taxonomy、rank 付与ルール、`distractor_group` の最小件数ルールを固定する
    - 欠損、重複、不正 rank、曖昧な列定義は hard fail とする
  - 実装メモ:
    - `LoadCoreWords` が runtime でも `core` 辞書契約を検証するようにし、required fields、`(lemma, pos)` 一意性、`frequency_rank` 一意性、`distractor_group` 最小件数を保証する方向へ寄せる
    - `import` 側は最小必須を維持しつつ、段階投入や pack 運用に備えて CSV の optional `frequency_rank` を受けられるようにする
    - `docs/design.md` に 30k 拡張時の `core` 入力契約を追記し、`core` と `import` の validation レベル差を明文化した
  - 依存: 30k 語彙の配布方針
  - 完了条件: 生データから最終辞書まで同じ契約で検証できる

- [x] 語彙生成・検証パイプラインを追加する
  - 種別: 完了
  - 現状: `assets/words_core.jsonl` は直接管理されているが、大規模辞書へ拡張する再現可能な生成/検証手順はまだ無い
  - 実装方針:
    - 生データから最終 JSONL を生成する scripts または CLI 導線を用意する
    - required fields、一意性、rank、`distractor_group` 件数、表記揺れを検査できるようにする
  - 実装メモ:
    - `eitango validate` を追加し、embedded core と外部 CSV/JSONL の validation を DB 非依存で回せるようにする
    - import CSV も runtime validation を通し、重複 `lemma/pos` や重複 `frequency_rank` を事前に拒否する
    - `eitango import` も `jsonl` を受けられるようにし、`core` と pack のフォーマット差を減らした
  - 依存: 30k 語彙のデータ契約
  - 完了条件: 語彙拡張を手作業の差し替えではなく再現可能な手順で回せる

- [ ] 語彙データを段階的に拡張する
  - 種別: blocked
  - 現状: Phase 1 の約 1,051 語は揃っているが、repo 内には追加語彙ソースが無く、30,000 語への実データ拡張はまだ始められない
  - 実装方針:
    - 5,000、10,000、30,000 の各段階でデータを増やす
    - 各段階でサンプルレビュー、`doctor`、quizability、seed/import 時間を確認する
  - 実装メモ:
    - `eitango validate --embedded-core` は現行 1,051 語で green
    - 追加の licensable source data が入り次第、`validate` → `import` / `words_core.jsonl` 生成 → `doctor` の順で段階投入に進める
  - 依存: 語彙生成・検証パイプライン
  - 完了条件: 各段階で品質ゲートを満たしつつ 30k へ進める

- [x] 大規模語彙向けの診断と性能確認を強化する
  - 種別: 完了
  - 現状: `doctor` と既存テストはあるが、30k 規模での 4 択生成安定性や seed/import 性能の確認はまだ薄い
  - 実装方針:
    - `doctor` と関連テストを 30k 前提でも有効な形に広げる
    - seed/import が律速なら `word_write` の upsert を chunked/bulk 寄りに改善する
  - 実装メモ:
    - `internal/store` に embedded core seed benchmark、`internal/quiz` に choice build benchmark を追加して基準化を始める
    - `doctor` に `word metadata` check を追加し、欠損 `level` / `frequency_rank` / `distractor_group` と same-source rank 重複を warning で拾えるようにした
    - `word_write` の upsert は source ごとの既存 row を先に map 化するように変え、seed benchmark は `21.9ms -> 18.4ms` 相当まで改善した
  - 依存: 語彙生成・検証パイプライン
  - 完了条件: 30k 想定でも quiz、stats、session の既存挙動を崩さずに回せる

- [ ] 移行導線とドキュメントを仕上げる
  - 種別: blocked
  - 現状: `CoreWordsVersion`、`reset --reseed`、CLI ヘルプはあるが、30k 版への移行前提では実データと version bump がまだ無い
  - 実装方針:
    - `CoreWordsVersion` 更新、reseed 動作確認、既存ユーザーの進捗リセット方針整理を行う
    - 関連ドキュメントと CLI ヘルプの説明を更新する
  - 実装メモ:
    - 配布方針、validation 導線、benchmark、metadata 診断までは先に整備済み
    - 30k データ本体が揃い次第、version bump と reseed 導線の最終仕上げに進める
  - 依存: 語彙データの段階拡張, 大規模語彙向けの診断と性能確認
  - 完了条件: 30k 版の配布・移行・運用方針が明文化される

## 推奨着手順

- 完了済み: `eitango review`, `config.toml` 読み込み, `--focus-mode`
- 完了済み: `eitango doctor`
- 完了済み: Phase 1 辞書パック拡張
- 完了済み: `eitango reset`
- 完了済み: `eitango export`
- 完了済み: `words.source` migration + `eitango import`
- 完了済み: `.goreleaser.yaml` の整理
- 完了済み: 30k 語彙の配布方針を固める
- 完了済み: 30k 語彙のデータ契約を定義する
- 完了済み: 語彙生成・検証パイプラインを追加する
- 完了済み: 大規模語彙向けの診断と性能確認を強化する
1. 追加語彙ソースを調達し、5k / 10k / 30k の段階投入を進める
2. `CoreWordsVersion` 更新と reseed / 移行導線を仕上げる
