# Contributing

eitango へのコントリビュートを検討していただきありがとうございます。最初に迷わないための最小ガイドをまとめています。

## 0. 基本方針

- bundled core 語彙 (`assets/words_core.jsonl`) への**新規語彙追加 PR は原則受け付けていません**。ローカルリポジトリで順番に整備しているためです
- バイナリに独自語彙を含めたい場合は fork して運用してください
- 既存登録済み語彙の誤字・意味・品詞・メタデータ修正 PR は歓迎します
- 個人の休日プロダクトのため、issue / PR への反応が遅れることがあります

## 1. セットアップ

### 必要なもの

- Go **1.26.1 以上**（`go.mod` 準拠）
- Python **3.11 以上**（`scripts/vocab/` を触る場合）
- `uv`（Python 依存解決用）
- `goimports`
- `golangci-lint` **v2.11.4**（CI と同じ）
- `shellcheck`（`install.sh` を触る場合）
- `lefthook`（任意ですが推奨）

アプリ本体の通常開発は Go だけで進められます。Python と `uv` は辞書生成パイプライン向けです。

### clone と初回セットアップ

```bash
git clone https://github.com/harumiWeb/eitango.git
cd eitango

uv sync
go test ./...
go build ./...
go run ./cmd/eitango --help
```

### hook の有効化

`lefthook` を入れている場合は、clone 後に一度だけ次を実行してください。

```bash
lefthook install
```

pre-commit では次が自動実行されます。

- `goimports -w cmd internal assets`
- `golangci-lint run`
- `go test ./...`

## 2. 開発フロー

### 先に issue を立ててほしいケース

- 新機能追加
- 既存挙動を変える変更
- 辞書フォーマット・生成フロー・配布方法に影響する変更
- 実装方針に複数案がありそうな変更

### そのまま PR してよいケース

- typo 修正
- README / CONTRIBUTING などの軽微なドキュメント改善
- 小さなバグ修正やテスト修正で、変更意図が明確なもの

### ブランチ・コミット・PR

- 厳密なブランチ命名ルールはありませんが、`fix/...` `docs/...` `chore/...` のように内容が分かる名前を推奨します
- コミットは「レビューしやすい単位」で分け、無関係な変更を同じ PR に混ぜないでください
- PR 説明には最低限、**何を直したか / なぜ必要か / どう確認したか** を書いてください
- TUI の見た目や操作が変わる場合は、スクリーンショットや短い説明があると助かります

## 3. 品質チェック

PR 前には少なくとも次を確認してください。

```bash
goimports -w cmd internal assets
golangci-lint run
go test ./...
go build ./...
```

`install.sh` を変更した場合は追加で次も実行してください。

```bash
shellcheck install.sh
```

hook を入れていない場合でも、上記コマンドは手動で実行してください。CI でも同等のチェックが走ります。

## 4. 変更対象ごとの補足

### TUI を変えるとき

- キーボード操作が崩れていないか
- 狭い terminal でも致命的にレイアウトが崩れないか
- 色やテーマ依存の変更なら `no_color` でも読めるか

### 辞書 / データ周りを変えるとき

- bundled core への新規語彙追加 PR は原則不可です
- 既存データ修正では、変更理由と影響範囲を PR に書いてください
- `assets/words_core.jsonl` や `third_party/licenses/` を触る場合は、出所とライセンス表記を壊さないでください
- `scripts/vocab/` はローカル入力（`tmp/` 配下）を前提にしており、通常利用者には不要です

### 配布系ファイルを変えるとき

- `install.sh`、`.goreleaser.yaml`、license / notice 類を触ったら、配布物に必要なファイルが残るか確認してください
- 変更後のインストール手順や配布手順が README とずれていないかも確認してください
