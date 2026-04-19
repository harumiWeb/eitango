# 2026-04-19 語彙追加 35000 seed batch

## Goal

- `34000seed` まで進んだ bundled core 語彙拡張を継続し、既存の Leipzig + Japanese WordNet ベースの review workflow に沿って次の batch を追加する。
- `tmp/generated_vocab` の生成物と `assets/words_core.jsonl` を矛盾なく更新する。

## Scope

- `34001-35000` の parallel review slice 作成と承認結果の反映
- `approved_review_candidates.tsv` / `approved_seed.csv` / `assets/words_core.jsonl` の更新
- 新規追加帯の noun / verb / adjective / adverb の spot audit と post-apply sync

## Non-Goals

- 既存 34000 rank 以下の承認済み語彙の再審査
- 語彙生成アルゴリズムや score 閾値の仕様変更
- DB schema やアプリ側ロジックの変更

## Required Behavior

- 既存の `scripts/vocab/*.py` workflow を使い、手作業で TSV を再構成しない。
- `34001-35000` の候補だけを今回の review 対象とし、既承認語は重複反映しない。
- `approved_slice_*.tsv` へ入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する。
- `merge_parallel_reviews.py` と `apply_review_batch.py` を使って承認結果を bundled core へ反映する。
- 反映後は新規追加帯に対して、代表義から外れた gloss、verb の述語形、人名詞 / place / daily / abstract noun 系の `distractor_group` ドリフト、quality / condition adjective と thinking / business / communication verb 系のずれを監査する。
- post-apply の監査で既存 entry を補正するときは、`approved_review_candidates.tsv` と `approved_slice_*.tsv` だけでなく `assets/words_core.jsonl` も同期更新する。`apply_review_batch.py` は既存 lemma/pos を再適用しない。
- `go test ./...`、`go build ./...`、`go run ./cmd/eitango validate --embedded-core`、fresh data dir の `go run ./cmd/eitango stats` → `go run ./cmd/eitango doctor` で整合性を確認する。

## Acceptance

- `tmp/generated_vocab/parallel_review_35000/approved_slice_*.tsv` が揃い、freeze 用の `parallel_review_35000_final` が作られる。
- `approved_review_candidates.tsv` / `approved_seed.csv` に `34001-35000` の承認語が追加される。
- `assets/words_core.jsonl` に新規 201 語が追加され、embedded core が 9423 entries になる。
- `laughingstock` / `monotheism` / `monotone` / `overcharge` / `oversimplification` / `poverty-stricken` / `recollect` / `restatement` / `ruminate` の post-apply 補正を反映した状態でも validate / doctor が問題なしを返す。

---

# 2026-04-19 語彙追加 34000 seed batch

## Goal

- `33000seed` まで進んだ bundled core 語彙拡張を継続し、既存の Leipzig + Japanese WordNet ベースの review workflow に沿って次の batch を追加する。
- `tmp/generated_vocab` の生成物と `assets/words_core.jsonl` を矛盾なく更新する。

## Scope

- `33001-34000` の parallel review slice 作成と承認結果の反映
- `approved_review_candidates.tsv` / `approved_seed.csv` / `assets/words_core.jsonl` の更新
- 新規追加帯の noun / verb / adjective / adverb の spot audit と post-apply sync

## Non-Goals

- 既存 33000 rank 以下の承認済み語彙の再審査
- 語彙生成アルゴリズムや score 閾値の仕様変更
- DB schema やアプリ側ロジックの変更

## Required Behavior

- 既存の `scripts/vocab/*.py` workflow を使い、手作業で TSV を再構成しない。
- `33001-34000` の候補だけを今回の review 対象とし、既承認語は重複反映しない。
- `approved_slice_*.tsv` へ入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する。
- `merge_parallel_reviews.py` と `apply_review_batch.py` を使って承認結果を bundled core へ反映する。
- 反映後は新規追加帯に対して、代表義から外れた gloss、verb の述語形、人名詞 / place / daily / abstract noun 系の `distractor_group` ドリフト、quality / condition adjective のずれを監査する。
- post-apply の監査で既存 entry を補正するときは、`approved_review_candidates.tsv` と `approved_slice_*.tsv` だけでなく `assets/words_core.jsonl` も同期更新する。`apply_review_batch.py` は既存 lemma/pos を再適用しない。
- `go test ./...`、`go build ./...`、`go run ./cmd/eitango validate --embedded-core`、fresh data dir の `go run ./cmd/eitango stats` → `go run ./cmd/eitango doctor` で整合性を確認する。

## Acceptance

- `tmp/generated_vocab/parallel_review_34000/approved_slice_*.tsv` が揃い、freeze 用の `parallel_review_34000_final` が作られる。
- `approved_review_candidates.tsv` / `approved_seed.csv` に `33001-34000` の承認語が追加される。
- `assets/words_core.jsonl` に新規 152 語が追加され、embedded core が 9222 entries になる。
- `admittance` / `aftershock` / `blare` / `brouhaha` / `curtsy` の `distractor_group` 補正と `enclose` / `flask` の gloss 補正を反映した状態でも validate / doctor が問題なしを返す。

---

# 2026-04-19 語彙追加 33000 seed batch

## Goal

- `32000seed` まで進んだ bundled core 語彙拡張を継続し、既存の Leipzig + Japanese WordNet ベースの review workflow に沿って次の batch を追加する。
- `tmp/generated_vocab` の生成物と `assets/words_core.jsonl` を矛盾なく更新する。

## Scope

- `32001-33000` の parallel review slice 作成と承認結果の反映
- `approved_review_candidates.tsv` / `approved_seed.csv` / `assets/words_core.jsonl` の更新
- 新規追加帯の abstract noun / adverb / people-place-daily 系 `distractor_group` の spot audit

## Non-Goals

- 既存 32000 rank 以下の承認済み語彙の再審査
- 語彙生成アルゴリズムや score 閾値の仕様変更
- DB schema やアプリ側ロジックの変更

## Required Behavior

