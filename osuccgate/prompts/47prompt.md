ccgateman-wails のフロントエンドUI全体を「Material Design 3 (Material You / M3)」の設計原則に沿った洗練されたデザインへ刷新（リファクタ）してください。

### 【全体方針・制約】
1. **技術スタックの維持**:
   - 新たな重いコンポーネントライブラリは導入せず、既存の **Svelte 4 + Tailwind CSS** のユーティリティクラスを最大限活用してM3デザインを表現してください。
   - バックエンド（Wails / Go / SQLite）とのバインディングや既存の業務ロジック（打刻、検索、ソート、モーダル、ページネーション等）は一切破壊せず、見た目・インタラクションのみを刷新してください。
2. **デザインシステムの導入 (M3トークン & カラー)**:
   - 必要に応じて `tailwind.config.cjs` のテーマ拡張を行い、M3の色体系（Primary, Secondary, Tertiary, Surface Container (Lowest〜Highest), Outline, Error）およびトーン配色（Surface Tint, Primary Container 等）を定義・適用してください。
   - 濃いドロップシャドウを排除し、M3特有の「面（Surface）のトーン階層」「Elevation 0〜3」「ボーダー（Outline Variant）」による美しい奥行きを表現してください。
3. **M3の形状・タイポグラフィ**:
   - カード類: `rounded-2xl` または `rounded-3xl`
   - ボタン: 完全な角丸 `rounded-full`（Pill形状）または `rounded-xl`（Filled, Tonal, Outlined, Text の各バリエーション）
   - モーダル/ダイアログ: `rounded-[28px]` のコンテナとトーン背景
   - チップ/バッジ: 区分表示やステータス表示に M3 Assist / Filter / Suggestion Chip スタイルを採用

### 【対象画面と具体的な改修要件】
1. **キオスク / ゲート画面 (`GateScreen.svelte`, `KioskSwipe.svelte`, `Header.svelte`, `PINModal.svelte`)**:
   - 打刻待機時: M3スタイルのクリーンな大カード（Surface Container）と柔らかなアイコン・タイポグラフィ。
   - 打刻結果表示: 入室（Success/Primary Container）、退室（Secondary/Tertiary Container）、未登録/今年度未登録（Warning Container）、エラー等のフィードバックをM3のトーンカラーで鮮やかに分かりやすく表示。
   - PIN入力モーダル: M3ダイアログ形状（角丸28px）、M3 Outlinedテキストフィールド、Tonal/Filledアクションボタン。

2. **管理者画面ヘッダー & ナビゲーション (`AdminDashboard.svelte`)**:
   - タブ切替部分を M3 Primary Tabs または Segmented Button / Navigation Rail スタイル（アクティブ時にPill型の背景インジケーターが付く形状）に刷新。
   - 年度バッジやヘッダー操作ボタンを M3 Tonal / Outlined Button に統一。

3. **ダッシュボードタブ (`DashboardTab.svelte`)**:
   - 4つの統計カードを M3 Elevated / Filled Card に刷新。アイコン背景とコンテナを M3 のトーンカラー（Primary Container, Secondary Container 等）に整理。
   - グラフ表示期間切替（週/月/年）を M3 Segmented Buttons に変更。
   - 月別統計テーブルを M3 Data Table スタイル（角丸コンテナ、ヘッダーの控えめなトーン、区分別チップ）に整流。

4. **在室者タブ & ログタブ & ユーザー管理タブ (`InsideTab.svelte`, `LogsTab.svelte`, `UsersTab.svelte`)**:
   - 検索バー: M3 Search Bar スタイル（`rounded-full` のピル形状、Surface Container背景、Leading Icon）。
   - テーブル一覧: M3のフラットで視認性の高いテーブルデザイン、行ホバー時の State Layer（薄いオーバーレイ）、列幅リサイズやソートアイコンの馴染み。
   - ページネーション: M3 Navigation Controls（角丸ボタン・アイコンボタン）。

5. **モーダルダイアログ (`UserModal.svelte`, `AdvancedOpsModal.svelte`)**:
   - M3 Dialog（角丸28px、背景ブラー、ヘッダー/コンテンツ/アクションの明確な分離）。
   - フォーム入力欄: M3 Outlined Text Field（枠線フォーカス時のPrimaryアクセント、ヘルパーテキスト、ラベル配置）。

### 【検証】
- 各ファイルの変更後、`cd osuccgate/frontend && npm run build` を実行して構文エラーやビルドエラーが一切ないことを確認してください。
