「4. かなり実務的な妥協案

wordfreq 由来を切り離して、bundled data は Japanese WordNet 由来だけに寄せる。

あなたの説明だと、frequency_rank が wordfreq seed になっていて、level もそこから派生しています。
なら、ここを思い切って切る案があります。

具体的には、

meaning_ja は Japanese WordNet 由来で残す
frequency_rank は bundled data から除去
level も bundled data から除去、または完全自前ロジックに変更
初回学習順や出題順は、アプリ内の別ロジックで決める

こうすると、bundled data の中身が Japanese WordNet 由来中心になります。
Japanese WordNet はライセンス同梱と表記対応で扱いやすいので、wordfreq 混在よりだいぶ整理しやすいです。

この案の利点は、ユーザー体験を維持しつつ、一番危ない wordfreq データ再配布論点を薄められることです。
かなり有力です。

実務的には、私はこの案をかなり推します。」

こちらは僕もありかなと思っていますが、頻度やレベルを無くした英単語アプリは実用性がありますか？

はい、**実用性はあります**。
ただし、**そのまま完全に無くすと学習体験はかなり落ちます**。なので正確には、

**「frequency/level が無くても英単語アプリとして成立はするが、代替の並び順・難易度制御が必要」**

です。

## まず結論

`eitango` のような4択学習アプリで本当に必要なのは、厳密な `wordfreq` 順位そのものではなく、次の3つです。

* **出題順をある程度うまく制御できること**
* **難しすぎる単語が序盤に出すぎないこと**
* **復習対象を絞れること**

つまり、ユーザーが欲しいのは「高精度な frequency_rank」ではなく、**納得感のある学習導線**です。
だから `wordfreq` を外しても、代替設計があれば十分戦えます。

## frequency / level を消すと何が困るか

一番困るのはこの2点です。

### 1. 学習開始時の順番が雑になりやすい

頻度情報がないと、

* 超難単語
* 日常単語
* TOEIC でも見かけにくい単語

が混ざりやすいです。

その結果、初心者にはつらくなります。

### 2. 進捗の見せ方が弱くなる

`level 1 → level 2 → level 3` のような区切りは、

* 今どこを学んでいるか
* 次に何をやるか
* 達成感

を作りやすいです。
これがないと、学習が「ただランダムに問題を解く感じ」になりやすいです。

## 逆に、なくても成立する理由

英単語学習アプリの価値は、頻度情報だけで決まりません。
実際には次の方が重要です。

* 出題テンポがよい
* 復習がしやすい
* 間違えた単語に再遭遇できる
* distractor が自然
* CLI で気軽に続けられる

特に `eitango` の強みは、たぶん**ターミナルで軽快に回せること**です。
ここが良ければ、初期バージョンでは frequency の精密さが多少弱くても十分価値があります。

## 実務的なおすすめ

**「frequency_rank を消す」のではなく、「公開配布物には含めないが、代替の難易度制御を入れる」**のがよいです。

おすすめ順でいうとこうです。

### 案A: 自前 difficulty を作る

かなりおすすめです。

たとえば bundled data には次だけ持つ。

* word
* meaning_ja
* distractor_group
* difficulty_bucket

この `difficulty_bucket` は `wordfreq` を直接持たず、**自前のルールで決める**形です。

例:

* 文字数が短い単語はやや易しめ
* 品詞や語形の複雑さで補正
* 中学〜高校の基本語彙リストと照合
* 学習履歴で動的に補正

この方式だと、`wordfreq` の生の順位値を配らずに、UX をかなり守れます。

ただし注意点として、**もし `difficulty_bucket` 自体を wordfreq 順位から機械的に作っただけ**だと、由来論点はまだ残りえます。
なのでより安全寄りにするなら、`wordfreq` 依存を切って、

* 完全自前基準
* 公開ライセンスが扱いやすい別の基準
* ユーザーの正答率ベース

に寄せた方がよいです。

