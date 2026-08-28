package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"osuccgate/internal/db"
	"osuccgate/internal/nfc"
	"sync"
	"time"
)

// SwipeResponse 打刻結果レスポンス
type SwipeResponse struct {
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	CardID       string `json:"cardId"`
	UserName     string `json:"userName"`
	RoleName     string `json:"roleName"`
	RoleCode     int    `json:"roleCode"`
	StudentNo    string `json:"studentNo"`
	EventType    string `json:"eventType"`    // "entry" or "exit"
	Timestamp    string `json:"timestamp"`    // "15:04:05"
	DurationText string `json:"durationText"` // "1時間30分" (退室時のみ)
	SoundType    string `json:"soundType"`    // "studentEntry", "studentExit", "staffEntry", "staffExit", "booboo"
	IsDebounced  bool   `json:"isDebounced"`  // デバウンスによるスキップ
}

type GateService struct {
	dbManager   *db.DBManager
	lastSwipes  map[string]time.Time
	mu          sync.Mutex
	debounceSec time.Duration
}

func NewGateService(dbMgr *db.DBManager) *GateService {
	return &GateService{
		dbManager:   dbMgr,
		lastSwipes:  make(map[string]time.Time),
		debounceSec: 2 * time.Second, // 要件: 2秒間の待機（受付不可）時間
	}
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

	// 音声演出タイプの決定 (ロール・入退室に応じて)
	soundType := "studentEntry"
	if user.RoleCode == 9 || user.RoleName == "教職員" || user.RoleName == "スタッフ" {
		if logRecord.EventType == "entry" {
			soundType = "staffEntry"
		} else {
			soundType = "staffExit"
		}
	} else {
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
	// ヘッダー行
	headers := []string{"ログID", "打刻日時", "識別ID", "学籍番号/職員番号", "氏名", "区分", "入退室", "滞在時間(秒)", "滞在時間"}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for _, l := range logs {
		eventTypeJa := "入室"
		if l.EventType == "exit" {
			eventTypeJa = "退室"
		}
		row := []string{
			fmt.Sprintf("%d", l.ID),
			l.Timestamp.Format("2006-01-02 15:04:05"),
			l.CardID,
			l.StudentNo,
			l.UserName,
			l.RoleName,
			eventTypeJa,
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
