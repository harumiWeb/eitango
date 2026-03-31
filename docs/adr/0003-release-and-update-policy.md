# ADR-0003: 配布物と Update Check の運用方針

## Status

`accepted`

## Background

初期リリース後の保守では、ユーザーへ何を配るか、更新導線をどこまで自動化するか、ライセンスと notice をどう同梱するかを固定しておく必要がある。現行実装は GitHub Releases と GoReleaser を前提にし、update check は best-effort の補助機能として分離されている。この方針を ADR にしておかないと、将来の配布や更新 UX が機能追加のたびにぶれやすい。

## Decision

- 正規の配布チャネルは GitHub Releases とし、GoReleaser で darwin / linux / windows 向け archive を生成する。
- release artifact には実行バイナリに加えて `LICENSE`, `README.md`, `README.en.md`, `THIRD_PARTY_NOTICES.md`, `third_party/licenses/**` を同梱する。
- macOS / linux では `install.sh` を bootstrap 導線として許可するが、installer 自体は GitHub Releases の archive と `checksums.txt` を取得する薄い wrapper に留める。
- `install.sh` は `checksums.txt` で SHA256 を必須検証し、法務ファイルを `~/.eitango/share/` へ保持する。
- 更新は自動適用しない。新しい版の取得は release archive の再取得か `go install ...@latest` の再実行で行う。
- update check は GitHub Releases の latest を参照する補助機能とし、学習フローを止めない best-effort 動作に限定する。
- update check の state は data dir の `update-check.json` に保存し、ホーム画面の通知は起動ごとに latest release を非同期で再検証する。HTTP timeout は 1.5 秒とし、保存 state は request failure 時の fallback に使う。
- cache を尊重する helper（例: `Check`）は TTL 24 時間を維持するが、TUI ホーム画面や `eitango version` などの対話 UI からは常に非 cache な `CheckNow` を呼び出し、鮮度判定には TTL を使わない。TTL 付き helper は将来のバッチ系 / 非対話コマンド向けに残すが、現状では呼び出し元を持たない前提とする。
- 初回の successful check では通知を出さず、2 回目以降に新しい版が確認されたときだけ通知対象にする。
- `EITANGO_DISABLE_UPDATE_CHECK=1` で update check を完全に無効化できるように保つ。
- オフライン、タイムアウト、API failure 時は黙って失敗し、通常の学習や CLI 実行は継続させる。

## Consequences

- release artifact の再配布条件と notice の同梱要件を、ビルド設定と文書で一貫して維持できる。
- `curl | sh` の最短導線を追加しても、配布物の単一ソースは GitHub Releases のまま保てる。
- SHA256 検証で download 途中の破損や取り違えは防ぎやすくなるが、署名付き provenance ではないため `install.sh` 本体の信頼境界は依然として HTTPS / GitHub 側にある。
- 自動更新を持たないため、更新失敗が学習データを壊す経路を増やさずに済む。
- update check は起動ごとに 1 回の best-effort request を行うが、最新 release 反映の遅延は大きく減り、更新作業は引き続き手動のまま保てる。
- 保存済み state があるため、GitHub API が失敗しても直前の latest 情報を fallback として使える。
- GitHub Releases が update metadata の単一ソースになるため、別チャネルを増やす場合は新しい判断が必要になる。

## Rationale

- Tests:
  - `internal/updatecheck/checker_test.go`
  - `cmd/eitango/main_test.go`
  - `install_test.go`
- Code:
  - `internal/updatecheck/checker.go`
  - `cmd/eitango/main.go`
  - `.goreleaser.yaml`
  - `install.sh`
- Related specs:
  - なし。コード、README、CHANGELOG、tests を正本とする。

## Supersedes

- None

## Superseded by

- None
