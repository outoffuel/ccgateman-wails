`.github/workflows/release.yml` を更新して、`dev` ブランチへの push 時にも GitHub Release を自動作成・更新できるようにしてください。

### 要件
1. トリガーの追加
   - `on.push.branches` に `dev` を追加してください。

2. バージョン番号の生成 (`version` ジョブ)
   - `dev` ブランチの場合は、バージョン文字列を `dev-vYYYY.MM.DD.${{ github.run_number }}`（または単に `dev`）として生成し、バイナリの `appVersion` に埋め込んでください。

3. リリースの分離 (`release` ジョブ)
   - `master` ブランチへの push 時はこれまで通りタグ名 `latest` のリリースを更新してください。
   - `dev` ブランチへの push 時は独立したタグ名 `dev` のプレリリースを作成・更新してください（`master` 側の `latest` リリースやバイナリは一切上書き・削除しないこと）。
   - タグ push 時はこれまで通りタグ名のリリースを作成してください。

