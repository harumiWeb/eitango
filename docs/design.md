では、実装に入れる粒度まで落とします。
今回は **MVP を2週間くらいで形にできる設計** を意識して、次の3つを出します。

1. 具体的なアーキテクチャ
2. DB schema と出題アルゴリズム
3. Go のディレクトリ構成と実装順

前提として、TUI は Bubble Tea、CLI コマンドは Cobra、辞書同梱は `embed`、ローカル永続化は pure Go の `modernc.org/sqlite`、配布は GoReleaser という組み合わせで進めるのが妥当です。Bubble Tea は Elm Architecture ベースの Go TUI フレームワークで、複雑な対話型ターミナルアプリにも向いています。Cobra は `git` や `go` のようなサブコマンド型 CLI を構築するための定番ライブラリです。`embed` はファイルをコンパイル時にバイナリへ埋め込めます。`modernc.org/sqlite` は pure Go の SQLite ドライバです。GoReleaser は複数 OS/アーキテクチャ向けのビルド・アーカイブ・リリース自動化を提供します。 ([GitHub][1])

---

# 1. まず決めるべきプロダクトの芯

このアプリは「英単語学習 CLI」ですが、真の価値はそこだけではありません。
芯は次です。

**AI コーディング待機時間に、思考を切り替えすぎず 1〜3 分で学習できること。**

なので設計判断はすべてこの軸で行います。

* 起動が速い
* オフラインで動く
* 途中離脱しやすい
* キーボードだけで回せる
* 1問ごとに保存される
* 次に開いた時に即再開できる

この要件だと、Web アプリやクライアントサーバ型より、**ローカル単体 TUI** が圧倒的に合っています。

---

# 2. 全体構成

おすすめはこうです。

```text
eitango/
├─ cmd/
│  └─ eitango/
│     └─ main.go
├─ internal/
│  ├─ app/          # Bubble Tea の state machine
│  ├─ tui/          # keymap, style, components
│  ├─ quiz/         # 出題ロジック
│  ├─ srs/          # 復習間隔計算
│  ├─ store/        # SQLite access
│  ├─ dict/         # 埋め込み辞書ロード
│  ├─ session/      # 学習セッション管理
│  ├─ stats/        # 統計集計
│  └─ config/       # パス/設定
├─ assets/
│  ├─ words_core.jsonl
│  └─ migrations/
│     ├─ 001_init.sql
│     └─ 002_indexes.sql
├─ .goreleaser.yaml
└─ go.mod
```

ポイントは、**Bubble Tea の Model に全部入れないこと**です。
UI は UI、出題は出題、DB は DB に分けます。

Bubble Tea は状態遷移を扱いやすいですが、TUI フレームワークにドメインロジックを埋め込むとテストしづらくなります。なので `quiz`, `srs`, `store` は pure Go で独立させます。Bubble Tea 自体も model / update / view の流れでアプリを組む想定なので、この分離と整合します。 ([GitHub][1])

---

# 3. 画面設計

最初は 4 画面で十分です。

## ホーム

表示するもの:

* 今日の復習数
* 新規学習候補数
* 連続学習日数
* 前回中断セッションの有無

キー:

* `Enter` 学習開始
* `r` 復習のみ
* `s` 統計
* `q` 終了

## 問題画面

表示するもの:

* 英単語
* 品詞
* 4択
* 進捗 `3/10`
* 残り復習数

キー:

* `1,2,3,4`
* `j/k` でカーソル移動
* `Enter` 決定
* `q` 中断保存
* `?` ヘルプ

## フィードバック画面

表示するもの:

* 正誤
* 正しい意味
* 必要なら補助メモ
* 自己評価ボタン

  * `a`: again
  * `h`: hard
  * `g`: good
  * `e`: easy

## セッション結果

表示するもの:

* 正答率
* 学習数
* 新規 / 復習の内訳
* 苦手トップ 5

---

# 4. コマンド設計

Cobra を使って、TUI 起動とメンテ系を分けます。Cobra はサブコマンド、ヘルプ、自動補完などを備えた CLI ライブラリです。 ([GitHub][2])

```bash
eitango learn
eitango review
eitango stats
eitango browse
eitango import --file words.csv
eitango reset
eitango doctor
```

最初はこれで十分です。

### `eitango learn`

通常学習。新規 + 復習を混ぜる。

### `eitango review`