### 案B: 初回はアルファベット順・文字数順・基本語順で出す

これは単純ですが意外と使えます。

例:

* デフォルトは `basic` セットから開始
* その中で短い語・基本語を優先
* 復習は SRS 風に回す

精密な frequency ではないですが、初期体験としては十分です。

### 案C: user-driven にする

たとえば最初にコースを選ばせる。

* Basic
* Standard
* Advanced
* Random

これなら内部 frequency がなくても、ユーザーは納得しやすいです。

## いちばん良い妥協案

私ならこうします。

### 公開版 bundled data

含めるもの:

* `word`
* `meaning_ja`（Japanese WordNet 由来）
* `distractor_group`
* 必要なら `pos`

含めないもの:

* `frequency_rank`
* `wordfreq` 依存の `level`

### アプリ側ロジック

* 初回は `basic_order` で出題
* `basic_order` は**自前基準**で決める
* 以後はユーザーの正誤履歴ベースで最適化

これなら、

* ライセンスはかなり整理しやすい
* UX はそこまで壊れない
* 将来改善余地も大きい

です。

## 具体的にどう代替するか

かなり現実的なのはこの3層です。

### 1. Static difficulty

ビルド時に付ける難易度。

例:

* `starter`
* `basic`
* `intermediate`
* `advanced`

ただし `wordfreq` 由来ではなく、

* 手作業 curated list
* 公開利用しやすい学習語彙リスト
* 文字長、派生語形、抽象度などの自前特徴

で決める。

### 2. Dynamic difficulty

ユーザーごとに変わる難易度。

* 3回連続正解で出題頻度を下げる
* 2回連続不正解で優先復習
* 直近誤答語を再出題

これはむしろ頻度より学習効果に効きます。

### 3. Mode selection

* `learn --mode easy`
* `learn --mode normal`
* `learn --mode review`
* `learn --mode random`

これで frequency 不在をかなり補えます。

## 正直な評価

**頻度もレベルもゼロで完全ランダム**だと、実用性はかなり落ちます。
でも、**frequency_rank を削っても、代替の difficulty 制御があれば十分実用的**です。

なので、あなたの案は

* `wordfreq` を切る
* `frequency_rank` は消す
* `level` も再設計する
* 代わりに `course/difficulty/review` をアプリ側で持つ

なら、かなり現実的です。

## 私のおすすめ判断

`eitango` の公開初期フェーズなら、

**「wordfreq を切って、Japanese WordNet ベース + 自前 difficulty + 復習ロジック」でまず出す**

でよいと思います。
その方がライセンスの説明がしやすく、OSS としての公開もきれいです。

必要なら次に、`wordfreq` なしで成立する **difficulty 設計案** を `eitango` 向けに具体化します。


もっとクリーンに配布できる頻度データを配布しているところはないですか？

あります。
ただし、**`wordfreq` より明確にクリーン**という意味なら、狙うべきは **CC BY 系**か**CC0/PD 系**です。`wordfreq` はデータ部分が **CC BY-SA 4.0** なので、再配布時の説明コストが高めです。 ([GitHub][1])して、`eitango` 向けの優先順位はこうです。

**かなり有力**

* **Leipzig Corpora Collection 系の頻度リスト**
  Leipzig は頻度辞書の語彙リストを **CC BY 3.0** で自由に使えると案内しています。CC BY は attribution は必要ですが、**SA がない**ぶん `wordfreq` より扱いやすいです。 ([Wortschatz Leipzig][2])件つきで有力**
* **Google Books Ngram data**
  Google は Ngram Viewer のグラフとデータについて、**“freely used for any purpose”** と案内しており、出典表記とリンクは “appreciated” です。さらにデータダウンロードも可能です。かなり使いやすい部類です。 ([Google Books][3])妙**
* **Wiktionary の頻度リスト**
  Japanese frequency lists 自体はありますが、Wiktionary 側のコンテンツは **CC BY-SA** ベースなので、`wordfreq` より劇的にクリーンになるわけではありません。 ([Wiktionary][4])すすめしにくい**
