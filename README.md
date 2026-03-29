<p align="center">
    <img width="600" alt="logo" src="https://github.com/user-attachments/assets/ab4d0cb7-d415-427f-9026-5b2faac52b11" />
</p>

<p align="center">
  <em>eitango - TUI English Vocabulary Tool</em>
</p>

# eitango

オフラインで動く英単語トレーニング TUI です。Bubble Tea ベースの対話UIとローカル SQLite を使い、待ち時間に短く回せる学習セッションを想定しています。

[English README](README.en.md)

## 現在できること

- `eitango learn` で通常学習セッションを開始
- `eitango review` で due-only の復習セッションを開始
- `eitango stats` で学習統計を表示
- `eitango doctor` で DB と辞書の read-only 診断を実行
- `eitango validate` で組み込み辞書や外部 CSV / JSONL を検証
- `eitango import` / `eitango export` / `eitango reset` で辞書と進捗を保守

## インストール

### 1. GitHub Releases から使う

公開アーカイブにはバイナリに加えて `LICENSE`、`THIRD_PARTY_NOTICES.md`、`third_party/licenses/` が同梱されます。自分のOS向けの成果物を展開して `eitango` を実行してください。

### 2. Go からインストールする

Go 1.26 以降を前提にしています。

```bash
go install github.com/harumiWeb/eitango/cmd/eitango@latest
```

## クイックスタート

```bash
eitango learn
eitango review --focus-mode
eitango stats
eitango doctor
```

学習データは初回起動時にローカル DB へ初期化されます。デフォルトでは組み込みの `assets/words_core.jsonl` を seed として使用します。

## データ保存先

- Windows: `%AppData%\\eitango-cli\\`
- macOS: `~/Library/Application Support/eitango-cli/`
- Linux: `~/.local/share/eitango-cli/`

次のファイルが作成されます。

- `user.db`
- `config.toml`
- `logs/`

保存先は `EITANGO_DATA_DIR` で上書きできます。

## コマンド一覧

| コマンド | 役割 |
| --- | --- |
| `eitango learn [--focus-mode] [--questions N]` | 通常学習セッションを開始 |
| `eitango review [--focus-mode] [--questions N] [--restart]` | due-only 復習を開始 |
| `eitango stats` | 統計を表示 |
| `eitango doctor` | DB / 辞書の診断 |
| `eitango validate --embedded-core` | 組み込み core 辞書を検証 |
| `eitango validate --file words.csv --format csv --kind import` | import 用辞書を検証 |
| `eitango import --file words.jsonl --format jsonl --source my-pack` | 外部辞書を取り込み |
| `eitango export wrong-words --output wrong.csv` | 苦手語を CSV 出力 |
| `eitango export progress --output progress.json` | 進捗を JSON 出力 |
| `eitango reset --progress` / `eitango reset --reseed` | 学習履歴の初期化 / 組み込み core 再投入 |

## 辞書データとライセンス

`eitango` のコードは [Apache License 2.0](LICENSE) です。ただし、配布物に含まれる `assets/words_core.jsonl` はコードとは別に出所を持つ語彙データであり、Apache-2.0 だけで完結するものとして扱っていません。

このリポジトリでは、bundled core の語彙由来を Leipzig Corpora Collection English News 2024 1M word list と Japanese WordNet (`wnjpn.db`) に限定しています。

- `assets/words_core.jsonl` はプロジェクトが編集・整備した core 語彙データです
- `meaning_ja` は Japanese WordNet を参照して整備した日本語意味データです
- `frequency_rank` は Leipzig Corpora Collection English News 2024 1M word list 由来の bundled-core ranking です
- `level` は `core-1` から `core-4` の内部バケットであり、上流データセットのラベルではありません
- 語彙生成スクリプトはローカル入力の `tmp/eng_news_2024_1M-words.txt` と `tmp/wnjpn.db` を参照します
- raw の Leipzig / WordNet 入力は配布物に含めず、生成条件は `scripts/vocab/source_manifest.json` に固定します
- Japanese WordNet を直接・間接に使った成果公開や再配布では、`third_party/licenses/Japanese-WordNet.txt` にまとめた上流推奨のクレジット文言・リンク・ライセンス案内を保持してください

公開成果物での Japanese WordNet 帰属表示は、少なくとも次のような文言を含める想定です（ローカル入力を別版に差し替えた場合は版番号も合わせて更新してください）。

```text
Japanese Wordnet (v1.1) © 2009-2011 NICT, 2012-2015 Francis Bond and 2016-2024 Francis Bond, Takayuki Kuribayashi
https://bond-lab.github.io/wnja/index.en.html
```

```text
日本語ワードネット（1.1版）© 2009-2011 NICT, 2012-2015 Francis Bond and 2016-2024 Francis Bond, Takayuki Kuribayashi
https://bond-lab.github.io/wnja/index.ja.html
```

再配布や派生利用の前に、必ず次を確認してください。

- [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
- [`third_party/licenses/`](third_party/licenses)

特に `words_core.jsonl` を含む再配布物では、第三者データ由来の注意書きに加えて、Japanese WordNet の帰属表示案内も保持する前提で扱ってください。

## 開発メモ

アプリ本体は Go だけで動作しますが、辞書生成パイプラインには Python 3.11 以降を使います。

```bash
uv sync
go test ./...
go run ./cmd/eitango --help
```

辞書生成スクリプトは `scripts/vocab/` にあり、`tmp/eng_news_2024_1M-words.txt` と `tmp/wnjpn.db` のようなローカル入力を前提とします。これらはエンドユーザーの通常利用には不要です。

## ライセンス

- アプリコード: [Apache License 2.0](LICENSE)
- 第三者ソフトウェアとデータ: [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)
- 参照用ライセンス原文とデータ出所メモ: [`third_party/licenses/`](third_party/licenses)