期限が来たものだけ。

### `eitango stats`

直近 7 日、30 日、総計。

### `eitango import`

将来の拡張用。自前単語帳を入れる。

### `eitango doctor`

DB や辞書整合性の確認。

---

# 5. データ設計

## 5.1 マスタ単語テーブル

```sql
CREATE TABLE words (
  id                INTEGER PRIMARY KEY,
  lemma             TEXT NOT NULL,
  pos               TEXT,
  meaning_ja        TEXT NOT NULL,
  level             TEXT,
  frequency_rank    INTEGER,
  distractor_group  TEXT,
  example_en        TEXT,
  example_ja        TEXT,
  created_at        TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### なぜ `distractor_group` を入れるのか

4択問題の質を上げるためです。

たとえば `abandon = 捨てる` に対し、誤答が

* 机
* 明るい
* ゆっくり

だと、問題として弱いです。
だから誤答候補を同系統に寄せる必要があります。

`distractor_group` 例:

* `basic-verb-action`
* `emotion-adjective`
* `business-noun`
* `education-verb`

---

## 5.2 学習進捗テーブル

```sql
CREATE TABLE progress (
  word_id             INTEGER PRIMARY KEY,
  state               TEXT NOT NULL,      -- new / learning / review / mastered
  due_at              TEXT,
  interval_days       REAL NOT NULL DEFAULT 0,
  ease_factor         REAL NOT NULL DEFAULT 2.5,
  last_seen_at        TEXT,
  streak_correct      INTEGER NOT NULL DEFAULT 0,
  total_correct       INTEGER NOT NULL DEFAULT 0,
  total_wrong         INTEGER NOT NULL DEFAULT 0,
  lapses              INTEGER NOT NULL DEFAULT 0,
  FOREIGN KEY(word_id) REFERENCES words(id)
);
```

---

## 5.3 回答履歴テーブル

```sql
CREATE TABLE reviews (
  id                  INTEGER PRIMARY KEY,
  word_id             INTEGER NOT NULL,
  session_id          TEXT NOT NULL,
  answered_at         TEXT NOT NULL,
  selected_choice     INTEGER NOT NULL,
  correct_choice      INTEGER NOT NULL,
  is_correct          INTEGER NOT NULL,
  response_ms         INTEGER,
  rating              TEXT,               -- again/hard/good/easy
  FOREIGN KEY(word_id) REFERENCES words(id)
);
```

---

## 5.4 セッションテーブル

```sql
CREATE TABLE sessions (
  id                  TEXT PRIMARY KEY,
  started_at          TEXT NOT NULL,
  finished_at         TEXT,
  mode                TEXT NOT NULL,      -- learn/review/cram
  total_questions     INTEGER NOT NULL,
  answered_questions  INTEGER NOT NULL DEFAULT 0,
  status              TEXT NOT NULL       -- active/completed/abandoned
);
```

これを持つと、「途中で抜けても次回再開」がきれいにできます。

---

# 6. SQLite をどう使うか

この用途なら SQLite がかなり適しています。
ローカル単体アプリ向けで、進捗保存・検索・集計に強く、SQLite の DB 本体は通常 1 ファイルで扱えます。WAL モードでは追加で `-wal` と `-shm` ファイルが使われ、SQLite 公式も WAL はしばしば高速で並行性が高いと説明しています。 ([SQLite][3])

初期化時はこうでよいです。

```sql
PRAGMA journal_mode=WAL;
PRAGMA synchronous=NORMAL;
PRAGMA foreign_keys=ON;
```

ただし 1 点だけ実務上の注意があります。
WAL は DB と同じディレクトリに `-wal`, `-shm` を作るので、**書き込み可能なユーザーデータ領域に保存する**前提で設計してください。SQLite 公式も WAL ファイルと shared-memory ファイルが同ディレクトリに置かれると説明しています。 ([SQLite][4])

---

# 7. 辞書データの持ち方

ここは次の構成がベストです。

## 配布時

* `assets/words_core.jsonl` を `embed` で埋め込む

## 初回起動時

* embed した辞書を SQLite に投入
* 以後は DB を使う

Go の `embed` は `//go:embed` でファイルやディレクトリをビルド時に埋め込めます。 ([Go Packages][5])

### なぜ DB 同梱ではなく JSONL→初回投入か

理由は 4 つあります。

