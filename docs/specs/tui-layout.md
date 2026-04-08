# TUI Layout

- width が未確定 (`WindowSizeMsg` 前で `RootModel.width == 0`) の間は、狭幅 guard を発火させない
- 現在の screen / overlay ごとに最小幅を固定し、未満では通常 UI を描かず narrow message に切り替える
- 最小幅 tier は次で固定する
  - `56` cols: `home`, `results`, `stats`, `quiz.write`, `feedback.write`
  - `64` cols: `settings overlay`, `home confirm`, `quiz.choice`, `feedback.rate`, `help`
  - `76` cols: `keymap editor`
- narrow message は現在幅と必要幅を明示し、terminal を広げると通常表示へ戻ることを案内する
- narrow message 本文と status line は、現在幅に収まるよう wrap する
- `Update` の key handling は狭幅 guard で変えない。描画だけを切り替え、runtime input 契約は維持する
