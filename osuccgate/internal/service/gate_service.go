package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
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