1. 辞書編集が楽
2. import/export が作りやすい
3. 標準辞書とユーザー進捗を分離できる
4. 将来、辞書差分更新がしやすい

### JSONL の 1 行例

```json
{"lemma":"abandon","pos":"verb","meaning_ja":"捨てる、断念する","level":"toeic600","frequency_rank":3500,"distractor_group":"basic-verb-action"}
```

### 30k 拡張時の配布方針

* 最終目標は `assets/words_core.jsonl` を 30,000 語まで育てた `core` 同梱
* ただし移行リスクを抑えるため、5,000 → 10,000 → 30,000 の段階投入で進める
* `words.source` は維持し、必要に応じて追加パック import も共存できる形を残す

### `core` 辞書の入力契約

`core` 辞書は出題品質を支える前提データなので、次を必須にします。

* lemma
* meaning_ja
* pos
* level
* frequency_rank
* distractor_group

さらに、

* `(lemma, pos)` は `core` 内で一意
* `frequency_rank` は正の整数で一意
* 各 `distractor_group` は最低 4 語

とします。user import は間口を狭めすぎないために最小必須を維持しつつ、`frequency_rank` までは任意で受けられるようにしておくのがよいです。

---

# 8. 3万語を扱う時の実務的な考え方

3万語を DB に入れるのは重くありません。
問題は「どう出題の質を保つか」です。

最初から 3 万語フル実装より、**まず 1000〜3000 語で品質を固める**のがいいです。
理由は、学習アプリの満足度は収録語数より次で決まるからです。

* 4択の自然さ
* 復習タイミング
* テンポ
* キー操作の気持ちよさ

なので段階的にはこうです。

現行実装は Phase 1 の品質固めを通過した前提として、次は 5,000 → 10,000 → 30,000 の順で広げるのが安全です。

### Phase 1

* 1000〜3000語
* TOEIC600帯や中学〜高校基礎
* 品詞と distractor_group 整備

### Phase 2

* 10000語
* 難易度タグ増強
* 頻度順出題

### Phase 3

* `core` 30000語
* 段階投入の最終到達点
* `source` を維持したまま必要なら用途別パックも追加可能

  * TOEIC
  * 英検
  * ビジネス
  * 大学受験

---

# 9. 出題アルゴリズム

ここがこのプロダクトのコアです。

## 問題選定ルール

優先順位はこれを推します。

1. `due_at <= now` の復習単語
2. 今セッション中に間違えた単語の再出題
3. 新規単語
4. 余裕があれば mastered の維持復習

これで「忘れる前に出る」「今日のミスが残らない」が両立します。

---

## 新規と復習の比率

最初のデフォルトはこうでいいです。

* 復習 70%
* 新規 30%

ユーザーが変えられるようにする。

### なぜか

英単語アプリは新規を増やしすぎると「覚えた気がするだけ」になりやすいからです。
待ち時間学習というコンセプトなら、復習中心の方が継続しやすいです。

---

## 4択の誤答生成

誤答は次の順で絞ります。

```text
1. 同じ品詞
2. level が近い
3. distractor_group が同じ or 近い
4. meaning が完全一致しない
5. 直近で使った誤答を避ける
```

擬似コードだとこうです。

```go
func BuildChoices(correct Word, pool []Word, n int) []Choice {
    candidates := filter(pool, func(w Word) bool {
        if w.ID == correct.ID { return false }
        if w.Pos != correct.Pos { return false }
        if abs(w.FrequencyRank-correct.FrequencyRank) > 3000 { return false }
        if w.MeaningJA == correct.MeaningJA { return false }
        return true
    })

    scored := scoreDistractors(correct, candidates)
    distractors := pickTopDistinct(scored, n-1)

    choices := append([]Word{correct}, distractors...)
    shuffle(choices)
    return toChoices(choices)
}
```

### スコア例

* 同じ `distractor_group`: +5
* 同じ `level`: +3
* 頻度ランク近い: +2
* 同じ品詞: 必須
* 最近使った誤答: -4

---

# 10. SRS 設計

MVP では Anki 級の複雑さは不要です。
まずは簡易 SM-2 風で十分です。

## 状態

* `new`
* `learning`
* `review`
* `mastered`

## レーティング

* `again`
* `hard`
* `good`
* `easy`

## 初期ルール

* `again` → 10分後
* `hard` → 1日後
* `good` → 3日後
* `easy` → 7日後

