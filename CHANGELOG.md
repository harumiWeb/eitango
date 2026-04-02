# Changelog

このファイルには、このプロジェクトの重要な変更を記録します。
今後は user-visible な変更を `Unreleased` に追記し、release 時に版セクションへ移します。

## [Unreleased]

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

[Unreleased]: https://github.com/harumiWeb/eitango/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/harumiWeb/eitango/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/harumiWeb/eitango/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/harumiWeb/eitango/compare/v0.2.0...v0.2.2
[0.2.0]: https://github.com/harumiWeb/eitango/compare/v0.1.1...v0.2.0
