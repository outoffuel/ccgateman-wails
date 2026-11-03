package db

import (
	"fmt"
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
	if stats.TodayLogCount != 2 || stats.TodayEntryCount != 1 || stats.TodayExitCount != 1 {
		t.Errorf("Expected TodayLogCount=2 (1 entry, 1 exit), got total=%d, entry=%d, exit=%d",
			stats.TodayLogCount, stats.TodayEntryCount, stats.TodayExitCount)
	}
	if stats.CurrentInsideCount != 0 {
		t.Errorf("Expected CurrentInsideCount=0, got %d", stats.CurrentInsideCount)
	}
}

func TestDashboardAndMonthlyStats_Over500AndFiscalYear(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "osuccgate_stats_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_stats.db")
	mgr, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer mgr.Close()

	currentFY := CurrentFiscalYear()
	pastFY := currentFY - 1

	// 500件制限を超える大量ログ（今年度 600件の入室 + 550件の退室 = 1150件、前年度 100件の入室 + 40件の退室 = 140件）を投入
	var bulkLogs []AccessLog
	now := time.Now()

	// 今年度：今日の日付で 600 entry + 550 exit (合計1150件 > 500件)
	for i := 0; i < 600; i++ {
		bulkLogs = append(bulkLogs, AccessLog{
			CardID:    fmt.Sprintf("ST_%04d", i),
			UserName:  "学生A",
			RoleName:  "学生",
			RoleCode:  1,
			StudentNo: fmt.Sprintf("B%04d", i),
			EventType: "entry",
			Timestamp: now.Add(-time.Duration(i) * time.Second),
		})
	}
	for i := 0; i < 550; i++ {
		ev := "exit"
		if i%10 == 0 {
			ev = "force_exit"
		}
		bulkLogs = append(bulkLogs, AccessLog{
			CardID:    fmt.Sprintf("ST_%04d", i),
			UserName:  "学生A",
			RoleName:  "学生",
			RoleCode:  1,
			StudentNo: fmt.Sprintf("B%04d", i),
			EventType: ev,
			Timestamp: now.Add(-time.Duration(i) * time.Second),
		})
	}

	// 前年度（pastFY年5月15日）：入室100件、退室40件（0.95推計なら95件になるが、実数は40件であること）
	pastDate := time.Date(pastFY, 5, 15, 12, 0, 0, 0, time.Local)
	for i := 0; i < 100; i++ {
		bulkLogs = append(bulkLogs, AccessLog{
			CardID:    "PAST_ST",
			UserName:  "過年度学生",
			RoleName:  "学生",
			RoleCode:  1,
			StudentNo: "B0001",
			EventType: "entry",
			Timestamp: pastDate.Add(time.Duration(i) * time.Second),
		})
	}
	for i := 0; i < 40; i++ {
		bulkLogs = append(bulkLogs, AccessLog{
			CardID:    "PAST_ST",
			UserName:  "過年度学生",
			RoleName:  "学生",
			RoleCode:  1,
			StudentNo: "B0001",
			EventType: "exit",
			Timestamp: pastDate.Add(time.Duration(i+200) * time.Second),
		})
	}

	if _, err := mgr.ImportLogs(bulkLogs); err != nil {
		t.Fatalf("ImportLogs failed: %v", err)
	}

	// 1. 今年度の統計検証 (500件制限を受けず1150件すべてが集計されること)
	curStats, err := mgr.GetDashboardStats(currentFY)
	if err != nil {
		t.Fatalf("GetDashboardStats(currentFY) failed: %v", err)
	}
	if curStats.WeekEntryCount != 600 || curStats.WeekExitCount != 550 || curStats.WeekLogCount != 1150 {
		t.Errorf("Expected week counts 600/550/1150, got %d/%d/%d",
			curStats.WeekEntryCount, curStats.WeekExitCount, curStats.WeekLogCount)
	}
	if curStats.YearEntryCount != 600 || curStats.YearExitCount != 550 || curStats.YearLogCount != 1150 {
		t.Errorf("Expected currentFY year counts 600/550/1150, got %d/%d/%d",
			curStats.YearEntryCount, curStats.YearExitCount, curStats.YearLogCount)
	}

	// 2. 前年度の統計検証 (選択年度に連動し、架空の0.95推計ではなく実数の100/40が返ること)
	pastStats, err := mgr.GetDashboardStats(pastFY)
	if err != nil {
		t.Fatalf("GetDashboardStats(pastFY) failed: %v", err)
	}
	if pastStats.YearEntryCount != 100 || pastStats.YearExitCount != 40 || pastStats.YearLogCount != 140 {
		t.Errorf("Expected pastFY year counts 100/40/140, got %d/%d/%d",
			pastStats.YearEntryCount, pastStats.YearExitCount, pastStats.YearLogCount)
	}
	// 5月 (index 1: 4月=0, 5月=1) のグラフ値が Entries=100, Exits=40 であること
	if len(pastStats.YearChart) != 12 {
		t.Fatalf("Expected 12 months in YearChart, got %d", len(pastStats.YearChart))
	}
	mayPoint := pastStats.YearChart[1]
	if mayPoint.Label != "5月" || mayPoint.Entries != 100 || mayPoint.Exits != 40 {
		t.Errorf("Expected May chart point {Label:5月, Entries:100, Exits:40}, got %+v", mayPoint)
	}

	// 3. 月別統計テーブルで学生 (role_code=1) が「その他」に誤分類されず Role1Count に集計されること
	pastMonthly, err := mgr.GetMonthlyStats(pastFY)
	if err != nil {
		t.Fatalf("GetMonthlyStats(pastFY) failed: %v", err)
	}
	if len(pastMonthly.Rows) != 1 {
		t.Fatalf("Expected 1 monthly row for pastFY, got %d", len(pastMonthly.Rows))
	}
	row := pastMonthly.Rows[0]
	if row.Role1Count != 100 || row.RoleOtherCount != 0 || row.MonthlyTotal != 100 || row.MonthlyExitCount != 40 || row.MonthlyLogTotal != 140 {
		t.Errorf("Unexpected monthly stat row: %+v", row)
	}
}
