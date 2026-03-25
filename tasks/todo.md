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

- [ ] Phase 1 の辞書パックを 1000〜3000 語へ拡張する
  - 種別: 一部着手
  - 現状: `assets/words_core.jsonl` は存在するが、現行データ量は smoke test 向けの小規模パックに留まる
  - 実装方針:
    - `lemma`, `pos`, `meaning_ja`, `level`, `frequency_rank`, `distractor_group` を優先し、まずは例文より出題品質を優先する
    - 同一 `pos` / 近い `level` / 同一 `distractor_group` で 4 択が成立するよう、グループ単位でデータを増やす
    - データ拡張とセットで loader / doctor 用の検証を追加する
  - 完了条件: 設計書 Phase 1 相当の語彙量と distractor の最低品質を満たす

## P1: まだ入口が無い運用コマンド群

- [ ] `eitango doctor` を追加する
  - 種別: 未着手
  - 実装方針:
    - 最初は read-only 診断に限定する
    - DB open/migrate, core words version, orphan progress/reviews/session_items, active session の不整合, 4 択を作れない単語群を検査する
    - テキスト出力と終了コードで CI / 手元調査の両方に使える形にする
  - 完了条件: DB/辞書の壊れ方を CLI から切り分けられる

- [ ] `eitango reset` を追加する
  - 種別: 未着手
  - 実装方針:
    - 破壊的操作をデフォルトにしない
    - まずは `--progress`（学習履歴だけ初期化）と `--reseed`（組み込み辞書の再投入）に分ける
    - 将来の import 実装に備えて、core 辞書と user 辞書を分離できるデータモデルを前提にする
  - 依存: import 用の source 管理が入ると拡張しやすい
  - 完了条件: 学習履歴のリセットと辞書再投入を安全に実行できる

- [ ] `eitango browse` を追加する
  - 種別: 未着手
  - 実装方針:
    - 最初は read-only の単語ブラウザに絞る
    - filter は `state`, `level`, `pos`、search は `lemma` ベースで十分
    - 既存の `store` を再利用し、編集機能は後回しにする
  - 完了条件: 学習前に語彙と進捗を一覧できる

- [ ] `eitango export` を追加する
  - 種別: 未着手
  - 実装方針:
    - 先に `wrong-words` CSV と `progress` JSON の 2 経路だけ実装する
    - 出力契約を先に固定し、Anki 取り込みやバックアップ用途を優先する
    - `reviews`, `progress`, `words` を join する read model を store 側に追加する
  - 完了条件: 苦手語 CSV と進捗 JSON を外部へ持ち出せる

- [ ] `eitango import` を追加する
  - 種別: 未着手
  - 実装方針:
    - CSV から始め、必須列は `lemma`, `meaning_ja` に限定する
    - import と reset / doctor を安全にするため、`words` に `source`（`core` / `import:<name>`）相当の識別子を持たせる migration を先に入れる
    - duplicate policy は「同じ source 内は upsert、source を跨ぐ重複は許可して doctor で警告」に寄せる
  - 依存: `words.source` 追加 migration, `doctor`, `reset`
  - 完了条件: CSV から追加辞書を取り込める

## P2: UI / 体験の未着手項目

- [ ] 専用の help 画面を追加する
  - 種別: 一部着手
  - 現状: `?` は status line のヒントだけで、設計書の `ScreenHelp` はまだ無い
  - 実装方針:
    - 画面ごとの keymap を共通の help renderer に寄せる
    - `?` で開く / `Esc` で戻るの往復だけにする
  - 完了条件: home / quiz / feedback / results / stats から一貫した help を見られる

- [ ] 例文表示を quiz / feedback に載せる
  - 種別: 一部着手
  - 現状: schema に `example_en` / `example_ja` はあるが、seed データと画面表示では未使用
  - 実装方針:
    - まずは quiz では非表示、feedback で正答確認と一緒に出す
    - seed データ拡張では例文が無い単語を許容し、表示側で空欄を自然に扱う
  - 依存: 辞書パック拡張
  - 完了条件: 例文付き単語が自然に学習フローへ出る

- [ ] `streak by waiting` を stats / home に追加する
  - 種別: 未着手
  - 実装方針:
    - 既存の `reviews.response_ms` から待機時間変換メトリクスを導出し、最初は追加テーブルを作らず集計で出す
    - 指標名は streak よりも `wait minutes` / `waiting converted` のように誤読しない形に寄せる
  - 完了条件: 通常の連続日数とは別に、待機時間由来の学習量を表示できる

## P3: 配布・将来機能の保留タスク

- [ ] `.goreleaser.yaml` を実運用向けに固める
  - 種別: 一部着手
  - 現状: テンプレートはあるが、成果物名・アーカイブ構成・リリース運用は未整理
  - 実装方針:
    - `cmd/eitango` を前提に成果物名を固定し、Windows zip / Unix tar.gz を最終仕様に合わせる
    - `go generate ./...` が不要なら hook から外す
  - 完了条件: ローカル dry-run で配布アーカイブを確認できる

- [ ] `--idle-hook` の外部連携契約を決める
  - 種別: 未着手
  - 実装方針:
    - まずは実装しない。どの親プロセス / エージェントとどう繋ぐかが未確定のため、CLI 契約から先に決める
    - 実装に入るなら「終了時コールバック」ではなく「開始条件を満たしたら短時間セッションを起動する」方向で設計する
  - 完了条件: 呼び出し側との I/O 契約が決まり、誤起動しない仕様になる

- [ ] MVP 後半の後回し機能を backlog 化する
  - 種別: 未着手
  - 対象: 発音, 同期, AI 解説, レベル別辞書パック, スペリング入力モード
  - 実装方針:
    - `tasks/todo.md` 本体では詳細設計まで掘らず、MVP 完了後に別 spec へ切り出す
    - まずは学習ループの質と運用コマンドを優先する
  - 完了条件: 直近 backlog と将来構想が混ざらない状態になる

## 推奨着手順

- 完了済み: `eitango review`, `config.toml` 読み込み, `--focus-mode`
1. `eitango doctor`
2. Phase 1 辞書パック拡張
3. `eitango reset`
4. `eitango export`
5. `words.source` migration + `eitango import`
6. `eitango browse`
7. help 画面 / 例文表示 / waiting metrics
