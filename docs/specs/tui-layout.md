# TUI Layout

- width が未確定 (`WindowSizeMsg` 前で `RootModel.width == 0`) の間は、狭幅 guard を発火させず、`renderHome` / `renderQuiz` / `renderFeedback` / `renderResults` / `renderStats` / `renderHelp` / `renderKeymapEditor` など全 screen の従来 renderer を維持する
- 現在の主要 screen / overlay は `adaptive / narrow` の 2 段階レイアウトを持つ
- adaptive 対象は `home`, `home confirm`, `settings overlay`, `help`, `quiz.choice`, `quiz.write`, `feedback.choice`, `feedback.write`, `results`, `stats`, `keymap editor` とする
- 幅 tier は次で固定する
  - `28` cols: `home`, `results`, `stats`, `quiz.write`, `feedback.write`
  - `32` cols: `settings overlay`, `home confirm`, `help`, `quiz.choice`, `feedback.choice`, `keymap editor`
- `width >= minWidth` では同じ adaptive renderer が panel を terminal 幅へ連続追従させる
- `width < minWidth` のときは narrow message に切り替える
- narrow message は現在幅と必要幅を明示し、terminal を広げると通常表示へ戻ることを案内する
- adaptive layout では border を残したまま panel を terminal 幅へ縮め、horizontal padding を減らす
- adaptive layout の単一行 UI は `...` による省略で current width に収める
- `...` の対象は key guide / keymap 表示 / settings row / quiz meta / results-stat summary / keymap editor row とする
- `quiz.choice` の選択肢本文と `results` の hard words のような主情報は `...` で省略せず、adaptive 幅でも wrap して全文を残す
- help 本文、settings note、feedback examples、narrow message 本文のような prose は wrap を維持する
- narrow message 本文と status line は、現在幅に収まるよう wrap する
- `Update` の key handling は狭幅 guard で変えない。描画だけを切り替え、runtime input 契約は維持する
