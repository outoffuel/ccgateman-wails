package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"osuccgate/internal/db"
	"osuccgate/internal/nfc"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/japanese"
)

// SwipeResponse 打刻結果レスポンス
type SwipeResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CardID       string `json:"cardId"`
	UserName     string `json:"userName"`
	RoleName     string `json:"roleName"`
	RoleCode     int    `json:"roleCode"` // 0: 教職員, 1: 学生, 9: 学生スタッフ
	StudentNo    string `json:"studentNo"`
	EventType    string `json:"eventType"`    // "entry", "exit", "force_exit"
	Timestamp    string `json:"timestamp"`    // "15:04:05"
	DurationText string `json:"durationText"` // "1時間30分" (退室・強制退室時)
	SoundType    string `json:"soundType"`    // "studentEntry", "studentExit", "staffEntry", "staffExit", "booboo"
	IsDebounced  bool   `json:"isDebounced"`  // デバウンスによるスキップ
}

type GateService struct {
	dbManager   *db.DBManager
	lastSwipes  map[string]time.Time
	mu          sync.Mutex
	debounceSec time.Duration

	schedulerCancel context.CancelFunc
	lastForceExitDay string // 23:00二重実行防止用 ("2026-08-28")
}

func NewGateService(dbMgr *db.DBManager) *GateService {
	return &GateService{
		dbManager:   dbMgr,
		lastSwipes:  make(map[string]time.Time),
		debounceSec: 2 * time.Second, // 要件: 2秒間の待機（受付不可）時間
	}
}

// StartScheduler 23:00強制退室スケジューラーの開始
func (s *GateService) StartScheduler() {
	ctx, cancel := context.WithCancel(context.Background())
	s.schedulerCancel = cancel

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				// 毎日 23:00 (23時00分〜23時01分の間) にチェック
				todayStr := now.Format("2006-01-02")
				if now.Hour() == 23 && now.Minute() == 0 {
					if s.lastForceExitDay != todayStr {
						s.lastForceExitDay = todayStr
						log.Printf("[Scheduler] 23:00 reached. Executing automatic force exit for all inside users...")
						count, err := s.dbManager.ForceExitAllInsideUsers(now)
						if err != nil {
							log.Printf("[Scheduler] Error during automatic force exit: %v", err)
						} else {
							log.Printf("[Scheduler] Automatic force exit completed. Processed %d users.", count)
						}
					}
				}
			}
		}
	}()
}

// StopScheduler スケジューラーの停止
func (s *GateService) StopScheduler() {
	if s.schedulerCancel != nil {
		s.schedulerCancel()
	}
}

// ForceExitAllInside 手動での一括強制退室処理
func (s *GateService) ForceExitAllInside() (int, error) {
	now := time.Now()
	return s.dbManager.ForceExitAllInsideUsers(now)
}

// ProcessSwipe 打刻受付メインロジック (磁気 / NFC 共通)
func (s *GateService) ProcessSwipe(rawCardID string) *SwipeResponse {
	cardID := nfc.NormalizeCardID(rawCardID)
	if cardID == "" {
		return &SwipeResponse{
			Success:      false,
			ErrorMessage: "カード番号が空です",
			SoundType:    "booboo",
		}
	}

	// デバウンス制御 (2秒以内は無視)
	s.mu.Lock()
	now := time.Now()
	if lastTime, exists := s.lastSwipes[cardID]; exists {
		if now.Sub(lastTime) < s.debounceSec {
			s.mu.Unlock()
			return &SwipeResponse{
				Success:      false,
				ErrorMessage: "連続読み込み防止中（2秒待機）",
				IsDebounced:  true,
			}
		}
	}
	s.lastSwipes[cardID] = now
	s.mu.Unlock()

	// ユーザー取得
	user, err := s.dbManager.GetUser(cardID)
	if err != nil {
		return &SwipeResponse{
			Success:      false,
			ErrorMessage: "データベースエラーが発生しました",
			SoundType:    "booboo",
		}
	}

	if user == nil {
		// 未登録ユーザー
		return &SwipeResponse{
			Success:      false,
			CardID:       cardID,
			ErrorMessage: fmt.Sprintf("未登録のカードです (ID: %s)", cardID),
			SoundType:    "booboo",
		}
	}

	// 打刻記録（入退室判定＆滞在時間算出）
	logRecord, err := s.dbManager.RecordSwipe(user)
	if err != nil {
		return &SwipeResponse{
			Success:      false,
			CardID:       cardID,
			ErrorMessage: "打刻の保存に失敗しました",
			SoundType:    "booboo",
		}
	}

	// 音声演出タイプの決定 (教職員: 0, 学生: 1, 学生スタッフ: 9)
	soundType := "studentEntry"
	if user.RoleCode == 0 || user.RoleName == "教職員" {
		if logRecord.EventType == "entry" {
			soundType = "staffEntry"
		} else {
			soundType = "staffExit"
		}
	} else {
		// 学生 (1) または 学生スタッフ (9)
		if logRecord.EventType == "entry" {
			soundType = "studentEntry"
		} else {
			soundType = "studentExit"
		}
	}

	return &SwipeResponse{
		Success:      true,
		CardID:       user.CardID,
		UserName:     user.Name,
		RoleName:     user.RoleName,
		RoleCode:     user.RoleCode,
		StudentNo:    user.StudentNo,
		EventType:    logRecord.EventType,
		Timestamp:    logRecord.Timestamp.Format("15:04:05"),
		DurationText: logRecord.DurationText,
		SoundType:    soundType,
	}
}

