# 指示
あなたはWails（Go + Svelte）、SQLite、Webフロントエンド開発に精通したエキスパートエンジニアです。
以下の前提条件およびシステム要件に基づき、[入退室管理システム]の全体アーキテクチャ設計書、データベース設計、および動作可能な実装コードを作成してください。

# 開発環境・技術スタック
* **フレームワーク**: Wails
* **フロントエンド**: Svelte, Tailwind CSS, Vite
* **バックエンド**: Go
* **データベース**: SQLite（CGO不要な `glebarez/go-sqlite` または `modernc.org/sqlite` を使用）
* **ベーステンプレート**: 
  https://github.com/PylotLight/wails-vite-svelte-tailwind-template

# 入力デバイス・ハードウェア連携
1. **磁気カードリーダ**: キーボード入力エミュレーション（HIDデバイスとして文字入力を検知・隠しinputへフォーカス）
2. **NFCリーダ**: SONY PaSoRi RC-S380（GoバックエンドからIDm等の固有識別情報を読み取り）

# システム仕様・制御ロジック
1. **入退室自動判定**:
   * 同一ユーザーの打刻回数に基づき判定（1回目/奇数回＝「入室」、2回目/偶数回＝「退室」）。
2. **連続読み込み防止（デバウンス制御）**:
   * 同一デバイス・同一ユーザーからの誤連打を防ぐため、打刻処理完了後に **2秒間の待機（受付不可）時間** を設ける。

# データベース設計（SQLite）
1. **利用登録者テーブル (`registered_users`)**:
   * 識別ID（磁気カード番号 / NFC IDm）、氏名、ロール/区分（学生/教職員/スタッフ等）、ロールコード（1:学生, 9:教職員 等）、登録日時など。
2. **入退室ログテーブル (`access_logs`)**:
   * ログID、ユーザーID、区分（入室/退室）、打刻日時、滞在時間（退室時に算出・記録）。
   * 各ユーザーの「現在の入退室ステータス（在室中/退室済）」および「現在の滞在時間」を即座に集計可能な構造。

# 画面要件 1：メイン打刻画面（ミニPC用）
※以下の通り提供されたHTML/CSS/JSのデザイン・レスポンシブ配置・音響演出ロジックを極力変更せずにSvelteコンポーネントへ移植してください。（`google.script.run` は Wails バインディング通信 `App.ProcessSwipe(id)` 等へ置換）

* **デザイン・UI構造**:
  * カード風大画面UI、中央配置、フォントアニメーション、ステータスカラー区分（`type-entry`: 青, `type-exit`: オレンジ, `type-error`: 赤）。
  * 常に非表示の `<input id="swipeInput">` にフォーカスを維持し、画面どこをクリックしても `focusInput()` が動く仕様。
* **音声演出 (Web Audio API)**:
  * サンプル内の `sounds` 関数（Web Audio APIによる効果音生成ロジック）を保持。
  * `roleCode` や入退室（`entryExit`）に応じた効果音の分岐再生（`studentEntry`, `studentExit`, `staffEntry`, `staffExit`, `booboo` など）。
* **表示情報**:
  * 正常打刻時: ロール名バッジ、氏名（「◯◯ 様」）、ステータス（「入室」または「退室」）、打刻時刻、退室時は「滞在時間」。
  * 読み込み完了後、音に応じた待機時間の経過後に「学生証を通してください（読み取り待機中...）」の初期状態にリセット。

