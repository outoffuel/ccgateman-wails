package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"osuccgate/internal/db"
	"osuccgate/internal/nfc"
	"osuccgate/internal/server"
	"osuccgate/internal/service"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx         context.Context
	dbManager   *db.DBManager
	gateService *service.GateService
	nfcReader   *nfc.NFCReader
	httpServer  *server.HTTPServer
	adminPin    string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		adminPin: "1234", // 管理者PINコード
	}
}

// startup is called at application startup
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// データベース保存パスの決定 (実行ディレクトリまたはAppData)
	dbPath := "osuccgate.db"
	execDir, err := os.Getwd()
	if err == nil {
		dbPath = filepath.Join(execDir, "osuccgate.db")
	}

	// SQLite初期化
	dbMgr, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	a.dbManager = dbMgr

	// ゲートサービス初期化
	a.gateService = service.NewGateService(dbMgr)

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
	a.httpServer = server.NewHTTPServer(a.dbManager, a.gateService, 8080, a.adminPin)
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

// GetDashboardStats ダッシュボード統計データの取得
func (a *App) GetDashboardStats() (*db.DashboardStats, error) {
	return a.dbManager.GetDashboardStats()
}

// GetAllUsers 登録ユーザー一覧の取得
func (a *App) GetAllUsers() ([]db.RegisteredUser, error) {
	return a.dbManager.GetAllUsers()
}

// SaveUser ユーザーの登録・更新
func (a *App) SaveUser(user db.RegisteredUser) error {
	return a.dbManager.UpsertUser(&user)
}

// DeleteUser ユーザーの削除
func (a *App) DeleteUser(cardID string) error {
	return a.dbManager.DeleteUser(cardID)
}

// GetRecentLogs 最新の入退室ログを取得
func (a *App) GetRecentLogs(limit int) ([]db.AccessLog, error) {
	if limit <= 0 {
		limit = 50
	}
	return a.dbManager.GetRecentLogs(limit)
}

// GetCurrentInsideUsers 現在在室中ユーザー一覧の取得
func (a *App) GetCurrentInsideUsers() ([]db.UserStatus, error) {
	return a.dbManager.GetCurrentInsideUsers()
}

// ClearAllLogs 全入退室ログの削除（高度な操作）
func (a *App) ClearAllLogs() error {
	return a.dbManager.ClearAllLogs()
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
