package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// RegisteredUser 登録利用者
type RegisteredUser struct {
	CardID    string    `json:"cardId"`    // 磁気カード番号 / NFC IDm (空の場合は学籍番号と同一)
	Name      string    `json:"name"`      // 氏名
	Furigana  string    `json:"furigana"`  // フリガナ
	Gender    string    `json:"gender"`    // 性別 ("男", "女" 等)
	RoleName  string    `json:"roleName"`  // 教職員, 学生, 学生スタッフ 等
	RoleCode  int       `json:"roleCode"`  // 0: 教職員, 1: 学生, 9: 学生スタッフ
	StudentNo string    `json:"studentNo"` // 学籍番号/職員番号 (必須)
	AdminNo   string    `json:"adminNo"`   // 管理番号 (自動採番 / CSV指定)
	Purpose   string    `json:"purpose"`   // 利用目的
	Contact   string    `json:"contact"`   // 連絡先 (電話番号/メール等)
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AccessLog 入退室ログ
type AccessLog struct {
	ID             int64     `json:"id"`
	CardID         string    `json:"cardId"`
	UserName       string    `json:"userName"`
	RoleName       string    `json:"roleName"`
	RoleCode       int       `json:"roleCode"`
	StudentNo      string    `json:"studentNo"`
	EventType      string    `json:"eventType"` // "entry" (入室), "exit" (退室), "force_exit" (強制退室)
	Timestamp      time.Time `json:"timestamp"`
	DurationSecond int64     `json:"durationSecond"` // 退室・強制退室時に算出 (秒単位)
	DurationText   string    `json:"durationText"`   // "1時間23分" 等
}

// UserStatus 在室状態・最新ログ
type UserStatus struct {
	CardID        string    `json:"cardId"`
	UserName      string    `json:"userName"`
	RoleName      string    `json:"roleName"`
	StudentNo     string    `json:"studentNo"`
	CurrentStatus string    `json:"currentStatus"` // "inside" (在室中) or "outside" (退室済)
	LastEventTime time.Time `json:"lastEventTime"`
	StayDuration  string    `json:"stayDuration"` // 在室中の場合は現在までの滞在時間
}

// DashboardStats ダッシュボード統計
type DashboardStats struct {
	CurrentInsideCount int `json:"currentInsideCount"` // 現在の在室者数
	TotalUserCount     int `json:"totalUserCount"`     // 登録総数
	TodayLogCount      int `json:"todayLogCount"`      // 本日の総打刻数
}

type DBManager struct {
	db *sql.DB
	mu sync.RWMutex
}

// InitDB SQLiteデータベースの初期化・テーブル作成
func InitDB(dbPath string) (*DBManager, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil && dir != "." {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// コネクションプール設定 (書き込み競合防止)
	db.SetMaxOpenConns(1)

	mgr := &DBManager{db: db}
	if err := mgr.createTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	return mgr, nil
}

func (m *DBManager) Close() error {
	return m.db.Close()
}

func (m *DBManager) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS registered_users (
		card_id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		furigana TEXT NOT NULL DEFAULT '',
		gender TEXT NOT NULL DEFAULT '',
		role_name TEXT NOT NULL,
		role_code INTEGER NOT NULL DEFAULT 1, -- 0:教職員, 1:学生, 9:学生スタッフ
		student_no TEXT NOT NULL DEFAULT '',
		admin_no TEXT NOT NULL DEFAULT '',
		purpose TEXT NOT NULL DEFAULT '',
		contact TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS access_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id TEXT NOT NULL,
		event_type TEXT NOT NULL, -- 'entry', 'exit', 'force_exit'
		timestamp DATETIME NOT NULL,
		duration_second INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY(card_id) REFERENCES registered_users(card_id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_access_logs_card_id ON access_logs(card_id);
	CREATE INDEX IF NOT EXISTS idx_access_logs_timestamp ON access_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_registered_users_student_no ON registered_users(student_no);
	`
	if _, err := m.db.Exec(schema); err != nil {
		return err
	}

	// 既存DB用カラムマイグレーション
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN furigana TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN gender TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN admin_no TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN purpose TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN contact TEXT NOT NULL DEFAULT ''`)
	return nil
}

// GetUser 登録ユーザー取得 (card_id または student_no のどちらでも検索可能)
func (m *DBManager) GetUser(identifier string) (*RegisteredUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `SELECT card_id, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at FROM registered_users WHERE card_id = ? OR (student_no != '' AND student_no = ?) LIMIT 1`
	row := m.db.QueryRow(query, identifier, identifier)

	var u RegisteredUser
	var createdAtStr, updatedAtStr string
	err := row.Scan(&u.CardID, &u.Name, &u.Furigana, &u.Gender, &u.RoleName, &u.RoleCode, &u.StudentNo, &u.AdminNo, &u.Purpose, &u.Contact, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	if u.CreatedAt.IsZero() {
		u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
	}
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
	if u.UpdatedAt.IsZero() {
		u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
	}
	return &u, nil
}

// GetAllUsers 全ユーザー一覧取得
func (m *DBManager) GetAllUsers() ([]RegisteredUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `SELECT card_id, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at FROM registered_users ORDER BY role_code ASC, student_no ASC, name ASC`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []RegisteredUser
	for rows.Next() {
		var u RegisteredUser
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&u.CardID, &u.Name, &u.Furigana, &u.Gender, &u.RoleName, &u.RoleCode, &u.StudentNo, &u.AdminNo, &u.Purpose, &u.Contact, &createdAtStr, &updatedAtStr); err != nil {
			return nil, err
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		if u.CreatedAt.IsZero() {
			u.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAtStr)
		}
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		if u.UpdatedAt.IsZero() {
			u.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAtStr)
		}
		users = append(users, u)
	}
	return users, nil
}

// GetMaxAdminNo 既存ユーザーに設定されている管理番号の最大整数値を取得 (自動採番用)
func (m *DBManager) GetMaxAdminNo() (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rows, err := m.db.Query("SELECT admin_no FROM registered_users WHERE admin_no != ''")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	maxNo := 0
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			s = strings.TrimSpace(s)
			if n, err := strconv.Atoi(s); err == nil {
				if n > maxNo {
					maxNo = n
				}
			}
		}
	}
	return maxNo, nil
}