- 既存の `scripts/vocab/*.py` workflow を使い、手作業で TSV を再構成しない。
- `32001-33000` の候補だけを今回の review 対象とし、既承認語は重複反映しない。
- `approved_slice_*.tsv` へ入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する。
- `merge_parallel_reviews.py` と `apply_review_batch.py` を使って承認結果を bundled core へ反映する。
- 反映後は新規追加帯に対して、adverb の副詞形、verb の述語形、人名詞 / place / daily / abstract noun 系の `distractor_group` ドリフトを監査する。
- post-apply の監査で既存 entry を補正するときは、`approved_review_candidates.tsv` と `approved_slice_*.tsv` だけでなく `assets/words_core.jsonl` も同期更新する。`apply_review_batch.py` は既存 lemma/pos を再適用しない。
- `go test ./...`、`go build ./...`、`go run ./cmd/eitango validate --embedded-core`、fresh data dir の `go run ./cmd/eitango stats` → `go run ./cmd/eitango doctor` で整合性を確認する。

## Acceptance

- `tmp/generated_vocab/parallel_review_33000/approved_slice_*.tsv` が揃い、freeze 用の `parallel_review_33000_final` が作られる。
- `approved_review_candidates.tsv` / `approved_seed.csv` に `32001-33000` の承認語が追加される。
- `assets/words_core.jsonl` に新規 264 語が追加され、embedded core が 9070 entries になる。
- `meanness` / `nonviolence` / `peacefulness` / `snobbery` / `vagueness` / `viciousness` / `wildness` / `dreamworld` / `gaiety` の `distractor_group` が `abstract-noun` に補正された状態でも validate / doctor が問題なしを返す。

---

# 2026-04-19 語彙追加 32000 seed batch

## Goal

- `31000seed` まで進んだ bundled core 語彙拡張を継続し、既存の Leipzig + Japanese WordNet ベースの review workflow に沿って次の batch を追加する。
- `tmp/generated_vocab` の生成物と `assets/words_core.jsonl` を矛盾なく更新する。

## Scope

- `31001-32000` の parallel review slice 作成と承認結果の反映
- `approved_review_candidates.tsv` / `approved_seed.csv` / `assets/words_core.jsonl` の更新
- 新規追加帯の adverb 代表訳と `distractor_group` の spot audit

## Non-Goals

- 既存 31000 rank 以下の承認済み語彙の再審査
- 語彙生成アルゴリズムや score 閾値の仕様変更
- DB schema やアプリ側ロジックの変更

## Required Behavior

- 既存の `scripts/vocab/*.py` workflow を使い、手作業で TSV を再構成しない。
- `31001-32000` の候補だけを今回の review 対象とし、既承認語は重複反映しない。
- `approved_slice_*.tsv` へ入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する。
- `merge_parallel_reviews.py` と `apply_review_batch.py` を使って承認結果を bundled core へ反映する。
- 反映前に新規追加帯に対して、adverb の非副詞形代表訳、verb の名詞形代表訳、人名詞 / place / daily / quality 系の `distractor_group` ドリフトを監査する。
- 語義と synset が噛み合わない誤承認行は、再 merge せず既存の `approved_review_candidates.tsv` / `approved_seed.csv` を直接補正する。
- `go test ./...`、`go run ./cmd/eitango validate --embedded-core`、fresh data dir の `go run ./cmd/eitango stats` → `go run ./cmd/eitango doctor` で整合性を確認する。

## Acceptance

- `tmp/generated_vocab/parallel_review_32000/approved_slice_*.tsv` が揃う。
- `approved_review_candidates.tsv` / `approved_seed.csv` に `31001-32000` の承認語が追加される。
- `assets/words_core.jsonl` の行数が増え、追加語が bundled core に含まれる。
- `accumulator` のような誤った語義行を除外した新規 397 語の反映後も validate / doctor が問題なしを返す。

---

# 2026-04-19 語彙追加 31000 seed batch

## Goal

- `30000seed` まで進んだ bundled core 語彙拡張を継続し、既存の Leipzig + Japanese WordNet ベースの review workflow に沿って次の batch を追加する。
- `tmp/generated_vocab` の生成物と `assets/words_core.jsonl` を矛盾なく更新する。

## Scope

- `30001-31000` の parallel review slice 作成と承認結果の反映
- `approved_review_candidates.tsv` / `approved_seed.csv` / `assets/words_core.jsonl` の更新
- 新規追加帯の `verb` 代表訳と `distractor_group` の spot audit

## Non-Goals

- 既存 30000 rank 以下の承認済み語彙の再審査
- 語彙生成アルゴリズムや score 閾値の仕様変更
- DB schema やアプリ側ロジックの変更

## Required Behavior

- 既存の `scripts/vocab/*.py` workflow を使い、手作業で TSV を再構成しない。
- `30001-31000` の候補だけを今回の review 対象とし、既承認語は重複反映しない。
- `approved_slice_*.tsv` へ入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する。
- `merge_parallel_reviews.py` と `apply_review_batch.py` を使って承認結果を bundled core へ反映する。
- 反映後は新規追加帯に対して、`verb` の名詞形代表訳と人名詞 / 食べ物 / 動物 / adverb group の `distractor_group` ドリフトを監査する。
- `go test ./...`、`go run ./cmd/eitango validate --embedded-core`、fresh data dir の `go run ./cmd/eitango stats` → `go run ./cmd/eitango doctor` で整合性を確認する。

## Acceptance

- `tmp/generated_vocab/parallel_review_31000/approved_slice_*.tsv` が揃う。
- `approved_review_candidates.tsv` / `approved_seed.csv` に `30001-31000` の承認語が追加される。
- `assets/words_core.jsonl` の行数が増え、追加語が bundled core に含まれる。
- 新規 371 語の反映後も validate / doctor が問題なしを返す。

---

# 2026-04-11 語彙追加 30000 seed batch

## Goal

- `29000seed` まで進んだ bundled core 語彙拡張を継続し、既存の Leipzig + Japanese WordNet ベースの review workflow に沿って次の batch を追加する。
- `tmp/generated_vocab` の生成物と `assets/words_core.jsonl` を矛盾なく更新する。

## Scope

