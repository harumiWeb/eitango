# Security Policy

`eitango` のセキュリティ報告窓口と、対応バージョンの考え方をまとめています。 This document describes how to report vulnerabilities privately and how supported versions are handled.

## Reporting a Vulnerability

脆弱性やセキュリティ上の懸念を見つけた場合は、**公開 issue ではなく**次のメールアドレスへ非公開で連絡してください。 If you find a vulnerability, please report it privately instead of opening a public issue.

- `harumiweb.security@gmail.com`

可能であれば、以下もあわせて記載してください。 When possible, include the following details:

- 影響を受ける version / install 方法
- 再現手順
- 想定される影響
- 必要であれば PoC やログ

個人開発プロジェクトのため、返信と初動は **best effort** です。ただし、確認できた報告には可能な範囲で対応します。 Response time is best effort, but confirmed reports will be handled as capacity allows.

## Disclosure Guidance

- まずは非公開で報告してください
- 修正または緩和策の準備前に、公開 issue / discussion / SNS などで詳細を共有することは避けてください
- 内容の確認後、必要に応じて公開タイミングや告知方法を相談します

Please avoid disclosing full details in public before a fix or mitigation is ready. After triage, we can coordinate disclosure timing if needed.

## Supported Versions

`eitango` は比較的新しいリリースを優先して保守します。原則として、サポート対象は次のいずれかです。 In practice, security support is focused on the newest release line.

- 最新リリース
- まだ置き換え期間中と判断できる、ごく最近のリリース

古いバージョンでは修正を backport しない場合があります。その場合は、最新の安全なリリースへの更新を案内します。 Older releases may not receive backports and may be asked to upgrade instead.

サポート範囲や運用方針は、プロジェクトの成長に応じて将来変更されることがあります。 This policy may change as the project evolves.
