# Changelog

このファイルには、このプロジェクトの重要な変更を記録します。
今後は user-visible な変更を `Unreleased` に追記し、release 時に版セクションへ移します。

## [Unreleased]

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

[Unreleased]: https://github.com/harumiWeb/eitango/compare/v0.2.2...HEAD
[0.2.2]: https://github.com/harumiWeb/eitango/compare/v0.2.0...v0.2.2
[0.2.0]: https://github.com/harumiWeb/eitango/compare/v0.1.1...v0.2.0