--- サンプルHTML (移植元) ---
<!DOCTYPE html>
<html>
<head>
<style>
html, body { height: 100%; margin: 0; padding: 0; overflow: hidden; background-color: #f4f7f9; }
body { font-family: 'Helvetica Neue', Arial, sans-serif; display: flex; justify-content: center; align-items: center; cursor: pointer; }
.card { background: white; width: 95%; max-width: 800px; padding: 60px; border-radius: 40px; box-shadow: 0 20px 50px rgba(0,0,0,0.15); text-align: center; }
#display { min-height: 500px; display: flex; flex-direction: column; justify-content: center; align-items: center; }
.title-text { color: #2c3e50; margin: 0 0 30px 0; font-size: 2.5em; font-weight: bold; }
.role-badge { background: #2c3e50; color: white; padding: 10px 30px; border-radius: 50px; font-size: 1.6em; margin-bottom: 20px; }
.user-name { font-size: 3.5em; font-weight: bold; color: #333; margin: 15px 0; }
.status-box { font-size: 6em; font-weight: 900; padding: 25px 0; border-radius: 30px; margin: 25px 0; width: 100%; letter-spacing: 5px; }
.info-text { font-size: 2em; color: #2c3e50; font-weight: bold; margin-top: 10px; }
.type-entry { background-color: #3498db; color: white; }
.type-exit { background-color: #e67e22; color: white; }
.type-error { background-color: #e74c3c; color: white; }
.loading-text { font-size: 3.5em; color: #34495e; font-weight: bold; animation: blink 0.8s infinite; }
@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0.2; } }
#swipeInput { position: absolute; left: -9999px; opacity: 0; }
</style>
</head>
<body onclick="focusInput();">
<div class="card">
  <div class="title-text">入退室管理システム</div>
  <input type="text" id="swipeInput" autofocus autocomplete="off">
  <div id="display">
    <div id="content">
      <div style="font-size: 2.8em; color: #34495e; font-weight: bold;">学生証を通してください</div>
      <div style="font-size: 1.6em; color: #bdc3c7; margin-top: 25px;">読み取り待機中...</div>
    </div>
  </div>
</div>
</body>
</html>

# 画面要件 2：管理者画面（同一LAN内の他PCからブラウザアクセス可能）
* **ネットワーク構成**: 
  * Wailsアプリ内部でHTTPサーバー（例: `:8080`）を構築、またはWailsのWeb Server機能を有効化し、同一ローカルPCおよび同一LAN内の他PCのブラウザからアクセス可能にする。
* **簡易セキュリティ**:
  * 管理者画面アクセス時に4桁PINまたはパスワード入力を求めるダイアログ/モーダルを配置（アクセス制限のハードルを設定）。
* **ダッシュボード統計表示**:
  * 現在の在室者数
  * ユーザー登録総数
  * 本日の総打刻数
* **データ管理機能**:
  * **新規利用者登録 & 編集**: カード識別ID、氏名、学籍番号/教職員区分等のCRUD機能。
  * **入退室ログ一覧**: 学籍番号/区分、現在のステータス（入室中/退室済）、最新打刻時刻、現在の滞在時間（滞在中含む）のリアルタイム表示。
* **高度な操作（破壊的操作＆エクスポート）**:
  * 「高度な操作」ボタンを配置。
  * クリック時に警告モーダルを表示：「システムの管理者及びはリーダーに了承を得て操作を行ってください」（継続 / 中止 ボタン）。
  * 中止時は元画面へ復帰。継続時のみ以下の操作を解禁：
    1. **登録者の削除**
    2. **入退室ログの削除**
    3. **yyyy年度の入退室ログ出力**: 指定した年度（例: 2026年度）のログデータをCSVまたはXLSX形式でダウンロード/エクスポートする機能。

# 求める出力内容
1. **システム全体構成およびデータフロー**（入力〜デバウンス〜SQLite〜メイン画面＆Web管理者画面への共有）
2. **SQLiteデータベース設計**（テーブル定義 DDL および Go用構造体）
3. **Goバックエンド実装コード**
   * SQLite接続・CRUD処理・年度別ログCSV/XLSX生成ロジック
   * NFC (PaSoRi RC-S380) / 磁気入力のバインディング
   * 2秒デバウンス制御および奇数/偶数入退室判定ロジック
   * ローカルLAN向けWebサーバー（管理者画面用HTTP API）の実装
4. **Svelteフロントエンド実装コード**
   * サンプルHTML/CSS/JSをベースとしたメイン打刻画面コンポーネント
   * PIN認証付きWeb管理者画面ダッシュボード＆「高度な操作」モーダルコンポーネント