- `tmp/generated_vocab/meaning_candidates.jsonl` / `review_candidates.tsv` の再生成
- `29001-30000` の parallel review slice 作成と承認結果の反映
- `approved_review_candidates.tsv` / `approved_seed.csv` / `assets/words_core.jsonl` の更新

## Non-Goals

- 既存 18000 rank 以下の承認済み語彙の再審査
- 語彙生成アルゴリズムや score 閾値の仕様変更
- DB schema やアプリ側ロジックの変更

## Required Behavior

- 既存の `scripts/vocab/*.py` workflow を使い、手作業で TSV を再構成しない。
- `29001-30000` の候補だけを今回の review 対象とし、既承認語は重複反映しない。
- `approved_slice_*.tsv` へ入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する。
- `merge_parallel_reviews.py` で承認済み TSV を統合し、`apply_review_batch.py` で bundled core へ反映する。
- 反映後は新規追加帯に対して、`verb` の名詞形代表訳と人名詞 / 食べ物 / 動物の `distractor_group` ドリフトを監査する。
- `go run ./cmd/eitango validate --embedded-core` と `go run ./cmd/eitango doctor` で語彙データの整合性を確認する。

## Acceptance

- `review_candidates.tsv` が `29000` より後ろの rank を含む状態へ更新される。
- `tmp/generated_vocab/parallel_review_30000/approved_slice_*.tsv` が揃う。
- `approved_review_candidates.tsv` / `approved_seed.csv` に `29001-30000` の承認語が追加される。
- `assets/words_core.jsonl` の行数が増え、追加語が bundled core に含まれる。

---

# 2026-04-04 winget 配布追加

## Goal

- GitHub Releases の Windows zip を使って winget community repository へ manifest を提出できるようにする。
- release 実行時に `harumiWeb/winget-pkgs` fork へ manifest を push し、`microsoft/winget-pkgs` への PR は手動で作成する。

## Scope

- `.goreleaser.yaml` の archive / winget publish 設定
- `.github/workflows/release.yml` の secret 注入
- README / README.en / ADR の Windows 導線更新

## Non-Goals

- Windows installer の新規実装
- GitHub Releases の asset naming 変更
- `install.sh` の Windows 対応
- update check の通知仕様変更

## Required Behavior

- darwin / linux の release archive は引き続き `tar.gz` を使い、`install.sh` が前提とする asset naming を変えない。
- Windows 向けには winget 用の zip archive を 1 系統だけ生成し、winget manifest はその zip だけを参照する。
- `winget.package_identifier` は `HarumiWeb.Eitango` に固定する。
- `winget.license` はリポジトリ実態に合わせて `Apache-2.0` を使う。
- winget publish 用の token は `repository.token: "{{ .Env.WINGET_GITHUB_TOKEN }}"` として `GITHUB_TOKEN` から分離する。
- release workflow は `WINGET_GITHUB_TOKEN` を GoReleaser step へ渡す。
- GoReleaser は fork への push までを自動化し、upstream への cross-repository PR 作成は行わない。
- README / README.en は Windows の primary install 導線として `winget install HarumiWeb.Eitango` を案内する。

## Acceptance

- `goreleaser check` が通る。
- `goreleaser release --snapshot --clean --skip=publish` が通る。
- snapshot の `dist/winget` に `HarumiWeb.Eitango` の manifest 群が生成される。
- snapshot の installer manifest が GitHub Releases の Windows zip を参照する。
- darwin / linux の `tar.gz` naming は従来どおり維持される。

---

# 2026-04-01 PR #12: Codacy follow-up

## Goal

- PR #12 の Codacy で出ている `doctor.go` の Security 指摘と SQLite migration への TSQLLint 誤検知を解消する。

## Scope

- `internal/store/doctor.go` の schema introspection query
- `internal/store/doctor_test.go` の回帰テスト
- Codacy 向けの tool-scoped ignore 設定

## Non-Goals

- `doctor` の診断仕様そのものの変更
- SQLite migration SQL の文法や適用順の変更
- 他 tool の有効/無効設定変更

## Required Behavior

- `tableHasColumn` は string-formatted SQL を使わず、許可した table 名に対する定数 PRAGMA だけを実行する。
- 想定外の table 名は query 実行前にエラーとして拒否する。
- Codacy では SQLite migration 群に `tsqllint` を適用しない。
- `play/review [choice|write]` は想定外の位置引数を受け付けず、`wrtie` のような typo を default `choice` として黙って実行しない。
- locale format string を触ったテストは、空文字確認だけで済ませず、引数個数と出力文字列まで検証する。

## Acceptance

- `go test ./internal/store` が通る。
- `go test ./cmd/eitango ./internal/i18n ./internal/store` が通る。
- Codacy issue の `internal/store/doctor.go` Security 2 件が再発しない。
- `assets/migrations/005_answer_modes.sql` の Compatibility 指摘が `.codacy.yaml` で抑止される。

---

# 2026-04-02 issue #14: Writeモードの難易度緩和

## Goal

- Write モードの初見難易度を下げるため、`basic` では Choice で一度見た語だけを Write の新規枠に出す。
- 上級者向けに現行挙動を維持する `hard` を残す。

## Scope

- `config.Settings` と `config.toml` の新設定
- ホーム設定 UI と locale 文言
- Learn + Write の session 候補選定
- store の Choice 既出語クエリ
- 回帰テストと README 更新

## Non-Goals

- CLI flag による一時 override
- Review モードの候補選定変更
- SRS アルゴリズム変更
- DB schema / migration 追加

## Required Behavior

- `write_mode_difficulty` 設定を追加し、値は `basic` / `hard`、未設定時は `basic` とする。
- ホーム設定に Write 難易度を追加し、`←/→` で切り替えて保存できる。
- `mode=learn` かつ `answer_mode=write` かつ `write_mode_difficulty=basic` のときだけ、新規候補を Choice 既出語から取得する。
- Choice 既出語の条件は `reviews.answer_mode = 'choice'` が 1 件以上あることとする。
- `basic` 用の新規候補は due 除外を守り、並び順は `ListNewWords` と同じ `frequency_rank ASC, id ASC` にする。
- `basic` では候補不足時の fallback 補完を入れず、利用可能件数だけで session を組む。
- `hard` は従来どおり `ListNewWords` を使う。

