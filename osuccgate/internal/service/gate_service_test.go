package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"osuccgate/internal/db"
)

func TestGateService_DebounceAndSwipe(t *testing.T) {
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

	// ユーザー登録
	_ = mgr.UpsertUser(&db.RegisteredUser{
		CardID:   "CARD999",
		Name:     "教員 先生",
		RoleName: "教職員",
		RoleCode: 9,
	})

	svc := NewGateService(mgr)

	// 1. 1回目の打刻 (入室・教員音)
	res1 := svc.ProcessSwipe("CARD999")
	if !res1.Success {
		t.Fatalf("Expected success, got error: %s", res1.ErrorMessage)
	}
	if res1.EventType != "entry" {
		t.Errorf("Expected 'entry', got '%s'", res1.EventType)
	}
	if res1.SoundType != "staffEntry" {
		t.Errorf("Expected SoundType 'staffEntry', got '%s'", res1.SoundType)
	}

	// 2. 直後の打刻（2秒デバウンスでブロックされること）
	res2 := svc.ProcessSwipe("CARD999")
	if !res2.IsDebounced {
		t.Errorf("Expected debounce to block swipe, but was not debounced")
	}

	// 3. 未登録カードの打刻
	resUnregistered := svc.ProcessSwipe("UNKNOWN_CARD")
	if resUnregistered.Success {
		t.Errorf("Expected unregistered card to fail")
	}
	if resUnregistered.SoundType != "booboo" {
		t.Errorf("Expected 'booboo' sound for error, got '%s'", resUnregistered.SoundType)
	}

	// 4. CSVエクスポート
	currentYear := time.Now().Year()
	csvBytes, err := svc.GenerateFiscalYearCSV(currentYear)
	if err != nil {
		t.Fatalf("GenerateFiscalYearCSV failed: %v", err)
	}
	csvContent := string(csvBytes)
	if !strings.Contains(csvContent, "教員 先生") {
		t.Errorf("CSV should contain user name '教員 先生'")
	}
}
