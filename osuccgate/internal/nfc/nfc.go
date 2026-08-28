package nfc

import (
	"context"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/ebfe/scard"
)

type SwipeCallback func(cardID string)

type NFCReader struct {
	ctx      context.Context
	cancel   context.CancelFunc
	callback SwipeCallback
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
	go r.pollingLoop()
}

// Stop 監視の停止
func (r *NFCReader) Stop() {
	r.cancel()
}

func (r *NFCReader) pollingLoop() {
	var sctx *scard.Context
	var err error

	defer func() {
		if sctx != nil {
			sctx.Release()
		}
	}()

	// 検出済みカードのID（離脱検知用）
	var lastCardID string

	for {
		select {
		case <-r.ctx.Done():
			return
		default:
		}

		// コンテキスト初期化
		isValid := false
		if sctx != nil {
			valid, vErr := sctx.IsValid()
			if vErr == nil && valid {
				isValid = true
			}
		}

		if !isValid {
			if sctx != nil {
				_ = sctx.Release()
				sctx = nil
			}
			sctx, err = scard.EstablishContext()
			if err != nil {
				// スマートカードサービスが起動していない or リーダーなし
				time.Sleep(3 * time.Second)
				continue
			}
		}

		readers, err := sctx.ListReaders()
		if err != nil || len(readers) == 0 {
			// リーダーが未接続
			time.Sleep(2 * time.Second)
			continue
		}

		// 最初のリーダー (PaSoRi等) を使用
		readerName := readers[0]

		// カード接続を試行
		card, err := sctx.Connect(readerName, scard.ShareShared, scard.ProtocolAny)
		if err != nil {
			// カードが置かれていない
			lastCardID = ""
			time.Sleep(300 * time.Millisecond)
			continue
		}

		// カードIDm (UID) 読み取りコマンド (PC/SC Get Data Command for UID / FeliCa IDm)
		// APDU: FF CA 00 00 00
		apdu := []byte{0xFF, 0xCA, 0x00, 0x00, 0x00}
		resp, err := card.Transmit(apdu)
		card.Disconnect(scard.LeaveCard)

		if err != nil || len(resp) < 2 {
			time.Sleep(300 * time.Millisecond)
			continue
		}

		// 正常終了ステータスバイト: 0x90 0x00
		sw1 := resp[len(resp)-2]
		sw2 := resp[len(resp)-1]
		if sw1 == 0x90 && sw2 == 0x00 {
			data := resp[:len(resp)-2]
			idStr := strings.ToUpper(hex.EncodeToString(data))
			if idStr != "" && idStr != lastCardID {
				lastCardID = idStr
				log.Printf("[NFC] Card Detected: %s (Reader: %s)", idStr, readerName)
				if r.callback != nil {
					r.callback(idStr)
				}
			}
		} else {
			lastCardID = ""
		}

		time.Sleep(300 * time.Millisecond)
	}
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