// GenerateFiscalYearCSV 指定年度のCSVバイト列を生成 (Excel対応 UTF-8 BOM付き)
func (s *GateService) GenerateFiscalYearCSV(year int) ([]byte, error) {
	logs, err := s.dbManager.GetFiscalYearLogs(year)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	// UTF-8 BOM を追加してExcelで文字化けしないようにする
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	writer := csv.NewWriter(&buf)
	// ヘッダー行 (入力種別・方法を明記)
	headers := []string{"ログID", "打刻日時", "識別ID", "学籍番号/職員番号", "氏名", "区分", "入退室種別", "入力方法", "滞在時間(秒)", "滞在時間"}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, l := range logs {
		eventTypeJa := "入室"
		inputMethod := "カード読み取り"

		if l.EventType == "exit" {
			eventTypeJa = "退室"
			inputMethod = "カード読み取り"
		} else if l.EventType == "force_exit" {
			eventTypeJa = "強制退室"
			inputMethod = "システム自動 (23:00強制退室)"
		}

		row := []string{
			fmt.Sprintf("%d", l.ID),
			l.Timestamp.Format("2006-01-02 15:04:05"),
			l.CardID,
			l.StudentNo,
			l.UserName,
			l.RoleName,
			eventTypeJa,
			inputMethod,
			fmt.Sprintf("%d", l.DurationSecond),
			l.DurationText,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}

// sanitizeCSVData BOMの削除とShift-JIS/UTF-8の正規化
func sanitizeCSVData(data []byte) ([]byte, error) {
	// UTF-8 BOM の除去
	if bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}) {
		data = data[3:]
	}

	// UTF-8として有効か判定
	if utf8.Valid(data) {
		return data, nil
	}

	// Shift-JIS (Windows-31J/CP932) からのデコードを試みる
	decoder := japanese.ShiftJIS.NewDecoder()
	utf8Data, err := decoder.Bytes(data)
	if err == nil && utf8.Valid(utf8Data) {
		return utf8Data, nil
	}

	// フォールバック
	return data, nil
}

