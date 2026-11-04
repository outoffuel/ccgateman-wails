package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"osuccgate/internal/db"
	"osuccgate/internal/nfc"
	"osuccgate/internal/server"
	"osuccgate/internal/service"
	"osuccgate/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx         context.Context
	dbManager   *db.DBManager
	gateService *service.GateService
	nfcReader   *nfc.NFCReader
	httpServer  *server.HTTPServer
	updater     *updater.Updater
	adminPin    string
	appVersion  string
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	if version == "" {
		version = "dev"
	}
	return &App{
		adminPin:   "8202",
		appVersion: version,
		updater:    updater.NewUpdater(version),
	}
}

// resolveDatabasePath データベースファイルの最適なパスを解決する
func resolveDatabasePath() string {
	// 1. 環境変数による明示的な指定があれば最優先
	if envPath := os.Getenv("OSUCCGATE_DB_PATH"); envPath != "" {
		return envPath
	}

	// 2. 実行ファイルと同じディレクトリのDB (本番バイナリ実行時)
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		// 一時ディレクトリ(go run等)でなければチェック
		if !strings.Contains(exeDir, "go-build") && !strings.Contains(exeDir, "/tmp") && !strings.Contains(exeDir, "\\Temp") {
			candidate := filepath.Join(exeDir, "osuccgate.db")
			if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
				return candidate
			}
		}
	}

	// 3. カレントディレクトリおよび関連パス候補
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "build", "bin", "osuccgate.db"),
		filepath.Join(cwd, "osuccgate", "build", "bin", "osuccgate.db"),
		filepath.Join(cwd, "osuccgate", "osuccgate.db"),
		filepath.Join(cwd, "osuccgate.db"),
	}

	// 既にサイズが存在する (データが入っている) DBがあれば最も大きいものを優先
	var bestCandidate string
	var maxBytes int64 = 0
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			if fi.Size() > maxBytes {
				maxBytes = fi.Size()
				bestCandidate = c
			}
		}
	}
	if bestCandidate != "" && maxBytes > 0 {
		return bestCandidate
	}

	// 4. フォールバック
	if cwd != "" {
		return filepath.Join(cwd, "osuccgate.db")
	}
	return "osuccgate.db"
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// データベース保存パスの決定 (実行ディレクトリまたはAppData)
	dbPath := resolveDatabasePath()
	log.Printf("[Startup] Using SQLite database: %s", dbPath)

	// SQLite初期化
	dbMgr, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	a.dbManager = dbMgr

	// ゲートサービス初期化 & スケジューラ開始
	a.gateService = service.NewGateService(dbMgr)
	a.gateService.SetUpdater(a.updater, a.appVersion)
	a.gateService.StartScheduler()

	// NFC PaSoRi リーダーの監視開始
	a.nfcReader = nfc.NewNFCReader(func(cardID string) {
		// NFCカード検出時のコールバック
		resp := a.gateService.ProcessSwipe(cardID)
		if !resp.IsDebounced {
			// Wailsフロントエンドへイベント通知
			runtime.EventsEmit(a.ctx, "card_swiped", resp)
		}
	})
	a.nfcReader.Start()

	// LAN向けWeb管理者サーバーの起動 (ポート 8080)
	a.httpServer = server.NewHTTPServer(a.dbManager, a.gateService, 8080, a.adminPin, assets, a.updater, a.appVersion)
	a.httpServer.Start()
}

// domReady is called after front-end resources have been loaded
func (a *App) domReady(ctx context.Context) {
}

// beforeClose is called when the application is about to quit
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	if a.gateService != nil {
		a.gateService.StopScheduler()
	}
	if a.nfcReader != nil {
		a.nfcReader.Stop()
	}
	if a.httpServer != nil {
		a.httpServer.Stop()
	}
	if a.dbManager != nil {
		_ = a.dbManager.Close()
	}
}

// ==========================================
// Wails Frontend Bindings
// ==========================================

// ProcessSwipe 磁気カード・キーボード入力・画面入力からの打刻処理
func (a *App) ProcessSwipe(cardID string) *service.SwipeResponse {
	return a.gateService.ProcessSwipe(cardID)
}

// VerifyPIN 管理者PINコードの検証
func (a *App) VerifyPIN(pin string) bool {
	return pin == a.adminPin
}

// GetAvailableFiscalYears 利用可能な年度一覧（降順）を取得
func (a *App) GetAvailableFiscalYears() ([]int, error) {
	return a.dbManager.GetAvailableFiscalYears()
}

// GetCurrentFiscalYear 現在の年度を取得
func (a *App) GetCurrentFiscalYear() int {
	return db.CurrentFiscalYear()
}

// GetDashboardStats ダッシュボード統計データの取得 (年度指定対応)
func (a *App) GetDashboardStats(fiscalYear int) (*db.DashboardStats, error) {
	return a.dbManager.GetDashboardStats(fiscalYear)
}

// GetMonthlyStats 月別・区分別統計データの取得 (01prompt, 19prompt要件、年度指定対応)
func (a *App) GetMonthlyStats(fiscalYear int) (*db.MonthlyStatsResponse, error) {
	return a.dbManager.GetMonthlyStats(fiscalYear)
}

// GetAllUsers 登録ユーザー一覧の取得 (年度指定対応)
func (a *App) GetAllUsers(fiscalYear int) ([]db.RegisteredUser, error) {
	return a.dbManager.GetAllUsers(fiscalYear)
}

