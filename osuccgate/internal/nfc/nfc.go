package nfc

import (
	"context"
	"encoding/hex"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ebfe/scard"
)

type SwipeCallback func(cardID string)

type NFCReader struct {
	ctx      context.Context
	cancel   context.CancelFunc
	callback SwipeCallback

	mu   sync.Mutex
	sctx *scard.Context
}

// preferredReaderKeywords RC-S380 / PaSoRi を優先選択するためのキーワード一覧
var preferredReaderKeywords = []string{
	"rc-s380",
	"pasori",
	"sony",
	"felica",
}

func NewNFCReader(callback SwipeCallback) *NFCReader {
	ctx, cancel := context.WithCancel(context.Background())
	return &NFCReader{
		ctx:      ctx,
		cancel:   cancel,
		callback: callback,
	}
}

// Start NFCリーダーの監視を開始 (バックグラウンドgoroutine)
func (r *NFCReader) Start() {
	go r.monitorLoop()
}

// Stop 監視の停止（ブロッキング中のGetStatusChangeも即座に解除）
func (r *NFCReader) Stop() {
	r.cancel()
	r.mu.Lock()
	if r.sctx != nil {
		_ = r.sctx.Cancel()
	}
	r.mu.Unlock()
}

// SelectPreferredReader 接続されているリーダー一覧から RC-S380 / PaSoRi を優先選択する
func SelectPreferredReader(readers []string) string {
	if len(readers) == 0 {
		return ""
	}
	for _, kw := range preferredReaderKeywords {
		for _, name := range readers {
			if strings.Contains(strings.ToLower(name), kw) {
				return name
			}
		}
	}
	return readers[0]
}

func (r *NFCReader) setContext(sctx *scard.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sctx = sctx
}

func (r *NFCReader) releaseContext() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sctx != nil {
		_ = r.sctx.Release()
		r.sctx = nil
	}
}

func (r *NFCReader) sleepWithContext(d time.Duration) bool {
	select {
	case <-r.ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

func (r *NFCReader) monitorLoop() {
	defer r.releaseContext()

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		// 1. PC/SC コンテキストの確立・有効性確認
		r.mu.Lock()
		sctx := r.sctx
		r.mu.Unlock()

		isValid := false
		if sctx != nil {
			if valid, err := sctx.IsValid(); err == nil && valid {
				isValid = true
			}
		}

		if !isValid {
			r.releaseContext()
			newCtx, err := scard.EstablishContext()
			if err != nil {
				// PC/SC サービス未起動時などは待機して再試行
				if !r.sleepWithContext(3 * time.Second) {
					return
				}
				continue
			}
			r.setContext(newCtx)
			sctx = newCtx
		}

		// 2. リーダー一覧の取得と RC-S380 / PaSoRi の自動選定
		readers, err := sctx.ListReaders()
		if err != nil || len(readers) == 0 {
			if errors.Is(err, scard.ErrNoService) || errors.Is(err, scard.ErrServiceStopped) || errors.Is(err, scard.ErrInvalidHandle) {
				r.releaseContext()
			}
			if !r.sleepWithContext(2 * time.Second) {
				return
			}
			continue
		}

		readerName := SelectPreferredReader(readers)
		log.Printf("[NFC] Monitoring reader: %s", readerName)

		// 3. 対象リーダーのイベント駆動監視ループ (GetStatusChange)
		r.watchReaderEvents(sctx, readerName)
	}
}

// watchReaderEvents GetStatusChange を用いてカードの載置・離脱イベントを待機する
func (r *NFCReader) watchReaderEvents(sctx *scard.Context, readerName string) {
	states := []scard.ReaderState{
		{
			Reader:       readerName,
			CurrentState: scard.StateUnaware,
		},
	}

	// カードが現在載置済みかどうかの状態フラグ（置いたままの連続打刻を防止）
	cardPresent := false

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		// イベント待機（2秒タイムアウトで定期的にコンテキスト・キャンセル状態を確認）
		err := sctx.GetStatusChange(states, 2*time.Second)
		if err != nil {
			if errors.Is(err, scard.ErrTimeout) {
				continue
			}
			if errors.Is(err, scard.ErrCancelled) && r.ctx.Err() != nil {
				return
			}
			log.Printf("[NFC] Reader status change error (%s): %v. Reconnecting...", readerName, err)
			if errors.Is(err, scard.ErrNoService) || errors.Is(err, scard.ErrServiceStopped) || errors.Is(err, scard.ErrInvalidHandle) {
				r.releaseContext()
			}
			r.sleepWithContext(1 * time.Second)
			return
		}

		eventState := states[0].EventState

		// リーダーが取り外された、または利用不可になった場合は再探索へ戻る
		if eventState&(scard.StateUnavailable|scard.StateUnknown) != 0 {
			log.Printf("[NFC] Reader unavailable or disconnected: %s", readerName)
			r.sleepWithContext(1 * time.Second)
			return
		}

		// カードが離脱した（Empty）場合 -> 再タッチを受け付けられるようにフラグリセット
		if eventState&scard.StateEmpty != 0 {
			cardPresent = false
		}

		// カードが載置された（Present）かつ未処理の場合のみ IDm を読み取る
		if eventState&scard.StatePresent != 0 && eventState&scard.StateMute == 0 {
			if !cardPresent {
				idStr, readErr := r.readCardIDWithRetry(sctx, readerName, 3)
				if readErr == nil && idStr != "" {
					cardPresent = true
					log.Printf("[NFC] Card Detected: %s (Reader: %s)", idStr, readerName)
					if r.callback != nil {
						r.callback(idStr)
					}
				} else if readErr != nil {
					log.Printf("[NFC] Failed to read card UID on %s: %v", readerName, readErr)
				}
			}
		}

		// 次回の変化検知のため、StateChanged ビットを除いた現在の状態を CurrentState に設定
		states[0].CurrentState = eventState &^ scard.StateChanged
	}
}

