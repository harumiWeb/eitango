# Changelog

このファイルには、このプロジェクトの重要な変更を記録します。
今後は user-visible な変更を `Unreleased` に追記し、release 時に版セクションへ移します。

## [Unreleased]

## [0.5.1] - 2026-04-04

### Added

- Windows 向けの `winget install HarumiWeb.Eitango` 導線を追加しました。

### Changed

- release フローから `harumiWeb/winget-pkgs` fork へ manifest を生成し、`microsoft/winget-pkgs` へ PR を作成できるようにしました。
- GoReleaser の archive 構成を整理し、winget では GitHub Releases に公開した Windows zip のみを参照するようにしました。
- README / README.en / 配布ポリシー ADR を Windows の winget 配布方針に合わせて更新しました。

## [0.5.0] - 2026-04-03

### Added

- macOS / Windows 向けの音声再生を追加し、quiz / feedback 画面で `Ctrl+P` から現在の単語を手動再生できるようにしました。
- ホーム設定画面と `config.toml` に `audio_enabled` / `audio_autoplay` を追加し、音声機能の既定値を保存できるようにしました。

### Changed

- quiz / feedback 画面に自動再生状態の表示を追加し、`Shift+Tab` でセッション単位の autoplay を切り替えられるようにしました。
- `write` モードでは答えを直接漏らさないため、音声再生と autoplay をフィードバック画面だけで有効にしました。

### Fixed

- 音声が無効、または未対応の環境では autoplay を ON に保持しないようにし、設定値、セッション state、UI 表示を実際の動作に合わせて正規化しました。
- macOS で `en_US` 音声が無い場合でも、`en_GB` など他の英語音声へフォールバックして英語読み上げを継続するようにしました。

## [0.4.2] - 2026-04-03

### Added

- 進行中のセッションがある状態で `Enter` / `n` / `r` から別セッションを始めるとき、破棄確認ダイアログを表示するようにしました。
- 破棄確認ダイアログに、現在のセッション状況と開始予定のモードを表示するようにしました。

### Fixed

- 既存セッションを破棄して新規開始しようとした直後に開始失敗した場合でも、ホーム画面に古い active session 表示が残らないようにしました。
- 破棄後の復旧で stats の再読込に失敗しても、ホーム画面の active session 状態だけは正しく再同期するようにしました。

## [0.4.1] - 2026-04-03

### Fixed

- `write` モードで最後のヒントまで使って答えがすべて開示された場合、そのまま自動でフィードバック画面へ進み、不正解 (`Again`) として保存するようにしました。
- 最後のヒントで正解文字列が入力欄に揃っていた場合でも、正答扱いで進捗が保存されないようにしました。

## [0.4.0] - 2026-04-02

### Added

- `write_mode_difficulty` 設定を追加し、Write モードの難易度を `basic / hard` で切り替えられるようにしました。
- ホーム設定画面に Write 難易度の項目を追加しました。

### Changed

- `write_mode_difficulty=basic` では、Write の新規問題候補を Choice モードで一度見た語から優先的に選ぶようにしました。
- `write_mode_difficulty=hard` は従来どおり、Choice 未出題の語も Write に出せる高難度設定として維持しました。
- README / README.en に Write 難易度設定と `basic` / `hard` の違いを追記しました。

### Fixed

- `basic` で Write 未出題語を優先しつつ、候補不足時だけ Choice 既出語へフォールバックするようにし、Write の初期難易度が不必要に上がる問題を抑えました。
- `basic` の候補選定で、due の復習語が新規枠へ再混入しないようにしました。

## [0.3.0] - 2026-04-01

### Added

- `write` モードを追加し、日本語の意味を見て英単語を入力する学習フローを使えるようにしました。
- CLI に `eitango play choice|write` と `eitango review choice|write` を追加しました。既存の `learn` は `play` の互換 alias として維持しています。
- ホーム画面に回答方式トグルを追加し、`Tab` で `choice / write` を切り替えられるようにしました。

### Changed

- セッション管理を `play/review × choice/write` の 2 軸に整理し、active session 再開時も回答方式を維持するようにしました。
- `write` モードの操作を text entry 前提に整理し、`Tab=ヒント`、`Ctrl+S=スキップ`、`Enter=決定/次へ`、`Esc=終了` に統一しました。
- `write` モードの入力欄は `Word` スロット表示に合わせて字間スペース付きで描画するようにし、文字数の見比べをしやすくしました。
- README / README.en を新しい `play/review` コマンド体系と `write` モード操作に合わせて更新しました。

### Fixed

- `write` モードで `h` / `s` / `q` がショートカット扱いされて文字入力できない問題を修正しました。
- `write` フィードバック画面の help / status 表示を Enter で保存して次へ進む実際の操作に合わせました。
- `eitango doctor` が pre-`005_answer_modes.sql` の DB を read-only で診断したとき、`sessions.answer_mode` 不在で失敗しないようにしました。旧スキーマは `choice` として診断し、migration drift は `migrations` check で報告します。

## [0.2.2] - 2026-03-31

### Added

- macOS / Linux 向けの `curl | sh` installer を追加しました。
- installer に `--version`, `--uninstall`, `--purge-data` を追加しました。

### Changed

- installer は GitHub Releases の `checksums.txt` を使って archive の SHA256 を必須検証するようにしました。
- README / README.en に curl installer、version pin、uninstall の導線を追記しました。
- ホーム画面の update 通知は起動ごとに latest release を再確認し、保存済み cache は失敗時の fallback に限定するようにしました。

## [0.2.0] - 2026-03-29

### Added

- GitHub Releases の latest を確認する自動 update check を追加しました。
- 新しい版があるときにホーム画面へ update 通知を表示するようにしました。
- `eitango version` コマンドを追加し、現在の build 情報と最新 release 情報を確認できるようにしました。

### Changed

- README / README.en に update check の挙動、手動更新方法、`EITANGO_DISABLE_UPDATE_CHECK=1` による無効化方法を追記しました。

### Fixed

- システム時計が戻った場合でも、古い cache を update check に使い続けないようにしました。
- 通知不要時に古い update tag が画面に残る問題を修正しました。
- `dev` など非 semver の build でも update availability を正しく判定するようにしました。

[Unreleased]: https://github.com/harumiWeb/eitango/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/harumiWeb/eitango/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/harumiWeb/eitango/compare/v0.4.2...v0.5.0
[0.4.2]: https://github.com/harumiWeb/eitango/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/harumiWeb/eitango/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/harumiWeb/eitango/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/harumiWeb/eitango/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/harumiWeb/eitango/compare/v0.2.0...v0.2.2
[0.2.0]: https://github.com/harumiWeb/eitango/compare/v0.1.1...v0.2.0