以後は `interval_days * ease_factor` をベースに伸ばす。

### 更新例

```go
func Update(progress Progress, rating Rating, now time.Time) Progress {
    switch rating {
    case Again:
        progress.State = "learning"
        progress.IntervalDays = 0
        progress.DueAt = now.Add(10 * time.Minute)
        progress.EaseFactor = max(1.3, progress.EaseFactor-0.2)
        progress.Lapses++
        progress.StreakCorrect = 0
    case Hard:
        progress.State = "review"
        progress.IntervalDays = max(1, progress.IntervalDays*1.2)
        progress.DueAt = now.Add(days(progress.IntervalDays))
        progress.EaseFactor = max(1.3, progress.EaseFactor-0.15)
    case Good:
        progress.State = "review"
        if progress.IntervalDays < 1 {
            progress.IntervalDays = 1
        } else {
            progress.IntervalDays = progress.IntervalDays * progress.EaseFactor
        }
        progress.DueAt = now.Add(days(progress.IntervalDays))
        progress.StreakCorrect++
    case Easy:
        progress.State = "review"
        if progress.IntervalDays < 1 {
            progress.IntervalDays = 3
        } else {
            progress.IntervalDays = progress.IntervalDays * (progress.EaseFactor + 0.3)
        }
        progress.DueAt = now.Add(days(progress.IntervalDays))
        progress.EaseFactor += 0.05
        progress.StreakCorrect++
    }
    return progress
}
```

---

# 11. Bubble Tea の状態設計

Bubble Tea は Elm 方式なので、`Model`, `Update`, `View` を整理しておくとブレません。Bubble Tea はこのアーキテクチャを前面に出しています。 ([GitHub][1])

## 画面状態 enum

```go
type Screen int

const (
    ScreenHome Screen = iota
    ScreenQuiz
    ScreenFeedback
    ScreenStats
    ScreenHelp
)
```

## ルート Model

```go
type RootModel struct {
    screen       Screen
    keymap       KeyMap
    session      *session.Runtime
    currentQ     *quiz.Question
    feedback     *quiz.Feedback
    stats        stats.Snapshot
    err          error
    width        int
    height       int
}
```

## 重要メッセージ

```go
type questionLoadedMsg struct {
    q quiz.Question
}

type answerSubmittedMsg struct {
    result quiz.Result
}

type sessionFinishedMsg struct {
    summary session.Summary
}

type errMsg struct {
    err error
}
```

### 実装上のコツ

DB I/O や重い計算は Bubble Tea の `Cmd` で非同期に回し、`Update` は状態更新だけに寄せると保守しやすいです。Bubble Tea は commands/messages で非同期処理を扱う前提の設計です。 ([GitHub][1])

---

# 12. 実装する keymap

```go
type KeyMap struct {
    Up        key.Binding
    Down      key.Binding
    Select1   key.Binding
    Select2   key.Binding
    Select3   key.Binding
    Select4   key.Binding
    Confirm   key.Binding
    Quit      key.Binding
    Help      key.Binding
    Again     key.Binding
    Hard      key.Binding
    Good      key.Binding
    Easy      key.Binding
}
```

Bubbles には key handling や list のような TUI コンポーネント群があります。必要な場所だけ使う形でよいです。 ([GitHub][6])

---

# 13. 学習セッション設計

あなたのコンセプトだと、セッション設計はかなり大事です。

## デフォルト

* 10問
* 途中終了可
* 問題ごとに保存
* 再起動時に再開

## セッション生成

```go
type Plan struct {
    NewCount    int
    ReviewCount int
    RetryCount  int
}
```

たとえば 10 問なら:

* review 6
* retry 1
* new 3

など。

## 途中離脱

`q` を押したら:

* 現在のセッションを `active` のまま保存
* 進捗はすでに保存済み
* 次回 `learn` 時に「前回の続きから？」と出す

---

# 14. 保存先

各 OS でユーザーデータ保存先を分けるのが自然です。
Go 標準ライブラリの `os.UserConfigDir` / `os.UserHomeDir` を使ってもよいですが、最初は自前ラップで十分です。

例:

