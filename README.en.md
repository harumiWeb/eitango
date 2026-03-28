# eitango

`eitango` is an offline English vocabulary trainer with a terminal UI. It uses Bubble Tea for the interactive interface and a local SQLite database for progress tracking.

[日本語README](README.md)

## What It Does

- `eitango learn` starts a standard learning session
- `eitango review` starts a due-only review session
- `eitango stats` prints review statistics
- `eitango doctor` runs read-only diagnostics
- `eitango validate` checks embedded or external dictionary files
- `eitango import`, `export`, and `reset` maintain word packs and learning data

## Installation

### Release archives

Published archives are expected to contain the executable plus `LICENSE`, `THIRD_PARTY_NOTICES.md`, and `third_party/licenses/`.

### Install with Go

Go 1.26 or newer is required.

```bash
go install github.com/harumiWeb/eitango/cmd/eitango@latest
```

## Quick Start

```bash
eitango learn
eitango review --focus-mode
eitango stats
eitango doctor
```

On first run, `eitango` seeds a local database from the embedded `assets/words_core.jsonl`.

## Data Directory

- Windows: `%AppData%\\eitango-cli\\`
- macOS: `~/Library/Application Support/eitango-cli/`
- Linux: `~/.local/share/eitango-cli/`

Set `EITANGO_DATA_DIR` to override the default location.

## Dictionary Data and Licensing

The application code is licensed under [Apache License 2.0](LICENSE). The bundled `assets/words_core.jsonl` should not be treated as if it were covered only by Apache-2.0.

The bundled core vocabulary in this repository is now sourced only from the Leipzig English News 2024 1M word list plus Japanese WordNet (`wnjpn.db`).

- `words_core.jsonl` is a project-curated core vocabulary file
- `frequency_rank` is a Leipzig-derived bundled-core ranking and `level` uses internal `core-1` through `core-4` buckets
- the generation pipeline reads local inputs from `tmp/eng_news_2024_1M-words.txt` and `tmp/wnjpn.db`
- raw upstream databases and corpora are not redistributed in the release artifacts; the reproducible source manifest lives at `scripts/vocab/source_manifest.json`

Before redistributing the repository or packaged artifacts, review:

- [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
- [`third_party/licenses/`](third_party/licenses)

## Development

The main app is Go-only at runtime. Python is only needed for vocabulary generation workflows.

```bash
uv sync
go test ./...
go run ./cmd/eitango --help
```

The scripts in `scripts/vocab/` expect local inputs such as `tmp/eng_news_2024_1M-words.txt` and `tmp/wnjpn.db`.
