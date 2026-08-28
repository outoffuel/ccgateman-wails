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