* Windows: `%AppData%\eitango-cli\`
* macOS: `~/Library/Application Support/eitango-cli/`
* Linux: `~/.local/share/eitango-cli/`

中身:

* `user.db`
* `config.toml`
* `logs/`

---

# 15. import/export

これは MVP 後半でよいですが、設計だけしておくと後で楽です。

## import

```bash
eitango import --file mywords.csv --format csv
eitango import --file business-pack.jsonl --format jsonl
eitango validate --file mywords.csv --kind import
```

必須列:

* lemma
* meaning_ja

任意列:

* pos
* level
* frequency_rank
* distractor_group
* example_en
* example_ja

`import` は user 辞書の取り込み口として最小必須を維持し、`core` 側より緩めにする。ただし 30k 拡張や pack 運用を見据えるなら、`frequency_rank` を入れられるようにしておくと出題品質を保ちやすい。

## export

```bash
eitango export wrong-words --format csv
eitango export progress --format json
```

これがあると、

* 苦手単語を Anki に持っていく
* 他学習ツールと連携
* 辞書メンテナンス
  がしやすくなります。

`core` 更新前には次も回せるようにしておくと安全です。

```bash
eitango validate --embedded-core
eitango validate --file assets/words_core.jsonl --kind core --format jsonl
```

---

# 16. MVP で必要なインデックス

SQLite は量的には余裕ですが、検索パターンに合わせて index は入れておくべきです。

```sql
CREATE INDEX idx_progress_due_at ON progress(due_at);
CREATE INDEX idx_words_level_pos ON words(level, pos);
CREATE INDEX idx_words_distractor_group ON words(distractor_group);
CREATE INDEX idx_reviews_word_id_answered_at ON reviews(word_id, answered_at);
```

---

# 17. エラーハンドリング方針

CLI/TUI はエラー体験がかなり重要です。

## 原則

* 1問ごとの保存失敗は即通知
* DB 初期化失敗は起動時に明示
* 辞書壊れは `doctor` で検出
* panic させない

TUI では画面下に status line を持つとよいです。

例:

* `Saved`
* `DB locked`
* `Imported 1200 words`
* `Resume session found`

---

# 18. テスト戦略

ここはちゃんと分けてテストできます。

## 単体テスト

* `quiz.BuildChoices`
* `srs.Update`
* `session.MakePlan`
* `dict.ParseJSONL`

## DB テスト

* migration
* due 単語取得
* review 保存
* resume

## TUI テスト

Bubble Tea そのものの snapshot テストを頑張りすぎる必要はありません。
重要なのは UI ではなくドメインロジックです。

---

# 19. 開発順序

これで進めるとかなり安全です。

## Step 1

Cobra で `eitango learn` だけ作る

## Step 2

Bubble Tea で固定 1 問表示

## Step 3

4択入力を実装

## Step 4

SQLite 初期化と 1問ごと保存

## Step 5

辞書 `embed` + 初回投入

## Step 6

SRS 導入

## Step 7

stats / resume / review モード

## Step 8

GoReleaser 設定

GoReleaser は複数 OS 向けのビルドやアーカイブを自動化できます。公式の Quick Start でも初期化からビルド/アーカイブ/リリースの流れが整理されています。 ([goreleaser.com][7])

---

# 20. 最初の MVP 仕様

私ならまずこれに絞ります。

## 必須

* `eitango learn`
* 10問セッション
* 4択
* 正誤表示
* 1問ごと保存
* due 復習
* resume
* `eitango stats`

## 後回し

* 例文
* 発音
* 同期
* AI 解説
* レベル別辞書パック
* スペリング入力モード

---

# 21. かなり重要なプロダクト上の提案

あなたのコンセプトに合うように、次の小機能を最初から入れると強いです。

## `--focus-mode`

1セッション 5問固定。
「待ち時間にちょっとだけ」のためのモード。

```bash
eitango learn --focus-mode
```

## `--idle-hook`

将来的に、AI エージェントの待機中に自動起動できる余地を残す。

たとえば将来:

```bash
codex ... && eitango learn --focus-mode
```

## `streak by waiting`

普通の学習アプリの streak ではなく、
**「待機時間をどれだけ学習へ変換したか」**
を見せると独自性が出ます。

---

# 22. この設計で最初に書くべきコード

最初に書くならこの3ファイルです。

### `cmd/eitango/main.go`

* root command
* learn command

### `internal/store/store.go`

* OpenDB
* Migrate
* SaveReview
* GetDueWords

### `internal/app/model.go`

* RootModel
* Init / Update / View