// ImportUsersCSV CSVデータから利用者を一括インポート
func (s *GateService) ImportUsersCSV(csvData []byte) (int, int, error) {
	cleanData, err := sanitizeCSVData(csvData)
	if err != nil {
		return 0, 0, fmt.Errorf("CSVデータの読み込みに失敗しました: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(cleanData))
	reader.FieldsPerRecord = -1 // 可変長対応
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return 0, 0, fmt.Errorf("CSVパースエラー: %w", err)
	}

	if len(rows) == 0 {
		return 0, 0, fmt.Errorf("CSVデータが空です")
	}

	// ヘッダー行判定
	headerMap := make(map[string]int)
	startIdx := 0

	// 1行目をヘッダーとして調査
	firstRow := rows[0]
	isHeader := false
	for colIdx, colVal := range firstRow {
		cleaned := strings.ToLower(strings.TrimSpace(colVal))
		cleaned = strings.ReplaceAll(cleaned, "_", "")
		cleaned = strings.ReplaceAll(cleaned, " ", "")
		cleaned = strings.ReplaceAll(cleaned, "（", "(")
		cleaned = strings.ReplaceAll(cleaned, "）", ")")

		if strings.Contains(cleaned, "学籍") || strings.Contains(cleaned, "学生番号") || strings.Contains(cleaned, "職員番号") || cleaned == "studentno" || cleaned == "studentid" {
			headerMap["student_no"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "フリガナ") || strings.Contains(cleaned, "ふりがな") || strings.Contains(cleaned, "カナ") || strings.Contains(cleaned, "かな") || cleaned == "furigana" || cleaned == "kana" {
			headerMap["furigana"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "氏名") || strings.Contains(cleaned, "名前") || strings.Contains(cleaned, "ユーザー名") || cleaned == "name" || cleaned == "username" {
			headerMap["name"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "性別") || strings.Contains(cleaned, "性") || cleaned == "gender" || cleaned == "sex" {
			headerMap["gender"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "区分") || strings.Contains(cleaned, "ロール") || strings.Contains(cleaned, "役職") || strings.Contains(cleaned, "属性") || cleaned == "role" || cleaned == "rolename" {
			headerMap["role_name"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "カード") || strings.Contains(cleaned, "識別") || strings.Contains(cleaned, "idm") || strings.Contains(cleaned, "nfc") || cleaned == "cardid" {
			headerMap["card_id"] = colIdx
			isHeader = true
		}
	}

	if isHeader {
		startIdx = 1
	}

	var users []db.RegisteredUser
	for i := startIdx; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		getCol := func(key string, fallbackIdx int) string {
			if idx, ok := headerMap[key]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			if !isHeader && fallbackIdx < len(row) {
				return strings.TrimSpace(row[fallbackIdx])
			}
			return ""
		}

		studentNo := getCol("student_no", 0)
		name := getCol("name", 1)
		furigana := getCol("furigana", 2)
		gender := getCol("gender", 3)
		roleStr := getCol("role_name", 4)
		cardID := getCol("card_id", 5)

		if studentNo == "" && name == "" && cardID == "" {
			continue // 空行スキップ
		}

		// 学籍番号がないがカードIDがある場合は学籍番号にカードIDを設定
		if studentNo == "" && cardID != "" {
			studentNo = cardID
		}

		// カードIDが空の場合は学籍番号と同じ値を扱う
		if cardID == "" {
			cardID = studentNo
		}

		if studentNo == "" {
			continue // 学籍番号/カードIDが特定できないものはスキップ
		}

		// 区分・ロールコードの解決
		roleName := "学生"
		roleCode := 1
		roleTrimmed := strings.TrimSpace(roleStr)

		if strings.Contains(roleTrimmed, "教職員") || strings.Contains(roleTrimmed, "教員") || strings.Contains(roleTrimmed, "職員") || roleTrimmed == "0" {
			roleName = "教職員"
			roleCode = 0
		} else if strings.Contains(roleTrimmed, "スタッフ") || roleTrimmed == "9" {
			roleName = "学生スタッフ"
			roleCode = 9
		} else {
			roleName = "学生"
			roleCode = 1
		}

		// 性別の正規化
		genderNorm := ""
		if strings.Contains(gender, "男") || strings.ToLower(gender) == "male" || strings.ToLower(gender) == "m" {
			genderNorm = "男"
		} else if strings.Contains(gender, "女") || strings.ToLower(gender) == "female" || strings.ToLower(gender) == "f" {
			genderNorm = "女"
		} else {
			genderNorm = gender
		}

		users = append(users, db.RegisteredUser{
			CardID:    cardID,
			Name:      name,
			Furigana:  furigana,
			Gender:    genderNorm,
			RoleName:  roleName,
			RoleCode:  roleCode,
			StudentNo: studentNo,
		})
	}

	if len(users) == 0 {
		return 0, len(rows) - startIdx, fmt.Errorf("インポート可能な利用者データが見つかりませんでした")
	}

	count, err := s.dbManager.ImportUsers(users)
	return count, len(users), err
}

// ImportLogsCSV CSVデータから入退室ログを一括インポート
func (s *GateService) ImportLogsCSV(csvData []byte) (int, int, error) {
	cleanData, err := sanitizeCSVData(csvData)
	if err != nil {
		return 0, 0, fmt.Errorf("CSVデータの読み込みに失敗しました: %w", err)
	}

	reader := csv.NewReader(bytes.NewReader(cleanData))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return 0, 0, fmt.Errorf("CSVパースエラー: %w", err)
	}

	if len(rows) == 0 {
		return 0, 0, fmt.Errorf("CSVデータが空です")
	}

	headerMap := make(map[string]int)
	startIdx := 0

	firstRow := rows[0]
	isHeader := false
	for colIdx, colVal := range firstRow {
		cleaned := strings.ToLower(strings.TrimSpace(colVal))
		cleaned = strings.ReplaceAll(cleaned, "_", "")
		cleaned = strings.ReplaceAll(cleaned, " ", "")
		cleaned = strings.ReplaceAll(cleaned, "（", "(")
		cleaned = strings.ReplaceAll(cleaned, "）", ")")

		if strings.Contains(cleaned, "日時") || strings.Contains(cleaned, "時刻") || strings.Contains(cleaned, "timestamp") || strings.Contains(cleaned, "date") || strings.Contains(cleaned, "time") {
			headerMap["timestamp"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "識別") || strings.Contains(cleaned, "カード") || strings.Contains(cleaned, "学籍") || strings.Contains(cleaned, "学生番号") || cleaned == "cardid" || cleaned == "studentno" {
			headerMap["card_id"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "種別") || strings.Contains(cleaned, "イベント") || strings.Contains(cleaned, "ステータス") || cleaned == "eventtype" || cleaned == "type" || cleaned == "status" {
			headerMap["event_type"] = colIdx
			isHeader = true
		} else if strings.Contains(cleaned, "秒") || strings.Contains(cleaned, "duration") {
			headerMap["duration_sec"] = colIdx
			isHeader = true
		}
	}

	if isHeader {
		startIdx = 1
	}

	var logs []db.AccessLog
	timeFormats := []string{
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006/01/02 15:04",
		"2006-01-02 15:04",
	}

	for i := startIdx; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}

		getCol := func(key string, fallbackIdx int) string {
			if idx, ok := headerMap[key]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			if !isHeader && fallbackIdx < len(row) {
				return strings.TrimSpace(row[fallbackIdx])
			}
			return ""
		}

		tsStr := getCol("timestamp", 1)
		cardID := getCol("card_id", 2)
		eventStr := getCol("event_type", 6)
		durStr := getCol("duration_sec", 8)

		if tsStr == "" && cardID == "" {
			continue
		}

		// 日時パース
		var parsedTime time.Time
		for _, layout := range timeFormats {
			if t, err := time.Parse(layout, tsStr); err == nil {
				parsedTime = t
				break
			}
		}
		if parsedTime.IsZero() {
			parsedTime = time.Now()
		}

		// 種別パース
		eventType := "entry"
		eventTrimmed := strings.TrimSpace(eventStr)
		if strings.Contains(eventTrimmed, "強制退室") || eventTrimmed == "force_exit" {
			eventType = "force_exit"
		} else if strings.Contains(eventTrimmed, "退室") || strings.Contains(eventTrimmed, "退") || strings.ToLower(eventTrimmed) == "exit" || strings.ToLower(eventTrimmed) == "out" {
			eventType = "exit"
		} else {
			eventType = "entry"
		}

		var durSec int64 = 0
		if durStr != "" {
			if n, err := strconv.ParseInt(durStr, 10, 64); err == nil {
				durSec = n
			}
		}

		// cardID のユーザー解決（学籍番号またはカードIDから既存カードIDを探す）
		if cardID != "" {
			if u, _ := s.dbManager.GetUser(cardID); u != nil {
				cardID = u.CardID
			}
		}

		logs = append(logs, db.AccessLog{
			CardID:         cardID,
			EventType:      eventType,
			Timestamp:      parsedTime,
			DurationSecond: durSec,
		})
	}

	if len(logs) == 0 {
		return 0, len(rows) - startIdx, fmt.Errorf("インポート可能なログデータが見つかりませんでした")
	}

	count, err := s.dbManager.ImportLogs(logs)
	return count, len(logs), err
}