## Acceptance

- `go test ./internal/config ./internal/app ./internal/store ./cmd/eitango` が通る。
- `basic` で Choice 履歴のない語が Write 新規枠に出ない回帰がある。
- `hard` で従来どおり未出題語が Write 新規枠に出る回帰がある。
- 設定未指定で `basic` が使われる回帰がある。
- README / README.en に設定項目と `basic` の制約が反映される。

---

# 2026-04-10 issue #44: 復習モードの reviewed-only fallback

## Goal

- due の復習予定が 0 件でも、過去に出題済みの語だけを使って復習モードを続けられるようにする。
- due-only の通常復習と、due が尽きたときの reviewed-only fallback を UI 上で明示的に切り替える。

## Scope

- review session 開始時の候補選定
- ホーム / startup review 開始時の確認 overlay
- locale / README の review 説明
- store / app の回帰テスト

## Non-Goals

- due が残っている session の途中で、同一 session 内に reviewed-only 問題を差し込むこと
- SRS の interval / due 計算変更
- DB schema / migration 追加

## Required Behavior

- review 開始時は、まず従来どおり due 語のみを優先する。
- due 語が 0 件で、かつ review 履歴が 1 件以上ある語が存在する場合は、即エラーにせず reviewed-only fallback 確認を返す。
- reviewed-only fallback の確認文言は「過去に出題済み語だけをランダム出題する」ことを明示する。
- fallback を承認した review session は、`reviews` に履歴のある語だけを `ORDER BY RANDOM()` で抽出し、`QuestionCount` 上限まで出題する。
- fallback session の item kind は通常 review と同じ `review` のままにする。
- fallback session は内部的に通常 review と別 mode として保持し、active session 再開時も「reviewed-only practice」であることを失わない。
- fallback session の回答は `progress` / `due_at` / interval など SRS 状態を更新しない。
- fallback session の回答結果は review row と統計には残してよいが、choice の feedback では 4 段階評価を出さず `Enter` だけで次へ進む。
- fallback session の write feedback も rating を保存せず、`Enter` だけで次へ進む。
- review 履歴が 0 件のときだけ、従来どおり `no words available for this session` を返す。
- startup の `eitango review [choice|write]` 経路でも、同じ fallback 確認 overlay を使う。
- active session を置き換える review 開始要求でも、fallback 確認を出した時点では active session をまだ abandon しない。

## Acceptance

- `go test ./internal/app ./internal/store ./cmd/eitango` が通る。
- due 語が 0 件かつ review 履歴ありのとき、review 開始コマンドが fallback 確認 message を返す回帰がある。
- fallback 承認後の review session が reviewed-only のランダム出題になる回帰がある。
- fallback session の回答で SRS progress が変化しない回帰がある。
- fallback session の choice feedback で rating key が消え、`Enter` だけで次へ進む回帰がある。
- fallback 確認をキャンセルしたとき、active session がある場合はそれが維持される回帰がある。
- due 語も review 履歴も 0 件のときは、従来どおりエラーになる回帰がある。
- README / README.en の review 説明が fallback 挙動に追従する。

---

# 2026-03-29 ドキュメント再編仕様

## Goal

- 旧設計書を廃止し、初期リリース後も参照価値がある判断だけを ADR に残す。
- `docs/specs/` には恒久的な内部仕様と制約だけを残し、コード・README・tests を実装の正本として維持する。

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
- `docs/specs/` への一時メモ追加や index/README の追加
- 旧設計書の全文アーカイブ

## Acceptance

- `docs/adr/` の 3 本が `Status`, `Background`, `Decision`, `Consequences`, `Rationale`, `Supersedes`, `Superseded by` を持つ。
- `rg -n "docs/design\.md|design\.md"` で削除済み設計書への壊れた参照が残らない。
- `go test ./...` が通る。

---

# 2026-03-31 issue #9: write モード追加と play/review × answer mode 整理

## Goal

- `choice` に加えて、日本語の意味を見て英単語を入力する `write` モードを追加する。
- 学習種別を `play/review`、回答方式を `choice/write` の 2 軸で整理する。
- ホーム画面のシンプルさを保ちつつ、`Tab` で回答方式だけを切り替えられるようにする。

## Scope

- TUI の home / quiz / feedback の mode 分岐
- CLI の `play/review [choice|write]` 契約と `learn` alias 維持
- `answer_mode` の session / review 永続化
- write 用の入力、ヒント、skip、auto-rating、回帰テスト
- README の操作説明更新

## Non-Goals

- typo 許容
- `default_answer_mode` 設定
- space / hyphen を含む lemma の write 対応
- typed answer 本文の履歴保存

## Required Behavior

- ホーム画面で `Tab` が `choice/write` を切り替え、`Enter` は play、`r` は review のままにする。
- `play` は正式コマンドとし、`learn` は互換 alias として残す。
- `play/review` は `choice/write` を省略した場合に `choice` を使う。
- `doctor` は read-only のまま pre-`005_answer_modes.sql` DB を診断できる。旧 `sessions` row は `choice` として扱い、migration drift は `migrations` check で報告する。
- `write` は `MeaningJA` を prompt にし、`trim + lower-case + exact match` で正誤判定する。
- `write` 中の通常入力を阻害しないため、ヒントと skip は `Tab` / `Ctrl+S` に割り当てる。
- `write` のヒントは初回に先頭文字と、5 文字以上なら末尾文字も開示し、その後は未開示文字を中心から外側へ 1 文字ずつ開示する。
- `write` の rating は自動で決める。ヒントなし正解は `Easy`、ヒントあり正解は `Good`、不正解と skip は `Again`。
- `write` の feedback は `Enter` だけで保存して次へ進む。help / status / quit 無効表示も採点ショートカットではなく Enter 操作に合わせる。
- active session は永続化した `answer_mode` で再開する。旧 row は `choice` として扱う。

## Acceptance

- `go test ./...` が通る。
- `play/review [choice|write]` の command tree と `learn` alias の回帰がある。
- home の mode toggle、write の入力/ヒント/skip/auto-rating、store の `answer_mode` 永続化に回帰テストがある。
- README / README.en が新しい操作体系を案内する。

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

