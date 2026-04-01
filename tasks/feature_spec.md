# 2026-03-29 ドキュメント再編仕様

## Goal

- 旧設計書を廃止し、初期リリース後も参照価値がある判断だけを ADR に残す。
- `docs/specs/` は増やさず、コード・README・tests を正本として維持する。

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
- `docs/specs/` の新設や index/README の追加
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
