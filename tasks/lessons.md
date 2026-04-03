# Lessons

- CSV や TSV の header を map に詰めるときは、後勝ち上書きを許さず duplicate column を即エラーにする。入力契約の曖昧さは静かに吸収しない。
- フラグ値や入力値を正規化して validation する場合は、判定だけでなく後段へ渡す返り値も正規化済みの値にする。normalized で許可したのに raw 値を返すと downstream で壊れる。
- 並列レビュー用の TSV は手で再構成しない。固定 schema の split / merge スクリプトを通し、row ごとの列数不一致と numeric-only `example_ja` を core 反映前に必ず弾く。
- 語彙データを追加・更新するときは `meaning_ja` と `distractor_group` をサンプル抽出で必ず目視確認する。意味の自他や品詞不一致、`people-noun` など意味とかけ離れた分類はテストでは落ちず学習画面の品質だけを壊す。
- 埋め込み辞書へ新規語を seed する前に、各語の中心義と品詞を最低1件ずつ確認する。特殊義や派生義、分数のような限定用法を代表訳として採用しない。
- `people-noun` は人そのものを指す名詞に限定し、活動名詞や身体部位を入れない。人物でない語を混ぜると誤答候補が人物語へ偏り、出題品質を崩す。
- 同じ生成物を読む後段コマンドを、前段の書き込みコマンドと並列で走らせない。`merge_parallel_reviews.py` の出力 TSV を `apply_review_batch.py` が読むような producer/consumer 関係は必ず直列にする。
- ローカルビルドで生成した `eitango.exe` や `bin/` 配下の実行ファイルはリポジトリへ含めない。配布用バイナリは Git ではなく release artifact で管理する。
- installer や配布補助スクリプトで release asset 名を組み立てるときは、推測した tag 文字列をそのまま埋め込まず、実際の GoReleaser naming と `checksums.txt` を基準に fixture も含めて一致確認する。
- destructive な削除は TTY 対話で後付け有効化しない。CLI 仕様で `--purge-data` のような明示 flag に限定すると決めたら、その契約をコードとテストの両方で固定する。
- 文字入力を受け付ける画面では、単文字ショートカットを回答文字と衝突させない。補助操作は `Tab` / `Ctrl+...` / `Esc` のような非文字キーに寄せ、`h/s/q` のような代表文字が通常入力として通る回帰テストを追加する。
- read-only 診断は最新 schema を直接仮定しない。`doctor` のように migration 未適用 DB を読む経路では列追加後も `PRAGMA table_info(...)` などで実スキーマを見て fallback し、欠けている migration は専用 check でだけ報告する。
- schema introspection 用の `PRAGMA table_info(...)` は table 名を `fmt.Sprintf` で組み立てない。現在の呼び出し元が定数でも、許可テーブルごとの定数 query に閉じて静的解析と将来の misuse を防ぐ。
- Cobra の runnable 親コマンドに subcommand を足したら、`Args` を明示して typo 位置引数を拒否する。未指定だと `play wrtie` が default 実行へ落ちる。
- format string を更新した i18n テストは「空でない」だけで終わらせない。`%!s(MISSING)` を見逃さないよう、期待文字列か少なくとも verb 数一致まで確認する。
- 状態復旧メッセージは primary な整合回復を secondary snapshot の成功に依存させない。active session を消す補正では `home` 更新を最優先にし、`stats` など付随情報は取れた場合だけ上書きする。
