入退室履歴（LogsTab）のCSV出力ボタンのWailsネイティブ対応と、GUI全体のスクロールおよび描画の軽量化を行ってください。

### 1. CSV出力ボタンの修正（Wails ネイティブファイル保存対応）
- **現状の問題**:
  `LogsTab.svelte` の `exportLogsCSV` がブラウザ用の `<a>` タグ＋Blob URL（`link.click()`）で実装されており、Wails デスクトップ環境（WebKitGTK / WebView2）ではダウンロードが無効化されて何も起きない。
- **改修要件**:
  1. `osuccgate/app.go` に、任意のCSV文字列をOS標準のファイル保存ダイアログで保存できるメソッドを追加してください。
     - メソッド名: `SaveCSVContent(defaultFilename string, content string) (string, error)`
     - `runtime.SaveFileDialog(a.ctx, ...)` でダイアログを開き、ユーザーが指定したパスにUTF-8（BOM付き）でファイル保存して保存先パスを返す。キャンセルされた場合は空文字を返す。
  2. `LogsTab.svelte` の CSV出力処理を改修:
     - Wails環境（`window.go?.main?.App`）がある場合は `SaveCSVContent` を呼び出し、保存成功時に「CSVファイルを保存しました: {パス}」とフィードバック（alertまたはメッセージ）を表示する。
     - リモートブラウザ閲覧時（Webモード）は従来のBlobダウンロードへフォールバックする。
     - 出力フォーマット（日時, ID, 氏名, 判定, 区分, 利用目的, 入退室, 滞在時間）は維持すること。

### 2. GUIの徹底的な軽量化（スクロール・描画のもたつき解消）
M3デザイン適用後、WebView環境でスクロールが重くなるボトルネックを解消してください。

1. **`backdrop-blur`（すりガラス効果）の完全排除**:
   - スクロール領域の固定ヘッダー（`AdminDashboard.svelte`）やモーダル背景で `backdrop-blur-md` や `backdrop-blur-sm` が使われていると、WebKitGTK/WebView2 がスクロール毎フレームで背面ピクセルのガウシアンブラー再計算（GPU/CPU負荷）を行い、フレームレートが著しく低下します。
   - すべて M3 のソリッドな不透明背景（例: `bg-m3-surface-container-lowest`）やシンプルな透過色（`bg-black/50`）に置き換えてください。
2. **スクロールコンテナの多重ネスト解消**:
   - `App.svelte` のルート `<main>` に `overflow-y-auto` が残っていると、内部の `AdminDashboard.svelte`（`overflow-y-auto`）とホイールスクロールイベントが競合してカクつきの原因になります。`App.svelte` 側は `h-screen overflow-hidden` にして、スクロールは内部コンテナ単一に集約してください。
   - `AdminDashboard.svelte` のスクロール領域 `<main>` に `[will-change:scroll-position]` を付与し、GPU合成レイヤーとして最適化してください。
3. **トランジション・シャドウの軽量化**:
   - `index.css` の `.m3-btn-*` や `.m3-input-*` に指定されている `transition-all`（レイアウト・ジオメトリ含む全プロパティの補間）を、負荷の低い `transition-colors duration-150` に変更してください。
   - テーブル行ホバー時のアルファ透過カラー（`hover:bg-m3-on-surface/[0.04]` 等）を、シンプルなソリッド系トーン（`hover:bg-m3-surface-container-high`）に変更し、ラスタライズ負荷を軽減してください。
4. **入退室履歴（LogsTab）の表示件数選択**:
   - ハードコードされている件数（10件）を動的な `logsPerPage`（10件, 20件, 50件, 100件）に変更し、テーブル上部から選択できるようにしてください（デフォルト20件）。

### 3. ビルドと動作確認
- 改修後、`cd osuccgate/frontend && npm run build` および Go側のビルドチェックを実行し、エラーがないことを確認してください。
