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