// UpsertUser ユーザー登録・更新
func (m *DBManager) UpsertUser(u *RegisteredUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u.CardID == "" {
		u.CardID = u.StudentNo
	}

	now := time.Now()
	query := `
	INSERT INTO registered_users (card_id, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(card_id) DO UPDATE SET
		name = excluded.name,
		furigana = excluded.furigana,
		gender = excluded.gender,
		role_name = excluded.role_name,
		role_code = excluded.role_code,
		student_no = excluded.student_no,
		admin_no = CASE WHEN excluded.admin_no != '' THEN excluded.admin_no ELSE registered_users.admin_no END,
		purpose = excluded.purpose,
		contact = excluded.contact,
		updated_at = excluded.updated_at
	`
	_, err := m.db.Exec(query, u.CardID, u.Name, u.Furigana, u.Gender, u.RoleName, u.RoleCode, u.StudentNo, u.AdminNo, u.Purpose, u.Contact, now.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

// ImportUsers 複数ユーザーの一括インポート・Upsert
func (m *DBManager) ImportUsers(users []RegisteredUser) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO registered_users (card_id, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(card_id) DO UPDATE SET
		name = excluded.name,
		furigana = excluded.furigana,
		gender = excluded.gender,
		role_name = excluded.role_name,
		role_code = excluded.role_code,
		student_no = excluded.student_no,
		admin_no = CASE WHEN excluded.admin_no != '' THEN excluded.admin_no ELSE registered_users.admin_no END,
		purpose = excluded.purpose,
		contact = excluded.contact,
		updated_at = excluded.updated_at
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	now := time.Now()
	count := 0
	for _, u := range users {
		cardID := u.CardID
		if cardID == "" {
			cardID = u.StudentNo
		}
		if cardID == "" {
			continue
		}
		createdAt := u.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		updatedAt := u.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		_, err := stmt.Exec(cardID, u.Name, u.Furigana, u.Gender, u.RoleName, u.RoleCode, u.StudentNo, u.AdminNo, u.Purpose, u.Contact, createdAt.Format(time.RFC3339), updatedAt.Format(time.RFC3339))
		if err != nil {
			return count, err
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

// ImportLogs 過去の入退室ログ一括インポート
func (m *DBManager) ImportLogs(logs []AccessLog) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO access_logs (card_id, event_type, timestamp, duration_second)
	VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, l := range logs {
		if l.CardID == "" {
			continue
		}
		ts := l.Timestamp
		if ts.IsZero() {
			ts = time.Now()
		}
		_, err := stmt.Exec(l.CardID, l.EventType, ts.Format(time.RFC3339), l.DurationSecond)
		if err != nil {
			return count, err
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

// DeleteUser ユーザー削除
func (m *DBManager) DeleteUser(cardID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec("DELETE FROM registered_users WHERE card_id = ? OR student_no = ?", cardID, cardID)
	return err
}

// RecordSwipe 打刻処理 (入退室判定＆ログ記録)
func (m *DBManager) RecordSwipe(user *RegisteredUser) (*AccessLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// 直近の打刻ログを確認して、奇数/偶数 (トグル) 判定
	var lastEventType string
	var lastTsStr string
	var lastID int64
	queryLast := `SELECT id, event_type, timestamp FROM access_logs WHERE card_id = ? ORDER BY id DESC LIMIT 1`
	row := m.db.QueryRow(queryLast, user.CardID)
	err := row.Scan(&lastID, &lastEventType, &lastTsStr)

	eventType := "entry" // デフォルトは入室 (1回目/奇数回、または前回退室/強制退室時)
	var durationSec int64 = 0

	if err == nil {
		if lastEventType == "entry" {
			// 前回入室 -> 今回は退室
			eventType = "exit"
			lastTime, _ := time.Parse(time.RFC3339, lastTsStr)
			if lastTime.IsZero() {
				lastTime, _ = time.Parse("2006-01-02 15:04:05", lastTsStr)
			}
			durationSec = int64(now.Sub(lastTime).Seconds())
			if durationSec < 0 {
				durationSec = 0
			}
		} else {
			// 前回 exit または force_exit -> 今回は入室
			eventType = "entry"
		}
	}

	// ログ書き込み
	insertQuery := `INSERT INTO access_logs (card_id, event_type, timestamp, duration_second) VALUES (?, ?, ?, ?)`
	res, err := m.db.Exec(insertQuery, user.CardID, eventType, now.Format(time.RFC3339), durationSec)
	if err != nil {
		return nil, err
	}

	newID, _ := res.LastInsertId()

	durationText := ""
	if (eventType == "exit" || eventType == "force_exit") && durationSec > 0 {
		durationText = FormatDuration(durationSec)
	}

	return &AccessLog{
		ID:             newID,
		CardID:         user.CardID,
		UserName:       user.Name,
		RoleName:       user.RoleName,
		RoleCode:       user.RoleCode,
		StudentNo:      user.StudentNo,
		EventType:      eventType,
		Timestamp:      now,
		DurationSecond: durationSec,
		DurationText:   durationText,
	}, nil
}

// ForceExitAllInsideUsers 23:00等に在室中の全ユーザーを一括強制退室処理する
func (m *DBManager) ForceExitAllInsideUsers(exitTime time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 現在在室中（最新ログが 'entry'）のカード一覧と入室日時を取得
	query := `
	SELECT 
		l.card_id, l.timestamp
	FROM access_logs l
	INNER JOIN (
		SELECT card_id, MAX(id) as max_id FROM access_logs GROUP BY card_id
	) latest ON l.id = latest.max_id
	WHERE l.event_type = 'entry'
	`
	rows, err := m.db.Query(query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type insideEntry struct {
		cardID    string
		entryTime time.Time
	}
	var targets []insideEntry

	for rows.Next() {
		var cardID, tsStr string
		if err := rows.Scan(&cardID, &tsStr); err != nil {
			return 0, err
		}
		t, _ := time.Parse(time.RFC3339, tsStr)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		}
		targets = append(targets, insideEntry{cardID: cardID, entryTime: t})
	}

	if len(targets) == 0 {
		return 0, nil
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	insertStmt, err := tx.Prepare(`INSERT INTO access_logs (card_id, event_type, timestamp, duration_second) VALUES (?, 'force_exit', ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer insertStmt.Close()

	for _, item := range targets {
		durationSec := int64(exitTime.Sub(item.entryTime).Seconds())
		if durationSec < 0 {
			durationSec = 0
		}
		_, err := insertStmt.Exec(item.cardID, exitTime.Format(time.RFC3339), durationSec)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(targets), nil
}

// GetRecentLogs 直近の打刻ログ一覧取得 (limit件)
func (m *DBManager) GetRecentLogs(limit int) ([]AccessLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
	SELECT 
		l.id, l.card_id, u.name, u.role_name, u.role_code, u.student_no,
		l.event_type, l.timestamp, l.duration_second
	FROM access_logs l
	LEFT JOIN registered_users u ON l.card_id = u.card_id
	ORDER BY l.id DESC
	LIMIT ?
	`
	rows, err := m.db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var l AccessLog
		var tsStr string
		var name, roleName, studentNo sql.NullString
		var roleCode sql.NullInt64

		if err := rows.Scan(&l.ID, &l.CardID, &name, &roleName, &roleCode, &studentNo, &l.EventType, &tsStr, &l.DurationSecond); err != nil {
			return nil, err
		}
		l.UserName = name.String
		l.RoleName = roleName.String
		l.RoleCode = int(roleCode.Int64)
		l.StudentNo = studentNo.String
		l.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		if l.Timestamp.IsZero() {
			l.Timestamp, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		}
		if (l.EventType == "exit" || l.EventType == "force_exit") && l.DurationSecond > 0 {
			l.DurationText = FormatDuration(l.DurationSecond)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// GetDashboardStats ダッシュボード用集計データ
func (m *DBManager) GetDashboardStats() (*DashboardStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := &DashboardStats{}

	// 1. ユーザー登録総数
	err := m.db.QueryRow("SELECT COUNT(*) FROM registered_users").Scan(&stats.TotalUserCount)
	if err != nil {
		return nil, err
	}

	// 2. 本日の総打刻数 (00:00:00以降)
	todayStart := time.Now().Format("2006-01-02") + "T00:00:00"
	err = m.db.QueryRow("SELECT COUNT(*) FROM access_logs WHERE timestamp >= ?", todayStart).Scan(&stats.TodayLogCount)
	if err != nil {
		todayDate := time.Now().Format("2006-01-02")
		_ = m.db.QueryRow("SELECT COUNT(*) FROM access_logs WHERE timestamp LIKE ?", todayDate+"%").Scan(&stats.TodayLogCount)
	}

	// 3. 現在の在室者数 (各カードの最新ログが 'entry' である件数)
	queryInside := `
	SELECT COUNT(*) FROM (
		SELECT card_id, event_type FROM access_logs
		WHERE id IN (
			SELECT MAX(id) FROM access_logs GROUP BY card_id
		) AND event_type = 'entry'
	)
	`
	_ = m.db.QueryRow(queryInside).Scan(&stats.CurrentInsideCount)

	return stats, nil
}

// GetCurrentInsideUsers 現在在室中のユーザー一覧
func (m *DBManager) GetCurrentInsideUsers() ([]UserStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
	SELECT 
		l.card_id, u.name, u.role_name, u.student_no, l.event_type, l.timestamp
	FROM access_logs l
	INNER JOIN (
		SELECT card_id, MAX(id) as max_id FROM access_logs GROUP BY card_id
	) latest ON l.id = latest.max_id
	LEFT JOIN registered_users u ON l.card_id = u.card_id
	WHERE l.event_type = 'entry'
	ORDER BY l.timestamp DESC
	`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	now := time.Now()
	var list []UserStatus
	for rows.Next() {
		var s UserStatus
		var name, roleName, studentNo sql.NullString
		var tsStr string
		if err := rows.Scan(&s.CardID, &name, &roleName, &studentNo, &s.CurrentStatus, &tsStr); err != nil {
			return nil, err
		}
		s.UserName = name.String
		s.RoleName = roleName.String
		s.StudentNo = studentNo.String
		s.LastEventTime, _ = time.Parse(time.RFC3339, tsStr)
		if s.LastEventTime.IsZero() {
			s.LastEventTime, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		}
		durationSec := int64(now.Sub(s.LastEventTime).Seconds())
		if durationSec > 0 {
			s.StayDuration = FormatDuration(durationSec)
		}
		s.CurrentStatus = "inside"
		list = append(list, s)
	}
	return list, nil
}

// ClearAllLogs 全入退室ログの削除 (高度な操作)
func (m *DBManager) ClearAllLogs() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.db.Exec("DELETE FROM access_logs")
	return err
}

// GetFiscalYearLogs 指定年度のログ取得 (4月1日〜翌年3月31日)
func (m *DBManager) GetFiscalYearLogs(fiscalYear int) ([]AccessLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	startDate := fmt.Sprintf("%d-04-01T00:00:00", fiscalYear)
	endDate := fmt.Sprintf("%d-03-31T23:59:59", fiscalYear+1)

	query := `
	SELECT 
		l.id, l.card_id, COALESCE(u.name, '未登録') as user_name,
		COALESCE(u.role_name, '未登録') as role_name,
		COALESCE(u.role_code, 1) as role_code,
		COALESCE(u.student_no, '') as student_no,
		l.event_type, l.timestamp, l.duration_second
	FROM access_logs l
	LEFT JOIN registered_users u ON l.card_id = u.card_id
	WHERE l.timestamp >= ? AND l.timestamp <= ?
	ORDER BY l.timestamp ASC
	`
	rows, err := m.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var l AccessLog
		var tsStr string
		if err := rows.Scan(&l.ID, &l.CardID, &l.UserName, &l.RoleName, &l.RoleCode, &l.StudentNo, &l.EventType, &tsStr, &l.DurationSecond); err != nil {
			return nil, err
		}
		l.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		if l.Timestamp.IsZero() {
			l.Timestamp, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		}
		if (l.EventType == "exit" || l.EventType == "force_exit") && l.DurationSecond > 0 {
			l.DurationText = FormatDuration(l.DurationSecond)
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// FormatDuration 秒数を "X時間Y分" または "X分Y秒" に変換
func FormatDuration(sec int64) string {
	if sec < 60 {
		return fmt.Sprintf("%d秒", sec)
	}
	minutes := sec / 60
	remainingSec := sec % 60
	if minutes < 60 {
		if remainingSec == 0 {
			return fmt.Sprintf("%d分", minutes)
		}
		return fmt.Sprintf("%d分%d秒", minutes, remainingSec)
	}
	hours := minutes / 60
	remainingMin := minutes % 60
	if remainingMin == 0 {
		return fmt.Sprintf("%d時間", hours)
	}
	return fmt.Sprintf("%d時間%d分", hours, remainingMin)
}