---

# 2026-03-30 issue #4: curl installer / uninstall / version pin

## Goal

- macOS / Linux ユーザー向けに `curl | sh` の最短 install 導線を追加する。
- installer から特定 release tag の導入と uninstall を実行できるようにする。
- download した archive は GitHub Releases の `checksums.txt` で SHA256 検証してから展開する。

## Scope

- ルート `install.sh`
- release archive を前提にした installer integration test
- README / README.en / CHANGELOG の install 説明
- 配布ポリシー ADR の更新

## Non-Goals

- Windows PowerShell installer の追加
- Go 本体への self-update サブコマンド追加
- update check の TTL / 表示仕様変更
- `.goreleaser.yaml` の artifact naming 変更

## Required Behavior

- `install.sh` は macOS / Linux のみを対象にし、`x86_64|arm64` を release archive 名へ正規化する。
- `install.sh --version 0.2.0` と `install.sh --version v0.2.0` は同じ release tag `v0.2.0` を解決する。
- `--version` 未指定時だけ GitHub Releases API の latest を参照する。
- installer は archive と同じ release の `checksums.txt` を取得し、SHA256 が一致しない限り install root を置き換えない。
- installer は `~/.eitango/bin/eitango`, `~/.eitango/version`, `~/.eitango/share/` を管理し、法務ファイルを `share/` に保持する。
- `--uninstall` は installer 管理下の `~/.eitango` を削除し、学習 data は既定で保持する。対話確認で purge を有効化しない。
- `--purge-data` 指定時だけ data dir も削除し、`EITANGO_DATA_DIR` がその実行で渡されていればその path を優先する。
- install root 置換に失敗した場合、旧 install の rollback copy を削除せず、復元または backup path に残す。
- PATH は自動変更しない。

## Acceptance

- `go test ./...` で installer の latest install / pinned install / checksum failure / uninstall / failed replace の回帰が通る。
- Ubuntu CI で `shellcheck install.sh` が通る。
- README に latest install, version pin, uninstall, purge-data の例が揃う。

---

# 2026-03-31 PR #7 review follow-up: installer archive layout

## Goal

- `install.sh` が release archive の layout を root 直下前提にせず、単一トップディレクトリで包まれた archive でも展開できるようにする。

## Scope

- `install.sh` の展開後 source dir 解決
- `install_test.go` の archive fixture と回帰テスト

## Non-Goals

- `.goreleaser.yaml` の artifact naming や archive 生成ポリシーの変更
- installer の checksum / uninstall / rollback 契約の変更

## Required Behavior

- 展開先に単一のトップディレクトリだけが存在する場合は、そのディレクトリ配下を release contents として扱う。
- 展開先に複数エントリがある場合は、従来どおり展開先 root を release contents として扱う。
- installer regression test は wrapped / unwrapped の両方の archive layout を検証する。

## Acceptance

- wrapped archive fixture で既存 install 系 test が通る。
- unwrapped archive fixture でも install success の回帰が通る。
- `go test ./...` が通る。

---

# 2026-03-31 issue #8: update 通知の stale latest tag

## Goal

- 新しい release 公開直後でも、ホーム画面の update 通知が前回 release tag を長時間表示し続けないようにする。

## Scope

- TUI 起動時の update check 呼び出し
- update check の forced refresh / cached fallback 回帰テスト
- README / README.en / CHANGELOG の update 通知説明
- 配布・update policy ADR の整合更新

## Non-Goals

- semver 比較ロジックの変更
- `install.sh` の latest release 解決方式変更
- GitHub Releases API を `/releases/latest` から全件走査へ切り替えること

## Required Behavior

- ホーム画面の update 通知は起動ごとに非同期で最新 release を再検証する。
- network request が timeout / offline / API failure になった場合だけ、保存済み `update-check.json` の latest 情報へ fallback する。
- 初回の successful check では通知を出さず、2 回目以降に新しい版が確認されたときだけ通知対象にする既存仕様は維持する。
- `eitango version` の latest release 表示と、manual update フローは従来どおり維持する。

## Acceptance

- ホーム画面の update check が `CheckNow` 経由で forced refresh を使うことを unit test で確認できる。
- cached `v0.2.0` が存在しても forced refresh 成功時は `v0.2.1` が返る回帰 test が通る。
- forced refresh 失敗時は cached latest 情報へ fallback する回帰 test が通る。
- `go test ./...` が通る。

---

# 2026-04-03 issue #15: 進行中セッション時のホーム開始競合確認

## Goal

- 進行中セッションがある状態でホームから別の開始操作を行ったとき、既存セッションを無言で再開せず、破棄確認を挟めるようにする。

## Scope

- `internal/app` のホーム画面 state machine / overlay / help
- locale 文言
- ホーム操作の回帰テスト

## Non-Goals

- CLI `play` / `review` の active session 再開ポリシー変更
- store / DB schema の変更
- review / write feedback 画面の操作変更

## Required Behavior

- active session がない場合、`Enter` / `r` / `n` は従来どおり即開始する。
- active session がある場合:
  - `Enter` は、ホームで選択中の `answer_mode` が active session と同じなら再開する。
  - `Enter` は、選択中 `answer_mode` が active session と異なるなら、active session 破棄確認を開く。
  - `r` は常に新しい review session 開始要求として扱い、active session があれば破棄確認を開く。
  - `n` は常に新しい learn session 開始要求として扱い、active session があれば破棄確認を開く。
- 破棄確認 overlay の `Enter` は active session を `abandoned` にしてから pending request を開始する。
- 破棄確認 overlay の `Esc` / `b` はキャンセルし、active session は保持する。
- 破棄確認をキャンセルしても、ホームで `Tab` で選んだ `answer_mode` は維持する。
- 破棄確認 overlay は current active session の進行状況と、これから始める target mode / answer mode を明示する。

## Acceptance

