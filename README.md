<p align="center">
    <img width="600" alt="logo" src="assets/images/logo.png" />
</p>

<p align="center">
  <em>eitango - TUI English Vocabulary Tool</em>
</p>

<div align="center" style="max-width: 600px; margin: auto;">

  ![GitHub Release](https://img.shields.io/github/v/release/harumiWeb/eitango) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/harumiWeb/eitango) [![Apache-2.0](https://custom-icon-badges.herokuapp.com/badge/license-Apache%202.0-8BB80A.svg?logo=law&logoColor=white)]()  [![CI](https://github.com/harumiWeb/eitango/actions/workflows/ci.yml/badge.svg)](https://github.com/harumiWeb/eitango/actions/workflows/ci.yml) [![Codacy Badge](https://app.codacy.com/project/badge/Grade/8c55fea55abd41a090db9253b79990d5)](https://app.codacy.com/gh/harumiWeb/eitango/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_grade)

</div>

---

# eitango

オフラインで動く英単語トレーニング TUI です。Bubble Tea ベースの対話UIとローカル SQLite を使い、待ち時間に短く回せる学習セッションを想定しています。

[English README](README.en.md)

<img alt="home" src="assets/images/home.png" />

## 現在できること

- `eitango` でホーム画面を表示し、TUIでモード選択や設定変更が可能

<p align="center">
  <img alt="プレイしている様子" src="assets/images/playing.gif" />
</p>

- `eitango learn` で通常学習セッションを開始
- `eitango review` で due-only の復習セッションを開始
- `eitango stats` で学習統計を表示
- `eitango version` で現在のビルド情報と最新 release を確認
- `eitango doctor` で DB と辞書の read-only 診断を実行
- `eitango validate` で組み込み辞書や外部 CSV / JSONL を検証
- `eitango import` / `eitango export` / `eitango reset` で辞書と進捗を保守

## インストール

### 1. macOS / Linux は `curl | sh` を使う

`install.sh` は `--version` を省略した場合に GitHub Releases API (`/releases/latest`) へアクセスして最新 version を解決し、そのうえで対応する archive と `checksums.txt` を取得します。SHA256 検証が通ったときだけ `~/.eitango/` へ展開し、shell rc は自動変更しません。

```bash
curl -fsSL https://raw.githubusercontent.com/harumiWeb/eitango/main/install.sh | sh
```

特定 version を入れるときは次を使ってください。

```bash
curl -fsSL https://raw.githubusercontent.com/harumiWeb/eitango/main/install.sh | sh -s -- --version v0.2.0
```

インストール後は次が配置されます。

- `~/.eitango/bin/eitango`
- `~/.eitango/version`
- `~/.eitango/share/`

法務ファイルと notice は `~/.eitango/share/` に保持されます。PATH に `~/.eitango/bin` が無い場合は次を shell 設定へ追加してください。

```bash
export PATH="$HOME/.eitango/bin:$PATH"
```

script を pipe せず確認してから実行したい場合は次でも同じです。

```bash
curl -fsSLo install.sh https://raw.githubusercontent.com/harumiWeb/eitango/main/install.sh
sh install.sh --version v0.2.0
```

アンインストールは次です。既定では学習データを残します。

```bash
curl -fsSL https://raw.githubusercontent.com/harumiWeb/eitango/main/install.sh | sh -s -- --uninstall
```

学習データも消す場合は `--purge-data` を付けます。`EITANGO_DATA_DIR` を使っている場合は、同じ env を付けて実行してください。

```bash
curl -fsSL https://raw.githubusercontent.com/harumiWeb/eitango/main/install.sh | sh -s -- --uninstall --purge-data
```

必要ツールは `sh`, `curl`, `tar`, `mktemp` と、`sha256sum` / `shasum` / `openssl` のいずれか 1 つです。Windows は今回の installer 対象外なので、release zip を使ってください。

### 2. GitHub Releases から使う

公開アーカイブにはバイナリに加えて `LICENSE`、`THIRD_PARTY_NOTICES.md`、`third_party/licenses/` が同梱されます。自分のOS向けの成果物を展開して `eitango` を実行してください。

### 3. Go からインストールする

Go 1.26 以降を前提にしています。

```bash
go install github.com/harumiWeb/eitango/cmd/eitango@latest
```

`go install` で導入した build でも、`eitango version` は埋め込まれた module version を表示します。

## クイックスタート

```bash
eitango
```

モード指定で起動することもできます。

```bash
eitango learn
eitango review --focus-mode
eitango stats
eitango version
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
- `update-check.json`

保存先は `EITANGO_DATA_DIR` で上書きできます。

## 更新通知

`eitango` / `eitango learn` / `eitango review` / `eitango version` は、GitHub Releases の latest release を確認できます。

- ホーム画面の通知は起動ごとに非同期で latest release を再確認します
- 初回の成功確認では通知せず、次回以降の起動で差分があればホーム画面に軽く表示します
- `update-check.json` には直前の successful check 結果を保存し、タイムアウトやオフライン時の fallback に使います
- `eitango version` は現在の build info に加えて latest release URL も表示します
- タイムアウトやオフライン時は黙ってスキップし、学習体験を止めません
- `EITANGO_DISABLE_UPDATE_CHECK=1` で完全に無効化できます

更新自体は自動化していません。必要に応じて GitHub Releases の最新成果物を取り直すか、Go インストールなら次を再実行してください。

```bash
go install github.com/harumiWeb/eitango/cmd/eitango@latest
```

`curl | sh` で入れた場合も self-update はしません。最新版へ上げるときは installer を再実行してください。

```bash
curl -fsSL https://raw.githubusercontent.com/harumiWeb/eitango/main/install.sh | sh
```

## コマンド一覧

| コマンド | 役割 |
| --- | --- |
| `eitango version` | 現在の build info と latest release を表示 |
| `eitango learn [--focus-mode] [--questions N]` | 通常学習セッションを開始 |
| `eitango review [--focus-mode] [--questions N] [--restart]` | due-only 復習を開始 |
| `eitango stats` | 統計を表示 |
| `eitango --license` | 同梱ライセンスと notice を表示 |
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
