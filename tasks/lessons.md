# Lessons

- CSV や TSV の header を map に詰めるときは、後勝ち上書きを許さず duplicate column を即エラーにする。入力契約の曖昧さは静かに吸収しない。
- フラグ値や入力値を正規化して validation する場合は、判定だけでなく後段へ渡す返り値も正規化済みの値にする。normalized で許可したのに raw 値を返すと downstream で壊れる。
- 並列レビュー用の TSV は手で再構成しない。固定 schema の split / merge スクリプトを通し、row ごとの列数不一致と numeric-only `example_ja` を core 反映前に必ず弾く。
- 語彙データを追加・更新するときは `meaning_ja` と `distractor_group` をサンプル抽出で必ず目視確認する。意味の自他や品詞不一致、`people-noun` など意味とかけ離れた分類はテストでは落ちず学習画面の品質だけを壊す。
- 埋め込み辞書へ新規語を seed する前に、各語の中心義と品詞を最低1件ずつ確認する。特殊義や派生義、分数のような限定用法を代表訳として採用しない。
- 同じ生成物を読む後段コマンドを、前段の書き込みコマンドと並列で走らせない。`merge_parallel_reviews.py` の出力 TSV を `apply_review_batch.py` が読むような producer/consumer 関係は必ず直列にする。
