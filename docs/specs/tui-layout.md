# TUI Layout

- width が未確定 (`WindowSizeMsg` 前で `RootModel.width == 0`) の間は、狭幅 guard を発火させない
- 現在の screen / overlay ごとに `normal / compact / narrow` の 3 段階レイアウトを持つ
- `compact` 対象は `home`, `home confirm`, `settings overlay`, `help`, `quiz.choice`, `quiz.write`, `feedback.choice`, `feedback.write` とする
- 幅 tier は次で固定する
  - `44 / 56` cols: `home`, `quiz.write`, `feedback.write`
  - `48 / 64` cols: `settings overlay`, `home confirm`, `help`, `quiz.choice`, `feedback.choice`
  - `56` cols only: `results`, `stats`
  - `76` cols only: `keymap editor`
- `width >= normalMin` では通常 UI を描く
- `compactMin <= width < normalMin` では compact layout に切り替える
- `width < compactMin`、または compact を持たない画面で `width < normalMin` のときは narrow message に切り替える
- narrow message は現在幅と必要幅を明示し、terminal を広げると通常表示へ戻ることを案内する
- compact layout では border と固定幅 label を減らし、key guide / help / label-value 行を現在幅に収まるよう wrap する
- narrow message 本文と status line は、現在幅に収まるよう wrap する
- `Update` の key handling は狭幅 guard で変えない。描画だけを切り替え、runtime input 契約は維持する