- active learn/choice 中に `Tab` で write へ切替えて `Enter` すると、即再開ではなく破棄確認 overlay が出る。
- 上記 overlay で `Enter` すると旧 session は `abandoned` になり、新 session は `learn/write` で開始する。
- 上記 overlay で `Esc` すると旧 session は active のまま残り、ホームへ戻る。
- active session 中の `r` と `n` も同様に破棄確認 overlay を経由する。
- `go test ./internal/app ./internal/i18n` と必要なら `go test ./...` が通る。

---

# 2026-04-03 issue #19: macOS / Windows 音声再生

## Goal

- macOS / Windows で現在の英単語を発話できるようにする。
- `Ctrl+P` の手動再生と、`Shift+Tab` で切り替えるセッション内自動再生を TUI に追加する。
- 音声が使えない環境でも学習フローを壊さず、Linux では no-op で継続できるようにする。

## Scope

- `internal/audio` の追加と OS 別 backend
- `config.Settings` / `config.toml` の音声設定
- `internal/app` / `internal/tui` の keymap・state・表示
- locale 文言、README、回帰テスト

## Non-Goals

- Linux の音声 backend 実装
- 例文読み上げ
- stop / pause / seek
- voice / speed / volume の詳細設定
- 音声ファイル cache

## Required Behavior

- `internal/audio` に `Speaker` interface を置き、音声 backend の差分を app 層から分離する。
- macOS は `say`、Windows は `powershell.exe` + `System.Speech.Synthesis.SpeechSynthesizer` を使う。
- `audio_enabled` と `audio_autoplay` を flat config key として追加し、既定値は `true` / `false` とする。
- ホーム設定 overlay で audio on/off と autoplay on/off を `←/→` で切り替えて保存できる。
- quiz / feedback 画面で `Ctrl+P` が現在語を再生する。
- quiz / feedback 画面で `Shift+Tab` が現在セッションだけの autoplay on/off を切り替える。config は即時保存しない。
- autoplay は session 開始直後と次の問題読込直後にだけ発火する。
- 音声 unavailable / 再生失敗は status line の非致命メッセージにとどめ、学習セッションは継続する。
- quiz / feedback に autoplay state を明示し、help / key guide / README も新操作へ追従する。

## Acceptance

- `go test ./internal/audio ./internal/config ./internal/app ./internal/i18n` が通る。
- `go test ./...` が通る。
- macOS / Windows CI で `internal/audio` が build / test できる。
- `Ctrl+P` 手動再生、`Shift+Tab` セッション内 toggle、`audio_enabled` / `audio_autoplay` の回帰テストがある。

---

# 2026-04-06 issue #28: 無色モードとテーマカラー

## Goal

- 色がなくても主要 UI を判読できるようにし、アクセシビリティを改善する。
- 設定で `default` / `no_color` / `neon` / `custom` のテーマモードを選べるようにする。
- `custom` では `config.toml` から role-based の RGB 値を指定できるようにする。

## Scope

- `config.Settings` / `config.toml` のテーマ設定
- `internal/tui` の style 生成
- `internal/app` のホーム設定 overlay と色非依存の識別
- locale 文言、README、回帰テスト
- `docs/specs/` の恒久仕様

## Non-Goals

- CLI `--no-color` flag の追加
- ホーム設定 overlay からの RGB 直接編集
- 背景色や画面ごとの完全自由なスタイル編集
- terminal capability に応じた dynamic theme 切替

## Required Behavior

- `config.Settings` に `theme_mode` と `theme_palette` を追加する。
- `theme_mode` は `default` / `no_color` / `neon` / `custom` の 4 値だけを受け付ける。
- `theme_palette` は `accent` / `success` / `danger` / `muted` / `border` の 5 slot を受け付ける。
- `theme_palette` の色文字列は `#RRGGBB` だけを受け付け、load/save 時に trim して大文字へ正規化する。
- `theme_palette` の未指定 slot は default palette に fallback する。
- `theme_palette` を保存するとき、未指定 slot は `""` で書かず key ごと omit する。
- `theme_mode != "custom"` のときも `theme_palette` は保存対象に残し、後で `custom` に戻したとき再利用できるようにする。
- `internal/tui` は固定色直書きをやめ、settings から theme を解決して `Styles` を生成する。
- `default` は旧 `NewStyles()` の見た目を維持し、`no_color` は色指定なし、`neon` はライトグリーン基調の高コントラスト preset を提供する。
- ホーム設定 overlay では theme mode だけを `←/→` で切り替えて保存できる。
- `custom` 選択中は `config.toml` の `theme_palette` 編集を案内する note を表示する。
- 選択状態やエラー状態は色だけに依存させない。
  - answer mode tab の選択中は bracket 記法で示す。
  - settings row の選択中は prefix 記号で示す。
  - status line は通常 `status:`、エラー時 `error:` で示す。
- `choice` の `▸`、feedback の `✓/✗`、audio の `ON/OFF` は維持する。

## Acceptance

- `theme_mode = "no_color"` で主要画面が色なしでも判読できる。
- `theme_mode = "neon"` でライトグリーン基調の preset が反映される。
- `theme_mode = "custom"` と `theme_palette` で role-based 色を保存・再読込できる。
- ホーム設定 overlay で theme mode を切り替えて保存できる。
- README / README.en に設定例が記載される。
- `go test ./internal/config ./internal/tui ./internal/app ./internal/i18n` が通る。
- 必要なら `go test ./...` が通る。

---

# 2026-04-07 issue #34: キーバインドカスタマイズ

## Goal

- TUI のキーバインドを `config.toml` とホーム設定内の editor から変更できるようにする。
- key 変更後の runtime input 判定、help、各画面の key guide を再起動なしで同期させる。

## Scope

- `config.Settings` / `config.toml` の keymap 設定
- context-aware keymap runtime
- ホーム設定から開く Key Bindings Editor
- locale / README / specs / 回帰テスト

## Non-Goals

- 複数 keymap profile
- import / export
- ファイル監視による自動 reload
- keymap editor 自身の自由カスタマイズ

## Required Behavior

