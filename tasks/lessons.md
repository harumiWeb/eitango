# Lessons

- CSV や TSV の header を map に詰めるときは、後勝ち上書きを許さず duplicate column を即エラーにする。入力契約の曖昧さは静かに吸収しない。
- フラグ値や入力値を正規化して validation する場合は、判定だけでなく後段へ渡す返り値も正規化済みの値にする。normalized で許可したのに raw 値を返すと downstream で壊れる。
