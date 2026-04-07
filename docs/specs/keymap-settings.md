# Keymap Settings

`eitango` は `config.toml` の `[keymap]` でキーバインド override を受け付ける。

## Contract

- default keymap はコード側の context/action registry を正本とする
- `config.toml` には override だけを保存し、未指定は default へ fallback する
- `keymap.version` は `1` のみ許可する
- unknown context / unknown action / invalid key token は load/save ともに error にする
- 同一 context 内で同じ key を複数 action に割り当てることはできない
- `quiz.write` では英字 1 文字の command binding を禁止する
- keymap editor から action を clear した場合は空配列 override を保存し、default binding を明示的に無効化する
- `help.back` と `help.quit` を同時に空配列にはできない。help 画面には常に少なくとも 1 つの脱出キーを残す

## Contexts

- `home`
- `home_confirm`
- `settings_overlay`
- `quiz.choice`
- `quiz.write`
- `feedback.rate`
- `feedback.write`
- `results`
- `stats`
- `help`

`[keymap.global]` は shared override の入力形式として受け付けるが、保存時は context ごとの canonical override に正規化してよい。

## Format

```toml
[keymap]
version = 1

[keymap.home]
toggle_answer_mode = ["x"]

[keymap.quiz.write]
hint = ["tab"]
skip = ["ctrl+s"]
confirm = ["enter"]
write_quit = ["esc"]
```

## UI Behavior

- ホーム設定 overlay に `Key bindings` 行を追加し、`Enter` で editor を開く
- editor は context filter、action 一覧、current/default 値、add/clear/reset/save を持つ
- keymap editor の save は keymap override だけでなく、settings overlay 上の未保存 draft も同時に永続化する
- record mode は `Esc` 自体も割り当て可能にし、cancel は `Ctrl+G` で行う
- editor 本体は terminal 高を超えて伸ばさず、表示可能高を超える action 一覧だけをスクロール表示する
- 保存後は再起動なしで runtime keymap と help/key guide に反映する
