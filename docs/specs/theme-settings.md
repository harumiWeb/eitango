# Theme Settings

`eitango` は表示テーマを `config.toml` の `theme_mode` と `theme_palette` で制御する。

## Contract

- `theme_mode` は `default` / `no_color` / `neon` / `custom` の 4 値
- `theme_palette` は `custom` 用の optional role-based palette
- `theme_palette` の key は `accent` / `success` / `danger` / `muted` / `border`
- 色値は `#RRGGBB` 形式のみを受け付ける
- `theme_palette` の未指定 slot は default palette へ fallback する
- 設定保存時も、未指定 slot は空文字で書き出さず key 自体を omit する
- `theme_mode != "custom"` のときも `theme_palette` は保存対象に残す

## Presets

- `default`: 旧 `NewStyles()` の配色と未着色 slot をそのまま維持する
- `no_color`: 色指定を外し、terminal default color に委ねる
- `neon`: ライトグリーン基調の高コントラスト preset
- `custom`: `theme_palette` を適用する

## UI Rules

- ホーム設定 overlay では theme mode だけを切り替える
- `custom` の色詳細編集は `config.toml` で行う
- 選択状態やエラー状態は色だけに依存させない
  - selected tab は bracket で表す
  - selected settings row は prefix 記号で表す
  - status line は `status:` と `error:` を使い分ける
