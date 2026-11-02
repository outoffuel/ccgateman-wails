システム全体のコード最適化および動作パフォーマンスの最適化を行ってください。既存の機能・UI・仕様は一切変更せず、以下の改善を実施してください。

### 1. SQLiteデータベースのクエリ・インデックス最適化 (`internal/db/db.go`)
- `createTables()` に複合インデックス（`idx_access_logs_card_id_id ON access_logs(card_id, id DESC)` および `idx_access_logs_event_timestamp ON access_logs(event_type, timestamp)`）を追加してください。
- `GetDashboardStats()` で個別に実行している `access_logs` の期間別カウント（本日・週間・月間・年間）を1回のクエリ（または最小限のクエリ）に統合し、DBロック時間とI/Oを削減してください。
- SQLite接続のPRAGMA設定を最適化（`_pragma=synchronous(NORMAL)` 等の追加）し、WALモードでの読み書き性能を向上させてください。

### 2. バックエンドのメモリ・リソース管理の最適化 (`internal/service/gate_service.go`)
- `lastSwipes` マップ（デバウンス管理用）に古いエントリが溜まり続けないよう、打刻時またはスケジューラで期限切れエントリを定期的にクリーンアップする処理を追加してください。

### 3. フロントエンドのデータ取得・再計算の最適化 (`AdminDashboard.svelte`, 各タブコンポーネント, `utils/`)
- `AdminDashboard.svelte` の `loadData()` に多重実行防止ガード（実行中は重複リクエストしない）を追加してください。また、5秒ごとの定期ポーリングでは現在開いている `activeTab` に必要なデータのみを取得し、タブ切り替え時やデータ更新時に必要なデータを取得するよう通信量を最適化してください。
- `AdminDashboard.svelte` で生成した `processedLogs` を `LogsTab.svelte` にも渡し、同じログ配列のフォーマット変換を二重に行わないようにしてください。また、未使用の `Counter.svelte` が残っていれば削除してください。
- `DashboardTab.svelte` のグラフデータ計算（`weekChartData`, `monthChartData`, `yearChartData`）について、現在選択されている `chartPeriod` のデータのみを計算するようにし、ログ走査の計算量を減らしてください。
- `utils/sorting.js` の `sortList` で毎回 `localeCompare` を初期化するのではなく、モジュールスコープで生成した `Intl.Collator('ja', { numeric: true })` を使ってソートを高速化してください。
- `audio.js` の `playTone` で `osc.onended` 時に `osc.disconnect()` と `gain.disconnect()` を行い、長期稼働時のWeb Audioノードを確実に解放してください。
- `LogsTab.svelte` の `exportLogsCSV` で生成した ObjectURL を `URL.revokeObjectURL(url)` で解放するようにしてください。

### 4. 確認事項
- 変更後、`go test ./...` とフロントエンドの `npm run build` がエラーなく通ることを確認してください。