// FindUserInAnyFiscalYear 過去年度を含め最新の登録情報を検索 (更新モーダル用)
func (a *App) FindUserInAnyFiscalYear(identifier string) (*db.RegisteredUser, error) {
	return a.dbManager.FindUserInAnyFiscalYear(identifier)
}

// SaveUser ユーザーの登録・更新
func (a *App) SaveUser(user db.RegisteredUser) error {
	return a.dbManager.UpsertUser(&user)
}

// DeleteUser ユーザーの削除 (年度指定対応)
func (a *App) DeleteUser(cardID string, fiscalYear int) error {
	return a.dbManager.DeleteUser(cardID, fiscalYear)
}

// GetRecentLogs 最新の入退室ログを取得 (年度指定対応)
func (a *App) GetRecentLogs(limit int, fiscalYear int) ([]db.AccessLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.dbManager.GetRecentLogs(limit, fiscalYear)
}

// GetCurrentInsideUsers 現在在室中ユーザー一覧の取得
func (a *App) GetCurrentInsideUsers() ([]db.UserStatus, error) {
	return a.dbManager.GetCurrentInsideUsers()
}

// ClearAllLogs 全入退室ログの削除（高度な操作）
func (a *App) ClearAllLogs() error {
	return a.dbManager.ClearAllLogs()
}

// ForceExitAllInside 在室中の全利用者を即時一括強制退室（高度な操作）
func (a *App) ForceExitAllInside() (int, error) {
	return a.gateService.ForceExitAllInside()
}

// ExportFiscalYearLogsCSV 指定年度のCSVデータを生成して保存ダイアログを表示
func (a *App) ExportFiscalYearLogsCSV(fiscalYear int) (string, error) {
	csvData, err := a.gateService.GenerateFiscalYearCSV(fiscalYear)
	if err != nil {
		return "", err
	}

	defaultFilename := fmt.Sprintf("access_logs_%d_fiscal_year.csv", fiscalYear)
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           fmt.Sprintf("%d年度 入退室ログCSVの保存", fiscalYear),
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if savePath == "" {
		return "", nil // キャンセルされた場合
	}

	if err := os.WriteFile(savePath, csvData, 0644); err != nil {
		return "", fmt.Errorf("failed to write CSV file: %w", err)
	}

	return savePath, nil
}

// SaveCSVContent 任意のCSV文字列をファイル保存ダイアログ経由でUTF-8(BOM付き)保存する
func (a *App) SaveCSVContent(defaultFilename string, content string) (string, error) {
	if strings.TrimSpace(defaultFilename) == "" {
		defaultFilename = fmt.Sprintf("access_history_%s.csv", time.Now().Format("2006-01-02"))
	}

	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultFilename,
		Title:           "CSVファイルの保存",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if savePath == "" {
		return "", nil // キャンセルされた場合
	}

	// UTF-8 BOM (\xEF\xBB\xBF) が先頭になければ付与する
	const utf8BOM = "\xEF\xBB\xBF"
	if !strings.HasPrefix(content, utf8BOM) {
		content = utf8BOM + content
	}

	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write CSV file: %w", err)
	}

	return savePath, nil
}

// ImportUsersCSV 利用者マスターCSVファイルの選択・インポート
func (a *App) ImportUsersCSV() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "利用者登録CSVファイルの選択",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil // キャンセル
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルの読み込みに失敗しました: %w", err)
	}

	imported, total, err := a.gateService.ImportUsersCSV(data)
	if err != nil {
		return "", fmt.Errorf("インポート中にエラーが発生しました: %w", err)
	}

	return fmt.Sprintf("利用者CSVのインポートが完了しました（%d 件登録 / 対象 %d 件）", imported, total), nil
}

// ImportLogsCSV 入退室ログCSVファイルの選択・インポート
func (a *App) ImportLogsCSV() (string, error) {
	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "入退室ログCSVファイルの選択",
		Filters: []runtime.FileFilter{
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
	if err != nil {
		return "", err
	}
	if filePath == "" {
		return "", nil // キャンセル
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルの読み込みに失敗しました: %w", err)
	}

	imported, total, err := a.gateService.ImportLogsCSV(data)
	if err != nil {
		return "", fmt.Errorf("インポート中にエラーが発生しました: %w", err)
	}

	return fmt.Sprintf("入退室ログCSVのインポートが完了しました（%d 件インポート / 対象 %d 件）", imported, total), nil
}

// ToggleFullscreen フルスクリーン表示の切り替え
func (a *App) ToggleFullscreen() bool {
	if a.ctx == nil {
		return false
	}
	if runtime.WindowIsFullscreen(a.ctx) {
		runtime.WindowUnfullscreen(a.ctx)
		return false
	}
	runtime.WindowFullscreen(a.ctx)
	return true
}

// GetAppVersion アプリケーションのバージョン文字列を取得
func (a *App) GetAppVersion() string {
	return a.appVersion
}

// CheckUpdate 最新バージョンの有無を確認
func (a *App) CheckUpdate() (*updater.ReleaseInfo, error) {
	if a.updater == nil {
		return nil, fmt.Errorf("updater is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.updater.CheckForUpdate(ctx)
}

// ApplyUpdate 最新バージョンへの更新を適用して再起動
func (a *App) ApplyUpdate() error {
	if a.updater == nil {
		return fmt.Errorf("updater is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	info, err := a.updater.CheckForUpdate(ctx)
	if err != nil {
		return fmt.Errorf("failed to check release before updating: %w", err)
	}
	return a.updater.ApplyUpdate(ctx, info)
}
