# Session Modes

`eitango` は session intent (`play` / `review`) と answer mode (`choice` / `write`) を組み合わせて学習フローを構成する。この spec では、単発の実装タスクではなく継続保守で守るべき契約だけを残す。

## CLI and Persistence

- public command は `eitango play [choice|write]` と `eitango review [choice|write]` とする
- `learn` は `play` の互換 alias として残す
- answer mode を省略した場合は `choice` を既定値とする
- active session は保存済みの `answer_mode` で再開する
- legacy DB row に `answer_mode` が無い場合は `choice` として扱い、`doctor` は read-only 診断を壊さず migration drift だけを報告する

## Home Start Rules

- ホーム画面では `Tab` で `choice / write` を切り替え、`Enter` は play、`r` は review、`n` は新しい learn 開始要求として扱う
- active session が無い場合は、`Enter` / `r` / `n` は即開始する
- active session がある場合、`Enter` はホームで選択中の `answer_mode` が active session と同じときだけ再開する
- active session がある状態で `Enter` の mode が食い違う場合、または `r` / `n` で新 session を要求した場合は、開始前に abandon 確認 overlay を出す
- abandon 確認の accept は active session を `abandoned` にしてから pending request を開始し、cancel は active session とホーム上の選択 answer mode を維持する

## Write Mode Contract

- `write` は `meaning_ja` を prompt にし、`trim + lower-case + exact match` で正誤判定する
- `write` 中の通常入力を阻害しないため、hint と skip は文字キーではなく補助操作へ割り当てる
- 既定操作では `Tab` が staged hint、`Ctrl+S` が skip、`Enter` が confirm、`Esc` が離脱に使われる
- hint は初回に先頭文字を開示し、5 文字以上なら末尾文字も開示する。その後は未開示文字を中心から外側へ 1 文字ずつ開示する
- write の rating は自動決定し、hint なし正解は `Easy`、hint あり正解は `Good`、不正解と skip は `Again` とする
- write の feedback は `Enter` だけで保存して次へ進み、choice feedback のような rating shortcut は出さない
- answer を露出しすぎないため、write 中の音声再生と autoplay は prompt 画面ではなく feedback 画面だけで有効にする

## Write Difficulty

- `write_mode_difficulty` は `basic` / `hard` の 2 値だけを受け付け、未指定時は `basic` とする
- `basic` は `mode=learn` かつ `answer_mode=write` の新規候補だけに作用し、Choice で一度見た語 (`reviews.answer_mode = 'choice'`) から出題する
- `basic` の候補は due 除外を維持し、通常の新規語と同じ順序で扱う
- `basic` は候補不足時に fallback 補完しない
- `hard` は従来どおり通常の新規語候補を使う
- write difficulty の設定面はホーム設定または `config.toml` とし、CLI flag による一時 override は持たない

## Review Fallback

- review 開始時は、まず due 語だけを対象にする
- due 語が 0 件で、かつ review 履歴が 1 件以上ある語が存在する場合は、即エラーではなく reviewed-only fallback 確認を返す
- reviewed-only fallback は過去に出題済み語だけをランダム出題する review practice として扱う
- fallback session は通常 review と別 mode として保持し、active session 再開後も reviewed-only practice であることを失わない
- fallback session の回答は `progress` / `due_at` / interval など SRS 状態を更新しない
- fallback session の choice / write feedback は `Enter` だけで次へ進み、rating は保存しない
- due 語も review 履歴も無い場合だけ、従来どおり `no words available for this session` を返す
- startup の `eitango review [choice|write]` 経路でも同じ fallback 確認を使う