- `config.toml` は `[keymap]` を受け付け、default からの override だけを保存する。
- supported context は `home`, `home_confirm`, `settings_overlay`, `quiz.choice`, `quiz.write`, `feedback.rate`, `feedback.write`, `results`, `stats`, `help` とする。
- `keymap.version` は `1` のみ許可する。
- unknown context / unknown action / invalid key token / same-context conflict は load/save で error にする。
- `quiz.write` では answer 入力と衝突する英字 1 文字 key を command として保存できない。
- settings 保存後と keymap editor 保存後のどちらでも、runtime keymap は再起動なしで差し替わる。
- keymap editor の save は keymap だけでなく settings overlay の未保存 draft も同時に保存し、overlay 上の変更を巻き戻さない。
- help 画面と各 screen の key guide は current keymap を使って描画する。
- `quiz.choice` のインライン key guide は横幅を抑えるため `select1-4` を表示せず、選択肢キーの詳細は help 画面だけに出す。インラインには help action を残して詳細導線を維持する。
- `help.back` を unbind する設定は保存できない。help 画面の復帰は `back` に依存し、`quit` だけでは代替できない。
- ホーム設定 overlay に keymap editor への導線を追加する。
- keymap editor は context filter、action 一覧、record mode、clear、reset、save を持つ。
- record mode では `Esc` も通常キーとして割り当て可能にし、cancel は別キーで行う。
- keymap editor は terminal 高を超えて伸びず、表示可能高を超える action 一覧だけをスクロール表示する。

## Acceptance

- `config.toml` の `[keymap]` で home / quiz / feedback の key を override できる。
- 未指定 action は既定値を使い続ける。
- keymap editor で変更した key が save 後すぐに反映される。
- help / key guide が custom key を反映する。
- 一般的な terminal 高でも keymap editor 全体が画面内に収まり、cursor 移動で非表示行へ到達できる。
- `go test ./internal/keymap ./internal/config ./internal/app` が通る。
- `go test ./...` が通る。

## Narrow Width Guard v1

### Scope

- `internal/app` の TUI 描画入口
- locale / README / specs / 回帰テスト

### Non-Goals

- 2 行 key guide
- 画面ごとの簡易レイアウト
- data-dependent な長文 overflow の包括対処

---

## 2026-04-11 issue #49: bundled core 更新時の SRS 維持

### Goal

- `dict_version` 更新で bundled core 語彙が差し替わっても、既存 core 語の `word_id` と SRS 進捗を保持したまま新語だけを追加する。
- `reset --reseed` だけを明示的な破壊的リセットとして残し、通常起動時の core sync は non-destructive にする。

### Required Behavior

- core 語の同一性は store の現行実装に合わせて `strings.ToLower(strings.TrimSpace(lemma) + "\x00" + strings.TrimSpace(pos))` で判定する。
- `words` に `is_active` を持たせ、version bump 時の core sync では一致 row を同じ `id` のまま更新し、新語だけを insert し、消えた旧 core は `is_active = 0` にする。
- metadata 更新時は `meaning_ja` / `level` / `frequency_rank` / `distractor_group` / `example_*` に加えて `lemma` / `pos` も embedded core 側の canonical 値へ揃える。
- `SeedWords()` は core 未投入時の初回 seed と、同一 version の no-op を維持しつつ、version bump 時だけ destructive reset ではなく core diff sync を行う。
- version bump 時に active session が存在する場合、question payload を snapshot していない現行設計では再開時に設問文面や distractor が drift するため、その session は sync transaction 内で `abandoned` にする。
- `reset --reseed` は従来どおり learning tables を全削除し、`source='core'` を active/inactive を問わず全削除して bundled core を再投入する。
- future planning に使う query は active core だけを対象にする。対象は `ListDueWords` / `ListNewWords` / `ListWriteBasicCandidates` / `ListReviewedWordsRandom` / `ListWordsByPOS` / `ListDistractorCandidates` / `countDueWords` / `countNewWords`。
- `GetWord()`、export、session summary、履歴参照は inactive core を読めるままにして、過去 session や retired word の review 履歴を壊さない。
- `doctor` は retired core を辞書破損として扱わず、active core と retired core を分けて報告する。pre-006 schema の read-only DB では schema introspection で fallback し、migration drift だけを報告できるようにする。
- DB-level unique 制約は今回追加しない。same-source duplicate は sync 時 validation と `doctor` の duplicate check で検出する。

### Acceptance

- version bump 後も、同一 normalized `lemma/pos` の core 語は `word_id` を保持し、`progress` / `reviews` と completed/abandoned session 履歴が失われない。
- version bump 時点で active だった session は `abandoned` になり、resume 対象から外れる。
- 新規追加された core 語だけが `new` 候補として増え、退役した core 語は新規セッション計画に出なくなる。
- retired core を参照する既存 session item / review history / export が壊れない。
- `reset --reseed` は引き続き learning tables を全削除し、bundled core を完全再投入する。
- `go test ./internal/store ./internal/quiz ./cmd/eitango` と `go test ./...` が通る。

### Required Behavior

- `RootModel.width` が既知で、現在 screen/overlay に対応する最小幅を下回るときは通常 UI を描かず narrow message に切り替える。
- 最小幅は `home/results/stats/quiz.write/feedback.write=56`, `settings overlay/home confirm/quiz.choice/feedback.rate/help=64`, `keymap editor=76` とする。
- `width == 0` の間は narrow guard を無効にし、初回 `WindowSizeMsg` 前の描画挙動は変えない。
- narrow message は現在幅と必要幅を表示し、横幅を広げる案内を出す。
- narrow message 自体と status line は現在幅に収まるよう wrap / width constrain する。
- `Update` の input handling は変えず、狭幅でも既存 key handling はそのまま動く。

### Acceptance

- 代表画面で、しきい値未満では narrow message に切り替わり、しきい値以上では通常画面へ戻る。
- `width == 0` では narrow message に切り替わらない。
- narrow case の `View().Content` は各行 display width が `model.width` を超えない。
- `go test ./internal/app` が通る。

## Narrow Width Compact Layout v2

### Scope

- `internal/app` の主要画面描画
- `docs/specs/tui-layout.md` / README / 回帰テスト

### Non-Goals

- `results` / `stats` / `keymap editor` の compact layout
- input handling や keymap 契約の変更
- locale key の追加

### Required Behavior

