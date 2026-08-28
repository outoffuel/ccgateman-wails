package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"osuccgate/internal/db"
)

func TestGateService_RoleCodesAndForceExit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "osuccgate_svc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	mgr, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer mgr.Close()

	// 1. 教職員 (RoleCode: 0)
	_ = mgr.UpsertUser(&db.RegisteredUser{
		CardID:   "STAFF001",
		Name:     "教員 先生",
		RoleName: "教職員",
		RoleCode: 0,
	})

	// 2. 学生 (RoleCode: 1)
	_ = mgr.UpsertUser(&db.RegisteredUser{
		CardID:   "STUDENT001",
		Name:     "学生 太郎",
		RoleName: "学生",
		RoleCode: 1,
	})

	// 3. 学生スタッフ (RoleCode: 9)
	_ = mgr.UpsertUser(&db.RegisteredUser{
		CardID:   "STAFFSTUDENT001",
		Name:     "学生 スタッフ次郎",
		RoleName: "学生スタッフ",
		RoleCode: 9,
	})

	svc := NewGateService(mgr)

	// 教職員打刻テスト (RoleCode 0 -> staffEntry)
	resStaff := svc.ProcessSwipe("STAFF001")
	if !resStaff.Success || resStaff.SoundType != "staffEntry" {
		t.Errorf("Expected staffEntry for roleCode 0, got %s", resStaff.SoundType)
	}

	// 学生打刻テスト (RoleCode 1 -> studentEntry)
	resStudent := svc.ProcessSwipe("STUDENT001")
	if !resStudent.Success || resStudent.SoundType != "studentEntry" {
		t.Errorf("Expected studentEntry for roleCode 1, got %s", resStudent.SoundType)
	}

	// 学生スタッフ打刻テスト (RoleCode 9 -> studentEntry)
	resStaffStudent := svc.ProcessSwipe("STAFFSTUDENT001")
	if !resStaffStudent.Success || resStaffStudent.SoundType != "studentEntry" {
		t.Errorf("Expected studentEntry for roleCode 9, got %s", resStaffStudent.SoundType)
	}

	// 現在3人在室中であることを確認
	stats1, _ := mgr.GetDashboardStats()
	if stats1.CurrentInsideCount != 3 {
		t.Fatalf("Expected 3 inside users, got %d", stats1.CurrentInsideCount)
	}

	// 23:00 一括強制退室のテスト
	count, err := svc.ForceExitAllInside()
	if err != nil {
		t.Fatalf("ForceExitAllInside failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 users force-exited, got %d", count)
	}

	// 在室者が0人になったことを確認
	stats2, _ := mgr.GetDashboardStats()
	if stats2.CurrentInsideCount != 0 {
		t.Errorf("Expected 0 inside users after force exit, got %d", stats2.CurrentInsideCount)
	}

	// 直近ログ取得の確認 (最新が force_exit であること)
	recentLogs, err := mgr.GetRecentLogs(5)
	if err != nil || len(recentLogs) == 0 {
		t.Fatalf("Failed to get recent logs")
	}
	if recentLogs[0].EventType != "force_exit" {
		t.Errorf("Expected latest log eventType to be 'force_exit', got '%s'", recentLogs[0].EventType)
	}

	// CSVエクスポートで「システム自動」が出力されていることを確認
	currentYear := time.Now().Year()
	csvBytes, err := svc.GenerateFiscalYearCSV(currentYear)
	if err != nil {
		t.Fatalf("GenerateFiscalYearCSV failed: %v", err)
	}
	csvContent := string(csvBytes)
	if !strings.Contains(csvContent, "強制退室") || !strings.Contains(csvContent, "システム自動") {
		t.Errorf("CSV should contain '強制退室' and 'システム自動'")
	}
}

func TestGateService_CSVImportAndDualAuth(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "osuccgate_csv_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	mgr, err := db.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer mgr.Close()

	svc := NewGateService(mgr)

	// 1. 利用者CSVインポートテスト (BOM付き、カードID空、フリガナ、性別等)
	userCSV := "\xEF\xBB\xBF学籍番号,氏名,フリガナ,性別,区分,カードID\n" +
		"B2026001,山田 太郎,ヤマダ タロウ,男,学生,\n" +
		"B2026002,佐藤 花子,サトウ ハナコ,女,学生スタッフ,NFC002\n" +
		"T9999001,鈴木 一郎,スズキ イチロウ,男,教職員,MAG003\n"

	imported, total, err := svc.ImportUsersCSV([]byte(userCSV))
	if err != nil {
		t.Fatalf("ImportUsersCSV failed: %v", err)
	}
	if imported != 3 || total != 3 {
		t.Errorf("Expected 3 imported users, got imported=%d, total=%d", imported, total)
	}

	// 2. 登録内容の検証
	u1, err := mgr.GetUser("B2026001")
	if err != nil || u1 == nil {
		t.Fatalf("Failed to get user 1 by studentNo: %v", err)
	}
	if u1.CardID != "B2026001" || u1.Furigana != "ヤマダ タロウ" || u1.Gender != "男" || u1.RoleCode != 1 {
		t.Errorf("Unexpected user 1 data: %+v", u1)
	}

	// 3. 学籍番号とカードIDの双方からの認証照合テスト
	// u2: StudentNo="B2026002", CardID="NFC002"
	// (a) カードID "NFC002" で打刻
	resA := svc.ProcessSwipe("NFC002")
	if !resA.Success || resA.UserName != "佐藤 花子" || resA.EventType != "entry" {
		t.Errorf("Failed swipe by CardID: %+v", resA)
	}

	// (b) 学籍番号 "B2026002" で打刻 (退室になるはず)
	time.Sleep(10 * time.Millisecond)
	// デバウンスを回避するためにlastSwipesをリセットまたは別IDとして通過
	svc.lastSwipes = make(map[string]time.Time)
	resB := svc.ProcessSwipe("B2026002")
	if !resB.Success || resB.UserName != "佐藤 花子" || resB.EventType != "exit" {
		t.Errorf("Failed swipe by StudentNo: %+v", resB)
	}

	// 4. ログCSVインポートテスト
	logCSV := "打刻日時,識別ID,氏名,区分,入退室種別,入力方法,滞在時間(秒)\n" +
		"2026-04-01 09:00:00,B2026001,山田 太郎,学生,入室,カード読み取り,0\n" +
		"2026-04-01 10:30:00,B2026001,山田 太郎,学生,退室,カード読み取り,5400\n" +
		"2026-04-01 23:00:00,NFC002,佐藤 花子,学生スタッフ,強制退室,システム自動,3600\n"

	logImported, logTotal, err := svc.ImportLogsCSV([]byte(logCSV))
	if err != nil {
		t.Fatalf("ImportLogsCSV failed: %v", err)
	}
	if logImported != 3 || logTotal != 3 {
		t.Errorf("Expected 3 imported logs, got imported=%d, total=%d", logImported, logTotal)
	}
}
