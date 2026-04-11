# Audio Settings

`eitango` は local TTS の既定動作を `config.toml` の `audio_enabled`, `audio_autoplay`, `audio_voice` で制御する。

## Contract

- `audio_enabled` は local audio backend を使うかどうかを表す bool
- `audio_autoplay` は session 中の自動再生既定値を表す bool
- `audio_voice` は local voice の識別子で、空文字または未指定は `auto` 扱い
- `audio_voice = auto` 相当では、platform ごとの既定英語 voice 選択を使う
- 保存済み `audio_voice` が実機から消えていた場合は、実行時に `auto` へ正規化して fallback する
- unsupported platform や backend unavailable の場合も設定画面の行自体は維持し、音声再生だけ安全側に倒す

## Platform Rules

- macOS は `say -v ?` の結果を voice catalog とする
- Windows は `System.Speech.Synthesis.SpeechSynthesizer` の installed voices を voice catalog とする
- Windows の `ConvertTo-Json -Compress` は installed voice が 1 件だけのとき配列ではなく単一 object を返すため、consumer は `{"Name":"Microsoft David Desktop","Locale":"en-US"}` と `[{"Name":"Microsoft David Desktop","Locale":"en-US"}]` の両方を受け入れる
- settings overlay では `audio_enabled = false` でも catalog を閲覧できる
- catalog / capability probe は cache し、settings render ごとに subprocess を再実行しない

## UI Rules

- ホーム設定 overlay に `Local voice` 行を表示する
- `Local voice` の選択肢は `auto` + installed voices
- `audio unavailable` / `audio disabled` の status 表示契約は autoplay 操作で維持する
- quiz / feedback の手動再生と autoplay は同じ resolved speaker を使う