- 画面ごとに `normal / compact / narrow` の 3 段階レイアウトを持つ。
- `width >= normalMin` では既存の通常表示を使う。
- `compactMin <= width < normalMin` では compact layout に切り替える。
- `width < compactMin` では v1 と同様に narrow message に切り替える。
- compact 対象は `home`, `home confirm`, `settings overlay`, `help`, `quiz.choice`, `quiz.write`, `feedback.choice`, `feedback.write` とする。
- しきい値は `home/quiz.write/feedback.write=44/56`, `home confirm/settings overlay/help/quiz.choice/feedback.choice=48/64` を `compactMin/normalMin` として固定する。
- `results`, `stats`, `keymap editor` は compact layout を持たず、v1 の narrow guard のまま維持する。
- compact layout では border と余白を減らし、固定幅 `AlignLabel` と単一行 key guide を避ける。
- compact の key guide は current width に収まるよう複数行へ詰めて表示する。
- compact の label/value 表示と help line は長文でも wrap し、各行 display width が `model.width` を超えない。
- `width == 0` の間は compact / narrow 判定を無効にし、初回 `WindowSizeMsg` 前の描画挙動は変えない。
- `Update` の input handling は変えず、compact/narrow でも既存 key handling はそのまま動く。

### Acceptance

- compact 対象画面で `compactMin <= width < normalMin` のとき narrow message に切り替わらず compact layout が出る。
- compact 対象画面で `width < compactMin` のとき narrow message に切り替わる。
- compact 対象画面で `width >= normalMin` のとき通常表示へ戻る。
- `results`, `stats`, `keymap editor` は v2 でも compact に入らず、従来どおり narrow fallback を使う。
- compact / narrow の `View().Content` は各行 display width が `model.width` を超えない。
- `go test ./internal/app` が通る。

## Narrow Width Shrink Panels v3

### Scope

- `internal/app` の主要画面描画全体
- `docs/specs/tui-layout.md` / README / 回帰テスト

### Non-Goals

- CLI / config / keymap schema の変更
- runtime input handling の変更
- locale key の追加

### Required Behavior

- 主要画面の compact layout は border を残したまま terminal 幅へ追従して縮む。
- `compact` は `home`, `home confirm`, `settings overlay`, `help`, `quiz.choice`, `quiz.write`, `feedback.choice`, `feedback.write`, `results`, `stats`, `keymap editor` を対象にする。
- しきい値は `home/results/stats/quiz.write/feedback.write=28/56`, `settings overlay/home confirm/help/quiz.choice/feedback.choice=32/64`, `keymap editor=32/76` を `compactMin/normalMin` として固定する。
- `width >= normalMin` では通常表示を使う。
- `compactMin <= width < normalMin` では border 付き compact layout に切り替える。
- `width < compactMin` では narrow message に切り替える。
- compact layout では panel の horizontal padding を減らし、単一行 UI は `...` による省略で current width に収める。
- `...` の対象は key guide / keymap 表示 / settings row / quiz meta / results-stat summary / keymap editor row とする。
- help 本文、settings note、feedback examples、narrow message 本文のような prose は wrap を維持する。
- `width == 0` の間は compact / narrow 判定を無効にし、初回 `WindowSizeMsg` 前の描画挙動は変えない。

### Acceptance

- compact 対象画面で `compactMin <= width < normalMin` のとき narrow message ではなく border 付き compact layout が出る。
- compact 対象画面で `width < compactMin` のとき narrow message に切り替わる。
- compact / narrow の `View().Content` は各行 display width が `model.width` を超えない。
- 長い custom key binding を入れた compact case で `...` が表示される。
- `go test ./internal/app` が通る。

## 2026-04-08 issue #29 v4: 横幅の連続追従

### Scope

- `internal/app` の主要画面描画
- `docs/specs/tui-layout.md` / README / 回帰テスト

### Non-Goals

- CLI / config / keymap schema の変更
- runtime input handling の変更
- locale key の追加

### Required Behavior

- 主要画面は `normal / compact` の段階切替をやめ、最小幅以上では同じ adaptive renderer で terminal 幅へ連続追従する。
- 最小幅は `home/results/stats/quiz.write/feedback.write=28`, `settings overlay/home confirm/help/quiz.choice/feedback.choice/keymap editor=32` とする。
- `width < minWidth` のときだけ narrow message に切り替える。
- adaptive renderer では border を維持し、horizontal padding を詰める。
- 単一行 UI は width budget に応じて `...` で省略する。
- `quiz.choice` の選択肢本文や `results` の hard words のような主情報は `...` で潰さず、adaptive 幅でも wrap して全文を読めるようにする。
- prose は wrap を維持する。
- `width == 0` の間は narrow 判定を無効にするだけでなく、`renderHome` / `renderQuiz` / `renderFeedback` / `renderResults` / `renderStats` / `renderHelp` / `renderKeymapEditor` など全 screen の従来 renderer を維持し、初回 `WindowSizeMsg` 前の描画挙動を変えない。

### Acceptance

- 主要画面は `width >= minWidth` の複数幅で narrow message に切り替わらない。
- 主要画面の `View().Content` は各行 display width が `model.width` を超えない。
- 長い custom key binding を入れた adaptive case で `...` が表示される。
- `width < minWidth` のとき narrow message に切り替わる。
- 十分な幅の `home` では answer mode の selected state が theme の accent color を保つ。
- 十分な幅の `quiz.write` では label 列が固定幅で整列し、panel の上下余白も残る。
- 十分な幅の `home` では `answer mode / due / new / streak / wait` が従来どおり縦並びの固定幅ラベルで整列する。
- adaptive panel は terminal に追従しつつも、左右の外側余白は持たない。
- adaptive panel の枠内には、左右に 2 文字ぶんの内側余白を残す。
- `quiz.choice` の長い選択肢でも、adaptive 幅で選択肢末尾の違いまで読める回帰テストがある。
- `results` の hard words は adaptive 幅でも全文を読める回帰テストがある。
- `width == 0` のときは `results` / `stats` / `keymap editor` を含む旧 renderer 経路を通す回帰テストがある。
- `go test ./internal/app` が通る。
