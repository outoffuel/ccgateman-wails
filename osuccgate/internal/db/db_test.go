package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDBManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "osuccgate_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	mgr, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer mgr.Close()

	// 1. ユーザー登録
	user := &RegisteredUser{
		CardID:    "CARD123456",
		Name:      "テスト 太郎",
		RoleName:  "学生",
		RoleCode:  1,
		StudentNo: "B2026001",
	}
	if err := mgr.UpsertUser(user); err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	// 2. ユーザー取得検証
	fetched, err := mgr.GetUser("CARD123456")
	if err != nil || fetched == nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if fetched.Name != "テスト 太郎" || fetched.StudentNo != "B2026001" {
		t.Errorf("Unexpected user data: %+v", fetched)
	}

	// 3. 打刻処理 (1回目: 入室)
	log1, err := mgr.RecordSwipe(user)
	if err != nil {
		t.Fatalf("RecordSwipe 1 failed: %v", err)
	}
	if log1.EventType != "entry" {
		t.Errorf("Expected eventType 'entry', got '%s'", log1.EventType)
	}

	// 4. 打刻処理 (2回目: 退室)
	time.Sleep(10 * time.Millisecond)
	log2, err := mgr.RecordSwipe(user)
	if err != nil {
		t.Fatalf("RecordSwipe 2 failed: %v", err)
	}
	if log2.EventType != "exit" {
		t.Errorf("Expected eventType 'exit', got '%s'", log2.EventType)
	}

	// 5. 統計取得
	stats, err := mgr.GetDashboardStats()
	if err != nil {
		t.Fatalf("GetDashboardStats failed: %v", err)
	}
	if stats.TotalUserCount != 1 {
		t.Errorf("Expected TotalUserCount=1, got %d", stats.TotalUserCount)
	}
	if stats.TodayLogCount != 2 {
		t.Errorf("Expected TodayLogCount=2, got %d", stats.TodayLogCount)
	}
	if stats.CurrentInsideCount != 0 {
		t.Errorf("Expected CurrentInsideCount=0, got %d", stats.CurrentInsideCount)
	}
}

func TestGetMonthlyStats(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "osuccgate_monthly_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "monthly_test.db")
	mgr, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer mgr.Close()

	// ユーザー作成
	teacher := &RegisteredUser{CardID: "CARD_TEACHER", Name: "教員1", RoleName: "教職員", RoleCode: 0, StudentNo: "T001"}
	student := &RegisteredUser{CardID: "CARD_STUDENT", Name: "学生1", RoleName: "学生", RoleCode: 1, StudentNo: "S001"}
	staff := &RegisteredUser{CardID: "CARD_STAFF", Name: "スタッフ1", RoleName: "学生スタッフ", RoleCode: 9, StudentNo: "ST001"}

	_ = mgr.UpsertUser(teacher)
	_ = mgr.UpsertUser(student)
	_ = mgr.UpsertUser(staff)

	// 過去のログを直接インポート
	logs := []AccessLog{
		{CardID: "CARD_TEACHER", EventType: "entry", Timestamp: time.Date(2026, 6, 15, 10, 0, 0, 0, time.Local)},
		{CardID: "CARD_STUDENT", EventType: "entry", Timestamp: time.Date(2026, 6, 16, 11, 0, 0, 0, time.Local)},
		{CardID: "CARD_STAFF", EventType: "entry", Timestamp: time.Date(2026, 7, 1, 9, 0, 0, 0, time.Local)},
		{CardID: "CARD_STUDENT", EventType: "entry", Timestamp: time.Date(2026, 7, 2, 14, 0, 0, 0, time.Local)},
		{CardID: "CARD_STUDENT", EventType: "entry", Timestamp: time.Date(2026, 8, 10, 15, 0, 0, 0, time.Local)},
	}
	_, err = mgr.ImportLogs(logs)
	if err != nil {
		t.Fatalf("ImportLogs failed: %v", err)
	}

	monthlyResp, err := mgr.GetMonthlyStats()
	if err != nil {
		t.Fatalf("GetMonthlyStats failed: %v", err)
	}

	if len(monthlyResp.Rows) != 3 {
		t.Fatalf("Expected 3 monthly stat rows, got %d", len(monthlyResp.Rows))
	}

	// 降順ソートチェック (2026-08, 2026-07, 2026-06)
	if monthlyResp.Rows[0].YearMonth != "2026-08" {
		t.Errorf("Expected first row to be 2026-08, got %s", monthlyResp.Rows[0].YearMonth)
	}
	if monthlyResp.Rows[1].YearMonth != "2026-07" {
		t.Errorf("Expected second row to be 2026-07, got %s", monthlyResp.Rows[1].YearMonth)
	}
	if monthlyResp.Rows[2].YearMonth != "2026-06" {
		t.Errorf("Expected third row to be 2026-06, got %s", monthlyResp.Rows[2].YearMonth)
	}

	// 2026-06 行検証: 教員1, 学生1 -> 月計 2, 4月〜6月累計 2
	row06 := monthlyResp.Rows[2]
	if row06.Role0Count != 1 || row06.Role1Count != 1 || row06.MonthlyTotal != 2 {
		t.Errorf("Unexpected 2026-06 row totals: %+v", row06)
	}
	if row06.FiscalYearCumulativeTotal != 2 {
		t.Errorf("Expected 2026-06 cumulative=2, got %d", row06.FiscalYearCumulativeTotal)
	}

	// 2026-07 行検証: スタッフ1, 学生1 -> 月計 2, 4月〜7月累計 4
	row07 := monthlyResp.Rows[1]
	if row07.Role9Count != 1 || row07.Role1Count != 1 || row07.MonthlyTotal != 2 {
		t.Errorf("Unexpected 2026-07 row totals: %+v", row07)
	}
	if row07.FiscalYearCumulativeTotal != 4 {
		t.Errorf("Expected 2026-07 cumulative=4, got %d", row07.FiscalYearCumulativeTotal)
	}

	// 2026-08 行検証: 学生1 -> 月計 1, 4月〜8月累計 5
	row08 := monthlyResp.Rows[0]
	if row08.Role1Count != 1 || row08.MonthlyTotal != 1 {
		t.Errorf("Unexpected 2026-08 row totals: %+v", row08)
	}
	if row08.FiscalYearCumulativeTotal != 5 {
		t.Errorf("Expected 2026-08 cumulative=5, got %d", row08.FiscalYearCumulativeTotal)
	}
}

