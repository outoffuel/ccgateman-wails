package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// RegisteredUser 登録利用者
type RegisteredUser struct {
	CardID     string    `json:"cardId"`     // 磁気カード番号 / NFC IDm (空の場合は学籍番号と同一)
	FiscalYear int       `json:"fiscalYear"` // 年度 (例: 2026)
	Name       string    `json:"name"`       // 氏名
	Furigana   string    `json:"furigana"`   // フリガナ
	Gender     string    `json:"gender"`     // 性別 ("男", "女" 等)
	RoleName   string    `json:"roleName"`   // 教職員, 学生, 学生スタッフ 等
	RoleCode   int       `json:"roleCode"`   // 0: 教職員, 1: 学生, 9: 学生スタッフ
	StudentNo  string    `json:"studentNo"`  // 学籍番号/職員番号 (必須)
	AdminNo    string    `json:"adminNo"`    // 管理番号 (自動採番 / CSV指定)
	Purpose    string    `json:"purpose"`    // 利用目的
	Contact    string    `json:"contact"`    // 連絡先 (電話番号/メール等)
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// GetFiscalYear 指定日時の年度 (4月1日〜翌年3月31日) を算出
func GetFiscalYear(t time.Time) int {
	y := t.Year()
	if t.Month() < 4 {
		y--
	}
	return y
}

// CurrentFiscalYear 現在の年度 (4月1日〜翌年3月31日) を取得
func CurrentFiscalYear() int {
	return GetFiscalYear(time.Now())
}

// AccessLog 入退室ログ
type AccessLog struct {
	ID             int64     `json:"id"`
	CardID         string    `json:"cardId"`
	UserName       string    `json:"userName"`
	RoleName       string    `json:"roleName"`
	RoleCode       int       `json:"roleCode"`
	StudentNo      string    `json:"studentNo"`
	Purpose        string    `json:"purpose"`        // 利用目的
	EventType      string    `json:"eventType"`      // "entry" (入室), "exit" (退室), "force_exit" (強制退室)
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

// ChartPoint グラフ描画用の期間別入退室集計ポイント
type ChartPoint struct {
	Label   string `json:"label"`   // X軸ラベル ("11/3(火)", "4週前", "4月" 等)
	Entries int    `json:"entries"` // 入室数 (event_type = 'entry')
	Exits   int    `json:"exits"`   // 退室数 (event_type IN ('exit', 'force_exit'))
}

// DashboardStats ダッシュボード統計
type DashboardStats struct {
	FiscalYear         int          `json:"fiscalYear"`         // 集計対象年度
	CurrentInsideCount int          `json:"currentInsideCount"` // 現在の在室者数
	TotalUserCount     int          `json:"totalUserCount"`     // 登録総数
	TodayLogCount      int          `json:"todayLogCount"`      // 本日の総入退室数 (entry + exit + force_exit)
	TodayEntryCount    int          `json:"todayEntryCount"`    // 本日の入室数 (entry)
	TodayExitCount     int          `json:"todayExitCount"`     // 本日の退室数 (exit + force_exit)
	WeekLogCount       int          `json:"weekLogCount"`       // 週間の総入退室数 (直近7日間: entry + exit + force_exit)
	WeekEntryCount     int          `json:"weekEntryCount"`     // 週間の入室数 (直近7日間: entry)
	WeekExitCount      int          `json:"weekExitCount"`      // 週間の退室数 (直近7日間: exit + force_exit)
	MonthLogCount      int          `json:"monthLogCount"`      // 月間の総入退室数 (当月: entry + exit + force_exit)
	MonthEntryCount    int          `json:"monthEntryCount"`    // 月間の入室数 (当月: entry)
	MonthExitCount     int          `json:"monthExitCount"`     // 月間の退室数 (当月: exit + force_exit)
	YearLogCount       int          `json:"yearLogCount"`       // 年間の総入退室数 (選択年度: entry + exit + force_exit)
	YearEntryCount     int          `json:"yearEntryCount"`     // 年間の入室数 (選択年度: entry)
	YearExitCount      int          `json:"yearExitCount"`      // 年間の退室数 (選択年度: exit + force_exit)
	WeekChart          []ChartPoint `json:"weekChart"`          // 週別グラフ (直近7日間・日ごと)
	MonthChart         []ChartPoint `json:"monthChart"`         // 月別グラフ (直近4週間・週ごと)
	YearChart          []ChartPoint `json:"yearChart"`          // 年別グラフ (選択年度4月〜翌3月・月ごと)
}

// MonthlyStatRow 月別・区分別打刻統計行データ (01prompt, 19prompt要件)
type MonthlyStatRow struct {
	YearMonth                 string `json:"yearMonth"`                 // "2026-09"
	RoleOtherCount            int    `json:"roleOtherCount"`            // "-" (区分未設定 / その他)
	Role0Count                int    `json:"role0Count"`                // "0（教職員）"
	Role1Count                int    `json:"role1Count"`                // "1（学生）"
	Role9Count                int    `json:"role9Count"`                // "9（スタッフ）"
	MonthlyTotal              int    `json:"monthlyTotal"`              // 当月入室合計 (のべ利用回数)
	MonthlyExitCount          int    `json:"monthlyExitCount"`          // 当月退室合計 (exit + force_exit)
	MonthlyLogTotal           int    `json:"monthlyLogTotal"`           // 当月総打刻数 (入室 + 退室)
	FiscalYear                int    `json:"fiscalYear"`                // 2026 (年度)
	FiscalYearCumulativeTotal int    `json:"fiscalYearCumulativeTotal"` // 当月までの合計 (年度累計入室数)
	QuarterPeriod             string `json:"quarterPeriod"`             // 四半期ラベル ("4月～6月", "7月～9月", "10月～12月", "1月～3月")
	QuarterTotal              int    `json:"quarterTotal"`              // 当該四半期合計
}

// MonthlyStatsResponse 月別統計レスポンス
type MonthlyStatsResponse struct {
	Rows []MonthlyStatRow `json:"rows"`
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

	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=cache_size(-2000)")
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
	// 1. registered_users テーブルの存在および fiscal_year カラムの有無を確認
	hasRegisteredUsers := false
	hasFiscalYear := false

	tableCheckRow := m.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='registered_users'")
	var tblCount int
	if err := tableCheckRow.Scan(&tblCount); err == nil && tblCount > 0 {
		hasRegisteredUsers = true
		infoRows, err := m.db.Query("PRAGMA table_info(registered_users)")
		if err == nil {
			for infoRows.Next() {
				var cid int
				var colName, ctype string
				var notnull, pk int
				var dfltValue sql.NullString
				if err := infoRows.Scan(&cid, &colName, &ctype, &notnull, &dfltValue, &pk); err == nil {
					if colName == "fiscal_year" {
						hasFiscalYear = true
					}
				}
			}
			infoRows.Close()
		}
	}

	currentFY := CurrentFiscalYear()

	if !hasRegisteredUsers {
		// テーブル未存在: 初回作成 (複合主キー: card_id, fiscal_year)
		schema := `
		CREATE TABLE IF NOT EXISTS registered_users (
			card_id TEXT NOT NULL,
			fiscal_year INTEGER NOT NULL,
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
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (card_id, fiscal_year)
		);
		`
		if _, err := m.db.Exec(schema); err != nil {
			return err
		}
	} else if !hasFiscalYear {
		// 既存テーブルあり & fiscal_year なし: 複合主キーテーブルへマイグレーション
		migrationSQL := `
		CREATE TABLE registered_users_v2 (
			card_id TEXT NOT NULL,
			fiscal_year INTEGER NOT NULL,
			name TEXT NOT NULL,
			furigana TEXT NOT NULL DEFAULT '',
			gender TEXT NOT NULL DEFAULT '',
			role_name TEXT NOT NULL,
			role_code INTEGER NOT NULL DEFAULT 1,
			student_no TEXT NOT NULL DEFAULT '',
			admin_no TEXT NOT NULL DEFAULT '',
			purpose TEXT NOT NULL DEFAULT '',
			contact TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (card_id, fiscal_year)
		);

		INSERT OR IGNORE INTO registered_users_v2 (
			card_id, fiscal_year, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at
		)
		SELECT 
			card_id, ?, name,
			COALESCE(furigana, ''),
			COALESCE(gender, ''),
			role_name,
			COALESCE(role_code, 1),
			COALESCE(student_no, ''),
			COALESCE(admin_no, ''),
			COALESCE(purpose, ''),
			COALESCE(contact, ''),
			created_at, updated_at
		FROM registered_users;

		DROP TABLE registered_users;
		ALTER TABLE registered_users_v2 RENAME TO registered_users;
		`
		if _, err := m.db.Exec(migrationSQL, currentFY); err != nil {
			return fmt.Errorf("failed to migrate registered_users to fiscal_year schema: %w", err)
		}
	}

	// 2. access_logs テーブルの作成 (スナップショットカラム付き)
	logsSchema := `
	CREATE TABLE IF NOT EXISTS access_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		card_id TEXT NOT NULL,
		event_type TEXT NOT NULL, -- 'entry', 'exit', 'force_exit', 'unregistered', 'unregistered_renewal'
		timestamp DATETIME NOT NULL,
		duration_second INTEGER NOT NULL DEFAULT 0,
		user_name TEXT NOT NULL DEFAULT '',
		role_name TEXT NOT NULL DEFAULT '',
		role_code INTEGER NOT NULL DEFAULT 1,
		student_no TEXT NOT NULL DEFAULT '',
		purpose TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_access_logs_card_id ON access_logs(card_id);
	CREATE INDEX IF NOT EXISTS idx_access_logs_timestamp ON access_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_access_logs_card_id_id ON access_logs(card_id, id DESC);
	CREATE INDEX IF NOT EXISTS idx_access_logs_event_timestamp ON access_logs(event_type, timestamp);
	CREATE INDEX IF NOT EXISTS idx_registered_users_student_no ON registered_users(student_no);
	CREATE INDEX IF NOT EXISTS idx_registered_users_fy ON registered_users(fiscal_year);
	CREATE INDEX IF NOT EXISTS idx_registered_users_student_fy ON registered_users(student_no, fiscal_year);
	`
	if _, err := m.db.Exec(logsSchema); err != nil {
		return err
	}

	// 既存DB用 access_logs スナップショットカラムマイグレーション
	_, _ = m.db.Exec(`ALTER TABLE access_logs ADD COLUMN user_name TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE access_logs ADD COLUMN role_name TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE access_logs ADD COLUMN role_code INTEGER NOT NULL DEFAULT 1`)
	_, _ = m.db.Exec(`ALTER TABLE access_logs ADD COLUMN student_no TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE access_logs ADD COLUMN purpose TEXT NOT NULL DEFAULT ''`)

	// 既存DB用 registered_users カラム補完
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN furigana TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN gender TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN admin_no TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN purpose TEXT NOT NULL DEFAULT ''`)
	_, _ = m.db.Exec(`ALTER TABLE registered_users ADD COLUMN contact TEXT NOT NULL DEFAULT ''`)

	// 既存ログのバックフィル (user_name が未設定の古いログに registered_users の情報を適用)
	_, _ = m.db.Exec(`
		UPDATE access_logs 
		SET 
			user_name = COALESCE((SELECT name FROM registered_users WHERE registered_users.card_id = access_logs.card_id ORDER BY fiscal_year DESC LIMIT 1), '未登録'),
			role_name = COALESCE((SELECT role_name FROM registered_users WHERE registered_users.card_id = access_logs.card_id ORDER BY fiscal_year DESC LIMIT 1), '未登録'),
			role_code = COALESCE((SELECT role_code FROM registered_users WHERE registered_users.card_id = access_logs.card_id ORDER BY fiscal_year DESC LIMIT 1), 1),
			student_no = COALESCE((SELECT student_no FROM registered_users WHERE registered_users.card_id = access_logs.card_id ORDER BY fiscal_year DESC LIMIT 1), ''),
			purpose = COALESCE((SELECT purpose FROM registered_users WHERE registered_users.card_id = access_logs.card_id ORDER BY fiscal_year DESC LIMIT 1), '')
		WHERE user_name = '' OR user_name IS NULL
	`)

	return nil
}

// GetUser 登録ユーザー取得 (card_id または student_no のどちらでも検索可能、指定年度または今年度)
func (m *DBManager) GetUser(identifier string, fiscalYears ...int) (*RegisteredUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fy := CurrentFiscalYear()
	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy = fiscalYears[0]
	}

	query := `SELECT card_id, fiscal_year, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at FROM registered_users WHERE fiscal_year = ? AND (card_id = ? OR (student_no != '' AND student_no = ?)) LIMIT 1`
	row := m.db.QueryRow(query, fy, identifier, identifier)

	var u RegisteredUser
	var createdAtStr, updatedAtStr string
	err := row.Scan(&u.CardID, &u.FiscalYear, &u.Name, &u.Furigana, &u.Gender, &u.RoleName, &u.RoleCode, &u.StudentNo, &u.AdminNo, &u.Purpose, &u.Contact, &createdAtStr, &updatedAtStr)
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

// FindUserInAnyFiscalYear 過去年度を含め、最新の登録ユーザー情報を検索 (未更新者の情報引き継ぎ用)
func (m *DBManager) FindUserInAnyFiscalYear(identifier string) (*RegisteredUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
	SELECT card_id, fiscal_year, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at 
	FROM registered_users 
	WHERE card_id = ? OR (student_no != '' AND student_no = ?)
	ORDER BY fiscal_year DESC, updated_at DESC
	LIMIT 1
	`
	row := m.db.QueryRow(query, identifier, identifier)

	var u RegisteredUser
	var createdAtStr, updatedAtStr string
	err := row.Scan(&u.CardID, &u.FiscalYear, &u.Name, &u.Furigana, &u.Gender, &u.RoleName, &u.RoleCode, &u.StudentNo, &u.AdminNo, &u.Purpose, &u.Contact, &createdAtStr, &updatedAtStr)
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

// GetAllUsers 全ユーザー一覧取得 (年度指定対応、省略時は今年度)
func (m *DBManager) GetAllUsers(fiscalYears ...int) ([]RegisteredUser, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fy := CurrentFiscalYear()
	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy = fiscalYears[0]
	}

	query := `SELECT card_id, fiscal_year, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at FROM registered_users WHERE fiscal_year = ? ORDER BY role_code ASC, student_no ASC, name ASC`
	rows, err := m.db.Query(query, fy)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []RegisteredUser
	for rows.Next() {
		var u RegisteredUser
		var createdAtStr, updatedAtStr string
		if err := rows.Scan(&u.CardID, &u.FiscalYear, &u.Name, &u.Furigana, &u.Gender, &u.RoleName, &u.RoleCode, &u.StudentNo, &u.AdminNo, &u.Purpose, &u.Contact, &createdAtStr, &updatedAtStr); err != nil {
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
	if users == nil {
		users = []RegisteredUser{}
	}
	return users, nil
}

// GetMaxAdminNo 既存ユーザーに設定されている管理番号の最大整数値を取得 (自動採番用)
func (m *DBManager) GetMaxAdminNo(fiscalYears ...int) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fy := CurrentFiscalYear()
	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy = fiscalYears[0]
	}

	rows, err := m.db.Query("SELECT admin_no FROM registered_users WHERE fiscal_year = ? AND admin_no != ''", fy)
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

// GetAvailableFiscalYears 利用可能な年度一覧を降順で取得 (登録者または打刻ログが存在する年度 + 今年度)
func (m *DBManager) GetAvailableFiscalYears() ([]int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
	SELECT DISTINCT fiscal_year FROM registered_users
	UNION
	SELECT DISTINCT (
		CASE 
			WHEN CAST(substr(timestamp, 6, 2) AS INT) < 4 
			THEN CAST(substr(timestamp, 1, 4) AS INT) - 1 
			ELSE CAST(substr(timestamp, 1, 4) AS INT) 
		END
	) AS fy FROM access_logs WHERE timestamp != ''
	ORDER BY 1 DESC
	`
	rows, err := m.db.Query(query)
	if err != nil {
		return []int{CurrentFiscalYear()}, nil
	}
	defer rows.Close()

	currentFY := CurrentFiscalYear()
	fyMap := map[int]bool{currentFY: true}
	var list []int

	for rows.Next() {
		var fy int
		if err := rows.Scan(&fy); err == nil && fy > 0 {
			fyMap[fy] = true
		}
	}

	for fy := range fyMap {
		list = append(list, fy)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(list)))
	return list, nil
}

// UpsertUser ユーザー登録・更新
func (m *DBManager) UpsertUser(u *RegisteredUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u.CardID == "" {
		u.CardID = u.StudentNo
	}

	if u.FiscalYear <= 0 {
		u.FiscalYear = CurrentFiscalYear()
	}

	// 管理番号が未指定の場合は、今年度の最大値+1を自動採番
	if strings.TrimSpace(u.AdminNo) == "" {
		var maxNo int
		row := m.db.QueryRow("SELECT COALESCE(MAX(CAST(admin_no AS INTEGER)), 0) FROM registered_users WHERE fiscal_year = ?", u.FiscalYear)
		_ = row.Scan(&maxNo)
		u.AdminNo = strconv.Itoa(maxNo + 1)
	}

	now := time.Now()
	query := `
	INSERT INTO registered_users (card_id, fiscal_year, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(card_id, fiscal_year) DO UPDATE SET
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
	_, err := m.db.Exec(query, u.CardID, u.FiscalYear, u.Name, u.Furigana, u.Gender, u.RoleName, u.RoleCode, u.StudentNo, u.AdminNo, u.Purpose, u.Contact, now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return err
	}

	// 登録・更新されたユーザー情報を、該当カードIDまたは学籍番号の過去ログ（未登録打刻含む）に適応する
	updateLogsQuery := `
	UPDATE access_logs
	SET 
		user_name = ?,
		role_name = ?,
		role_code = ?,
		student_no = ?,
		purpose = CASE WHEN purpose = '' THEN ? ELSE purpose END
	WHERE card_id = ? OR (student_no != '' AND student_no = ?)
	`
	_, _ = m.db.Exec(updateLogsQuery, u.Name, u.RoleName, u.RoleCode, u.StudentNo, u.Purpose, u.CardID, u.StudentNo)

	return nil
}

// ImportUsers 複数ユーザーの一括インポート・Upsert
func (m *DBManager) ImportUsers(users []RegisteredUser, fiscalYears ...int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	defaultFY := CurrentFiscalYear()
	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		defaultFY = fiscalYears[0]
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO registered_users (card_id, fiscal_year, name, furigana, gender, role_name, role_code, student_no, admin_no, purpose, contact, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(card_id, fiscal_year) DO UPDATE SET
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
		fy := u.FiscalYear
		if fy <= 0 {
			fy = defaultFY
		}
		createdAt := u.CreatedAt
		if createdAt.IsZero() {
			createdAt = now
		}
		updatedAt := u.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		_, err := stmt.Exec(cardID, fy, u.Name, u.Furigana, u.Gender, u.RoleName, u.RoleCode, u.StudentNo, u.AdminNo, u.Purpose, u.Contact, createdAt.Format(time.RFC3339), updatedAt.Format(time.RFC3339))
		if err != nil {
			return count, err
		}
		count++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	// インポートされたユーザー情報を該当カードの未登録ログに適応
	updateStmt, err := m.db.Prepare(`
	UPDATE access_logs
	SET 
		user_name = ?,
		role_name = ?,
		role_code = ?,
		student_no = ?,
		purpose = CASE WHEN purpose = '' THEN ? ELSE purpose END
	WHERE (card_id = ? OR (student_no != '' AND student_no = ?))
	  AND (user_name = '' OR user_name = '未登録' OR role_name = '' OR role_name = '未登録')
	`)
	if err == nil {
		defer updateStmt.Close()
		for _, u := range users {
			cardID := u.CardID
			if cardID == "" {
				cardID = u.StudentNo
			}
			_, _ = updateStmt.Exec(u.Name, u.RoleName, u.RoleCode, u.StudentNo, u.Purpose, cardID, u.StudentNo)
		}
	}

	return count, nil
}

// ImportLogs 過去の入退室ログ一括インポート (スナップショット対応)
func (m *DBManager) ImportLogs(logs []AccessLog) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO access_logs (card_id, event_type, timestamp, duration_second, user_name, role_name, role_code, student_no, purpose)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		roleCode := l.RoleCode
		if roleCode == 0 && l.RoleName != "教職員" {
			roleCode = 1
		}
		_, err := stmt.Exec(l.CardID, l.EventType, ts.Format(time.RFC3339), l.DurationSecond, l.UserName, l.RoleName, roleCode, l.StudentNo, l.Purpose)
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

// DeleteUser ユーザー削除 (年度指定対応、省略時は今年度)
func (m *DBManager) DeleteUser(cardID string, fiscalYears ...int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	fy := CurrentFiscalYear()
	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy = fiscalYears[0]
	}

	_, err := m.db.Exec("DELETE FROM registered_users WHERE (card_id = ? OR student_no = ?) AND fiscal_year = ?", cardID, cardID, fy)
	return err
}

// RecordSwipe 打刻処理 (入退室判定＆ログ記録、スナップショット保存)
func (m *DBManager) RecordSwipe(user *RegisteredUser) (*AccessLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// 直近の有効な打刻ログを確認して、奇数/偶数 (トグル) 判定 (未登録打刻はトグルに影響させない)
	var lastEventType string
	var lastTsStr string
	var lastID int64
	queryLast := `SELECT id, event_type, timestamp FROM access_logs WHERE card_id = ? AND event_type IN ('entry', 'exit', 'force_exit') ORDER BY id DESC LIMIT 1`
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

	// ログ書き込み (打刻時点のユーザー情報をスナップショットとして保存)
	insertQuery := `
	INSERT INTO access_logs (card_id, event_type, timestamp, duration_second, user_name, role_name, role_code, student_no, purpose) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := m.db.Exec(insertQuery, user.CardID, eventType, now.Format(time.RFC3339), durationSec, user.Name, user.RoleName, user.RoleCode, user.StudentNo, user.Purpose)
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
		Purpose:        user.Purpose,
		EventType:      eventType,
		Timestamp:      now,
		DurationSecond: durationSec,
		DurationText:   durationText,
	}, nil
}

// RecordUnregisteredSwipe 未登録 / 今年度未登録カードの打刻ログ記録 (スナップショット保存)
func (m *DBManager) RecordUnregisteredSwipe(cardID string, eventType string, userSnapshot *RegisteredUser) (*AccessLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	userName := "未登録"
	roleName := "未登録"
	roleCode := 1
	studentNo := cardID
	purpose := ""

	if userSnapshot != nil {
		if userSnapshot.Name != "" {
			userName = userSnapshot.Name
		}
		if userSnapshot.RoleName != "" {
			roleName = userSnapshot.RoleName
		}
		roleCode = userSnapshot.RoleCode
		if userSnapshot.StudentNo != "" {
			studentNo = userSnapshot.StudentNo
		}
		purpose = userSnapshot.Purpose
	}

	insertQuery := `
	INSERT INTO access_logs (card_id, event_type, timestamp, duration_second, user_name, role_name, role_code, student_no, purpose) 
	VALUES (?, ?, ?, 0, ?, ?, ?, ?, ?)
	`
	res, err := m.db.Exec(insertQuery, cardID, eventType, now.Format(time.RFC3339), userName, roleName, roleCode, studentNo, purpose)
	if err != nil {
		return nil, err
	}

	newID, _ := res.LastInsertId()
	return &AccessLog{
		ID:             newID,
		CardID:         cardID,
		UserName:       userName,
		RoleName:       roleName,
		RoleCode:       roleCode,
		StudentNo:      studentNo,
		Purpose:        purpose,
		EventType:      eventType,
		Timestamp:      now,
		DurationSecond: 0,
		DurationText:   "",
	}, nil
}

// ForceExitAllInsideUsers 23:00等に在室中の全ユーザーを一括強制退室処理する
func (m *DBManager) ForceExitAllInsideUsers(exitTime time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 現在在室中（最新ログが 'entry'）のカード一覧とスナップショット情報を取得
	query := `
	SELECT 
		l.card_id, l.timestamp, l.user_name, l.role_name, l.role_code, l.student_no, l.purpose
	FROM access_logs l
	INNER JOIN (
		SELECT card_id, MAX(id) as max_id FROM access_logs WHERE event_type IN ('entry', 'exit', 'force_exit') GROUP BY card_id
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
		userName  string
		roleName  string
		roleCode  int
		studentNo string
		purpose   string
	}
	var targets []insideEntry

	for rows.Next() {
		var cardID, tsStr, userName, roleName, studentNo, purpose string
		var roleCode int
		if err := rows.Scan(&cardID, &tsStr, &userName, &roleName, &roleCode, &studentNo, &purpose); err != nil {
			return 0, err
		}
		t, _ := time.Parse(time.RFC3339, tsStr)
		if t.IsZero() {
			t, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		}
		targets = append(targets, insideEntry{
			cardID:    cardID,
			entryTime: t,
			userName:  userName,
			roleName:  roleName,
			roleCode:  roleCode,
			studentNo: studentNo,
			purpose:   purpose,
		})
	}

	if len(targets) == 0 {
		return 0, nil
	}

	tx, err := m.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	insertStmt, err := tx.Prepare(`
	INSERT INTO access_logs (card_id, event_type, timestamp, duration_second, user_name, role_name, role_code, student_no, purpose) 
	VALUES (?, 'force_exit', ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer insertStmt.Close()

	for _, item := range targets {
		durationSec := int64(exitTime.Sub(item.entryTime).Seconds())
		if durationSec < 0 {
			durationSec = 0
		}
		_, err := insertStmt.Exec(item.cardID, exitTime.Format(time.RFC3339), durationSec, item.userName, item.roleName, item.roleCode, item.studentNo, item.purpose)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return len(targets), nil
}

// GetRecentLogs 直近の打刻ログ一覧取得 (limit件、年度指定対応)
func (m *DBManager) GetRecentLogs(limit int, fiscalYears ...int) ([]AccessLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var query string
	var args []interface{}

	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy := fiscalYears[0]
		startDate := fmt.Sprintf("%04d-04-01", fy)
		endDate := fmt.Sprintf("%04d-04-01", fy+1)
		query = `
		SELECT 
			l.id, l.card_id,
			COALESCE(NULLIF(NULLIF(l.user_name, ''), '未登録'), u.name, '未登録') as user_name,
			COALESCE(NULLIF(NULLIF(l.role_name, ''), '未登録'), u.role_name, '未登録') as role_name,
			COALESCE(NULLIF(l.role_code, 1), u.role_code, 1) as role_code,
			COALESCE(NULLIF(l.student_no, ''), u.student_no, '') as student_no,
			COALESCE(NULLIF(l.purpose, ''), u.purpose, '') as purpose,
			l.event_type, l.timestamp, l.duration_second
		FROM access_logs l
		LEFT JOIN registered_users u ON l.card_id = u.card_id AND u.fiscal_year = ?
		WHERE l.timestamp >= ? AND l.timestamp < ?
		ORDER BY l.id DESC
		LIMIT ?
		`
		args = []interface{}{fy, startDate, endDate, limit}
	} else {
		currentFY := CurrentFiscalYear()
		query = `
		SELECT 
			l.id, l.card_id,
			COALESCE(NULLIF(NULLIF(l.user_name, ''), '未登録'), u.name, '未登録') as user_name,
			COALESCE(NULLIF(NULLIF(l.role_name, ''), '未登録'), u.role_name, '未登録') as role_name,
			COALESCE(NULLIF(l.role_code, 1), u.role_code, 1) as role_code,
			COALESCE(NULLIF(l.student_no, ''), u.student_no, '') as student_no,
			COALESCE(NULLIF(l.purpose, ''), u.purpose, '') as purpose,
			l.event_type, l.timestamp, l.duration_second
		FROM access_logs l
		LEFT JOIN registered_users u ON l.card_id = u.card_id AND u.fiscal_year = ?
		ORDER BY l.id DESC
		LIMIT ?
		`
		args = []interface{}{currentFY, limit}
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var l AccessLog
		var tsStr string
		var name, roleName, studentNo, purpose sql.NullString
		var roleCode sql.NullInt64

		if err := rows.Scan(&l.ID, &l.CardID, &name, &roleName, &roleCode, &studentNo, &purpose, &l.EventType, &tsStr, &l.DurationSecond); err != nil {
			return nil, err
		}
		l.UserName = name.String
		l.RoleName = roleName.String
		l.RoleCode = int(roleCode.Int64)
		l.StudentNo = studentNo.String
		l.Purpose = purpose.String
		l.Timestamp, _ = time.Parse(time.RFC3339, tsStr)
		if l.Timestamp.IsZero() {
			l.Timestamp, _ = time.Parse("2006-01-02 15:04:05", tsStr)
		}
		if (l.EventType == "exit" || l.EventType == "force_exit") && l.DurationSecond > 0 {
			l.DurationText = FormatDuration(l.DurationSecond)
		}
		logs = append(logs, l)
	}
	if logs == nil {
		logs = []AccessLog{}
	}
	return logs, nil
}

// GetDashboardStats ダッシュボード用集計データ (年度指定対応、省略時は今年度、SQLによる全件正確集計)
func (m *DBManager) GetDashboardStats(fiscalYears ...int) (*DashboardStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	fy := CurrentFiscalYear()
	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy = fiscalYears[0]
	}

	stats := &DashboardStats{
		FiscalYear: fy,
		WeekChart:  []ChartPoint{},
		MonthChart: []ChartPoint{},
		YearChart:  []ChartPoint{},
	}

	// 1. 指定年度のユーザー登録総数
	err := m.db.QueryRow("SELECT COUNT(*) FROM registered_users WHERE fiscal_year = ?", fy).Scan(&stats.TotalUserCount)
	if err != nil {
		return nil, err
	}

	// 2. 現在の在室者数 (本日00:00以降にentryし、最新ログが 'entry' である件数、未登録ログ除外)
	now := time.Now()
	todayPrefix := now.Format("2006-01-02")
	tomorrowPrefix := now.AddDate(0, 0, 1).Format("2006-01-02")
	queryInside := `
	SELECT COUNT(*) FROM (
		SELECT l.card_id, l.event_type FROM access_logs l
		INNER JOIN (
			SELECT card_id, MAX(id) as max_id FROM access_logs 
			WHERE event_type IN ('entry', 'exit', 'force_exit') 
			GROUP BY card_id
		) latest ON l.id = latest.max_id
		WHERE l.event_type = 'entry'
		  AND l.timestamp >= ? AND l.timestamp < ?
	)
	`
	_ = m.db.QueryRow(queryInside, todayPrefix, tomorrowPrefix).Scan(&stats.CurrentInsideCount)

	// 3. 期間別（本日・週間・月間・選択年度）の入室数・退室数をSQLで正確に集計
	weekStart := now.AddDate(0, 0, -6).Format("2006-01-02")
	monthStart := fmt.Sprintf("%04d-%02d-01", now.Year(), int(now.Month()))
	nextMonthStart := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	fyStart := fmt.Sprintf("%04d-04-01", fy)
	fyNextStart := fmt.Sprintf("%04d-04-01", fy+1)

	queryPeriods := `
	SELECT
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type = 'entry' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type IN ('exit', 'force_exit') THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type = 'entry' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type IN ('exit', 'force_exit') THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type = 'entry' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type IN ('exit', 'force_exit') THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type = 'entry' THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN timestamp >= ? AND timestamp < ? AND event_type IN ('exit', 'force_exit') THEN 1 ELSE 0 END), 0)
	FROM access_logs
	WHERE event_type IN ('entry', 'exit', 'force_exit')
	`
	err = m.db.QueryRow(queryPeriods,
		todayPrefix, tomorrowPrefix,
		todayPrefix, tomorrowPrefix,
		weekStart, tomorrowPrefix,
		weekStart, tomorrowPrefix,
		monthStart, nextMonthStart,
		monthStart, nextMonthStart,
		fyStart, fyNextStart,
		fyStart, fyNextStart,
	).Scan(
		&stats.TodayEntryCount,
		&stats.TodayExitCount,
		&stats.WeekEntryCount,
		&stats.WeekExitCount,
		&stats.MonthEntryCount,
		&stats.MonthExitCount,
		&stats.YearEntryCount,
		&stats.YearExitCount,
	)
	if err != nil {
		log.Printf("[DB] Warning: failed to aggregate period stats: %v", err)
	}

	stats.TodayLogCount = stats.TodayEntryCount + stats.TodayExitCount
	stats.WeekLogCount = stats.WeekEntryCount + stats.WeekExitCount
	stats.MonthLogCount = stats.MonthEntryCount + stats.MonthExitCount
	stats.YearLogCount = stats.YearEntryCount + stats.YearExitCount

	// 4. 週別グラフ（直近7日間・日ごと）および月別グラフ（直近4週間・週ごと）のSQL集計
	type entryExitCount struct {
		entries int
		exits   int
	}
	dayMap := make(map[string]entryExitCount)
	twentyEightDaysAgo := now.AddDate(0, 0, -27).Format("2006-01-02")

	rows28, err := m.db.Query(`
		SELECT
			substr(timestamp, 1, 10) AS day_str,
			COUNT(CASE WHEN event_type = 'entry' THEN 1 END) AS entries,
			COUNT(CASE WHEN event_type IN ('exit', 'force_exit') THEN 1 END) AS exits
		FROM access_logs
		WHERE timestamp >= ? AND timestamp < ?
		  AND event_type IN ('entry', 'exit', 'force_exit')
		GROUP BY day_str
	`, twentyEightDaysAgo, tomorrowPrefix)
	if err == nil {
		for rows28.Next() {
			var d string
			var en, ex int
			if err := rows28.Scan(&d, &en, &ex); err == nil {
				dayMap[d] = entryExitCount{entries: en, exits: ex}
			}
		}
		rows28.Close()
	}

	weekdays := []string{"日", "月", "火", "水", "木", "金", "土"}
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, -(6 - i))
		key := d.Format("2006-01-02")
		c := dayMap[key]
		stats.WeekChart = append(stats.WeekChart, ChartPoint{
			Label:   fmt.Sprintf("%d/%d(%s)", int(d.Month()), d.Day(), weekdays[d.Weekday()]),
			Entries: c.entries,
			Exits:   c.exits,
		})
	}

	for i := 0; i < 4; i++ {
		endOffset := (3 - i) * 7
		startOffset := endOffset + 6
		var enSum, exSum int
		for offset := startOffset; offset >= endOffset; offset-- {
			d := now.AddDate(0, 0, -offset)
			key := d.Format("2006-01-02")
			c := dayMap[key]
			enSum += c.entries
			exSum += c.exits
		}
		stats.MonthChart = append(stats.MonthChart, ChartPoint{
			Label:   fmt.Sprintf("%d週前", 4-i),
			Entries: enSum,
			Exits:   exSum,
		})
	}

	// 5. 年別グラフ（選択年度の4月〜翌3月・月ごと）のSQL集計 (架空の推計値を排除し実数を返却)
	ymMap := make(map[string]entryExitCount)
	rowsYear, err := m.db.Query(`
		SELECT
			substr(timestamp, 1, 7) AS ym,
			COUNT(CASE WHEN event_type = 'entry' THEN 1 END) AS entries,
			COUNT(CASE WHEN event_type IN ('exit', 'force_exit') THEN 1 END) AS exits
		FROM access_logs
		WHERE timestamp >= ? AND timestamp < ?
		  AND event_type IN ('entry', 'exit', 'force_exit')
		GROUP BY ym
	`, fyStart, fyNextStart)
	if err == nil {
		for rowsYear.Next() {
			var ym string
			var en, ex int
			if err := rowsYear.Scan(&ym, &en, &ex); err == nil {
				ymMap[ym] = entryExitCount{entries: en, exits: ex}
			}
		}
		rowsYear.Close()
	}

	fyMonths := []int{4, 5, 6, 7, 8, 9, 10, 11, 12, 1, 2, 3}
	for _, mNum := range fyMonths {
		y := fy
		if mNum < 4 {
			y = fy + 1
		}
		ym := fmt.Sprintf("%04d-%02d", y, mNum)
		c := ymMap[ym]
		stats.YearChart = append(stats.YearChart, ChartPoint{
			Label:   fmt.Sprintf("%d月", mNum),
			Entries: c.entries,
			Exits:   c.exits,
		})
	}

	return stats, nil
}

// GetMonthlyStats 月別・区分別打刻統計データの取得 (年月, -, 0教職員, 1学生, 9スタッフ, 当月合計, 当月までの合計、年度指定対応)
func (m *DBManager) GetMonthlyStats(fiscalYears ...int) (*MonthlyStatsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var query string
	var args []interface{}

	roleCaseExpr := `
		CASE
			WHEN l.role_name = '教職員' OR (l.role_name != '' AND l.role_name != '未登録' AND l.role_code = 0) THEN 0
			WHEN l.role_name IN ('学生スタッフ', 'スタッフ') OR (l.role_name != '' AND l.role_name != '未登録' AND l.role_code = 9) THEN 9
			WHEN l.role_name = '学生' OR (l.role_name != '' AND l.role_name != '未登録' AND l.role_code = 1) THEN 1
			WHEN u.role_code IS NOT NULL THEN u.role_code
			WHEN l.user_name != '' AND l.user_name != '未登録' THEN l.role_code
			ELSE -1
		END
	`

	if len(fiscalYears) > 0 && fiscalYears[0] > 0 {
		fy := fiscalYears[0]
		startDate := fmt.Sprintf("%04d-04-01", fy)
		endDate := fmt.Sprintf("%04d-04-01", fy+1)
		query = fmt.Sprintf(`
		SELECT 
			substr(l.timestamp, 1, 7) AS ym,
			%s AS role_code,
			COUNT(CASE WHEN l.event_type = 'entry' THEN 1 END) AS entry_cnt,
			COUNT(CASE WHEN l.event_type IN ('exit', 'force_exit') THEN 1 END) AS exit_cnt
		FROM access_logs l
		LEFT JOIN registered_users u ON l.card_id = u.card_id AND u.fiscal_year = ?
		WHERE l.event_type IN ('entry', 'exit', 'force_exit')
		  AND l.timestamp >= ? AND l.timestamp < ?
		GROUP BY 1, 2
		ORDER BY ym ASC
		`, roleCaseExpr)
		args = []interface{}{fy, startDate, endDate}
	} else {
		currentFY := CurrentFiscalYear()
		query = fmt.Sprintf(`
		SELECT 
			substr(l.timestamp, 1, 7) AS ym,
			%s AS role_code,
			COUNT(CASE WHEN l.event_type = 'entry' THEN 1 END) AS entry_cnt,
			COUNT(CASE WHEN l.event_type IN ('exit', 'force_exit') THEN 1 END) AS exit_cnt
		FROM access_logs l
		LEFT JOIN registered_users u ON l.card_id = u.card_id AND u.fiscal_year = ?
		WHERE l.event_type IN ('entry', 'exit', 'force_exit')
		GROUP BY 1, 2
		ORDER BY ym ASC
		`, roleCaseExpr)
		args = []interface{}{currentFY}
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type monthGroup struct {
		yearMonth string
		roleOther int
		role0     int
		role1     int
		role9     int
		exits     int
	}

	monthMap := make(map[string]*monthGroup)
	var orderedYM []string

	for rows.Next() {
		var ym string
		var roleCode, entryCnt, exitCnt int
		if err := rows.Scan(&ym, &roleCode, &entryCnt, &exitCnt); err != nil {
			return nil, err
		}
		if ym == "" {
			continue
		}
		mg, exists := monthMap[ym]
		if !exists {
			mg = &monthGroup{yearMonth: ym}
			monthMap[ym] = mg
			orderedYM = append(orderedYM, ym)
		}
		mg.exits += exitCnt
		switch roleCode {
		case 0:
			mg.role0 += entryCnt
		case 1:
			mg.role1 += entryCnt
		case 9:
			mg.role9 += entryCnt
		default:
			mg.roleOther += entryCnt
		}
	}

	var resultRows []MonthlyStatRow
	quarterTotals := make(map[string]int)

	currentFiscalYear := -1
	fyCumulative := 0

	for _, ym := range orderedYM {
		mg := monthMap[ym]
		var year, month int
		fmt.Sscanf(ym, "%d-%d", &year, &month)

		fiscalYear := year
		if month < 4 {
			fiscalYear = year - 1
		}

		quarterPeriod := ""
		if month >= 4 && month <= 6 {
			quarterPeriod = "4月～6月"
		} else if month >= 7 && month <= 9 {
			quarterPeriod = "7月～9月"
		} else if month >= 10 && month <= 12 {
			quarterPeriod = "10月～12月"
		} else {
			quarterPeriod = "1月～3月"
		}

		if fiscalYear != currentFiscalYear {
			currentFiscalYear = fiscalYear
			fyCumulative = 0
		}

		monthlyTotal := mg.roleOther + mg.role0 + mg.role1 + mg.role9
		if monthlyTotal == 0 && mg.exits == 0 {
			continue
		}
		fyCumulative += monthlyTotal

		qKey := fmt.Sprintf("%d_%s", fiscalYear, quarterPeriod)
		quarterTotals[qKey] += monthlyTotal

		resultRows = append(resultRows, MonthlyStatRow{
			YearMonth:                 ym,
			RoleOtherCount:            mg.roleOther,
			Role0Count:                mg.role0,
			Role1Count:                mg.role1,
			Role9Count:                mg.role9,
			MonthlyTotal:              monthlyTotal,
			MonthlyExitCount:          mg.exits,
			MonthlyLogTotal:           monthlyTotal + mg.exits,
			FiscalYear:                fiscalYear,
			FiscalYearCumulativeTotal: fyCumulative,
			QuarterPeriod:             quarterPeriod,
			QuarterTotal:              0, // 2パス目で設定
		})
	}

	// 四半期合計の反映
	for i := range resultRows {
		qKey := fmt.Sprintf("%d_%s", resultRows[i].FiscalYear, resultRows[i].QuarterPeriod)
		resultRows[i].QuarterTotal = quarterTotals[qKey]
	}

	// 年月降順（新しい順）
	for i, j := 0, len(resultRows)-1; i < j; i, j = i+1, j-1 {
		resultRows[i], resultRows[j] = resultRows[j], resultRows[i]
	}

	if resultRows == nil {
		resultRows = []MonthlyStatRow{}
	}

	return &MonthlyStatsResponse{
		Rows: resultRows,
	}, nil
}

// GetCurrentInsideUsers 現在在室中のユーザー一覧 (本日00:00以降にentryした利用者を対象)
func (m *DBManager) GetCurrentInsideUsers() ([]UserStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	todayPrefix := now.Format("2006-01-02")
	tomorrowPrefix := now.AddDate(0, 0, 1).Format("2006-01-02")

	query := `
	SELECT 
		l.card_id,
		COALESCE(NULLIF(l.user_name, ''), u.name, '未登録') as name,
		COALESCE(NULLIF(l.role_name, ''), u.role_name, '未登録') as role_name,
		COALESCE(NULLIF(l.student_no, ''), u.student_no, '') as student_no,
		l.event_type, l.timestamp
	FROM access_logs l
	INNER JOIN (
		SELECT card_id, MAX(id) as max_id FROM access_logs 
		WHERE event_type IN ('entry', 'exit', 'force_exit')
		GROUP BY card_id
	) latest ON l.id = latest.max_id
	LEFT JOIN registered_users u ON l.card_id = u.card_id AND u.fiscal_year = ?
	WHERE l.event_type = 'entry'
	  AND l.timestamp >= ? AND l.timestamp < ?
	ORDER BY l.timestamp DESC
	`
	currentFY := CurrentFiscalYear()
	rows, err := m.db.Query(query, currentFY, todayPrefix, tomorrowPrefix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
	if list == nil {
		list = []UserStatus{}
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

// GetFiscalYearLogs 指定年度のログ取得 (4月1日〜翌年3月31日、スナップショット優先)
func (m *DBManager) GetFiscalYearLogs(fiscalYear int) ([]AccessLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	startDate := fmt.Sprintf("%04d-04-01", fiscalYear)
	endDate := fmt.Sprintf("%04d-04-01", fiscalYear+1)

	query := `
	SELECT 
		l.id, l.card_id,
		COALESCE(NULLIF(NULLIF(l.user_name, ''), '未登録'), u.name, '未登録') as user_name,
		COALESCE(NULLIF(NULLIF(l.role_name, ''), '未登録'), u.role_name, '未登録') as role_name,
		COALESCE(NULLIF(l.role_code, 1), u.role_code, 1) as role_code,
		COALESCE(NULLIF(l.student_no, ''), u.student_no, '') as student_no,
		COALESCE(NULLIF(l.purpose, ''), u.purpose, '') as purpose,
		l.event_type, l.timestamp, l.duration_second
	FROM access_logs l
	LEFT JOIN registered_users u ON l.card_id = u.card_id AND u.fiscal_year = ?
	WHERE l.timestamp >= ? AND l.timestamp < ?
	ORDER BY l.timestamp ASC
	`
	rows, err := m.db.Query(query, fiscalYear, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AccessLog
	for rows.Next() {
		var l AccessLog
		var tsStr string
		if err := rows.Scan(&l.ID, &l.CardID, &l.UserName, &l.RoleName, &l.RoleCode, &l.StudentNo, &l.Purpose, &l.EventType, &tsStr, &l.DurationSecond); err != nil {
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
	if logs == nil {
		logs = []AccessLog{}
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
