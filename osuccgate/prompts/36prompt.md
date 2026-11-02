日付ベースの自動バージョニング（CalVer: `vYYYY.MM.DD.<run_number>`）と、GitHub Releasesを利用した自動アップデート（自己更新・再起動）機能を実装してください。

### 1. GitHub Actions (`.github/workflows/release.yml`) の更新
- `build` ジョブでバージョン文字列を生成してください（タグpush時はタグ名、それ以外は `TZ=Asia/Tokyo` の `vYYYY.MM.DD.${{ github.run_number }}`）。
- `wails build` 実行時に `-ldflags "-X main.appVersion=<生成したバージョン>"` を渡してバイナリにバージョンを埋め込んでください。
- `release` ジョブで `latest` リリースを作成する際、リリースの `name`（タイトル）にこのバージョン文字列が設定されるようにしてください。

### 2. バージョン情報の保持と公開 (`main.go`, `app.go`, `http_server.go`, `adminApi.js`)
- `main.go`（または `app.go`）に `var appVersion = "dev"` を定義し、Wailsバインディング (`GetAppVersion`) および HTTP API (`GET /api/version` または `/api/system/update-status`) で現在のバージョンを取得できるようにしてください。
- `AdminDashboard.svelte` の左サイドバー下部フッターに、現在のバージョン番号（例: `v2026.11.02.1`）を表示してください。

### 3. 自己アップデート機能 (`osuccgate/internal/updater/updater.go`)
- 起動時に過去の更新で残った `<実行ファイルパス>.old` があれば削除するクリーンアップ処理を入れてください。
- GitHub Releases API (`https://api.github.com/repos/outoffuel/ccgateman-wails/releases/tags/latest`) から最新リリース情報を取得し、現在の `appVersion` と最新リリースのバージョンおよびOS別アセット（Windows: `osuccgate-windows-amd64.exe`, Linux: `osuccgate-linux-amd64.tar.gz`）を比較する機能を実装してください。
- 更新を適用する際は、一時ファイルへダウンロード（Linuxは `tar.gz` からバイナリ展開）後、現在の実行ファイルを `.old` にリネームし、新しい実行ファイルを元のパスへ配置（実行権限付与）した上で、新プロセスを `exec.Command` で起動してから現在のプロセスを終了するようにしてください。

### 4. 自動アップデートスケジューラと管理画面の手動更新ボタン
- `gate_service.go`（または `app.go`）のスケジューラで、毎日深夜（23:00の強制退室処理の後、23:05など）に自動でアップデートチェックを行い、新バージョンがあれば自動更新・再起動するようにしてください（`appVersion == "dev"` のローカル開発時は自動更新をスキップすること）。
- `AdvancedOpsModal.svelte` に「システムアップデート」セクションを追加し、現在のバージョン表示、「更新を確認」ボタン、新バージョンがある場合の「今すぐ更新して再起動」ボタンを実装してください（Wails本体・8080番Web管理画面の両方で動作すること）。