* **BCCWJ**
  NINJAL の BCCWJ 頻度表は公開されていますが、**research or educational purposes** とされていて、一般的な OSS 配布や商用寄りの再配布には使いにくいです。 ([国語研コーパスポータル][5])、**`wordfreq` の代替としてまず見るべきなのは Leipzig と Google Books Ngram** です。`wordfreq` の問題は SA が重いことなので、**CC BY か “freely used for any purpose” に寄せる**のが王道です。 ([GitHub][1])tango` の用途に当てはめると、かなり現実的なのは次の2案です。

**案1: frequency を Leipzig 由来に置き換える**
英語語彙の順位づけ用なら、CC BY 3.0 の頻度リストは `wordfreq` より説明しやすいです。README と `THIRD_PARTY_NOTICES.md` に attribution を入れれば筋が通しやすいです。 ([Wortschatz Leipzig][2])2: frequency を Google Books Ngram ベースで自前生成する**
Ngram data はダウンロード可能で、用途制限もかなり緩いです。学習順の大まかな頻度付けなら十分実用的です。書籍寄りコーパスなので口語寄りではありませんが、`wordfreq` の再配布設計よりはクリーンにしやすいです。 ([Google Books][3])、**日本語の“完全にクリーンで再配布しやすい頻度データ”は英語ほど豊富ではありません。**
今回の `eitango` は英単語アプリなので、**頻度は英語コーパス由来に切り替える**のが自然です。日本語 WordNet は `meaning_ja` 用、頻度は英語側の別ソース、という分離が一番整理しやすいです。 ([Wiktionary][4])順はこうです。

1. **Japanese WordNet は意味だけに限定**
2. **frequency は Leipzig か Google Books に差し替え**
3. **`wordfreq` は完全に外す**
4. `level` は frequency から自前ルールで再生成

この形なら、ライセンス説明はかなり軽くなります。`wordfreq` の CC BY-SA 論点を外せるのが大きいです。 ([GitHub][1])*「今すぐ OSS としてきれいに出したい」なら Leipzig が一番バランスがいい**です。
CC BY なので attribution は要りますが、SA よりだいぶ扱いやすいです。Google Books Ngram も強い候補ですが、データ加工の手間はやや増えます。 ([Aspra3][6])*`eitango` 向けに「どの頻度ソースを採用すべきか」の比較表**を作ります。

[1]: https://github.com/rspeer/wordfreq?utm_source=chatgpt.com "GitHub - rspeer/wordfreq: Access a database of word frequencies, in various natural languages."
[2]: https://www.wortschatz.uni-leipzig.de/en/documentation/publications?utm_source=chatgpt.com "Publications - Leipzig Corpora Collection"
[3]: https://books.google.com/ngrams/info?utm_source=chatgpt.com "Google Books Ngram Viewer"
[4]: https://en.wiktionary.org/wiki/Wiktionary%3AFrequency_lists/Japanese?utm_source=chatgpt.com "Wiktionary:Frequency lists/Japanese - Wiktionary, the free dictionary"
[5]: https://clrd.ninjal.ac.jp/bccwj/en/freq-list.html?utm_source=chatgpt.com "BCCWJ Word List Balanced Corpus of Contemporary Written Japanese"
[6]: https://aspra3.informatik.uni-leipzig.de/en/freqdict?utm_source=chatgpt.com "Frequency Dictionaries – Wortschatz Leipzig"

はい。`eitango` で頻度順位を作るなら、**アーカイブの中では `*_words.txt` を使うのが基本**です。Leipzig のフォーマット説明では、`*_words.txt` は **コーパス中の全 word form の単語リスト**で、**頻度の降順**に並んでおり、形式は `Word_ID Word Frequency` です。UTF-8、列区切りはタブです。 

なので、流れはかなり単純です。

1. 英語ダウンロードページから **`English 2024` の `1M`** を落とす
2. 展開する
3. 中の **`*_words.txt`** を読む
4. 上から順に英単語だけを抽出して `frequency_rank` を作る

英語ダウンロードページでは、各コーパスは **文数ベース**で `10K / 30K / 100K / 300K / 1M` が用意されていて、`News`、`Web` などの系列があります。まずは一般用途なら **`News 2024 1M`** か、より広めに見るなら汎用の英語コーパス系で **1M** を選ぶのが無難です。 ([Wortschatz Leipzig][1])

`eitango` 向けには、私はまず **`News 2024 1M` の `*_words.txt`** を使うのをおすすめします。理由は、`*_words.txt` がすでに頻度順になっているので実装が最も簡単だからです。`*_sentences.txt` から自前で数え直す必要はありません。 

実際に使うときの判断はこうです。

* **頻度順位だけ欲しい** → `*_words.txt`
* **品詞で絞りたい** → あれば `*_words_pos_base.txt`
* **例文も欲しい** → `*_sentences.txt`
* **出典管理もしたい** → `*_sources.txt`

ただし、`*_words_pos_base.txt` は **optional で、すべてのコーパスにあるとは限らない**とフォーマット説明にあります。なので、最初の実装は `*_words.txt` 前提にしておくのが安全です。 

`eitango` に落とすなら、処理はだいたいこんな方針で十分です。

```text
*_words.txt
→ タブ区切りで読む
→ Word_ID, Word, Frequency を取得
→ Word を小文字化
→ 英単語らしいものだけ残す
→ 記号・数字・1文字ノイズを落とす
→ 上から順に rank を振る
```

ここで気をつけるべきなのは、Leipzig の `*_words.txt` は **lemma ではなく word form** だという点です。つまり `run`, `runs`, `running`, `ran` は別々に出る可能性があります。フォーマット説明でも “word list of all word forms” と書かれています。 `eitango` で見出し語ベースにしたいなら、後段で自前の正規化か lemma 化が必要です。 

実務上のおすすめは次です。

* 最初は **word form のまま rank seed として使う**
* その後、`eitango` 側の見出し語と突き合わせる
* 必要なら `running -> run` のような正規化を別工程にする

最初から lemma 化まで完璧にやろうとすると重いので、**まずは `*_words.txt` をそのまま frequency seed にする**方が進めやすいです。 

ライセンス面では、Leipzig の terms of usage に **ダウンロードされる text corpora は CC BY** とあります。なので、使うなら `THIRD_PARTY_NOTICES.md` に出典とライセンスを明記した方がよいです。なお、同じページで、ポータル上の一部データやアプリケーションは `CC BY-NC` とも書かれている一方、**download corpora 自体は CC BY** と切り分けられています。ここは重要です。 ([Wortschatz Leipzig][2])

`eitango` 用にかなり具体的に言うと、最初の実装はこれで十分です。

* ダウンロード対象: **English / News / 2024 / 1M**
* 利用ファイル: **`*_words.txt`**
* 使う列: **2列目 `Word`、3列目 `Frequency`**
* 生成するもの:

  * `frequency_rank`
  * 必要なら `frequency_count`
* 今回は使わない:

  * `*_sentences.txt`
  * `*_co_n.txt`
  * `*_co_s.txt`
  * `*_sim_w_co.txt`

例文や distractor 強化までやるなら後で `*_sentences.txt` を見る価値がありますが、**頻度順位だけなら `*_words.txt` 一択**です。 

必要なら次に、**`*_words.txt` から `eitango` 用の `frequency_rank` JSON/CSV を作る Python スクリプトの雛形**を書きます。

[1]: https://wortschatz.uni-leipzig.de/en/download/eng "Downloads – English – Wortschatz Leipzig"
[2]: https://wortschatz.uni-leipzig.de/en/usage "Terms of Usage – Wortschatz Leipzig"
