# Bundled Core Vocabulary Workflow

bundled core 語彙の追加・補正は、都度の作業ログではなく再現可能な review/apply workflow として保守する。版管理と runtime sync の方針自体は `docs/adr/0002-bundled-core-dictionary-lifecycle.md` に従い、この spec では日常的な語彙メンテ手順だけを固定する。

## Workflow Contract

- bundled core の review/apply は `scripts/vocab/*.py` の既存 workflow を使い、手作業で TSV を再構成しない
- review 対象はその回の rank band だけに絞り、既承認の `lemma/pos` を重複反映しない
- `approved_slice_*.tsv` に入れる行は `status=approved` を守り、`meaning_ja_candidate` と `distractor_group_candidate` を目視確認する
- 承認 slice の統合は `merge_parallel_reviews.py`、bundled core への反映は `apply_review_batch.py` を使う
- `apply_review_batch.py` は既存 `lemma/pos` を再適用しないため、post-apply 監査で既存 entry を補正した場合は `approved_review_candidates.tsv` / `approved_seed.csv` だけでなく `assets/words_core.jsonl` も同期更新する
- retry や比較用の TSV は merge 対象ディレクトリへ `approved_slice*.tsv` 名で置かない。比較用ファイルが必要なら別ディレクトリへ退避する

## Review Rules

- 新規追加帯では learner-dictionary の中心義を優先し、特殊義や派生義を代表訳にしない
- verb の `meaning_ja` は述語形で監査し、名詞形の gloss を残さない
- `distractor_group` は出題時の誤答品質を基準に監査し、特に people / place / daily / abstract noun 系の drift を spot check する
- row の追加後に silent quality drift を残さないため、新規帯ごとに representative gloss と `distractor_group` の spot audit を行う

## Validation

- bundled core 更新後は `go test ./...` を実行する
- bundled core 更新後は `go build ./...` を実行する
- bundled core 更新後は `go run ./cmd/eitango validate --embedded-core` を実行する
- fresh data dir を `EITANGO_DATA_DIR` で分離し、`go run ./cmd/eitango stats` の後に `go run ./cmd/eitango doctor` を実行する