// readCardIDWithRetry タッチ直後の通信安定性を高めるため小間隔リトライ付きでUID/IDmを取得する
func (r *NFCReader) readCardIDWithRetry(sctx *scard.Context, readerName string, maxAttempts int) (string, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if !r.sleepWithContext(40 * time.Millisecond) {
				return "", r.ctx.Err()
			}
		}

		idStr, err := readCardUID(sctx, readerName)
		if err == nil && idStr != "" {
			return idStr, nil
		}
		lastErr = err
	}
	return "", lastErr
}

// readCardUID PC/SC Get Data Command (FF CA 00 00 00) で FeliCa IDm / NFC UID を読み取る
func readCardUID(sctx *scard.Context, readerName string) (string, error) {
	card, err := sctx.Connect(readerName, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = card.Disconnect(scard.LeaveCard)
	}()

	// PC/SC Get Data Command for UID / FeliCa IDm: FF CA 00 00 00
	apdu := []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}
	resp, err := card.Transmit(apdu)
	if err != nil {
		return "", err
	}
	if len(resp) < 2 {
		return "", errors.New("short APDU response")
	}

	sw1 := resp[len(resp)-2]
	sw2 := resp[len(resp)-1]
	if sw1 != 0x90 || sw2 != 0x00 {
		return "", errors.New("APDU status not 9000")
	}

	data := resp[:len(resp)-2]
	idStr := NormalizeCardID(hex.EncodeToString(data))
	return idStr, nil
}

// NormalizeCardID 入力されたカードID（磁気カードまたはNFC IDm）の正規化
func NormalizeCardID(input string) string {
	cleaned := strings.TrimSpace(input)
	// 磁気カードの改行や制御文字を除去
	cleaned = strings.TrimPrefix(cleaned, ";")
	cleaned = strings.TrimPrefix(cleaned, "%")
	cleaned = strings.TrimSuffix(cleaned, "?")
	cleaned = strings.TrimSuffix(cleaned, "\r")
	cleaned = strings.TrimSuffix(cleaned, "\n")
	cleaned = strings.TrimSpace(cleaned)
	return strings.ToUpper(cleaned)
}
